package client

import "strings"

// AutoHeader 是发送时会由系统自动补充的 Header，在界面中以只读行展示，可单独关闭。
type AutoHeader struct {
	Key         string
	Value       string
	Description string
	Enabled     bool // false 表示用户已关闭该自动 Header
}

// deriveAutoContentType 根据请求体类型推导自动 Content-Type。
func deriveAutoContentType(d *Draft) string {
	switch d.Body.Type {
	case BodyJSON:
		if strings.TrimSpace(d.Body.JSON) != "" {
			return "application/json"
		}
	case BodyRaw:
		if strings.TrimSpace(d.Body.Raw) != "" {
			return "text/plain"
		}
	case BodyURLEncoded:
		return "application/x-www-form-urlencoded"
	}
	return ""
}

// AutoHeaders 列出当前请求发送时会自动补充的 Header。t 用于按界面语言选择文案。
func AutoHeaders(d *Draft, set Settings, jar *Jar, vars []Var, t func(zh, en string) string) []AutoHeader {
	present := map[string]bool{}
	for _, h := range d.Headers {
		if h.Enabled && strings.TrimSpace(h.Key) != "" && !IsAutoHeaderMarker(h.Key) {
			present[NormalizeHeaderName(h.Key)] = true
		}
	}
	suppressed := SuppressedAutoHeaders(d.Headers)
	var items []AutoHeader
	push := func(key, value, desc string) {
		items = append(items, AutoHeader{Key: key, Value: value, Description: desc, Enabled: !suppressed[NormalizeHeaderName(key)]})
	}
	runtimeValue := t("<发送请求时自动计算>", "<calculated when request is sent>")
	engineValue := t("<由运行时自动计算>", "<calculated by runtime>")
	contentType := deriveAutoContentType(d)

	if set.SendPostmanTokenHeader && !present["minipost-token"] && !present["postman-token"] {
		push(tokenHeaderName, runtimeValue, t("请求发送时自动生成随机 token", "Automatically generates random token when sending"))
	}
	if set.SendNoCacheHeader && !present["cache-control"] {
		push("Cache-Control", "no-cache", t("根据设置自动注入", "Auto-injected from settings"))
	}
	if contentType != "" && !present["content-type"] {
		push("Content-Type", contentType, t("根据 Body 类型自动生成", "Auto-generated from Body type"))
	}
	if (d.Body.Type != BodyNone || contentType != "") && !present["content-length"] {
		push("Content-Length", runtimeValue, t("运行时计算请求体长度", "Body length calculated at runtime"))
	}
	if !present["host"] {
		push("Host", runtimeValue, t("根据请求 URL 自动计算", "Auto-calculated from request URL"))
	}
	if !present["user-agent"] {
		push("User-Agent", "MiniPost/1.0", t("客户端默认标识", "Default client identifier"))
	}
	if !present["accept"] {
		push("Accept", "*/*", t("默认接受任意响应类型", "Accept any response type by default"))
	}
	if !present["accept-encoding"] {
		push("Accept-Encoding", engineValue, t("由底层 HTTP 运行时管理压缩协商", "Compression negotiation managed by HTTP runtime"))
	}
	if !present["connection"] {
		push("Connection", engineValue, t("由底层 HTTP 运行时管理连接复用", "Connection reuse managed by HTTP runtime"))
	}
	if !present["authorization"] {
		switch {
		case d.Auth.Type == AuthBearer && d.Auth.Token != "":
			push("Authorization", "Bearer "+d.Auth.Token, t("由 Auth(Bearer) 自动注入", "Auto-injected by Auth (Bearer)"))
		case d.Auth.Type == AuthBasic && (d.Auth.Username != "" || d.Auth.Password != ""):
			push("Authorization", t("Basic <由认证信息计算>", "Basic <calculated from auth>"), t("由 Auth(Basic) 自动注入", "Auto-injected by Auth (Basic)"))
		}
	}
	if d.Auth.Type == AuthAPIKey && d.Auth.APIKeyLocation == "header" {
		if key := strings.TrimSpace(d.Auth.APIKeyName); key != "" && !present[NormalizeHeaderName(key)] {
			push(key, d.Auth.APIKeyValue, t("由 Auth(API Key) 自动注入到 Header", "Auto-injected into header by Auth (API Key)"))
		}
	}
	if !set.DisableCookies && jar != nil && !present["cookie"] {
		url := ResolveTemplate(d.URL, vars)
		if url == "" {
			url = d.URL
		}
		if header := jar.Header(EnsureProtocol(url)); header != "" {
			push("Cookie", header, t("由 Cookie Jar 自动注入", "Auto-injected by Cookie Jar"))
		}
	}
	return items
}

// SetAutoHeaderEnabled 开关某个自动 Header：关闭时追加标记行，开启时移除标记行。
func SetAutoHeaderEnabled(d *Draft, name string, enabled bool) {
	marker := AutoHeaderMarkerKey(NormalizeHeaderName(name))
	kept := d.Headers[:0:0]
	found := false
	for _, h := range d.Headers {
		if NormalizeHeaderName(h.Key) == marker {
			found = true
			if enabled {
				continue
			}
		}
		kept = append(kept, h)
	}
	if !enabled && !found {
		r := NewRow()
		r.Key, r.Enabled, r.Description = marker, false, "auto-header-disabled"
		kept = append(kept, r)
	}
	d.Headers = kept
}
