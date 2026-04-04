package service

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"minipost/internal/model"
	appErrors "minipost/internal/pkg/errors"
)

type HttpService struct {
	client *http.Client
}

func NewHttpService() *HttpService {
	return &HttpService{
		client: &http.Client{
			Timeout: 30 * time.Second,
			// 不自动跟随重定向，让用户看到原始响应
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// SendRequest 执行 HTTP 请求并返回响应
func (s *HttpService) SendRequest(input model.SendRequestInput) (*model.HttpResponse, error) {
	reqURL, err := s.buildURL(input.URL, input.Params)
	if err != nil {
		return nil, appErrors.Wrap("INVALID_URL", "URL 解析失败", err)
	}

	bodyReader, contentType, err := s.buildBody(input.Body)
	if err != nil {
		return nil, appErrors.Wrap("INVALID_BODY", "请求体构建失败", err)
	}

	req, err := http.NewRequest(input.Method, reqURL, bodyReader)
	if err != nil {
		return nil, appErrors.Wrap("REQUEST_BUILD_FAILED", "请求构建失败", err)
	}

	// 设置 Content-Type（仅在有 body 时）
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	// 设置自定义 headers
	for _, h := range input.Headers {
		req.Header.Set(h.Key, h.Value)
	}

	// 设置认证
	s.applyAuth(req, input.Auth)

	// 设置默认 User-Agent
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "MiniPost/1.0")
	}

	start := time.Now()
	resp, err := s.client.Do(req)
	duration := time.Since(start).Milliseconds()

	if err != nil {
		return nil, appErrors.Wrap("REQUEST_FAILED", "请求发送失败", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, appErrors.Wrap("READ_BODY_FAILED", "读取响应体失败", err)
	}

	respContentType := resp.Header.Get("Content-Type")

	return &model.HttpResponse{
		StatusCode:  resp.StatusCode,
		StatusText:  http.StatusText(resp.StatusCode),
		Headers:     resp.Header,
		Body:        string(bodyBytes),
		Duration:    float64(duration),
		Size:        int64(len(bodyBytes)),
		ContentType: respContentType,
	}, nil
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

	default:
		return nil, "", nil
	}
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
