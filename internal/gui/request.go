package gui

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"minipost/internal/client"
	"minipost/internal/pkg/logger"
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

// crumb 是面包屑里的一段：显示名，以及点击后要在侧栏定位的文件夹 ID（项目段为空）。
type crumb struct{ name, folderID string }

// breadcrumbSegments 返回“项目 > 文件夹…”路径（不含请求名）。
func (a *app) breadcrumbSegments(t *tab) []crumb {
	var segs []crumb
	if pr := a.project(); pr != nil {
		segs = append(segs, crumb{name: pr.Name})
	}
	var chain []crumb
	if n, ok := a.tree.nodes[t.draft.ID]; ok {
		for id := n.parent; id != ""; id = a.tree.nodes[id].parent {
			chain = append([]crumb{{name: a.tree.nodes[id].name, folderID: id}}, chain...)
		}
	}
	return append(segs, chain...)
}

// clickCrumb 处理面包屑某一段的点击：文件夹段在侧栏里展开并定位到该文件夹，
// 项目段打开项目下拉面板（与标题栏的项目按钮一致）。
func (a *app) clickCrumb(cr crumb) {
	if cr.folderID == "" {
		a.projMenu.open, a.projMenu.query = true, ""
		return
	}
	if _, ok := a.tree.nodes[cr.folderID]; !ok {
		return
	}
	if a.set.SidebarCollapsed {
		a.set.SidebarCollapsed = false
		a.saveSettings()
	}
	a.side.tab, a.side.query = 0, "" // 回到请求列表并清掉搜索过滤，保证目标可见
	a.side.outline.Open.Add(cr.folderID)
	for p := a.tree.nodes[cr.folderID].parent; p != ""; p = a.tree.nodes[p].parent {
		a.side.outline.Open.Add(p)
	}
	a.side.sel = cr.folderID
	a.side.reveal, a.side.revealFocus = cr.folderID, false
	logger.Info("已从面包屑定位到侧栏文件夹", "folderID", cr.folderID)
}

// breadcrumb 是地址栏上方的一行：HTTP 徽标、路径、可点击改名的请求名，以及保存/复制 cURL。
func (a *app) breadcrumb(c *ui.Context, t *tab) {
	p := a.p
	sky, skyBg := ui.RGB(2, 132, 199), ui.RGBA(14, 165, 233, 0.15)
	if p.dark {
		sky = ui.RGB(56, 189, 248)
	}
	ui.Row(c).FillWidth().Shrink(0).Padding(4, 12, 0, 12).Children(func() {
		ui.Row(c).FillWidth().Height(28).PaddingX(8).AlignItems(ui.Center).Children(func() {
			ui.Row(c).Height(16).MinWidth(24).PaddingX(4).Radius(4).Background(skyBg).AlignItems(ui.Center).Justify(ui.Center).Shrink(0).Children(func() {
				txtMono(c, "HTTP", 9, sky, 600)
			})
			ui.Box(c).Width(4).Shrink(0)
			for i, seg := range a.breadcrumbSegments(t) {
				seg := seg
				tip := a.t("在侧栏中定位", "Reveal in sidebar")
				if seg.folderID == "" {
					tip = a.t("切换项目", "Switch project")
				}
				if a.textOverflows(c, seg.name, fsBase, 0, 208) {
					tip = seg.name + "\n" + tip // 名称被截断时，提示里带上全名
				}
				sb := ui.ButtonBase(c.Key("seg" + itoa(i))).Height(24).PaddingX(6).Radius(6).AlignItems(ui.Center).Shrink(0).Tooltip(tip)
				if sb.Hovered() {
					sb = sb.Background(p.selected)
				}
				sb.Children(func() {
					txtE(c, seg.name, fsBase, p.muted, 0).MaxWidth(220)
				})
				if sb.Clicked() {
					a.later(func() { a.clickCrumb(seg) })
				}
				// 旧版间距：路径段 →2px→ 箭头 →6px→ 下一段
				ui.Box(c.Key("bs" + itoa(i))).Width(2).Shrink(0)
				icon(c, "arrowRight", 10, p.muted)
				ui.Box(c.Key("bt" + itoa(i))).Width(6).Shrink(0)
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
		btn.Children(func() { txtEFlat(c, t.title(), fsBase, p.fg, 600) })
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

// urlBar 是地址栏（设计稿 2A 一体式）：方法选择、URL 输入与发送按钮组合成一个 34px 高的圆角控件，
// 发送按钮以强调色块内嵌在右端，整体只有一圈边框。
func (a *app) urlBar(c *ui.Context, t *tab) {
	p := a.p
	d := t.draft
	ui.Row(c).FillWidth().Shrink(0).Padding(4, 12).AlignItems(ui.Center).Children(func() {
		box := ui.Row(c.Key("urlbox")).Grow(1).Basis(0).Height(sizeURLBar).Radius(10).Background(p.surface).AlignItems(ui.Stretch)
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
			wrap := ui.Row(c.Key("urlwrap")).Grow(1).Basis(0).FillHeight()
			refs := client.FindTemplateRefs(d.URL, a.vars())
			hl := len(refs) > 0 && !wrap.FocusWithin()
			wrap.Children(func() {
				url := ui.TextInputBase(c.Key("url"), &d.URL).Placeholder(a.t("输入请求 URL...", "Enter request URL...")).PlaceholderColor(p.muted).
					FontSize(fsVXs).Font(monoFont).TextColor(p.fg).Grow(1).Basis(0).FillHeight().Padding(0, 12, 2, 12)
				if hl {
					url = url.TextColor(ui.RGBA(0, 0, 0, 0))
				}
				if url.Changed() {
					d.Params = client.SyncParamsWithURL(d.URL, d.Params)
				}
				if url.Submitted() && !t.sending {
					a.later(func() { a.sendTab(t) })
				}
				if hl {
					a.urlOverlay(c, d.URL, refs)
				}
			})
			a.undefinedHint(c, t)
			a.sendGroup(c, t)
		})
	})
}

// sendGroup 是地址栏右端内嵌的发送按钮组：发送 + 下拉（发送并下载）；发送中显示取消。
// 高度比地址栏小 6px（四周各留 3px），与设计稿一体式地址栏一致。
func (a *app) sendGroup(c *ui.Context, t *tab) {
	p := a.p
	empty := strings.TrimSpace(t.draft.URL) == ""
	h := float32(sizeURLBar - 8)
	if t.sending {
		bg := p.surface2
		if a.button(c, btnOpts{w: 88, h: h, radius: 7, bg: &bg, key: "cancel"}, func(bool) {
			txt(c, a.t("取消", "Cancel"), fsBase, p.fg, 500)
		}) {
			a.be.CancelHTTPRequest(t.sendID)
		}
		return
	}
	box := ui.Row(c.Key("sendgrp")).Width(104).Height(h).Shrink(0).Radius(7).Background(p.accent).Clip().Margin(3, 3, 3, 0).Gap(0)
	if empty {
		box = box.Opacity(0.5)
	}
	box.Children(func() {
		send := ui.ButtonBase(c.Key("send")).Grow(1).Basis(0).FillHeight().AlignItems(ui.Center).Justify(ui.Center).Gap(5).Disabled(empty).Tooltip("⌘↩")
		if send.Hovered() && !empty {
			send = send.Background(p.accentHover)
		}
		send.Children(func() {
			icon(c, "send", 12, p.accentFg)
			txt(c, a.t("发送", "Send"), fsBase, p.accentFg, 600)
		})
		if send.Clicked() {
			a.later(func() { a.sendTab(t) })
		}
		arrow := ui.ButtonBase(c.Key("sendmore")).Width(24).FillHeight().Shrink(0).AlignItems(ui.Center).Justify(ui.Center).Disabled(empty).
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
			ui.Box(c).Width(1).Height(14).Background(ui.RGBA(255, 255, 255, 0.28)).Absolute()
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
		// 设计稿：Body 标签后显示请求体类型，Auth 已配置时显示锁
		extras := make([]tabExtra, 4)
		switch d.Body.Type {
		case client.BodyJSON:
			extras[2].badge = "JSON"
		case client.BodyRaw:
			extras[2].badge = "RAW"
		case client.BodyFormData:
			extras[2].badge = "FORM"
		case client.BodyURLEncoded:
			extras[2].badge = "URL"
		}
		extras[3].lock = d.Auth.Type != client.AuthNone && d.Auth.Type != ""
		ui.Row(c).FillWidth().Height(30).Shrink(0).PaddingX(8).Gap(2).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(a.p.border).Children(func() {
			a.underlineTabs(c, "rq", &t.sub, labels, counts, extras, 30)
		})
		ui.Column(c).Grow(1).Basis(0).Children(func() {
			switch t.sub {
			case 0:
				if a.kvTableOpts(c, &d.Params, kvOpts{id: t.draft.ID + ":params", keyPH: a.t("参数名", "Key"), valPH: a.t("参数值", "Value"), hint: a.t("查询参数 · 修改后自动同步到地址栏", "Query params · synced to the URL")}) {
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
