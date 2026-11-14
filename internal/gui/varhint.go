package gui

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	"minipost/internal/client"
)

// varSpans 把 s 切成富文本片段：{{变量}} 已定义的用强调色，未定义的用警告色加波浪线。
func (a *app) varSpans(s string, refs []client.TemplateRef) []ui.Span {
	p := a.p
	var out []ui.Span
	pos := 0
	for _, r := range refs {
		if r.Start > pos {
			out = append(out, ui.Span{Text: s[pos:r.Start]})
		}
		sp := ui.Span{Text: s[r.Start:r.End], Radius: 3}
		if r.Defined {
			sp.Color, sp.Background = p.accent, p.accent.Alpha(0.12)
		} else {
			sp.Color, sp.Background = p.warning, p.warning.Alpha(0.14)
			sp.WavyUnderline, sp.DecorationColor = true, p.warning
		}
		out = append(out, sp)
		pos = r.End
	}
	if pos < len(s) {
		out = append(out, ui.Span{Text: s[pos:]})
	}
	return out
}

// urlOverlay 在失焦的地址输入框上盖一层带变量高亮的文字（不拦截鼠标，点击仍落到输入框上）。
func (a *app) urlOverlay(c *ui.Context, url string, refs []client.TemplateRef) {
	ui.Row(c.Key("urlhl")).Absolute().Top(0).Bottom(2).Left(12).Right(12).AlignItems(ui.Center).Clip().PassThrough().Children(func() {
		// 与下面输入框的内容重复，对辅助技术隐藏，只读输入框
		ui.RichText(c, a.varSpans(url, refs)...).NoWrap().Font(monoFont).FontSize(fsVXs).TextColor(a.p.fg).AccessHidden(true)
	})
}

// undefinedVars 返回请求里引用了、但当前环境没有定义的变量名。
func (a *app) undefinedVars(t *tab) []string {
	if t == nil || t.env != nil {
		return nil
	}
	return client.UndefinedVarNames(a.vars(), t.draft.TemplateTexts()...)
}

// undefinedHint 在地址栏右端显示一个警告图标，悬停时列出未定义的变量；没有则不显示。
func (a *app) undefinedHint(c *ui.Context, t *tab) {
	names := a.undefinedVars(t)
	if len(names) == 0 {
		return
	}
	tip := a.t("未定义的变量：", "Undefined variables: ") + strings.Join(names, ", ")
	if a.env() == nil {
		tip += a.t("（当前未选择环境）", " (no environment selected)")
	}
	ui.Row(c.Key("undefhint")).Width(28).FillHeight().Shrink(0).AlignItems(ui.Center).Justify(ui.Center).Tooltip(tip).Children(func() {
		icon(c, "info", 14, a.p.warning)
	})
}

// markUndefinedInEditor 给请求体编辑器里未定义的 {{变量}} 标上警告波浪线。
// 仅在文本或环境变量变化时重新计算。
func (a *app) markUndefinedInEditor(b *bodyEditor) {
	vars := a.vars()
	var sig strings.Builder
	for _, v := range vars {
		sig.WriteString(v.Key)
		sig.WriteByte(0)
	}
	ver := b.st.Buffer.Version()
	if b.diagInit && b.diagVer == ver && b.diagSig == sig.String() {
		return
	}
	b.diagInit, b.diagVer, b.diagSig = true, ver, sig.String()
	var ds []editor.Diagnostic
	for _, r := range client.FindTemplateRefs(b.st.Buffer.String(), vars) {
		if !r.Defined {
			ds = append(ds, editor.Diagnostic{Start: r.Start, End: r.End, Severity: editor.SeverityWarning,
				Message: a.t("未定义的变量：", "Undefined variable: ") + r.Name, Source: "MiniPost"})
		}
	}
	b.st.SetDiagnostics(ver, ds)
}
