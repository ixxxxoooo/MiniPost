package model

// HttpResponse 返回给前端的 HTTP 响应结构
type HttpResponse struct {
	StatusCode  int                 `json:"statusCode"`
	StatusText  string              `json:"statusText"`
	Headers     map[string][]string `json:"headers"`
	Body        string              `json:"body"`
	Duration    float64             `json:"duration"`
	Size        int64               `json:"size"`
	ContentType string              `json:"contentType"`
}
