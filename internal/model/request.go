package model

// SendRequestInput 前端发送请求时传入的参数
type SendRequestInput struct {
	Method  string         `json:"method"`
	URL     string         `json:"url"`
	Params  []KeyValue     `json:"params"`
	Headers []KeyValue     `json:"headers"`
	Body    RequestBody    `json:"body"`
	Auth    AuthConfig     `json:"auth"`
}

type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type RequestBody struct {
	Type           string     `json:"type"`
	Raw            string     `json:"raw"`
	JSON           string     `json:"json"`
	FormUrlEncoded []KeyValue `json:"formUrlEncoded"`
}

type AuthConfig struct {
	Type   string         `json:"type"`
	Basic  BasicAuth      `json:"basic"`
	Bearer BearerAuth     `json:"bearer"`
	APIKey APIKeyAuth     `json:"apiKey"`
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
