package service

import (
	"net/http"
	"testing"

	"minipost/internal/model"
)

func pairs(kv ...string) []model.KeyValue {
	var out []model.KeyValue
	for i := 0; i+1 < len(kv); i += 2 {
		out = append(out, model.KeyValue{Key: kv[i], Value: kv[i+1]})
	}
	return out
}

func TestEncodePairsKeepsOrderAndSkipsBlankKeys(t *testing.T) {
	got := encodePairs(pairs("z", "1", "a", "b c", "z", "2", "", "x", "  ", "y", "名", "值&"))
	if want := "z=1&a=b+c&z=2&%E5%90%8D=%E5%80%BC%26"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if encodePairs(nil) != "" {
		t.Fatalf("空输入应得到空串")
	}
}

func TestAppendQuery(t *testing.T) {
	cases := []struct{ raw, want string }{
		{"", "k=v"},
		{"a=1", "a=1&k=v"},
		{"a=1&", "a=1&k=v"},
		{"z=%20&a=b+c", "z=%20&a=b+c&k=v"}, // 已有部分原样保留
	}
	for _, c := range cases {
		if got := appendQuery(c.raw, pairs("k", "v")); got != c.want {
			t.Errorf("appendQuery(%q) = %q, want %q", c.raw, got, c.want)
		}
	}
	if got := appendQuery("a=1", nil); got != "a=1" {
		t.Errorf("没有追加项时应原样返回，实际 %q", got)
	}
}

func TestSetQueryParamReplacesAndKeepsOthers(t *testing.T) {
	cases := []struct{ raw, want string }{
		{"", "key=v"},
		{"z=1&a=2", "z=1&a=2&key=v"},
		{"key=old&z=1", "z=1&key=v"},
		{"z=1&key=a&key=b&m=2", "z=1&m=2&key=v"},
		{"k%65y=old", "key=v"}, // 编码后的同名键也被替换
		{"%zz=1&a=2", "%zz=1&a=2&key=v"},
	}
	for _, c := range cases {
		if got := setQueryParam(c.raw, "key", "v"); got != c.want {
			t.Errorf("setQueryParam(%q) = %q, want %q", c.raw, got, c.want)
		}
	}
}

func TestBuildURL(t *testing.T) {
	s := NewHttpService()
	cases := []struct {
		name, in string
		params   []model.KeyValue
		want     string
	}{
		{"补协议", "example.com/p", nil, "http://example.com/p"},
		{"大写协议不重复补", "HTTPS://example.com/p", nil, "https://example.com/p"},
		{"首尾空白", "  http://example.com/p  ", nil, "http://example.com/p"},
		{"已有查询串原样保留", "http://e.com/p?z=1&a=b%20c", pairs("m", "2"), "http://e.com/p?z=1&a=b%20c&m=2"},
		{"参数保持顺序", "http://e.com/p", pairs("z", "1", "a", "2", "z", "3"), "http://e.com/p?z=1&a=2&z=3"},
		{"跳过空键名", "http://e.com/p", pairs("", "x", "k", "v"), "http://e.com/p?k=v"},
		{"保留锚点", "http://e.com/p?a=1#top", pairs("b", "2"), "http://e.com/p?a=1&b=2#top"},
	}
	for _, c := range cases {
		got, err := s.buildURL(c.in, c.params)
		if err != nil || got != c.want {
			t.Errorf("%s: buildURL(%q) = %q, %v; want %q", c.name, c.in, got, err, c.want)
		}
	}
}

func TestBuildBodyURLEncodedOrdered(t *testing.T) {
	s := NewHttpService()
	r, ct, err := s.buildBody(model.RequestBody{Type: "form-urlencoded", FormUrlEncoded: pairs("z", "1", "a", "x y", "", "skip")})
	if err != nil || ct != "application/x-www-form-urlencoded" {
		t.Fatalf("ct=%q err=%v", ct, err)
	}
	buf := make([]byte, 64)
	n, _ := r.Read(buf)
	if got := string(buf[:n]); got != "z=1&a=x+y" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyAuthAPIKeyQueryKeepsExisting(t *testing.T) {
	s := NewHttpService()
	req, _ := http.NewRequest("GET", "http://e.com/p?z=1&a=2", nil)
	s.applyAuth(req, model.AuthConfig{Type: "api-key", APIKey: model.APIKeyAuth{Key: "key", Value: "a b", AddTo: "query"}})
	if got := req.URL.RawQuery; got != "z=1&a=2&key=a+b" {
		t.Fatalf("got %q", got)
	}
}
