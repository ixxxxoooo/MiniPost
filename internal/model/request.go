package model

// SendRequestInput 前端发送请求时传入的参数
type SendRequestInput struct {
	Method  string          `json:"method"`
	URL     string          `json:"url"`
	Params  []KeyValue      `json:"params"`
	Headers []KeyValue      `json:"headers"`
	Body    RequestBody     `json:"body"`
	Auth    AuthConfig      `json:"auth"`
	Options *RequestOptions `json:"options,omitempty"`
}

type RequestOptions struct {
	FollowRedirects         bool   `json:"followRedirects"`
	TimeoutMS               int    `json:"timeoutMs"`
	MaxResponseSizeMB       int    `json:"maxResponseSizeMB"`
	SSLVerify               bool   `json:"sslVerify"`
	HTTPVersion             string `json:"httpVersion"`
	DisableDefaultUserAgent bool   `json:"disableDefaultUserAgent"`
	DisableDefaultAccept    bool   `json:"disableDefaultAccept"`
	DisableAutoContentType  bool   `json:"disableAutoContentType"`
}

type KeyValue struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
}

type RequestBody struct {
	Type           string     `json:"type"`
	Raw            string     `json:"raw"`
	JSON           string     `json:"json"`
	FormUrlEncoded []KeyValue `json:"formUrlEncoded"`
	FormData       []FormData `json:"formData"`
}

type FormData struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type"` // text | file
	FilePath    string `json:"filePath,omitempty"`
	FileName    string `json:"fileName,omitempty"`
}

type AuthConfig struct {
	Type   string     `json:"type"`
	Basic  BasicAuth  `json:"basic"`
	Bearer BearerAuth `json:"bearer"`
	APIKey APIKeyAuth `json:"apiKey"`
}

type BasicAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type BearerAuth struct {
	Token string `json:"token"`
}

type APIKeyAuth struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	AddTo string `json:"addTo"`
}
