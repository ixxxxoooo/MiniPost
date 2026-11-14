package service

import (
	"fmt"
	"strings"

	"github.com/google/uuid"

	"minipost/internal/model"
	"minipost/internal/pkg/logger"
)

// SampleRequest 是示例项目里的一条请求；URL 里的 {{baseUrl}} 等变量由示例环境提供。
type SampleRequest struct {
	Name    string
	Method  string
	URL     string
	Headers []model.KeyValue
	Body    model.RequestBody
	Auth    model.AuthConfig
}

// SampleFolder 是示例项目里的文件夹。
type SampleFolder struct {
	Name     string
	Requests []SampleRequest
}

// SampleProject 描述一个用于功能测试的示例项目。
type SampleProject struct {
	Name        string
	Description string
	Variables   []model.Variable // 非空时创建名为「示例环境」的环境
	Folders     []SampleFolder
}

func kv(k, v string) model.KeyValue { return model.KeyValue{Key: k, Value: v} }

func get(name, url string) SampleRequest { return SampleRequest{Name: name, Method: "GET", URL: url} }

func jsonReq(name, method, url, body string) SampleRequest {
	return SampleRequest{Name: name, Method: method, URL: url,
		Headers: []model.KeyValue{kv("Content-Type", "application/json")},
		Body:    model.RequestBody{Type: "json", JSON: body}}
}

func v(key, value string) model.Variable {
	return model.Variable{ID: uuid.NewString(), Key: key, Value: value, Enabled: true}
}

// SampleProjects 返回内置的示例项目：全部指向公共的 httpbin 服务，覆盖请求方法、认证、
// 请求体类型、Cookie、各类响应格式与下载、状态码与网络错误、流式响应。
func SampleProjects() []SampleProject {
	return []SampleProject{
		{
			Name:        "示例 · 请求与认证",
			Description: "基于 httpbin 的请求方法、请求体类型、认证方式与 Cookie 示例，使用环境变量 baseUrl / token。",
			Variables:   []model.Variable{v("baseUrl", "https://httpbin.org"), v("token", "demo-token"), v("username", "user"), v("password", "passwd")},
			Folders: []SampleFolder{
				{Name: "请求方法", Requests: []SampleRequest{
					get("GET 带查询参数", "{{baseUrl}}/get?foo=bar&page=1&size=20"),
					jsonReq("POST JSON", "POST", "{{baseUrl}}/post", "{\n  \"name\": \"MiniPost\",\n  \"tags\": [\"api\", \"debug\"],\n  \"nested\": {\"ok\": true, \"count\": 3}\n}"),
					jsonReq("PUT JSON", "PUT", "{{baseUrl}}/put", "{\"id\": 1, \"name\": \"updated\"}"),
					jsonReq("PATCH JSON", "PATCH", "{{baseUrl}}/patch", "{\"name\": \"patched\"}"),
					{Name: "DELETE", Method: "DELETE", URL: "{{baseUrl}}/delete?id=1"},
					{Name: "HEAD", Method: "HEAD", URL: "{{baseUrl}}/get"},
					{Name: "OPTIONS", Method: "OPTIONS", URL: "{{baseUrl}}/get"},
				}},
				{Name: "请求体类型", Requests: []SampleRequest{
					{Name: "Raw 文本", Method: "POST", URL: "{{baseUrl}}/post",
						Headers: []model.KeyValue{kv("Content-Type", "text/plain")},
						Body:    model.RequestBody{Type: "raw", Raw: "hello from MiniPost"}},
					{Name: "表单 urlencoded", Method: "POST", URL: "{{baseUrl}}/post",
						Body: model.RequestBody{Type: "form-urlencoded", FormUrlEncoded: []model.KeyValue{kv("username", "alice"), kv("password", "secret"), kv("remember", "true")}}},
					{Name: "表单 form-data", Method: "POST", URL: "{{baseUrl}}/post",
						Body: model.RequestBody{Type: "form-data", FormData: []model.FormData{
							{Key: "title", Value: "文件上传示例", Type: "text"},
							{Key: "desc", Value: "multipart 文本字段", Type: "text"},
						}}},
				}},
				{Name: "认证", Requests: []SampleRequest{
					{Name: "Basic Auth", Method: "GET", URL: "{{baseUrl}}/basic-auth/{{username}}/{{password}}",
						Auth: model.AuthConfig{Type: "basic", Basic: model.BasicAuth{Username: "{{username}}", Password: "{{password}}"}}},
					{Name: "Bearer Token", Method: "GET", URL: "{{baseUrl}}/bearer",
						Auth: model.AuthConfig{Type: "bearer", Bearer: model.BearerAuth{Token: "{{token}}"}}},
					{Name: "API Key (Header)", Method: "GET", URL: "{{baseUrl}}/headers",
						Auth: model.AuthConfig{Type: "api-key", APIKey: model.APIKeyAuth{Key: "X-API-Key", Value: "{{token}}", AddTo: "header"}}},
					{Name: "API Key (Query)", Method: "GET", URL: "{{baseUrl}}/get",
						Auth: model.AuthConfig{Type: "api-key", APIKey: model.APIKeyAuth{Key: "api_key", Value: "{{token}}", AddTo: "query"}}},
				}},
				{Name: "请求头与 Cookie", Requests: []SampleRequest{
					{Name: "自定义请求头", Method: "GET", URL: "{{baseUrl}}/headers",
						Headers: []model.KeyValue{kv("X-Request-Id", "minipost-001"), kv("X-Env", "demo"), {Key: "X-Disabled", Value: "不会发送", Disabled: true}}},
					get("设置 Cookie", "{{baseUrl}}/cookies/set?session=abc123&theme=dark"),
					get("查看 Cookie", "{{baseUrl}}/cookies"),
					get("删除 Cookie", "{{baseUrl}}/cookies/delete?session=&theme="),
					get("User-Agent", "{{baseUrl}}/user-agent"),
				}},
			},
		},
		{
			Name:        "示例 · 响应格式与下载",
			Description: "覆盖 JSON / XML / HTML / 文本 / 图片 / 压缩 / 二进制 / 附件下载等响应，用来测试格式切换、预览、过滤、复制与下载。",
			Variables:   []model.Variable{v("baseUrl", "https://httpbin.org")},
			Folders: []SampleFolder{
				{Name: "文本类", Requests: []SampleRequest{
					get("JSON", "{{baseUrl}}/json"),
					get("XML", "{{baseUrl}}/xml"),
					get("HTML", "{{baseUrl}}/html"),
					get("纯文本", "{{baseUrl}}/robots.txt"),
					get("UTF-8 文本", "{{baseUrl}}/encoding/utf8"),
					get("深层嵌套 JSON (anything)", "{{baseUrl}}/anything/a/b/c?x=1&y=2"),
					get("NDJSON 逐行 JSON (stream/5)", "{{baseUrl}}/stream/5"),
				}},
				{Name: "压缩", Requests: []SampleRequest{
					get("gzip", "{{baseUrl}}/gzip"),
					get("deflate", "{{baseUrl}}/deflate"),
					get("brotli", "{{baseUrl}}/brotli"),
				}},
				{Name: "二进制与下载", Requests: []SampleRequest{
					get("PNG 图片", "{{baseUrl}}/image/png"),
					get("JPEG 图片", "{{baseUrl}}/image/jpeg"),
					get("WebP 图片", "{{baseUrl}}/image/webp"),
					get("SVG 图片", "{{baseUrl}}/image/svg"),
					get("随机字节 1KB", "{{baseUrl}}/bytes/1024"),
					get("下载 (attachment 文本)", "{{baseUrl}}/response-headers?Content-Type=text/plain&Content-Disposition=attachment%3B%20filename%3D%22demo.txt%22"),
					get("下载 (attachment JSON)", "{{baseUrl}}/response-headers?Content-Type=application/json&Content-Disposition=attachment%3B%20filename%3D%22data.json%22"),
					get("PDF 文件", "https://www.w3.org/WAI/ER/tests/xhtml/testfiles/resources/pdf/dummy.pdf"),
				}},
				{Name: "响应头", Requests: []SampleRequest{
					get("自定义响应头", "{{baseUrl}}/response-headers?X-Demo=hello&X-Count=3"),
					get("多个 Set-Cookie", "{{baseUrl}}/cookies/set?a=1&b=2&c=3"),
					get("缓存头 (cache/60)", "{{baseUrl}}/cache/60"),
				}},
			},
		},
		{
			Name:        "示例 · 状态码与异常",
			Description: "各类状态码、重定向、延迟/超时，以及 DNS、连接被拒绝、TLS 证书等网络错误，用来测试响应卡片、错误提示与设置里的重定向/超时/SSL 选项。",
			Variables:   []model.Variable{v("baseUrl", "https://httpbin.org")},
			Folders: []SampleFolder{
				{Name: "成功与重定向", Requests: []SampleRequest{
					get("200 OK", "{{baseUrl}}/status/200"),
					{Name: "201 Created", Method: "POST", URL: "{{baseUrl}}/status/201"},
					{Name: "204 No Content", Method: "DELETE", URL: "{{baseUrl}}/status/204"},
					get("301 永久重定向", "{{baseUrl}}/redirect-to?url=%2Fget&status_code=301"),
					get("302 临时重定向", "{{baseUrl}}/redirect-to?url=%2Fget&status_code=302"),
					get("连续重定向 3 次", "{{baseUrl}}/redirect/3"),
					get("304 Not Modified", "{{baseUrl}}/status/304"),
				}},
				{Name: "客户端 / 服务端错误", Requests: []SampleRequest{
					get("400 Bad Request", "{{baseUrl}}/status/400"),
					get("401 Unauthorized", "{{baseUrl}}/status/401"),
					get("403 Forbidden", "{{baseUrl}}/status/403"),
					get("404 Not Found", "{{baseUrl}}/status/404"),
					get("418 I'm a teapot", "{{baseUrl}}/status/418"),
					get("429 Too Many Requests", "{{baseUrl}}/status/429"),
					get("500 Internal Server Error", "{{baseUrl}}/status/500"),
					get("502 Bad Gateway", "{{baseUrl}}/status/502"),
					get("503 Service Unavailable", "{{baseUrl}}/status/503"),
				}},
				{Name: "延迟与超时", Requests: []SampleRequest{
					get("延迟 2 秒", "{{baseUrl}}/delay/2"),
					get("延迟 10 秒 (测试超时/取消)", "{{baseUrl}}/delay/10"),
					get("逐字节下发 5 秒 (drip)", "{{baseUrl}}/drip?duration=5&numbytes=20&code=200"),
				}},
				{Name: "网络错误", Requests: []SampleRequest{
					get("DNS 解析失败", "https://no-such-host.minipost.invalid/"),
					get("连接被拒绝", "http://127.0.0.1:1/"),
					get("TLS 证书过期", "https://expired.badssl.com/"),
					get("TLS 自签名证书", "https://self-signed.badssl.com/"),
					get("TLS 主机名不匹配", "https://wrong.host.badssl.com/"),
				}},
			},
		},
		{
			Name:        "示例 · 流式响应",
			Description: "SSE 与分块流式响应，用来测试 Stream 标签、搜索、消息展开与取消请求。",
			Folders: []SampleFolder{
				{Name: "SSE / 流", Requests: []SampleRequest{
					get("SSE 5 条消息 (httpbingo)", "https://httpbingo.org/sse?count=5&delay=500ms"),
					get("SSE 20 条 / 间隔 200ms", "https://httpbingo.org/sse?count=20&delay=200ms"),
					get("JSON 流 10 条 (httpbin)", "https://httpbin.org/stream/10"),
					get("分块下发 (drip)", "https://httpbin.org/drip?duration=3&numbytes=30&code=200"),
				}},
			},
		},
	}
}

// SampleInstaller 抽象示例安装所需的后端能力（便于测试与界面层解耦）。
type SampleInstaller struct {
	Projects *ProjectService
	Requests *RequestService
	Envs     *EnvironmentService
}

// InstallSamples 创建内置示例项目；已存在同名项目时跳过，返回本次新建的项目名与总数。
func (in SampleInstaller) InstallSamples() (created []string, err error) {
	existing, err := in.Projects.ListProjects()
	if err != nil {
		return nil, err
	}
	have := make(map[string]bool, len(existing))
	for _, p := range existing {
		have[p.Name] = true
	}
	for _, sp := range SampleProjects() {
		if have[sp.Name] {
			logger.Info("示例项目已存在，跳过", "name", sp.Name)
			continue
		}
		if err := in.installOne(sp); err != nil {
			logger.Error("创建示例项目失败", "name", sp.Name, "error", err.Error())
			return created, fmt.Errorf("创建示例项目「%s」失败: %w", sp.Name, err)
		}
		created = append(created, sp.Name)
		logger.Info("已创建示例项目", "name", sp.Name)
	}
	return created, nil
}

func (in SampleInstaller) installOne(sp SampleProject) error {
	project, err := in.Projects.CreateProject(sp.Name)
	if err != nil {
		return err
	}
	if sp.Description != "" {
		if _, err := in.Projects.UpdateProjectDescription(project.ID, sp.Description); err != nil {
			return err
		}
	}
	if len(sp.Variables) > 0 {
		env, err := in.Envs.CreateEnvironment(project.ID, "示例环境")
		if err != nil {
			return err
		}
		env.Variables = sp.Variables
		if err := in.Envs.SaveEnvironment(env); err != nil {
			return err
		}
	}
	for _, f := range sp.Folders {
		folder, err := in.Requests.CreateFolder(project.ID, "", f.Name)
		if err != nil {
			return err
		}
		for _, r := range f.Requests {
			item, err := in.Requests.CreateRequest(project.ID, folder.ID, r.Name)
			if err != nil {
				return err
			}
			item.Method = strings.ToUpper(r.Method)
			item.URL = r.URL
			if len(r.Headers) > 0 {
				item.Headers = r.Headers
			}
			if r.Body.Type != "" {
				item.Body = r.Body
			}
			if r.Auth.Type != "" {
				item.Auth = r.Auth
			}
			if err := in.Requests.SaveRequest(item); err != nil {
				return err
			}
		}
	}
	return nil
}
