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
	// 类型分段控件：与 Params / Headers 共用同一条 34 高的子工具条（右侧放该类型的动作）
	typeSeg := func() {
		if a.pillSeg(c, "body", &kind, []string{"none", "form-data", "raw", "JSON", "urlencoded"}, nil, segOpts{h: 22, padX: 11, fs: fsSmall + 0.5}) {
			d.Body.Type = bodyKinds[kind]
		}
	}
	ui.Column(c).Grow(1).Basis(0).Children(func() {
		if d.Body.Type != client.BodyURLEncoded {
			a.subBar(c, typeSeg, func() {
				if d.Body.Type == client.BodyJSON || d.Body.Type == client.BodyRaw {
					if a.button(c, btnOpts{padX: 6, h: 22, gap: 4, noHover: true, key: "beautify"}, func(hover bool) {
						col := p.accent
						if hover {
							col = p.accentHover
						}
						icon(c, "sparkle", 12, col)
						txt(c, a.t("格式化", "Beautify"), fsSmall, col, 600)
					}) {
						a.formatBody(t)
					}
				}
			})
		}
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
			a.kvTableOpts(c, &d.Body.URLEncoded, kvOpts{id: d.ID + ":urlenc", keyPH: a.t("键", "Key"), valPH: a.t("值", "Value"), noDesc: true, lead: typeSeg})
		default:
			// 没有请求体：居中的图标与两行提示
			ui.Column(c).FillWidth().Grow(1).Basis(0).Gap(6).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
				ui.Row(c).Size(36, 36).Radius(18).Background(p.surface2).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
					icon(c, "cube", 16, p.muted)
				})
				txt(c, a.t("此请求没有请求体", "This request has no body"), fsBase, p.fg2, 500)
				txt(c, a.t("在上方选择 JSON、form-data 等类型即可添加", "Pick JSON, form-data, etc. above to add one"), fsSmall, p.muted, 0)
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
