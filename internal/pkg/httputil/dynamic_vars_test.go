package httputil

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"minipost/internal/model"
)

func withFixedClock(t *testing.T) {
	t.Helper()
	oldNow, oldInt, oldUUID := nowFunc, randIntFunc, uuidFunc
	nowFunc = func() time.Time { return time.Date(2026, 11, 14, 10, 30, 0, 123_000_000, time.UTC) }
	n := 0
	randIntFunc = func() int { n++; return 40 + n }
	m := 0
	uuidFunc = func() string { m++; return "uuid-" + string(rune('0'+m)) }
	t.Cleanup(func() { nowFunc, randIntFunc, uuidFunc = oldNow, oldInt, oldUUID })
}

func TestResolveDynamicVars(t *testing.T) {
	withFixedClock(t)
	cases := map[string]string{
		"{{$timestamp}}":     "1794652200",
		"{{$timestampMs}}":   "1794652200123",
		"{{$isoTimestamp}}":  "2026-11-14T10:30:00Z",
		"t={{ $timestamp }}": "t=1794652200", // 允许花括号内有空白
		"{{$unknownThing}}":  "{{$unknownThing}}",
		"{{notDynamic}}":     "{{notDynamic}}",
		"无变量":                "无变量",
	}
	for in, want := range cases {
		if got := ResolveVariables(in, nil); got != want {
			t.Errorf("ResolveVariables(%q)=%q, 想要 %q", in, got, want)
		}
	}
}

// 随机类变量按调用顺序取值（测试里用递增的假随机源，所以单独验证，避免与 map 遍历顺序耦合）。
func TestResolveRandomDynamicVars(t *testing.T) {
	withFixedClock(t)
	if got := ResolveVariables("{{$randomInt}}", nil); got != "41" {
		t.Fatalf("randomInt: %q", got)
	}
	if got := ResolveVariables("{{$randomUUID}}", nil); got != "uuid-1" {
		t.Fatalf("randomUUID: %q", got)
	}
}

// 同一变量出现多次时，每次单独求值。
func TestResolveDynamicVarsEvaluatedPerOccurrence(t *testing.T) {
	withFixedClock(t)
	got := ResolveVariables("{{$randomUUID}}/{{$randomUUID}}", nil)
	if got != "uuid-1/uuid-2" {
		t.Fatalf("两次 randomUUID 应不同: %q", got)
	}
}

// 环境变量优先：环境里自己定义了同名的 $ 变量时，用环境的值。
func TestEnvironmentVariableWinsOverDynamic(t *testing.T) {
	withFixedClock(t)
	vars := []model.Variable{{Key: "$timestamp", Value: "fixed", Enabled: true}}
	if got := ResolveVariables("{{$timestamp}}", vars); got != "fixed" {
		t.Fatalf("环境变量应优先: %q", got)
	}
}

// 整个请求（地址、请求头、请求体、鉴权）里的动态变量都会被求值。
func TestResolveRequestInputDynamicVars(t *testing.T) {
	withFixedClock(t)
	vars := []model.Variable{{Key: "baseUrl", Value: "https://x.test", Enabled: true}}
	in := model.SendRequestInput{
		URL:     "{{baseUrl}}/t/{{$timestamp}}",
		Headers: []model.KeyValue{{Key: "X-Id", Value: "{{$randomUUID}}"}},
		Body:    model.RequestBody{Type: "json", JSON: `{"at":"{{$isoTimestamp}}"}`},
		Auth:    model.AuthConfig{Type: "bearer", Bearer: model.BearerAuth{Token: "{{$randomInt}}"}},
	}
	out := ResolveRequestInput(in, vars)
	if out.URL != "https://x.test/t/1794652200" {
		t.Fatalf("URL: %q", out.URL)
	}
	if out.Headers[0].Value != "uuid-1" || out.Body.JSON != `{"at":"2026-11-14T10:30:00Z"}` || out.Auth.Bearer.Token != "41" {
		t.Fatalf("其余字段未求值: %+v", out)
	}
	if strings.Contains(in.URL, "1794652200") {
		t.Fatal("不应修改传入的原始请求")
	}
}

func TestIsDynamicVar(t *testing.T) {
	for _, n := range DynamicVarNames {
		if !IsDynamicVar("$" + n) {
			t.Errorf("$%s 应是动态变量", n)
		}
	}
	for _, n := range []string{"", "$", "timestamp", "$nope", "$ timestamp"} {
		if IsDynamicVar(n) {
			t.Errorf("%q 不应是动态变量", n)
		}
	}
}

// 真实时钟下的格式健全性检查。
func TestDynamicValuesRealClockShape(t *testing.T) {
	if got := ResolveVariables("{{$timestamp}}", nil); !regexp.MustCompile(`^\d{10}$`).MatchString(got) {
		t.Fatalf("timestamp 应是 10 位秒: %q", got)
	}
	if got := ResolveVariables("{{$randomUUID}}", nil); !regexp.MustCompile(`^[0-9a-f-]{36}$`).MatchString(got) {
		t.Fatalf("randomUUID 格式不对: %q", got)
	}
}
