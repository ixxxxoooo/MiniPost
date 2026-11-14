package client

import "testing"

func TestParseRequestError(t *testing.T) {
	p := ParseRequestError("[DNS_LOOKUP_FAILED] 域名解析失败: no such host")
	if p.Code != "DNS_LOOKUP_FAILED" || p.Message != "域名解析失败" || p.Detail != "no such host" {
		t.Fatalf("解析不符: %+v", p)
	}
	p = ParseRequestError("[X_Y] 只有消息")
	if p.Code != "X_Y" || p.Message != "只有消息" || p.Detail != "" {
		t.Fatalf("无详情: %+v", p)
	}
	p = ParseRequestError("  boom  ")
	if p.Code != "REQUEST_FAILED" || p.Detail != "boom" {
		t.Fatalf("格式不符应回落: %+v", p)
	}
}

func TestDescribeRequestError(t *testing.T) {
	cases := []struct {
		raw  string
		tone ErrorTone
		desc string
	}{
		{"[DNS_LOOKUP_FAILED] x: y", ToneDanger, "无法解析目标域名，DNS 查询失败。"},
		{"dial tcp: connection refused", ToneDanger, "连接被目标地址拒绝，服务可能未启动或端口不可达。"},
		{"context deadline exceeded", ToneWarning, "请求在超时时间内未完成。"},
		{"x509: certificate signed by unknown authority", ToneWarning, "TLS 握手或证书校验失败。"},
		{"unexpected EOF", ToneDanger, "连接在请求过程中被提前关闭。"},
		{"something odd", ToneDanger, "请求未能成功发送。"},
	}
	for _, c := range cases {
		info := DescribeRequestError(ParseRequestError(c.raw))
		if info.Tone != c.tone || info.DescZH != c.desc || info.SuggestZH == "" || info.DescEN == "" || info.Detail == "" {
			t.Errorf("%q => %+v", c.raw, info)
		}
	}
}
