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

// 诊断清单：出错之前的步骤为通过，出错的那一步为失败，且主机名写进文案。
func TestDiagnoseRequestError(t *testing.T) {
	cases := []struct {
		name, err, url string
		okCount        int // 通过的步骤数（最后一步一定失败）
		lastZH         string
	}{
		{"dns", "[DNS_LOOKUP_FAILED] 请求失败: lookup api.exmaple.dev: no such host", "https://api.exmaple.dev/x", 1, "域名解析失败 api.exmaple.dev"},
		{"refused", "dial tcp 127.0.0.1:1: connection refused", "http://127.0.0.1:1/a", 2, "连接被拒绝 127.0.0.1:1"},
		{"timeout", "context deadline exceeded", "example.com/a", 2, "请求超时 example.com"},
		{"tls", "x509: certificate signed by unknown authority", "https://self.signed", 3, "TLS 握手或证书校验失败"},
		{"closed", "unexpected EOF", "https://a.b", 2, "连接被提前关闭 a.b"},
		{"other", "boom", "https://a.b", 1, "请求未能发送"},
	}
	for _, c := range cases {
		steps := DiagnoseRequestError(ParseRequestError(c.err), c.url)
		if len(steps) != c.okCount+1 {
			t.Fatalf("%s: 步骤数 %d，应为 %d：%+v", c.name, len(steps), c.okCount+1, steps)
		}
		for i, s := range steps[:c.okCount] {
			if !s.OK {
				t.Errorf("%s: 第 %d 步应通过", c.name, i)
			}
		}
		if last := steps[len(steps)-1]; last.OK || last.ZH != c.lastZH {
			t.Errorf("%s: 最后一步应为失败「%s」，实际 %+v", c.name, c.lastZH, last)
		}
	}
	if s := DiagnoseRequestError(ParsedRequestError{}, ""); len(s) != 1 || s[0].OK {
		t.Errorf("空 URL 应只给出一步失败：%+v", s)
	}
}
