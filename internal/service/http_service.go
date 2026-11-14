package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"minipost/internal/model"
	appErrors "minipost/internal/pkg/errors"
	"minipost/internal/pkg/httputil"
	"minipost/internal/pkg/logger"
)

type HttpService struct {
	client     *http.Client
	inflightMu sync.Mutex
	inflight   map[string]context.CancelFunc
	// transportMu 保护 transports 缓存。
	transportMu sync.Mutex
	// transports 按 (sslVerify, httpVersion) 维度缓存 *http.Transport，
	// 以便跨请求复用连接池（keep-alive），避免每次请求都新建 transport 导致重复握手与空闲连接堆积。
	transports map[string]*http.Transport
	// keyLogPath 是 TLS 会话密钥日志（NSS Key Log 格式）的写入路径，空表示不支持。
	keyLogMu   sync.Mutex
	keyLogPath string
	keyLogFile *os.File
}

type requestOptions struct {
	followRedirects         bool
	timeout                 time.Duration
	maxResponseBytes        int64
	sslVerify               bool
	sslKeyLog               bool
	httpVersion             string
	disableDefaultUserAgent bool
	disableDefaultAccept    bool
	disableAutoContentType  bool
}

func NewHttpService() *HttpService {
	return &HttpService{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		inflight:   make(map[string]context.CancelFunc),
		transports: make(map[string]*http.Transport),
	}
}

// SetKeyLogPath 设置 TLS 会话密钥日志文件路径；设置里开启「SSL/TLS Key Log」后，握手密钥会追加写入该文件，
// 可供 Wireshark 等工具解密抓包。环境变量 SSLKEYLOGFILE 优先于此路径。
func (s *HttpService) SetKeyLogPath(path string) {
	s.keyLogMu.Lock()
	defer s.keyLogMu.Unlock()
	s.keyLogPath = path
}

// KeyLogPath 返回当前生效的密钥日志路径（可能为空）。
func (s *HttpService) KeyLogPath() string {
	s.keyLogMu.Lock()
	defer s.keyLogMu.Unlock()
	if p := strings.TrimSpace(os.Getenv("SSLKEYLOGFILE")); p != "" {
		return p
	}
	return s.keyLogPath
}

// keyLogWriter 惰性打开密钥日志文件并返回写入器；路径为空或打开失败时返回 nil。
func (s *HttpService) keyLogWriter() io.Writer {
	path := s.KeyLogPath()
	if path == "" {
		return nil
	}
	s.keyLogMu.Lock()
	defer s.keyLogMu.Unlock()
	if s.keyLogFile != nil && s.keyLogFile.Name() == path {
		return s.keyLogFile
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		logger.Warn("创建 TLS 密钥日志目录失败", "error", err.Error())
		return nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		logger.Warn("打开 TLS 密钥日志失败", "error", err.Error())
		return nil
	}
	if s.keyLogFile != nil {
		_ = s.keyLogFile.Close()
	}
	s.keyLogFile = f
	logger.Info("已启用 TLS 密钥日志", "path", path)
	return f
}

func (s *HttpService) registerInflight(requestID string, cancel context.CancelFunc) {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" || cancel == nil {
		return
	}
	s.inflightMu.Lock()
	defer s.inflightMu.Unlock()
	if s.inflight == nil {
		s.inflight = make(map[string]context.CancelFunc)
	}
	s.inflight[requestID] = cancel
}

func (s *HttpService) unregisterInflight(requestID string) {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return
	}
	s.inflightMu.Lock()
	defer s.inflightMu.Unlock()
	delete(s.inflight, requestID)
}

// CancelRequest 取消仍在进行中的 HTTP 请求。
func (s *HttpService) CancelRequest(requestID string) bool {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return false
	}
	s.inflightMu.Lock()
	cancel, ok := s.inflight[requestID]
	if ok {
		delete(s.inflight, requestID)
	}
	s.inflightMu.Unlock()
	if !ok || cancel == nil {
		return false
	}
	cancel()
	logger.Info("HTTP 请求已取消", "requestID", requestID)
	return true
}

// SendRequest 执行 HTTP 请求并返回响应
func (s *HttpService) SendRequest(input model.SendRequestInput) (*model.HttpResponse, error) {
	return s.sendRequest(input, nil)
}

// SendRequestStreaming 执行 HTTP 请求并在读取响应体时实时回调 chunk
func (s *HttpService) SendRequestStreaming(input model.SendRequestInput, onChunk func(model.StreamChunk)) (*model.HttpResponse, error) {
	return s.sendRequest(input, onChunk)
}

func (s *HttpService) sendRequest(input model.SendRequestInput, onChunk func(model.StreamChunk)) (*model.HttpResponse, error) {
	requestID := newHTTPRequestLogID()
	originalTarget := logger.RedactRequestTarget(input.URL)
	logger.Debug("HTTP 请求输入开始",
		"requestID", requestID,
		"method", input.Method,
		"target", originalTarget,
		"paramCount", len(input.Params),
		"headerCount", len(input.Headers),
		"bodyType", input.Body.Type,
		"streaming", onChunk != nil,
	)

	normalizedInput, err := s.normalizeInput(input)
	if err != nil {
		logger.Warn("HTTP 请求输入规范化失败", "requestID", requestID, "target", originalTarget, "error", err.Error())
		return nil, err
	}
	if normalizedInput.URL != input.URL {
		logger.Info("HTTP 请求已从 cURL 解析", "requestID", requestID, "method", normalizedInput.Method, "target", logger.RedactRequestTarget(normalizedInput.URL))
	}
	input = normalizedInput
	options := s.extractRequestOptions(input.Options)
	logger.Debug("HTTP 请求选项已解析",
		"requestID", requestID,
		"followRedirects", options.followRedirects,
		"timeoutMs", options.timeout.Milliseconds(),
		"maxResponseBytes", options.maxResponseBytes,
		"sslVerify", options.sslVerify,
		"httpVersion", options.httpVersion,
		"disableDefaultUserAgent", options.disableDefaultUserAgent,
		"disableDefaultAccept", options.disableDefaultAccept,
		"disableAutoContentType", options.disableAutoContentType,
	)

	reqURL, err := s.buildURL(input.URL, input.Params)
	if err != nil {
		logger.Warn("HTTP 请求 URL 构建失败", "requestID", requestID, "target", logger.RedactRequestTarget(input.URL), "error", err.Error())
		return nil, appErrors.Wrap("INVALID_URL", "URL 解析失败", err)
	}
	logURL := logger.RedactURL(reqURL)

	bodyReader, contentType, err := s.buildBody(input.Body)
	if err != nil {
		logger.Warn("HTTP 请求体构建失败", "requestID", requestID, "method", input.Method, "url", logURL, "bodyType", input.Body.Type, "error", err.Error())
		return nil, appErrors.Wrap("INVALID_BODY", "请求体构建失败", err)
	}
	requestBodyBytes := estimateRequestBodyBytes(input.Body)
	logger.Debug("HTTP 请求体已构建",
		"requestID", requestID,
		"bodyType", input.Body.Type,
		"requestBodyBytes", requestBodyBytes,
		"autoContentType", contentType != "",
	)

	req, err := http.NewRequest(input.Method, reqURL, bodyReader)
	if err != nil {
		logger.Warn("HTTP 请求对象构建失败", "requestID", requestID, "method", input.Method, "url", logURL, "error", err.Error())
		return nil, appErrors.Wrap("REQUEST_BUILD_FAILED", "请求构建失败", err)
	}

	requestCtx := req.Context()
	if strings.TrimSpace(input.RequestID) != "" {
		var cancel context.CancelFunc
		requestCtx, cancel = context.WithCancel(requestCtx)
		s.registerInflight(input.RequestID, cancel)
		defer func() {
			cancel()
			s.unregisterInflight(input.RequestID)
		}()
		req = req.WithContext(requestCtx)
	}

	// 设置 Content-Type（仅在有 body 时）
	if contentType != "" && !options.disableAutoContentType {
		req.Header.Set("Content-Type", contentType)
	}

	// 设置自定义 headers
	for _, h := range input.Headers {
		key, value := strings.TrimSpace(h.Key), strings.TrimSpace(h.Value)
		switch {
		case key == "":
			continue
		case strings.EqualFold(key, "Host"):
			// Go 会忽略 Header 里的 Host，必须设置在 req.Host 上才会真正发出。
			req.Host = value
			logger.Debug("已改写 Host 请求头", "requestID", requestID, "host", value)
		default:
			req.Header.Set(key, value)
		}
	}

	// 设置认证
	s.applyAuth(req, input.Auth)

	// 设置默认 User-Agent
	if !options.disableDefaultUserAgent && req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "MiniPost/1.0")
	}
	// 与 Postman 一致：未指定时补默认 Accept
	if !options.disableDefaultAccept && req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "*/*")
	}

	client := s.buildClient(options)
	trace := requestTrace{}
	clientTrace := &httptrace.ClientTrace{
		GetConn: func(_ string) {
			if trace.getConn.IsZero() {
				trace.getConn = time.Now()
			}
		},
		GotConn: func(info httptrace.GotConnInfo) {
			if trace.gotConn.IsZero() {
				trace.gotConn = time.Now()
			}
			if info.Conn != nil {
				if trace.localAddr == "" {
					trace.localAddr = normalizeSocketAddress(info.Conn.LocalAddr())
				}
				if trace.remoteAddr == "" {
					trace.remoteAddr = normalizeSocketAddress(info.Conn.RemoteAddr())
				}
			}
		},
		DNSStart: func(_ httptrace.DNSStartInfo) {
			if trace.dnsStart.IsZero() {
				trace.dnsStart = time.Now()
			}
		},
		DNSDone: func(_ httptrace.DNSDoneInfo) {
			if trace.dnsDone.IsZero() {
				trace.dnsDone = time.Now()
			}
		},
		ConnectStart: func(_, _ string) {
			if trace.connectStart.IsZero() {
				trace.connectStart = time.Now()
			}
		},
		ConnectDone: func(_, _ string, _ error) {
			if trace.connectDone.IsZero() {
				trace.connectDone = time.Now()
			}
		},
		TLSHandshakeStart: func() {
			if trace.tlsStart.IsZero() {
				trace.tlsStart = time.Now()
			}
		},
		TLSHandshakeDone: func(_ tls.ConnectionState, _ error) {
			if trace.tlsDone.IsZero() {
				trace.tlsDone = time.Now()
			}
		},
		WroteRequest: func(_ httptrace.WroteRequestInfo) {
			if trace.wroteRequest.IsZero() {
				trace.wroteRequest = time.Now()
			}
		},
		GotFirstResponseByte: func() {
			if trace.firstByte.IsZero() {
				trace.firstByte = time.Now()
			}
		},
	}

	start := time.Now()
	trace.requestStart = start
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), clientTrace))
	logger.Info("HTTP 请求发送开始",
		"requestID", requestID,
		"method", req.Method,
		"url", logURL,
		"headerCount", len(req.Header),
		"requestBodyBytes", requestBodyBytes,
		"streaming", onChunk != nil,
	)
	resp, err := client.Do(req)

	if err != nil {
		if requestCtx.Err() != nil {
			logger.Info("HTTP 请求已取消", "requestID", requestID, "method", req.Method, "url", logURL)
			return nil, &appErrors.AppError{
				Code:    "REQUEST_CANCELLED",
				Message: "请求已取消",
			}
		}
		code, message, detail := classifyRequestSendError(err, reqURL)
		logger.Warn("HTTP 请求发送失败",
			"requestID", requestID,
			"method", req.Method,
			"url", logURL,
			"code", code,
			"message", message,
			"error", err.Error(),
		)
		return nil, &appErrors.AppError{
			Code:    code,
			Message: message,
			Detail:  detail,
		}
	}
	defer resp.Body.Close()
	responseHeaderBytes := estimateResponseHeaderBytes(resp)
	network := buildNetworkDetails(resp, trace)
	logger.Info("HTTP 响应头已收到",
		"requestID", requestID,
		"status", resp.StatusCode,
		"protocol", resp.Proto,
		"contentType", resp.Header.Get("Content-Type"),
		"responseHeaderBytes", responseHeaderBytes,
		"remoteAddress", networkValue(network, "remoteAddress"),
		"tlsProtocol", networkValue(network, "tlsProtocol"),
	)

	if onChunk != nil {
		type responseStartPayload struct {
			StatusCode  int                   `json:"statusCode"`
			StatusText  string                `json:"statusText"`
			Headers     map[string][]string   `json:"headers"`
			ContentType string                `json:"contentType"`
			Protocol    string                `json:"protocol,omitempty"`
			Network     *model.NetworkDetails `json:"network,omitempty"`
			HeaderBytes int64                 `json:"headerBytes"`
		}
		startPayload := responseStartPayload{
			StatusCode:  resp.StatusCode,
			StatusText:  http.StatusText(resp.StatusCode),
			Headers:     resp.Header,
			ContentType: resp.Header.Get("Content-Type"),
			Protocol:    resp.Proto,
			Network:     network,
			HeaderBytes: responseHeaderBytes,
		}
		encoded, _ := json.Marshal(startPayload)
		onChunk(model.StreamChunk{
			Kind:       "response_start",
			Data:       string(encoded),
			Timestamp:  time.Now().Format(time.RFC3339Nano),
			Sequence:   1,
			BytesTotal: responseHeaderBytes,
		})
		logger.Debug("HTTP 流式响应开始", "requestID", requestID, "status", resp.StatusCode, "contentType", resp.Header.Get("Content-Type"))
	}

	wrappedOnChunk := onChunk
	if onChunk != nil {
		wrappedOnChunk = func(chunk model.StreamChunk) {
			chunk.Sequence += 1
			chunk.BytesTotal += responseHeaderBytes
			onChunk(chunk)
		}
	}
	bodyBytes, err := s.readResponseBody(resp.Body, options.maxResponseBytes, resp.Header.Get("Content-Type"), wrappedOnChunk)
	if err != nil {
		if requestCtx.Err() != nil {
			logger.Info("HTTP 响应读取已取消", "requestID", requestID, "status", resp.StatusCode)
			return nil, &appErrors.AppError{
				Code:    "REQUEST_CANCELLED",
				Message: "请求已取消",
			}
		}
		logger.Warn("HTTP 响应体读取失败",
			"requestID", requestID,
			"status", resp.StatusCode,
			"contentType", resp.Header.Get("Content-Type"),
			"maxResponseBytes", options.maxResponseBytes,
			"error", err.Error(),
		)
		return nil, err
	}
	bodyDone := time.Now()

	respContentType := resp.Header.Get("Content-Type")
	respContentDisposition := resp.Header.Get("Content-Disposition")
	bodyIsBinary := shouldTreatResponseAsBinary(respContentType, respContentDisposition, bodyBytes)
	bodyText := string(bodyBytes)
	bodyBase64 := ""
	if bodyIsBinary {
		bodyBase64 = base64.StdEncoding.EncodeToString(bodyBytes)
		bodyText = ""
	}
	warnings := []string{}
	if !options.sslVerify {
		if warning := detectTLSWarning(reqURL, resp.TLS); warning != "" {
			warnings = append(warnings, warning)
		}
	}
	done := time.Now()

	// 使用响应协议信息估算请求头大小，便于与 Postman 的展示口径保持接近。
	requestHeaderBytes := estimateRequestHeaderBytes(req, resp.Proto)
	responseBodyBytes := int64(len(bodyBytes))
	sizeDetails := model.SizeBreakdown{
		ResponseHeaders: responseHeaderBytes,
		ResponseBody:    responseBodyBytes,
		ResponseTotal:   responseHeaderBytes + responseBodyBytes,
		RequestHeaders:  requestHeaderBytes,
		RequestBody:     requestBodyBytes,
		RequestTotal:    requestHeaderBytes + requestBodyBytes,
	}
	timings := buildTimingBreakdown(trace, bodyDone, done)
	if timings.Total <= 0 {
		timings.Total = float64(done.Sub(start)) / float64(time.Millisecond)
	}
	logger.Info("HTTP 响应读取完成",
		"requestID", requestID,
		"status", resp.StatusCode,
		"durationMs", timings.Total,
		"responseBodyBytes", responseBodyBytes,
		"responseTotalBytes", sizeDetails.ResponseTotal,
		"requestTotalBytes", sizeDetails.RequestTotal,
		"bodyIsBinary", bodyIsBinary,
		"warningCount", len(warnings),
		"dnsMs", timings.DNSLookup,
		"tcpMs", timings.TCPHandshake,
		"tlsMs", timings.SSLHandshake,
		"ttfbMs", timings.WaitingTTFB,
		"downloadMs", timings.Download,
	)

	return &model.HttpResponse{
		StatusCode:   resp.StatusCode,
		StatusText:   http.StatusText(resp.StatusCode),
		Headers:      resp.Header,
		Body:         bodyText,
		BodyBase64:   bodyBase64,
		BodyIsBinary: bodyIsBinary,
		Duration:     timings.Total,
		Size:         sizeDetails.ResponseTotal,
		ContentType:  respContentType,
		Protocol:     resp.Proto,
		Warnings:     warnings,
		Network:      network,
		Timings:      timings,
		SizeDetails:  sizeDetails,
	}, nil
}

func newHTTPRequestLogID() string {
	return fmt.Sprintf("http-%x", time.Now().UnixNano())
}

func (s *HttpService) buildURL(rawURL string, params []model.KeyValue) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if lower := strings.ToLower(rawURL); !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		rawURL = "http://" + rawURL
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	// 地址里已有的查询串原样保留（不排序、不重新编码），Params 按顺序追加在后面。
	u.RawQuery = appendQuery(u.RawQuery, params)

	return u.String(), nil
}

func (s *HttpService) buildBody(body model.RequestBody) (io.Reader, string, error) {
	switch body.Type {
	case "none":
		return nil, "", nil

	case "json":
		if body.JSON == "" {
			return nil, "", nil
		}
		return strings.NewReader(body.JSON), "application/json", nil

	case "raw":
		if body.Raw == "" {
			return nil, "", nil
		}
		return strings.NewReader(body.Raw), "text/plain", nil

	case "form-urlencoded":
		// 按行顺序编码，不排序；没有键名的行跳过。
		return strings.NewReader(encodePairs(body.FormUrlEncoded)), "application/x-www-form-urlencoded", nil

	case "form-data":
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)

		for _, field := range body.FormData {
			key := strings.TrimSpace(field.Key)
			if key == "" {
				continue
			}
			if strings.EqualFold(field.Type, "file") {
				filePath := strings.TrimSpace(field.FilePath)
				if filePath == "" {
					filePath = strings.TrimSpace(field.Value)
				}
				if filePath == "" {
					continue
				}
				file, err := os.Open(filePath)
				if err != nil {
					return nil, "", appErrors.Wrap("INVALID_FORM_DATA_FILE", fmt.Sprintf("读取 form-data 文件失败: %s", filePath), err)
				}
				part, err := writer.CreateFormFile(key, filepath.Base(filePath))
				if err != nil {
					_ = file.Close()
					return nil, "", appErrors.Wrap("INVALID_FORM_DATA_FILE", "构建 form-data 文件字段失败", err)
				}
				if _, err := io.Copy(part, file); err != nil {
					_ = file.Close()
					return nil, "", appErrors.Wrap("INVALID_FORM_DATA_FILE", "写入 form-data 文件内容失败", err)
				}
				_ = file.Close()
				continue
			}

			if err := writer.WriteField(key, field.Value); err != nil {
				return nil, "", appErrors.Wrap("INVALID_FORM_DATA", "构建 form-data 文本字段失败", err)
			}
		}

		if err := writer.Close(); err != nil {
			return nil, "", appErrors.Wrap("INVALID_FORM_DATA", "构建 form-data 失败", err)
		}
		return &buf, writer.FormDataContentType(), nil

	default:
		return nil, "", nil
	}
}

func (s *HttpService) normalizeInput(input model.SendRequestInput) (model.SendRequestInput, error) {
	fields := strings.Fields(strings.TrimSpace(input.URL))
	if len(fields) == 0 || !strings.EqualFold(fields[0], "curl") {
		return input, nil
	}

	parsed, err := httputil.ParseCurlCommand(input.URL)
	if err != nil {
		return input, appErrors.Wrap("INVALID_CURL", "cURL 命令解析失败", err)
	}
	parsed.Options = input.Options
	return *parsed, nil
}

func (s *HttpService) extractRequestOptions(inputOptions *model.RequestOptions) requestOptions {
	options := requestOptions{
		followRedirects:         true,
		timeout:                 s.client.Timeout,
		maxResponseBytes:        0,
		sslVerify:               true,
		httpVersion:             "auto",
		disableDefaultUserAgent: false,
		disableDefaultAccept:    false,
		disableAutoContentType:  false,
	}

	if inputOptions == nil {
		return options
	}

	options.followRedirects = inputOptions.FollowRedirects
	if inputOptions.TimeoutMS <= 0 {
		options.timeout = 0
	} else {
		options.timeout = time.Duration(inputOptions.TimeoutMS) * time.Millisecond
	}
	if inputOptions.MaxResponseSizeMB <= 0 {
		options.maxResponseBytes = 0
	} else {
		options.maxResponseBytes = int64(inputOptions.MaxResponseSizeMB) * 1024 * 1024
	}
	options.sslVerify = inputOptions.SSLVerify
	options.sslKeyLog = inputOptions.SSLTLSKeyLog
	switch strings.ToLower(strings.TrimSpace(inputOptions.HTTPVersion)) {
	case "http1", "http2":
		options.httpVersion = strings.ToLower(strings.TrimSpace(inputOptions.HTTPVersion))
	default:
		options.httpVersion = "auto"
	}
	options.disableDefaultUserAgent = inputOptions.DisableDefaultUserAgent
	options.disableDefaultAccept = inputOptions.DisableDefaultAccept
	options.disableAutoContentType = inputOptions.DisableAutoContentType

	return options
}

func (s *HttpService) buildClient(options requestOptions) *http.Client {
	client := *s.client
	client.Timeout = options.timeout

	if options.followRedirects {
		client.CheckRedirect = nil
	} else {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	// 复用按 (sslVerify, httpVersion) 缓存的 transport，保证连接池跨请求生效。
	client.Transport = s.transportFor(options)
	return &client
}

// transportFor 返回与给定选项匹配的 *http.Transport，命中缓存则直接复用，否则构建后缓存。
// 单个 *http.Transport 可并发安全使用，缓存后可在多次请求间复用连接池。
func (s *HttpService) transportFor(options requestOptions) *http.Transport {
	keyLog := options.sslKeyLog && s.keyLogWriter() != nil
	key := fmt.Sprintf("ssl=%t|ver=%s|keylog=%t", options.sslVerify, options.httpVersion, keyLog)

	s.transportMu.Lock()
	defer s.transportMu.Unlock()

	if s.transports == nil {
		s.transports = make(map[string]*http.Transport)
	}
	if cached, ok := s.transports[key]; ok && cached != nil {
		return cached
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if !options.sslVerify || keyLog {
		if !options.sslVerify {
			// 只在用户明确关闭「SSL 证书验证」时才跳过校验；这里不记录目标地址，避免泄露内网信息。
			logger.Warn("已关闭 SSL 证书验证，本次连接不校验服务端证书")
		}
		cfg := &tls.Config{InsecureSkipVerify: !options.sslVerify}
		if keyLog {
			cfg.KeyLogWriter = s.keyLogWriter()
		}
		transport.TLSClientConfig = cfg
	}
	switch options.httpVersion {
	case "http1":
		transport.ForceAttemptHTTP2 = false
	case "http2":
		transport.ForceAttemptHTTP2 = true
	}

	s.transports[key] = transport
	return transport
}

func (s *HttpService) applyAuth(req *http.Request, auth model.AuthConfig) {
	switch auth.Type {
	case "basic":
		// 账号与密码都为空时不附加 Authorization，避免发送无意义的空凭证（Basic Og==）。
		if auth.Basic.Username == "" && auth.Basic.Password == "" {
			return
		}
		encoded := base64.StdEncoding.EncodeToString(
			[]byte(fmt.Sprintf("%s:%s", auth.Basic.Username, auth.Basic.Password)),
		)
		req.Header.Set("Authorization", "Basic "+encoded)

	case "bearer":
		if auth.Bearer.Token != "" {
			req.Header.Set("Authorization", "Bearer "+auth.Bearer.Token)
		}

	case "api-key":
		if auth.APIKey.Key != "" && auth.APIKey.Value != "" {
			if auth.APIKey.AddTo == "query" {
				req.URL.RawQuery = setQueryParam(req.URL.RawQuery, auth.APIKey.Key, auth.APIKey.Value)
			} else {
				req.Header.Set(auth.APIKey.Key, auth.APIKey.Value)
			}
		}
	}
}
