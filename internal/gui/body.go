package gui

import (
	"bytes"
	"encoding/json"

	"github.com/egoist/mygo/ui"

	"minipost/internal/client"
)

// bodyKinds 与界面类型标签一一对应（顺序同旧版：none、form-data、raw、JSON、x-www-form-urlencoded）。
var bodyKinds = []string{client.BodyNone, client.BodyFormData, client.BodyRaw, client.BodyJSON, client.BodyURLEncoded}

// bodyTab 绘制 Body 子标签：类型标签行（右侧是 Beautify）与对应的编辑区。
func (a *app) bodyTab(c *ui.Context, t *tab) {
	p := a.p
	d := t.draft
	kind := 0
	for i, k := range bodyKinds {
		if k == d.Body.Type {
			kind = i
		}
	}
	ui.Column(c).Grow(1).Basis(0).Children(func() {
		ui.Row(c).FillWidth().Shrink(0).PaddingX(8).Padding(6, 8).Gap(4).AlignItems(ui.Center).Children(func() {
			if a.kindPills(c, "body", &kind, []string{"none", "form-data", "raw", "JSON", "urlencoded"}, 10) {
				d.Body.Type = bodyKinds[kind]
			}
			ui.Spacer(c)
			if d.Body.Type == client.BodyJSON || d.Body.Type == client.BodyRaw {
				if a.button(c, btnOpts{padX: 6, h: 20, gap: 4, noHover: true, key: "beautify"}, func(hover bool) {
					col := p.accent
					if hover {
						col = p.accentHover
					}
					icon(c, "formatList", 11, col)
					txt(c, a.t("格式化", "Beautify"), fsSmall, col, 500)
				}) {
					a.formatBody(t)
				}
			}
		})
		switch d.Body.Type {
		case client.BodyJSON:
			a.markUndefinedInEditor(t.jsonEd)
			a.codeView(c, t.jsonEd.st).Grow(1).FillWidth()
		case client.BodyRaw:
			a.markUndefinedInEditor(t.rawEd)
			a.codeView(c, t.rawEd.st).Grow(1).FillWidth()
		case client.BodyFormData:
			a.formTable(c, &d.Body.FormData)
		case client.BodyURLEncoded:
			a.kvTableOpts(c, &d.Body.URLEncoded, kvOpts{id: d.ID + ":urlenc", keyPH: "Key", valPH: "Value", noDesc: true})
		default:
			ui.Row(c).FillWidth().Padding(32, 0).Justify(ui.Center).Children(func() {
				txt(c, a.t("此请求没有请求体", "This request has no body"), 10.4, p.muted, 0)
			})
		}
	})
}

// formatBody 格式化当前请求体：JSON 缩进两空格，raw 尝试按 JSON 处理，失败时提示。
func (a *app) formatBody(t *tab) {
	if t.draft.Body.Type == client.BodyRaw {
		t.flush()
		var out bytes.Buffer
		if err := json.Indent(&out, []byte(t.draft.Body.Raw), "", "  "); err != nil {
			a.toastWarn(a.t("不是合法的 JSON：", "Invalid JSON: ") + err.Error())
			return
		}
		t.rawEd.st.Buffer.Set(out.String())
		return
	}
	a.formatJSON(t)
}

// formatJSON 把 JSON 请求体格式化为两空格缩进；不是合法 JSON 时提示用户。
func (a *app) formatJSON(t *tab) {
	t.flush()
	var out bytes.Buffer
	if err := json.Indent(&out, []byte(t.draft.Body.JSON), "", "  "); err != nil {
		a.toastWarn(a.t("不是合法的 JSON：", "Invalid JSON: ") + err.Error())
		return
	}
	t.jsonEd.st.Buffer.Set(out.String())
}
