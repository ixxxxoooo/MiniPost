package service

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"minipost/internal/model"
	appErrors "minipost/internal/pkg/errors"
	"minipost/internal/pkg/httputil"
)

type HttpService struct {
	client *http.Client
}

type requestOptions struct {
	followRedirects         bool
	timeout                 time.Duration
	maxResponseBytes        int64
	sslVerify               bool
	httpVersion             string
	disableDefaultUserAgent bool
	disableDefaultAccept    bool
	disableAutoContentType  bool
}

type requestTrace struct {
	requestStart time.Time
	getConn      time.Time
	gotConn      time.Time
	localAddr    string
	remoteAddr   string
	dnsStart     time.Time
	dnsDone      time.Time
	connectStart time.Time
	connectDone  time.Time
	tlsStart     time.Time
	tlsDone      time.Time
	wroteRequest time.Time
	firstByte    time.Time
}

func msBetween(start time.Time, end time.Time) float64 {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return 0
	}
	return float64(end.Sub(start)) / float64(time.Millisecond)
}

func firstNonZero(values ...time.Time) time.Time {
	for _, value := range values {
		if !value.IsZero() {
			return value
		}
	}
	return time.Time{}
}

func estimateRequestBodyBytes(body model.RequestBody) int64 {
	switch body.Type {
	case "json":
		return int64(len(body.JSON))
	case "raw":
		return int64(len(body.Raw))
	case "form-urlencoded":
		values := url.Values{}
		for _, kv := range body.FormUrlEncoded {
			if kv.Key == "" {
				continue
			}
			values.Add(kv.Key, kv.Value)
		}
		return int64(len(values.Encode()))
	case "form-data":
		total := 0
		for _, field := range body.FormData {
			total += len(field.Key) + len(field.Value)
			if strings.EqualFold(field.Type, "file") {
				filePath := strings.TrimSpace(field.FilePath)
				if filePath == "" {
					filePath = strings.TrimSpace(field.Value)
				}
				if filePath != "" {
					if stat, err := os.Stat(filePath); err == nil {
						total += int(stat.Size())
					}
				}
			}
		}
		return int64(total)
	default:
		return 0
	}
}

func estimateRequestHeaderBytes(req *http.Request, proto string) int64 {
	if proto == "" {
		proto = "HTTP/1.1"
	}

	size := int64(len(fmt.Sprintf("%s %s %s\r\n", req.Method, req.URL.RequestURI(), proto)))
	hostValue := req.Host
	if hostValue == "" {
		hostValue = req.URL.Host
	}
	hostAlreadyPresent := false
	for key, values := range req.Header {
		if strings.EqualFold(key, "host") {
			hostAlreadyPresent = true
		}
		for _, value := range values {
			size += int64(len(key) + 2 + len(value) + 2)
		}
	}

	if hostValue != "" && !hostAlreadyPresent {
		size += int64(len("Host") + 2 + len(hostValue) + 2)
	}
	size += 2 // 头结束空行
	return size
}

func estimateResponseHeaderBytes(resp *http.Response) int64 {
	proto := resp.Proto
	if proto == "" {
		proto = "HTTP/1.1"
	}
	statusLine := resp.Status
	if statusLine == "" {
		statusLine = fmt.Sprintf("%d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	size := int64(len(fmt.Sprintf("%s %s\r\n", proto, statusLine)))
	for key, values := range resp.Header {
		for _, value := range values {
			size += int64(len(key) + 2 + len(value) + 2)
		}
	}
	size += 2
	return size
}

func buildTimingBreakdown(trace requestTrace, bodyDone time.Time, done time.Time) model.TimingBreakdown {
	prepareEnd := firstNonZero(trace.getConn, trace.dnsStart, trace.connectStart, trace.wroteRequest, trace.firstByte, bodyDone)
	socketEnd := firstNonZero(trace.dnsStart, trace.connectStart, trace.gotConn, trace.wroteRequest, trace.firstByte, bodyDone)
	waitingStart := firstNonZero(trace.wroteRequest, trace.gotConn, trace.connectDone, trace.dnsDone, trace.getConn, trace.requestStart)
	downloadStart := firstNonZero(trace.firstByte, waitingStart)

	return model.TimingBreakdown{
		Prepare:              msBetween(trace.requestStart, prepareEnd),
		SocketInitialization: msBetween(trace.getConn, socketEnd),
		DNSLookup:            msBetween(trace.dnsStart, trace.dnsDone),
		TCPHandshake:         msBetween(trace.connectStart, trace.connectDone),
		SSLHandshake:         msBetween(trace.tlsStart, trace.tlsDone),
		WaitingTTFB:          msBetween(waitingStart, trace.firstByte),
		Download:             msBetween(downloadStart, bodyDone),
		Process:              msBetween(bodyDone, done),
		Total:                msBetween(trace.requestStart, done),
	}
}

func NewHttpService() *HttpService {
	return &HttpService{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
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
	normalizedInput, err := s.normalizeInput(input)
	if err != nil {
		return nil, err
	}
	input = normalizedInput
	options, headers := s.extractRequestOptions(input.Headers)
	input.Headers = headers

	reqURL, err := s.buildURL(input.URL, input.Params)
	if err != nil {
		return nil, appErrors.Wrap("INVALID_URL", "URL 解析失败", err)
	}

	bodyReader, contentType, err := s.buildBody(input.Body)
	if err != nil {
		return nil, appErrors.Wrap("INVALID_BODY", "请求体构建失败", err)
	}
	requestBodyBytes := estimateRequestBodyBytes(input.Body)

	req, err := http.NewRequest(input.Method, reqURL, bodyReader)
	if err != nil {
		return nil, appErrors.Wrap("REQUEST_BUILD_FAILED", "请求构建失败", err)
	}

	// 设置 Content-Type（仅在有 body 时）
	if contentType != "" && !options.disableAutoContentType {
		req.Header.Set("Content-Type", contentType)
	}

	// 设置自定义 headers
	for _, h := range input.Headers {
		req.Header.Set(h.Key, h.Value)
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
	resp, err := client.Do(req)

	if err != nil {
		code, message, detail := classifyRequestSendError(err, reqURL)
		return nil, &appErrors.AppError{
			Code:    code,
			Message: message,
			Detail:  detail,
		}
	}
	defer resp.Body.Close()
	responseHeaderBytes := estimateResponseHeaderBytes(resp)
	network := buildNetworkDetails(resp, trace)

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
		return nil, err
	}
	bodyDone := time.Now()

	respContentType := resp.Header.Get("Content-Type")
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

	return &model.HttpResponse{
		StatusCode:  resp.StatusCode,
		StatusText:  http.StatusText(resp.StatusCode),
		Headers:     resp.Header,
		Body:        string(bodyBytes),
		Duration:    timings.Total,
		Size:        sizeDetails.ResponseTotal,
		ContentType: respContentType,
		Protocol:    resp.Proto,
		Warnings:    warnings,
		Network:     network,
		Timings:     timings,
		SizeDetails: sizeDetails,
	}, nil
}

func buildNetworkDetails(resp *http.Response, trace requestTrace) *model.NetworkDetails {
	details := &model.NetworkDetails{
		HTTPVersion:   fmt.Sprintf("%d.%d", resp.ProtoMajor, resp.ProtoMinor),
		LocalAddress:  trace.localAddr,
		RemoteAddress: trace.remoteAddr,
	}

	if resp.TLS != nil {
		details.TLSProtocol = tlsVersionLabel(resp.TLS.Version)
		details.CipherName = tls.CipherSuiteName(resp.TLS.CipherSuite)
		if details.CipherName == "" && resp.TLS.CipherSuite != 0 {
			details.CipherName = fmt.Sprintf("0x%04x", resp.TLS.CipherSuite)
		}

		if len(resp.TLS.PeerCertificates) > 0 {
			leaf := resp.TLS.PeerCertificates[0]
			if leaf.Subject.CommonName != "" {
				details.CertificateCN = leaf.Subject.CommonName
			} else if len(leaf.DNSNames) > 0 {
				details.CertificateCN = leaf.DNSNames[0]
			}
			details.IssuerCN = leaf.Issuer.CommonName
			details.ValidUntil = leaf.NotAfter.UTC().Format(time.RFC3339)
		}
	}

	if details.HTTPVersion == "" &&
		details.LocalAddress == "" &&
		details.RemoteAddress == "" &&
		details.TLSProtocol == "" &&
		details.CipherName == "" &&
		details.CertificateCN == "" &&
		details.IssuerCN == "" &&
		details.ValidUntil == "" {
		return nil
	}
	return details
}

func tlsVersionLabel(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLSv1.0"
	case tls.VersionTLS11:
		return "TLSv1.1"
	case tls.VersionTLS12:
		return "TLSv1.2"
	case tls.VersionTLS13:
		return "TLSv1.3"
	default:
		return ""
	}
}

func normalizeSocketAddress(addr net.Addr) string {
	if addr == nil {
		return ""
	}
	raw := strings.TrimSpace(addr.String())
	if raw == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(raw)
	if err == nil {
		return strings.Trim(host, "[]")
	}
	return strings.Trim(raw, "[]")
}

func (s *HttpService) buildURL(rawURL string, params []model.KeyValue) (string, error) {
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	q := u.Query()
	for _, p := range params {
		q.Add(p.Key, p.Value)
	}
	u.RawQuery = q.Encode()

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
		form := url.Values{}
		for _, kv := range body.FormUrlEncoded {
			form.Add(kv.Key, kv.Value)
		}
		return strings.NewReader(form.Encode()), "application/x-www-form-urlencoded", nil

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
	return *parsed, nil
}

func (s *HttpService) extractRequestOptions(headers []model.KeyValue) (requestOptions, []model.KeyValue) {
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

	cleaned := make([]model.KeyValue, 0, len(headers))
	for _, header := range headers {
		key := strings.ToLower(strings.TrimSpace(header.Key))
		value := strings.TrimSpace(header.Value)

		switch key {
		case "x-minipost-option-follow-redirects":
			options.followRedirects = parseBoolOption(value, options.followRedirects)
			continue
		case "x-minipost-option-timeout-ms":
			if parsed, err := strconv.Atoi(value); err == nil {
				if parsed <= 0 {
					options.timeout = 0
				} else {
					options.timeout = time.Duration(parsed) * time.Millisecond
				}
			}
			continue
		case "x-minipost-option-max-response-size-mb":
			if parsed, err := strconv.Atoi(value); err == nil {
				if parsed <= 0 {
					options.maxResponseBytes = 0
				} else {
					options.maxResponseBytes = int64(parsed) * 1024 * 1024
				}
			}
			continue
		case "x-minipost-option-ssl-verify":
			options.sslVerify = parseBoolOption(value, options.sslVerify)
			continue
		case "x-minipost-option-http-version":
			switch strings.ToLower(value) {
			case "auto", "http1", "http2":
				options.httpVersion = strings.ToLower(value)
			}
			continue
		case "x-minipost-option-disable-default-user-agent":
			options.disableDefaultUserAgent = parseBoolOption(value, options.disableDefaultUserAgent)
			continue
		case "x-minipost-option-disable-default-accept":
			options.disableDefaultAccept = parseBoolOption(value, options.disableDefaultAccept)
			continue
		case "x-minipost-option-disable-auto-content-type":
			options.disableAutoContentType = parseBoolOption(value, options.disableAutoContentType)
			continue
		default:
			cleaned = append(cleaned, header)
		}
	}

	return options, cleaned
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

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if currentTransport, ok := s.client.Transport.(*http.Transport); ok && currentTransport != nil {
		transport = currentTransport.Clone()
	}

	if !options.sslVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	switch options.httpVersion {
	case "http1":
		transport.ForceAttemptHTTP2 = false
	case "http2":
		transport.ForceAttemptHTTP2 = true
	}

	client.Transport = transport
	return &client
}

func (s *HttpService) readResponseBody(body io.Reader, maxResponseBytes int64, contentType string, onChunk func(model.StreamChunk)) ([]byte, error) {
	if onChunk != nil {
		if isSSEContentType(contentType) {
			return s.readSSEStreamBody(body, maxResponseBytes, onChunk)
		}
	}

	if maxResponseBytes <= 0 {
		bodyBytes, err := io.ReadAll(body)
		if err != nil {
			return nil, appErrors.Wrap("READ_BODY_FAILED", "读取响应体失败", err)
		}
		return bodyBytes, nil
	}

	limited := io.LimitReader(body, maxResponseBytes+1)
	bodyBytes, err := io.ReadAll(limited)
	if err != nil {
		return nil, appErrors.Wrap("READ_BODY_FAILED", "读取响应体失败", err)
	}
	if int64(len(bodyBytes)) > maxResponseBytes {
		return nil, newMaxBodySizeExceededError(maxResponseBytes)
	}
	return bodyBytes, nil
}

func (s *HttpService) readSSEStreamBody(body io.Reader, maxResponseBytes int64, onChunk func(model.StreamChunk)) ([]byte, error) {
	reader := bufio.NewReader(body)
	buffer := bytes.NewBuffer(nil)
	eventLines := make([]string, 0, 8)
	sequence := 0

	emit := func(kind string, data string, raw string) {
		sequence++
		onChunk(model.StreamChunk{
			Kind:       kind,
			Data:       data,
			Raw:        raw,
			Timestamp:  time.Now().Format(time.RFC3339Nano),
			Sequence:   sequence,
			BytesTotal: int64(buffer.Len()),
		})
	}

	flushEvent := func() {
		if len(eventLines) == 0 {
			return
		}
		raw := strings.Join(eventLines, "\n")
		dataLines := make([]string, 0, len(eventLines))
		for _, line := range eventLines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "data:") {
				dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(trimmed, "data:")))
			}
		}
		if len(dataLines) > 0 {
			emit("data", strings.Join(dataLines, "\n"), raw)
		} else {
			emit("event", raw, raw)
		}
		eventLines = eventLines[:0]
	}

	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			if _, writeErr := buffer.WriteString(line); writeErr != nil {
				return nil, appErrors.Wrap("READ_BODY_FAILED", "读取响应体失败", writeErr)
			}
			if maxResponseBytes > 0 && int64(buffer.Len()) > maxResponseBytes {
				return nil, newMaxBodySizeExceededError(maxResponseBytes)
			}

			trimmedLine := strings.TrimRight(line, "\r\n")
			if trimmedLine == "" {
				flushEvent()
			} else {
				eventLines = append(eventLines, trimmedLine)
			}
		}

		if err == nil {
			continue
		}
		if errors.Is(err, io.EOF) {
			break
		}
		return nil, appErrors.Wrap("READ_BODY_FAILED", "读取响应体失败", err)
	}

	flushEvent()
	return buffer.Bytes(), nil
}

func isSSEContentType(contentType string) bool {
	return strings.Contains(strings.ToLower(contentType), "text/event-stream")
}

func newMaxBodySizeExceededError(maxResponseBytes int64) error {
	limitMB := maxResponseBytes / (1024 * 1024)
	return appErrors.New("MAX_RESPONSE_SIZE_EXCEEDED", fmt.Sprintf("响应体超过最大大小限制（%d MB）", limitMB))
}

func parseBoolOption(raw string, defaultValue bool) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return defaultValue
	}
}

func classifyRequestSendError(err error, requestURL string) (code string, message string, detail string) {
	lower := strings.ToLower(err.Error())
	endpoint, host := resolveRequestEndpoint(requestURL, true)
	if host == "" {
		endpoint = requestURL
	}
	rawDetail := stripRequestPrefix(err.Error(), requestURL)

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) || strings.Contains(lower, "no such host") {
		target := host
		if target == "" {
			target = requestURL
		}
		return "DNS_LOOKUP_FAILED", "域名解析失败", fmt.Sprintf("getaddrinfo ENOTFOUND %s", target)
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "REQUEST_TIMEOUT", "请求超时", fmt.Sprintf("connect ETIMEDOUT %s", endpoint)
	}
	if strings.Contains(lower, "deadline exceeded") || strings.Contains(lower, "timeout") {
		return "REQUEST_TIMEOUT", "请求超时", fmt.Sprintf("connect ETIMEDOUT %s", endpoint)
	}

	if strings.Contains(lower, "connection refused") || strings.Contains(lower, "econnrefused") {
		return "CONNECTION_REFUSED", "连接被目标服务拒绝", fmt.Sprintf("connect ECONNREFUSED %s", endpoint)
	}

	if strings.Contains(lower, "x509") || strings.Contains(lower, "tls") || strings.Contains(lower, "certificate") {
		return "TLS_HANDSHAKE_FAILED", "TLS/证书校验失败", rawDetail
	}

	if strings.Contains(lower, "eof") {
		// 某些网络环境下，无法连通目标时 Go 可能返回 EOF，这里统一转换为更可读的连接拒绝样式。
		return "CONNECTION_REFUSED", "连接被目标服务拒绝", fmt.Sprintf("connect ECONNREFUSED %s", endpoint)
	}

	return "REQUEST_FAILED", "请求发送失败", rawDetail
}

func stripRequestPrefix(rawError string, requestURL string) string {
	prefix := fmt.Sprintf(`Get "%s": `, requestURL)
	if strings.HasPrefix(rawError, prefix) {
		return strings.TrimSpace(strings.TrimPrefix(rawError, prefix))
	}
	return rawError
}

func resolveRequestEndpoint(requestURL string, preferIP bool) (endpoint string, host string) {
	u, err := url.Parse(requestURL)
	if err != nil || u.Hostname() == "" {
		return requestURL, ""
	}

	host = u.Hostname()
	port := u.Port()
	if port == "" {
		if strings.EqualFold(u.Scheme, "https") {
			port = "443"
		} else {
			port = "80"
		}
	}

	if preferIP {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		addrs, lookupErr := net.DefaultResolver.LookupIPAddr(ctx, host)
		if lookupErr == nil && len(addrs) > 0 {
			return fmt.Sprintf("%s:%s", addrs[0].IP.String(), port), host
		}
	}

	return fmt.Sprintf("%s:%s", host, port), host
}

func detectTLSWarning(requestURL string, tlsState *tls.ConnectionState) string {
	if tlsState == nil || len(tlsState.PeerCertificates) == 0 {
		return ""
	}

	parsedURL, err := url.Parse(requestURL)
	if err != nil {
		return ""
	}

	leaf := tlsState.PeerCertificates[0]
	verifyOpts := x509.VerifyOptions{
		DNSName:       parsedURL.Hostname(),
		Intermediates: x509.NewCertPool(),
	}
	for _, cert := range tlsState.PeerCertificates[1:] {
		verifyOpts.Intermediates.AddCert(cert)
	}
	if roots, rootsErr := x509.SystemCertPool(); rootsErr == nil && roots != nil {
		verifyOpts.Roots = roots
	}

	if _, err := leaf.Verify(verifyOpts); err != nil {
		lower := strings.ToLower(err.Error())
		switch {
		case strings.Contains(lower, "expired"):
			return "Certificate has expired"
		case strings.Contains(lower, "not yet valid"):
			return "Certificate is not yet valid"
		case strings.Contains(lower, "unknown authority"):
			return "Certificate is not trusted"
		default:
			return "TLS certificate verification warning"
		}
	}

	return ""
}

func (s *HttpService) applyAuth(req *http.Request, auth model.AuthConfig) {
	switch auth.Type {
	case "basic":
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
				q := req.URL.Query()
				q.Set(auth.APIKey.Key, auth.APIKey.Value)
				req.URL.RawQuery = q.Encode()
			} else {
				req.Header.Set(auth.APIKey.Key, auth.APIKey.Value)
			}
		}
	}
}
