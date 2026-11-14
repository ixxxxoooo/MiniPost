package client

import (
	"strings"
	"testing"
)

func urlOf(d *Draft, vars []Var) string { return buildURLWithQuery(d, vars, formEscape) }

// 表格里的变量先解析再编码；同一个参数只出现一次。
func TestBuildURLWithQueryResolvesTableVars(t *testing.T) {
	d := baseDraft()
	d.URL = "https://x.test/p"
	d.Params = []*Row{row("token", "{{tk}}", true), row("{{kn}}", "v", true), NewRow()}
	d.URL = BuildURLWithParams(d.URL, d.Params)
	got := urlOf(d, []Var{{"tk", "a b&c"}, {"kn", "key"}})
	if got != "https://x.test/p?token=a+b%26c&key=v" {
		t.Fatalf("got %s", got)
	}
}

// 地址栏手写的查询串和同步后的表格不会重复；表格独有的参数追加在后面。
func TestBuildURLWithQueryDedupAndAppend(t *testing.T) {
	d := baseDraft()
	d.URL = "https://x.test/p?a=1&b={{v}}"
	d.Params = []*Row{row("a", "1", true), row("b", "{{v}}", true), row("c", "3", true), row("off", "x", false), row("", "nokey", true), NewRow()}
	if got := urlOf(d, []Var{{"v", "2"}}); got != "https://x.test/p?a=1&b=2&c=3" {
		t.Fatalf("got %s", got)
	}
}

// 保持顺序、同名参数、没有等号的参数、空值。
func TestBuildURLWithQueryOrderAndShape(t *testing.T) {
	d := baseDraft()
	d.URL = "https://x.test/p?z=1&a=2&z=3&flag&empty="
	if got := urlOf(d, nil); got != "https://x.test/p?z=1&a=2&z=3&flag=&empty=" {
		t.Fatalf("got %s", got)
	}
}

// 没有参数时不留下 ?；锚点保留；变量展开后自带查询串时用 & 连接。
func TestBuildURLWithQueryHeadVariants(t *testing.T) {
	d := baseDraft()
	d.URL = "https://x.test/p#top"
	if got := urlOf(d, nil); got != "https://x.test/p#top" {
		t.Fatalf("got %s", got)
	}
	d.URL = "{{base}}"
	d.Params = []*Row{row("b", "2", true), NewRow()}
	if got := urlOf(d, []Var{{"base", "https://x.test/p?a=1"}}); got != "https://x.test/p?a=1&b=2" {
		t.Fatalf("got %s", got)
	}
	if got := urlOf(d, []Var{{"base", "https://x.test/p?"}}); got != "https://x.test/p?b=2" {
		t.Fatalf("got %s", got)
	}
}

// 动态变量在构建地址时求值，且只求值一次。
func TestBuildURLWithQueryDynamicVars(t *testing.T) {
	d := baseDraft()
	d.Params = []*Row{row("id", "{{$randomUUID}}", true), NewRow()}
	d.URL = BuildURLWithParams("https://x.test/p", d.Params)
	got := urlOf(d, nil)
	if strings.Count(got, "id=") != 1 || strings.Contains(got, "%7B") || len(got) != len("https://x.test/p?id=")+36 {
		t.Fatalf("got %s", got)
	}
}

// 未定义的变量原样保留（并被编码），不丢参数。
func TestBuildURLWithQueryUndefinedVarKept(t *testing.T) {
	d := baseDraft()
	d.URL = "https://x.test/p?a={{nope}}"
	if got := urlOf(d, nil); got != "https://x.test/p?a=%7B%7Bnope%7D%7D" {
		t.Fatalf("got %s", got)
	}
}

// cURL 导出：查询参数只出现一次，变量已解析，其余原有行为不变。
func TestBuildCurlCommandParamsNotDuplicated(t *testing.T) {
	d := baseDraft()
	d.URL = "https://x.test/p"
	d.Params = []*Row{row("a", "1", true), row("t", "{{tk}}", true), row("off", "x", false), NewRow()}
	d.URL = BuildURLWithParams(d.URL, d.Params)
	cmd := BuildCurlCommand(d, []Var{{"tk", "S S"}}, "")
	if !strings.Contains(cmd, "'https://x.test/p?a=1&t=S%20S'") {
		t.Fatalf("cURL 应只含一份查询参数: %s", cmd)
	}
}

// 关闭自动 Header 的标记行即使处于启用状态，也不会进入发送的 Header。
func TestEnabledKVSkipsAutoHeaderMarker(t *testing.T) {
	rows := []*Row{row(AutoHeaderMarkerKey("Accept"), "", true), row("X-A", "1", true)}
	got := enabledKV(rows)
	if len(got) != 1 || got[0].Key != "X-A" {
		t.Fatalf("got %+v", got)
	}
}

// 地址栏里是整条 cURL 命令时原样交给后端，不补 http://、不拆查询串。
func TestCurlCommandInURLPassesThrough(t *testing.T) {
	cmd := `curl -X POST 'https://x.test/p?a=1&b=2' -H 'X: y'`
	d := baseDraft()
	d.URL = cmd
	d.Params = []*Row{row("extra", "1", true), NewRow()}
	in, _ := BuildSendInput(d, "r", SendEnv{Settings: DefaultSettings(), Jar: NewJar("")})
	if in.URL != cmd {
		t.Fatalf("cURL 命令应原样透传，实际 %q", in.URL)
	}
	if !IsCurlCommand(cmd) || IsCurlCommand("curl") || IsCurlCommand("https://x.test?q=curl x") || IsCurlCommand("curlx -a") {
		t.Fatalf("IsCurlCommand 判断有误")
	}
}
