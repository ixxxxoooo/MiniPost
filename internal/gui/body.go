package gui

import (
	"bytes"
	"encoding/json"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	"minipost/internal/client"
)

// bodyKinds 与界面分段控件一一对应。
var bodyKinds = []string{client.BodyNone, client.BodyJSON, client.BodyRaw, client.BodyFormData, client.BodyURLEncoded}

// bodyTab 绘制 Body 子标签。
func (a *app) bodyTab(c *ui.Context, t *tab) {
	d := t.draft
	kind := 0
	for i, k := range bodyKinds {
		if k == d.Body.Type {
			kind = i
		}
	}
	ui.Column(c).Grow(1).Children(func() {
		ui.Row(c).Gap(8).Padding(8).AlignItems(ui.Center).Children(func() {
			if ui.Segmented(c, &kind, "none", "JSON", "raw", "form-data", "x-www-form-urlencoded").Changed() {
				d.Body.Type = bodyKinds[kind]
			}
			ui.Spacer(c)
			if d.Body.Type == client.BodyJSON && ui.Button(c, a.t("格式化", "Format")).Clicked() {
				a.formatJSON(t)
			}
		})
		switch d.Body.Type {
		case client.BodyJSON:
			editor.View(c, t.jsonEd.st).Grow(1).FillWidth()
		case client.BodyRaw:
			editor.View(c, t.rawEd.st).Grow(1).FillWidth()
		case client.BodyFormData:
			a.formTable(c, &d.Body.FormData)
		case client.BodyURLEncoded:
			a.kvTable(c, &d.Body.URLEncoded, "Key", a.t("值", "Value"))
		default:
			ui.Column(c).Grow(1).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
				ui.Text(c, a.t("此请求没有请求体", "This request has no body")).TextColor(c.Theme().TextMuted)
			})
		}
	})
}

// formatJSON 把 JSON 请求体格式化为两空格缩进；不是合法 JSON 时提示用户。
func (a *app) formatJSON(t *tab) {
	t.flush()
	var out bytes.Buffer
	if err := json.Indent(&out, []byte(t.draft.Body.JSON), "", "  "); err != nil {
		a.toast(a.t("不是合法的 JSON：", "Invalid JSON: ") + err.Error())
		return
	}
	t.jsonEd.st.Buffer.Set(out.String())
}
