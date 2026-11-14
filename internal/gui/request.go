package gui

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"minipost/internal/client"
)

// requestView 是当前标签的编辑区：面包屑、地址栏，下方是请求区与响应区的分栏。
func (a *app) requestView(c *ui.Context, t *tab) {
	p := a.p
	ui.Column(c).Grow(1).Basis(0).Background(p.surface).Children(func() {
		a.breadcrumb(c, t)
		a.urlBar(c, t)
		reqPane := func() { a.requestPane(c, t) }
		respPane := func() { a.responsePane(c, t) }
		var split ui.Element
		if a.set.LayoutDirection == "horizontal" {
			split = ui.SplitPercent(c, &a.splitPct, reqPane, respPane)
		} else {
			split = ui.SplitVerticalPercent(c, &a.splitPct, reqPane, respPane)
		}
		split.Grow(1).Basis(0)
	})
}

// breadcrumbSegments 返回“项目 > 文件夹…”路径（不含请求名）。
func (a *app) breadcrumbSegments(t *tab) []string {
	var segs []string
	if pr := a.project(); pr != nil {
		segs = append(segs, pr.Name)
	}
	var chain []string
	if n, ok := a.tree.nodes[t.draft.ID]; ok {
		for id := n.parent; id != ""; id = a.tree.nodes[id].parent {
			chain = append([]string{a.tree.nodes[id].name}, chain...)
		}
	}
	return append(segs, chain...)
}

// breadcrumb 是地址栏上方的一行：HTTP 徽标、路径、可点击改名的请求名，以及保存/复制 cURL。
func (a *app) breadcrumb(c *ui.Context, t *tab) {
	p := a.p
	sky, skyBg := ui.RGB(2, 132, 199), ui.RGBA(14, 165, 233, 0.15)
	if p.dark {
		sky = ui.RGB(56, 189, 248)
	}
	ui.Row(c).FillWidth().Shrink(0).Padding(4, 12, 0, 12).Children(func() {
		ui.Row(c).FillWidth().Height(28).PaddingX(8).Gap(4).AlignItems(ui.Center).Children(func() {
			ui.Row(c).Height(16).MinWidth(24).PaddingX(4).Radius(4).Background(skyBg).AlignItems(ui.Center).Justify(ui.Center).Shrink(0).Children(func() {
				txtMono(c, "HTTP", 9, sky, 600)
			})
			for i, seg := range a.breadcrumbSegments(t) {
				ui.Row(c.Key("seg" + itoa(i))).Height(24).PaddingX(6).Radius(6).AlignItems(ui.Center).Shrink(0).Children(func() {
					txt(c, seg, fsBase, p.muted, 0).MaxWidth(220).SingleLine()
				})
				icon(c, "arrowRight", 10, p.muted).MarginX(2)
			}
			a.requestName(c, t)
			ui.Row(c).MarginStart(4).Height(24).Gap(4).AlignItems(ui.Center).Shrink(0).Children(func() {
				dirty := t.dirty
				col := p.muted
				if dirty {
					col = p.fg2
				}
				hv := p.sideHover
				if a.button(c, btnOpts{h: 24, radius: 6, padX: 8, gap: 4, disabled: !dirty, hoverBg: &hv, tip: a.t("保存", "Save") + " (⌘S)", key: "bc-save"}, func(bool) {
					icon(c, "save", 12, col)
					txt(c, "Save", fsSmall, col, 500)
				}) {
					a.later(func() { a.saveTab(t) })
				}
				cc := p.fg2
				noURL := strings.TrimSpace(t.draft.URL) == ""
				if noURL {
					cc = p.muted
				}
				if a.button(c, btnOpts{w: 24, h: 24, radius: 6, disabled: noURL, hoverBg: &hv, tip: a.t("复制为 cURL", "Copy as cURL"), key: "bc-curl"}, func(bool) {
					icon(c, "copy", 12, cc)
				}) {
					a.later(func() { a.copyCurl(t) })
				}
			})
		})
	})
}

// requestName 是面包屑末尾的请求名：点击后原地编辑，回车或失焦提交，Esc 取消。
func (a *app) requestName(c *ui.Context, t *tab) {
	p := a.p
	if !t.editName {
		hv := p.selected
		btn := ui.ButtonBase(c.Key("reqname")).Height(24).Radius(6).PaddingX(6).AlignItems(ui.Center).Justify(ui.Start).Tooltip(a.t("单击编辑请求名称", "Click to edit request name"))
		if btn.Hovered() {
			btn = btn.Background(hv)
		}
		btn.Children(func() { txt(c, t.title(), fsBase, p.fg, 600).SingleLine() })
		if btn.Clicked() {
			t.editName, t.nameBuf = true, t.draft.Name
		}
		ui.Spacer(c)
		return
	}
	in := ui.TextInputBase(c.Key("reqname-edit"), &t.nameBuf).FontSize(fsBase).FontWeight(600).TextColor(p.fg).Height(24).PaddingX(6).Radius(6).
		Border(1, p.accent.Alpha(0.4)).MinWidth(160).AutoFocus()
	commit := in.Submitted() || in.FocusLeft()
	if in.Shortcut(0, ui.KeyEscape) {
		t.editName = false
		return
	}
	if commit {
		t.editName = false
		name := strings.TrimSpace(t.nameBuf)
		if name != "" && name != t.draft.Name {
			t.draft.Name = name
		}
	}
	ui.Spacer(c)
}

// urlBar 是地址栏：方法选择 + URL 输入（共用一个圆角边框）与 Send 按钮组。
func (a *app) urlBar(c *ui.Context, t *tab) {
	p := a.p
	d := t.draft
	ui.Row(c).FillWidth().Shrink(0).Padding(4, 12).Gap(8).AlignItems(ui.Center).Children(func() {
		box := ui.Row(c.Key("urlbox")).Grow(1).Basis(0).Height(32).Radius(10).Background(p.surface).AlignItems(ui.Stretch)
		bc := p.btnBorder
		if box.FocusWithin() {
			bc = p.accent
		}
		box.Border(1, bc).Children(func() {
			mc := p.method(d.Method)
			ui.ButtonBase(c.Key("method")).Width(96).Shrink(0).PaddingX(12).AlignItems(ui.Center).Justify(ui.SpaceBetween).
				BorderWidth(0, 1, 0, 0).BorderColor(p.btnBorder).
				PopoverMenuButton(func(m *ui.Menu) {
					for _, mm := range client.Methods {
						mm := mm
						if m.Item(mm).Checked(d.Method == mm).Chosen() {
							a.later(func() { d.Method = mm })
						}
					}
				}).Children(func() {
				txtMono(c, d.Method, fsSmall, mc, 600)
				icon(c, "arrowDown", 14, mc)
			})
			url := ui.TextInputBase(c.Key("url"), &d.URL).Placeholder(a.t("输入请求 URL...", "Enter request URL...")).PlaceholderColor(p.muted).
				FontSize(fsLarge).Font(monoFont).TextColor(p.fg).Grow(1).Basis(0).FillHeight().PaddingX(12)
			if url.Changed() {
				d.Params = client.SyncParamsWithURL(d.URL, d.Params)
			}
			if url.Submitted() && !t.sending {
				a.later(func() { a.sendTab(t) })
			}
		})
		a.sendGroup(c, t)
	})
}

// sendGroup 是 112px 宽的发送按钮组：Send + 下拉（发送并下载）；发送中显示 Cancel。
func (a *app) sendGroup(c *ui.Context, t *tab) {
	p := a.p
	empty := strings.TrimSpace(t.draft.URL) == ""
	if t.sending {
		bg := ui.RGB(230, 230, 230)
		if p.dark {
			bg = ui.RGB(68, 68, 68)
		}
		if a.button(c, btnOpts{w: 112, h: 30, radius: 8, bg: &bg, key: "cancel"}, func(bool) {
			txt(c, "Cancel", fsLarge, p.fg, 500)
		}) {
			a.be.CancelHTTPRequest(t.sendID)
		}
		return
	}
	box := ui.Row(c.Key("sendgrp")).Width(112).Height(30).Shrink(0).Radius(8).Background(p.accent).Clip()
	if empty {
		box = box.Opacity(0.5)
	}
	box.Children(func() {
		send := ui.ButtonBase(c.Key("send")).Grow(1).Basis(0).FillHeight().AlignItems(ui.Center).Justify(ui.Center).Disabled(empty).Tooltip("⌘↩")
		if send.Hovered() && !empty {
			send = send.Background(p.accentHover)
		}
		send.Children(func() { txt(c, "Send", fsLarge, p.accentFg, 500) })
		if send.Clicked() {
			a.later(func() { a.sendTab(t) })
		}
		arrow := ui.ButtonBase(c.Key("sendmore")).Width(28).FillHeight().Shrink(0).AlignItems(ui.Center).Justify(ui.Center).Disabled(empty).
			PopoverMenuButton(func(m *ui.Menu) {
				if m.Item("Send and Download").Chosen() {
					a.later(func() {
						t.downloadAfter = true
						a.sendTab(t)
					})
				}
			})
		if arrow.Hovered() && !empty {
			arrow = arrow.Background(p.accentHover)
		}
		arrow.Children(func() {
			ui.Box(c).Width(1).Height(18).Background(ui.RGBA(255, 255, 255, 0.25)).Absolute()
			icon(c, "arrowDown", 10, p.accentFg)
		})
	})
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
	ui.Column(c).Grow(1).Basis(0).Children(func() {
		labels := []string{"Params", "Headers", "Body", "Auth"}
		counts := []int{count(d.Params, false), count(d.Headers, true), 0, 0}
		ui.Row(c).FillWidth().Height(28).Shrink(0).PaddingX(8).Gap(4).AlignItems(ui.Center).Children(func() {
			a.pillTabs(c, "rq", &t.sub, labels, counts)
		})
		ui.Column(c).Grow(1).Basis(0).Padding(4, 0, 0, 0).Children(func() {
			switch t.sub {
			case 0:
				if a.kvTableOpts(c, &d.Params, kvOpts{id: t.draft.ID + ":params", keyPH: a.t("参数名", "Key"), valPH: a.t("参数值", "Value")}) {
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
	})
}

// withCount 在标签名后附加数量（旧调用兼容）。
func withCount(label string, n int) string {
	if n == 0 {
		return label
	}
	return label + " (" + itoa(n) + ")"
}
