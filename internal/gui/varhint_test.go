package gui

import (
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	"minipost/internal/client"
	"minipost/internal/model"
)

func varApp(t *testing.T, url string) (*app, *tab, *ui.Tester) {
	a, tt, _ := zhApp(t)
	a.envs = []model.Environment{{ID: "e1", Name: "dev", Variables: []model.Variable{{Key: "baseUrl", Value: "https://x", Enabled: true}}}}
	a.envID = "e1"
	a.openRequest(a.coll.Requests[0].ID)
	tab := a.activeTab()
	tab.draft.URL = url
	tt.Frame()
	return a, tab, tt
}

// 只有当前环境没有定义的变量才算“未定义”。
func TestUndefinedVarsDetectsMissingOnly(t *testing.T) {
	a, tb, _ := varApp(t, "{{baseUrl}}/a?x={{token}}")
	if got := a.undefinedVars(tb); !reflect.DeepEqual(got, []string{"token"}) {
		t.Fatalf("应只报 token: %v", got)
	}
	tb.draft.URL = "{{baseUrl}}/a"
	if got := a.undefinedVars(tb); len(got) != 0 {
		t.Fatalf("变量都已定义时不应提示: %v", got)
	}
	a.envID = ""
	if got := a.undefinedVars(tb); !reflect.DeepEqual(got, []string{"baseUrl"}) {
		t.Fatalf("未选择环境时所有引用都应视为未定义: %v", got)
	}
}

// 含变量的地址在失焦、聚焦两种状态下都能正常渲染且不改动地址内容。
func TestURLBarRendersWithVariables(t *testing.T) {
	_, tb, tt := varApp(t, "{{baseUrl}}/a?x={{token}}")
	tt.Frame()
	if tb.draft.URL != "{{baseUrl}}/a?x={{token}}" {
		t.Fatalf("渲染不应改动地址: %q", tb.draft.URL)
	}
}

func TestVarSpansSplitsText(t *testing.T) {
	a, _, _ := varApp(t, "")
	s := "a{{baseUrl}}b{{nope}}"
	spans := a.varSpans(s, client.FindTemplateRefs(s, a.vars()))
	if len(spans) != 4 {
		t.Fatalf("应切成 4 段: %+v", spans)
	}
	joined := ""
	for _, sp := range spans {
		joined += sp.Text
	}
	if joined != s {
		t.Fatalf("拼回后应与原文一致: %q", joined)
	}
	if spans[1].WavyUnderline || !spans[3].WavyUnderline {
		t.Fatal("已定义变量不应有波浪线，未定义变量应有")
	}
}

// 请求体编辑器里未定义的变量会被标上警告，已定义的不标。
func TestBodyEditorMarksUndefinedVariables(t *testing.T) {
	a, tb, _ := varApp(t, "")
	tb.jsonEd.st.Buffer.Set(`{"u":"{{baseUrl}}","t":"{{token}}"}`)
	a.markUndefinedInEditor(tb.jsonEd)
	ds := tb.jsonEd.st.Diagnostics()
	if len(ds) != 1 || ds[0].Severity != editor.SeverityWarning {
		t.Fatalf("应只有 token 一条警告: %+v", ds)
	}
	if got := tb.jsonEd.st.Buffer.String()[ds[0].Start:ds[0].End]; got != "{{token}}" {
		t.Fatalf("警告范围应覆盖 {{token}}: %q", got)
	}
	// 补上变量后警告消失
	a.envs[0].Variables = append(a.envs[0].Variables, model.Variable{Key: "token", Value: "t", Enabled: true})
	a.markUndefinedInEditor(tb.jsonEd)
	if n := len(tb.jsonEd.st.Diagnostics()); n != 0 {
		t.Fatalf("变量已定义后应清除警告: %d", n)
	}
}

// 变量高亮层不拦截鼠标：点在地址栏上仍能聚焦输入框并直接键入。
func TestURLOverlayDoesNotBlockClickAndTyping(t *testing.T) {
	_, tb, tt := varApp(t, "{{baseUrl}}/a")
	r, ok := tt.Find("{{baseUrl}}/a")
	if !ok {
		t.Fatal("找不到地址栏文字")
	}
	tt.ClickAt(r.X+r.W+200, r.Y+r.H/2)
	tt.Frame()
	tt.Type("Q")
	tt.Frame()
	if tb.draft.URL != "{{baseUrl}}/aQ" {
		t.Fatalf("点击高亮层后应能在输入框末尾键入，实际 %q", tb.draft.URL)
	}
}
