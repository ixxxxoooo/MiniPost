package model

type TimingBreakdown struct {
	Prepare              float64 `json:"prepare"`
	SocketInitialization float64 `json:"socketInitialization"`
	DNSLookup            float64 `json:"dnsLookup"`
	TCPHandshake         float64 `json:"tcpHandshake"`
	SSLHandshake         float64 `json:"sslHandshake"`
	WaitingTTFB          float64 `json:"waitingTTFB"`
	Download             float64 `json:"download"`
	Process              float64 `json:"process"`
	Total                float64 `json:"total"`
}

type SizeBreakdown struct {
	ResponseHeaders int64 `json:"responseHeaders"`
	ResponseBody    int64 `json:"responseBody"`
	ResponseTotal   int64 `json:"responseTotal"`
	RequestHeaders  int64 `json:"requestHeaders"`
	RequestBody     int64 `json:"requestBody"`
	RequestTotal    int64 `json:"requestTotal"`
}

// HttpResponse 返回给前端的 HTTP 响应结构
type HttpResponse struct {
	StatusCode  int                 `json:"statusCode"`
	StatusText  string              `json:"statusText"`
	Headers     map[string][]string `json:"headers"`
	Body        string              `json:"body"`
	Duration    float64             `json:"duration"`
	Size        int64               `json:"size"`
	ContentType string              `json:"contentType"`
	Protocol    string              `json:"protocol,omitempty"`
	Warnings    []string            `json:"warnings,omitempty"`
	Timings     TimingBreakdown     `json:"timings"`
	SizeDetails SizeBreakdown       `json:"sizeDetails"`
}
