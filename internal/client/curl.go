package client

import (
	"strings"
)

// shellQuote 用单引号包裹，并转义内部单引号。
func shellQuote(v string) string { return "'" + strings.ReplaceAll(v, "'", `'"'"'`) + "'" }

// encodeURIComponent 与 JavaScript 的同名函数一致：保留 A-Za-z0-9-_.!~*'()，其余按 UTF-8 百分号编码。
func encodeURIComponent(s string) string {
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	for _, c := range []byte(s) {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9',
			strings.IndexByte("-_.!~*'()", c) >= 0:
			b.WriteByte(c)
		default:
			b.WriteByte('%')
			b.WriteByte(hex[c>>4])
			b.WriteByte(hex[c&15])
		}
	}
	return b.String()
}

func appendQuery(u, key, value string) string {
	sep := "?"
	if strings.Contains(u, "?") {
		sep = "&"
	}
	return u + sep + encodeURIComponent(key) + "=" + encodeURIComponent(value)
}

// ResolveCurlURL 解析变量并补全协议后的请求地址。
func ResolveCurlURL(d *Draft, vars []Var) string {
	return EnsureProtocol(ResolveTemplate(d.URL, vars))
}

// BuildCurlCommand 把草稿导出为 cURL 命令；cookieHeader 为要追加的 Jar Cookie。
func BuildCurlCommand(d *Draft, vars []Var, cookieHeader string) string {
	res := func(s string) string { return ResolveTemplate(s, vars) }
	parts := []string{"curl"}
	method := strings.ToUpper(orDefault(d.Method, "GET"))
	if method != "GET" {
		parts = append(parts, "-X "+method)
	}

	full := EnsureProtocol(buildURLWithQuery(d, vars, encodeURIComponent))
	if d.Auth.Type == AuthAPIKey && d.Auth.APIKeyLocation == "query" {
		if key := res(d.Auth.APIKeyName); key != "" {
			full = appendQuery(full, key, res(d.Auth.APIKeyValue))
		}
	}
	parts = append(parts, shellQuote(full))

	manualCookie := ""
	for _, h := range d.Headers {
		if !h.Enabled || IsAutoHeaderMarker(h.Key) {
			continue
		}
		key := res(h.Key)
		if key == "" {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(key), "cookie") {
			manualCookie = res(h.Value)
			continue
		}
		parts = append(parts, "-H "+shellQuote(key+": "+res(h.Value)))
	}
	if merged := MergeCookieHeaders(manualCookie, cookieHeader); merged != "" {
		parts = append(parts, "-H "+shellQuote("Cookie: "+merged))
	}

	switch d.Auth.Type {
	case AuthBearer:
		if d.Auth.Token != "" {
			parts = append(parts, "-H "+shellQuote("Authorization: Bearer "+res(d.Auth.Token)))
		}
	case AuthBasic:
		parts = append(parts, "-u "+shellQuote(res(d.Auth.Username)+":"+res(d.Auth.Password)))
	case AuthAPIKey:
		if d.Auth.APIKeyLocation != "query" {
			if key := res(d.Auth.APIKeyName); key != "" {
				parts = append(parts, "-H "+shellQuote(key+": "+res(d.Auth.APIKeyValue)))
			}
		}
	}

	parts = append(parts, curlBodyParts(d, res)...)
	return strings.Join(parts, " \\\n  ")
}

func curlBodyParts(d *Draft, res func(string) string) []string {
	var parts []string
	switch d.Body.Type {
	case BodyJSON:
		if d.Body.JSON != "" {
			parts = append(parts, "-H "+shellQuote("Content-Type: application/json"), "-d "+shellQuote(res(d.Body.JSON)))
		}
	case BodyRaw:
		if d.Body.Raw != "" {
			parts = append(parts, "-d "+shellQuote(res(d.Body.Raw)))
		}
	case BodyURLEncoded:
		var fields []string
		for _, f := range d.Body.URLEncoded {
			if key := res(f.Key); f.Enabled && key != "" {
				fields = append(fields, encodeURIComponent(key)+"="+encodeURIComponent(res(f.Value)))
			}
		}
		if len(fields) > 0 {
			parts = append(parts, "-H "+shellQuote("Content-Type: application/x-www-form-urlencoded"),
				"-d "+shellQuote(strings.Join(fields, "&")))
		}
	case BodyFormData:
		for _, f := range d.Body.FormData {
			key := res(f.Key)
			if !f.Enabled || key == "" {
				continue
			}
			if f.Type == "file" {
				path := strings.TrimSpace(res(orDefault(f.FilePath, f.Value)))
				if path != "" {
					parts = append(parts, "-F "+shellQuote(key+"=@"+path))
				}
			} else {
				parts = append(parts, "--form-string "+shellQuote(key+"="+res(f.Value)))
			}
		}
	}
	return parts
}

// RuntimeCurl 导出带运行时 Cookie 的 cURL：Cookie 被全局禁用或该 Header 被关闭时不带 Jar Cookie。
func RuntimeCurl(d *Draft, vars []Var, set Settings, jar *Jar) (command string, cookieIncluded bool) {
	cookie := ""
	if !set.DisableCookies && jar != nil && !SuppressedAutoHeaders(d.Headers)["cookie"] {
		cookie = jar.Header(ResolveCurlURL(d, vars))
	}
	return BuildCurlCommand(d, vars, cookie), cookie != ""
}
