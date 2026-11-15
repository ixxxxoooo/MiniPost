package client

import (
	"strings"

	"github.com/google/uuid"

	"minipost/internal/model"
	"minipost/internal/pkg/jsonutil"
)

// tokenHeaderName 自动附加的请求标识 Header 名。
const tokenHeaderName = "MiniPost-Token"

// SendEnv 构造发送载荷所需的外部状态。
type SendEnv struct {
	Vars     []Var
	Settings Settings
	Jar      *Jar
}

// SendInfo 记录构造载荷过程中得到的附加信息。
type SendInfo struct {
	ResolvedURL    string // 解析变量并补全协议后的地址
	CookieIncluded bool   // 是否自动携带了 Jar 中的 Cookie
}

func dedupeHeaders(headers []model.KeyValue) []model.KeyValue {
	out := make([]model.KeyValue, 0, len(headers))
	index := map[string]int{}
	for _, h := range headers {
		key := strings.TrimSpace(h.Key)
		if key == "" {
			continue
		}
		name := NormalizeHeaderName(key)
		if i, ok := index[name]; ok {
			out[i] = model.KeyValue{Key: key, Value: h.Value}
			continue
		}
		index[name] = len(out)
		out = append(out, model.KeyValue{Key: key, Value: h.Value})
	}
	return out
}

func hasHeader(headers []model.KeyValue, name string) bool {
	target := NormalizeHeaderName(name)
	for _, h := range headers {
		if NormalizeHeaderName(h.Key) == target {
			return true
		}
	}
	return false
}

// enabledKV 过滤出已启用且有键名的行。
func enabledKV(rows []*Row) []model.KeyValue {
	var out []model.KeyValue
	for _, r := range rows {
		// 关闭自动 Header 的标记行只是开关，绝不能当成真实 Header 发出（旧数据里它可能处于启用状态）。
		if r.Enabled && r.Key != "" && !IsAutoHeaderMarker(r.Key) {
			out = append(out, model.KeyValue{Key: r.Key, Value: r.Value})
		}
	}
	return out
}

func resolveKV(items []model.KeyValue, vars []Var) []model.KeyValue {
	out := make([]model.KeyValue, len(items))
	for i, it := range items {
		out[i] = model.KeyValue{Key: ResolveTemplate(it.Key, vars), Value: ResolveTemplate(it.Value, vars)}
	}
	return out
}

// buildPayload 把草稿转换为发送载荷（仅保留启用行，并解析环境变量）。
func buildPayload(d *Draft, requestID string, vars []Var) model.SendRequestInput {
	in := model.SendRequestInput{
		RequestID: requestID, SourceID: d.ID, Method: d.Method,
		// 查询参数已全部合并进地址（见 buildURLWithQuery），Params 留空，避免后端重复追加。
		URL:     buildURLWithQuery(d, vars, formEscape),
		Params:  []model.KeyValue{},
		Headers: dedupeHeaders(resolveKV(enabledKV(d.Headers), vars)),
	}
	in.Body = model.RequestBody{
		Type:           d.Body.Type,
		Raw:            ResolveTemplate(d.Body.Raw, vars),
		JSON:           ResolveTemplate(string(jsonutil.StripComments([]byte(d.Body.JSON))), vars),
		FormUrlEncoded: resolveKV(enabledKV(d.Body.URLEncoded), vars),
		FormData:       []model.FormData{},
	}
	for _, f := range d.Body.FormData {
		if !f.Enabled || f.Key == "" {
			continue
		}
		in.Body.FormData = append(in.Body.FormData, model.FormData{
			Key: ResolveTemplate(f.Key, vars), Value: ResolveTemplate(f.Value, vars), Type: f.Type,
			FilePath: ResolveTemplate(f.FilePath, vars), FileName: ResolveTemplate(f.FileName, vars),
		})
	}
	in.Auth = model.AuthConfig{
		Type:   d.Auth.Type,
		Basic:  model.BasicAuth{Username: ResolveTemplate(d.Auth.Username, vars), Password: ResolveTemplate(d.Auth.Password, vars)},
		Bearer: model.BearerAuth{Token: ResolveTemplate(d.Auth.Token, vars)},
		APIKey: model.APIKeyAuth{
			Key: ResolveTemplate(d.Auth.APIKeyName, vars), Value: ResolveTemplate(d.Auth.APIKeyValue, vars),
			AddTo: orDefault(d.Auth.APIKeyLocation, "header"),
		},
	}
	return in
}

// BuildSendInput 构造最终发送给后端的请求：解析变量、补全协议、合并 Cookie 与自动 Header，并带上请求选项。
func BuildSendInput(d *Draft, requestID string, env SendEnv) (model.SendRequestInput, SendInfo) {
	in := buildPayload(d, requestID, env.Vars)
	in.URL = EnsureProtocol(in.URL)

	set := env.Settings
	suppressed := SuppressedAutoHeaders(d.Headers)
	isSuppressed := func(name string) bool { return suppressed[NormalizeHeaderName(name)] }
	info := SendInfo{ResolvedURL: in.URL}

	if !set.DisableCookies && env.Jar != nil && !isSuppressed("cookie") {
		if header := env.Jar.Header(in.URL); header != "" {
			info.CookieIncluded = true
			merged := false
			for i, h := range in.Headers {
				if strings.EqualFold(h.Key, "cookie") {
					in.Headers[i].Value = MergeCookieHeaders(h.Value, header)
					merged = true
					break
				}
			}
			if !merged {
				in.Headers = append(in.Headers, model.KeyValue{Key: "Cookie", Value: header})
			}
		}
	}
	if set.SendNoCacheHeader && !isSuppressed("cache-control") && !hasHeader(in.Headers, "Cache-Control") {
		in.Headers = append(in.Headers, model.KeyValue{Key: "Cache-Control", Value: "no-cache"})
	}
	tokenSuppressed := isSuppressed("minipost-token") || isSuppressed("postman-token")
	if set.SendPostmanTokenHeader && !tokenSuppressed && !hasHeader(in.Headers, tokenHeaderName) && !hasHeader(in.Headers, "Postman-Token") {
		in.Headers = append(in.Headers, model.KeyValue{Key: tokenHeaderName, Value: uuid.NewString()})
	}

	in.Options = &model.RequestOptions{
		FollowRedirects:         set.FollowRedirects,
		TimeoutMS:               max(0, set.RequestTimeoutMs),
		MaxResponseSizeMB:       max(0, set.MaxResponseSizeMB),
		SSLVerify:               set.SSLVerify,
		SSLTLSKeyLog:            set.SSLTLSKeyLog,
		HTTPVersion:             set.HTTPVersion,
		DisableDefaultUserAgent: isSuppressed("user-agent"),
		DisableDefaultAccept:    isSuppressed("accept"),
		DisableAutoContentType:  isSuppressed("content-type"),
	}
	return in, info
}

// ActiveVars 提取环境中已启用且有键名的变量。
func ActiveVars(env *model.Environment) []Var {
	if env == nil {
		return nil
	}
	var out []Var
	for _, v := range env.Variables {
		if v.Enabled && v.Key != "" {
			out = append(out, Var{Key: v.Key, Value: v.Effective()})
		}
	}
	return out
}
