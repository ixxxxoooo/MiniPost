package client

import (
	"net/url"
	"regexp"
	"strings"
)

// ParsedRequestError 是把后端错误文本 `[CODE] 消息: 详情` 拆开后的结果。
type ParsedRequestError struct {
	Code    string
	Message string
	Detail  string
}

// ErrorTone 决定错误徽标的配色：Danger 红色，Warning 橙色。
type ErrorTone int

const (
	ToneDanger ErrorTone = iota
	ToneWarning
)

// RequestErrorInfo 是响应区失败卡片展示的诊断信息，文案为中英文两套。
type RequestErrorInfo struct {
	Tone                 ErrorTone
	Detail               string // 去掉 "Error:" 前缀的错误详情
	DescZH, DescEN       string
	SuggestZH, SuggestEN string
}

var errPrefixRe = regexp.MustCompile(`^\[([A-Z0-9_]+)\]\s*([^:]+)(?::\s*(.*))?$`)

// ParseRequestError 解析错误文本；格式不符时 Code 为 REQUEST_FAILED，详情为原文。
func ParseRequestError(raw string) ParsedRequestError {
	trimmed := strings.TrimSpace(raw)
	m := errPrefixRe.FindStringSubmatch(trimmed)
	if m == nil {
		return ParsedRequestError{Code: "REQUEST_FAILED", Message: "", Detail: trimmed}
	}
	code := m[1]
	if code == "" {
		code = "REQUEST_FAILED"
	}
	return ParsedRequestError{Code: code, Message: strings.TrimSpace(m[2]), Detail: strings.TrimSpace(m[3])}
}

// 请求失败的归类，由 errKind 给出，描述与诊断步骤共用。
const (
	errKindDNS     = "dns"
	errKindRefused = "refused"
	errKindTimeout = "timeout"
	errKindTLS     = "tls"
	errKindClosed  = "closed"
	errKindOther   = "other"
)

// errKind 按错误码与错误文本把失败归到一类（同旧版 getRequestErrorPresentation 的判断顺序）。
func errKind(p ParsedRequestError) string {
	detail := p.Detail
	if detail == "" {
		detail = p.Message
	}
	lower := strings.ToLower(detail)
	has := func(subs ...string) bool {
		for _, s := range subs {
			if strings.Contains(lower, s) {
				return true
			}
		}
		return false
	}
	switch {
	case p.Code == "DNS_LOOKUP_FAILED" || has("no such host"):
		return errKindDNS
	case p.Code == "CONNECTION_REFUSED" || has("connection refused", "econnrefused"):
		return errKindRefused
	case p.Code == "REQUEST_TIMEOUT" || has("timeout", "deadline exceeded"):
		return errKindTimeout
	case p.Code == "TLS_HANDSHAKE_FAILED" || has("x509", "certificate", "tls"):
		return errKindTLS
	case p.Code == "CONNECTION_CLOSED" || has("eof"):
		return errKindClosed
	}
	return errKindOther
}

// DescribeRequestError 按错误码与错误文本给出原因说明与排查建议（同旧版 getRequestErrorPresentation）。
func DescribeRequestError(p ParsedRequestError) RequestErrorInfo {
	detail := p.Detail
	if detail == "" {
		detail = p.Message
	}
	info := RequestErrorInfo{Detail: detail}
	switch errKind(p) {
	case errKindDNS:
		info.DescZH, info.DescEN = "无法解析目标域名，DNS 查询失败。", "Failed to resolve target hostname (DNS lookup failed)."
		info.SuggestZH, info.SuggestEN = "请检查域名拼写、网络 DNS 设置，或尝试切换网络后重试。", "Check domain spelling and DNS settings, or switch network and retry."
	case errKindRefused:
		info.DescZH, info.DescEN = "连接被目标地址拒绝，服务可能未启动或端口不可达。", "Connection was refused by target. Service may be down or port unreachable."
		info.SuggestZH, info.SuggestEN = "请检查目标服务状态、端口、代理或防火墙策略。", "Check service status, port, proxy, or firewall policy."
	case errKindTimeout:
		info.Tone = ToneWarning
		info.DescZH, info.DescEN = "请求在超时时间内未完成。", "Request did not complete within timeout."
		info.SuggestZH, info.SuggestEN = "请增大请求超时、检查网络连通性，或确认服务是否响应较慢。", "Increase timeout, check network connectivity, or verify whether the service responds slowly."
	case errKindTLS:
		info.Tone = ToneWarning
		info.DescZH, info.DescEN = "TLS 握手或证书校验失败。", "TLS handshake or certificate validation failed."
		info.SuggestZH, info.SuggestEN = "请检查证书有效期、证书链配置，或在调试时临时关闭 SSL 校验。", "Check certificate validity and chain configuration, or temporarily disable SSL verification for debugging."
	case errKindClosed:
		info.DescZH, info.DescEN = "连接在请求过程中被提前关闭。", "Connection was closed before the request finished."
		info.SuggestZH, info.SuggestEN = "这通常与网络链路、代理拦截或目标服务异常有关，请查看控制台日志定位。", "Usually related to network path, proxy interception, or target service issues. Check console logs."
	default:
		info.DescZH, info.DescEN = "请求未能成功发送。", "The request could not be sent."
		info.SuggestZH, info.SuggestEN = "请检查 URL、协议、代理、证书或超时设置后重试。", "Check URL, protocol, proxy, certificate, or timeout settings and retry."
	}
	return info
}

// DiagStep 是失败诊断清单里的一步：OK 为通过（✓），否则为失败（✗）。
type DiagStep struct {
	OK     bool
	ZH, EN string
}

// DiagnoseRequestError 把一次失败拆成“URL → 域名解析 → TCP 连接 → TLS/响应”的逐步清单，
// 前面的步骤标为通过，出错的那一步标为失败，让用户一眼看到卡在哪。rawURL 为解析变量后的地址。
func DiagnoseRequestError(p ParsedRequestError, rawURL string) []DiagStep {
	target := strings.TrimSpace(rawURL)
	if target != "" && !strings.Contains(target, "://") {
		target = "http://" + target
	}
	u, err := url.Parse(target)
	if err != nil || u.Host == "" {
		return []DiagStep{{false, "URL 格式不正确", "The URL is malformed"}}
	}
	host, hostPort := u.Hostname(), u.Host
	steps := []DiagStep{{true, "URL 格式正确", "URL is well-formed"}}
	ok := func(zh, en string) { steps = append(steps, DiagStep{true, zh, en}) }
	fail := func(zh, en string) { steps = append(steps, DiagStep{false, zh, en}) }
	switch errKind(p) {
	case errKindDNS:
		fail("域名解析失败 "+host, "DNS lookup failed for "+host)
	case errKindRefused:
		ok("域名解析成功 "+host, "Resolved "+host)
		fail("连接被拒绝 "+hostPort, "Connection refused by "+hostPort)
	case errKindTimeout:
		ok("域名解析成功 "+host, "Resolved "+host)
		fail("请求超时 "+hostPort, "Timed out waiting for "+hostPort)
	case errKindTLS:
		ok("域名解析成功 "+host, "Resolved "+host)
		ok("TCP 连接成功 "+hostPort, "TCP connected to "+hostPort)
		fail("TLS 握手或证书校验失败", "TLS handshake or certificate check failed")
	case errKindClosed:
		ok("域名解析成功 "+host, "Resolved "+host)
		fail("连接被提前关闭 "+hostPort, "Connection closed early by "+hostPort)
	default:
		fail("请求未能发送", "The request could not be sent")
	}
	return steps
}
