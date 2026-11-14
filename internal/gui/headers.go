package gui

import (
	"github.com/egoist/mygo/ui"

	"minipost/internal/client"
)

// headersTab 绘制 Headers 子标签：可编辑的键值表，以及系统自动补充的 Header 列表。
func (a *app) headersTab(c *ui.Context, t *tab) {
	d := t.draft
	var visible, markers []*client.Row
	for _, h := range d.Headers {
		if client.IsAutoHeaderMarker(h.Key) {
			markers = append(markers, h)
		} else {
			visible = append(visible, h)
		}
	}
	ui.Column(c).Grow(1).Children(func() {
		a.kvTable(c, &visible, a.t("Header 名", "Header Name"), a.t("Header 值", "Header Value"))
		d.Headers = append(visible, markers...)
		a.autoHeaderList(c, t)
	})
}

// autoHeaderList 列出自动补充的 Header，可折叠，并允许逐项关闭。
func (a *app) autoHeaderList(c *ui.Context, t *tab) {
	th := c.Theme()
	items := client.AutoHeaders(t.draft, a.set, a.jar, a.vars(), a.t)
	ui.Divider(c)
	head := ui.Row(c).Padding(6, 10).Gap(6).AlignItems(ui.Center).Children(func() {
		arrow := "▸"
		if t.autoOpen {
			arrow = "▾"
		}
		ui.Text(c, arrow)
		ui.Textf(c, a.t("自动生成的请求头 (%d)", "Auto-generated headers (%d)"), len(items)).TextColor(th.TextMuted)
	})
	if head.Clicked() {
		t.autoOpen = !t.autoOpen
	}
	if !t.autoOpen {
		return
	}
	ui.Scroll(c).MaxHeight(190).Children(func() {
		ui.Column(c).Gap(4).Padding(4, 8).Children(func() {
			for _, it := range items {
				it := it
				ui.Row(c.Key("auto:" + it.Key)).Gap(6).AlignItems(ui.Center).Children(func() {
					on := it.Enabled
					if ui.Checkbox(c, &on, "").Width(22).Shrink(0).Changed() {
						client.SetAutoHeaderEnabled(t.draft, it.Key, on)
					}
					muted := th.TextMuted
					if on {
						muted = th.Text
					}
					ui.Text(c, it.Key).SingleLine().Grow(1).Basis(0).TextColor(muted)
					ui.Text(c, it.Value).SingleLine().Grow(1).Basis(0).TextColor(muted)
					ui.Text(c, it.Description).SingleLine().Grow(1).Basis(0).TextColor(th.TextMuted)
				})
			}
		})
	})
}
