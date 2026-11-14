package gui

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"minipost/internal/client"
)

// requestView 是当前标签的编辑区：地址栏、请求参数与响应。
func (a *app) requestView(c *ui.Context, t *tab) {
	ui.Column(c).Grow(1).Children(func() {
		a.urlBar(c, t)
		ui.Divider(c)
		reqPane := func() { a.requestPane(c, t) }
		respPane := func() { a.responsePane(c, t) }
		var split ui.Element
		if a.set.LayoutDirection == "horizontal" {
			split = ui.SplitPercent(c, &a.splitPct, reqPane, respPane)
		} else {
			split = ui.SplitVerticalPercent(c, &a.splitPct, reqPane, respPane)
		}
		split.Grow(1)
	})
}

// urlBar 绘制方法选择、地址输入与发送/保存按钮。
func (a *app) urlBar(c *ui.Context, t *tab) {
	d := t.draft
	ui.Row(c).Gap(6).Padding(8).AlignItems(ui.Center).Children(func() {
		ui.Select(c, &d.Method, client.Methods).Width(100).Shrink(0)
		url := ui.TextInput(c, &d.URL).Label("URL").Placeholder(a.t("输入请求地址，例如 https://api.example.com/users", "Enter request URL, e.g. https://api.example.com/users")).Grow(1)
		if url.Changed() {
			d.Params = client.SyncParamsWithURL(d.URL, d.Params)
		}
		if url.Submitted() && !t.sending {
			a.later(func() { a.sendTab(t) })
		}
		if t.sending {
			if ui.Button(c, a.t("取消", "Cancel")).Width(72).Shrink(0).Clicked() {
				a.be.CancelHTTPRequest(t.sendID)
			}
		} else if ui.PrimaryButton(c, a.t("发送", "Send")).Width(72).Shrink(0).Tooltip("⌘↩").Clicked() {
			a.later(func() { a.sendTab(t) })
		}
		if ui.Button(c, a.t("保存", "Save")).Disabled(!t.dirty).Tooltip("⌘S").Clicked() {
			a.later(func() { a.saveTab(t) })
		}
		ui.MenuButton(c, "···", func(m *ui.Menu) { a.requestMenu(m, t) }).Shrink(0)
	})
}

func (a *app) requestMenu(m *ui.Menu, t *tab) {
	if m.Item(a.t("复制为 cURL", "Copy as cURL")).Chosen() {
		a.later(func() { a.copyCurl(t) })
	}
	if m.Item(a.t("导入 cURL…", "Import cURL…")).Chosen() {
		a.later(func() { a.overlays.curl = curlState{open: true} })
	}
}

// requestPane 是请求参数区：Params / Headers / Body / Auth 子标签。
func (a *app) requestPane(c *ui.Context, t *tab) {
	d := t.draft
	count := func(rows []*client.Row, skipMarkers bool) int {
		n := 0
		for _, r := range rows {
			if r.Enabled && !r.IsPlaceholder() && !(skipMarkers && client.IsAutoHeaderMarker(r.Key)) {
				n++
			}
		}
		return n
	}
	ui.Column(c).Grow(1).Children(func() {
		labels := []string{
			withCount("Params", count(d.Params, false)),
			withCount("Headers", count(d.Headers, true)),
			"Body",
			"Auth",
		}
		ui.Tabs(c, &t.sub, labels...).Padding(0, 8)
		ui.Divider(c)
		switch t.sub {
		case 0:
			if a.kvTable(c, &d.Params, "Key", a.t("值", "Value")) {
				d.URL = client.BuildURLWithParams(d.URL, d.Params)
			}
		case 1:
			a.headersTab(c, t)
		case 2:
			a.bodyTab(c, t)
		case 3:
			a.authTab(c, t)
		}
	})
}

func withCount(label string, n int) string {
	if n == 0 {
		return label
	}
	return fmt.Sprintf("%s (%d)", label, n)
}
