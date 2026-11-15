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

// crumbBar 是请求子标签行右侧的面包屑（设计稿 .crumb）：项目 › 文件夹 › 请求名（可点击改名），
// 之后是竖线、保存（⌘S）、复制为 cURL 与在侧栏中定位。窄窗口下路径段先让位，只留请求名和按钮。
func (a *app) crumbBar(c *ui.Context, t *tab) {
	p := a.p
	ui.Row(c.Key("crumb")).Grow(1).Basis(0).MinWidth(0).Gap(4).AlignItems(ui.Center).Justify(ui.End).Children(func() {
		for i, seg := range a.breadcrumbSegments(t) {
			seg := seg
			tip := a.t("在侧栏中定位", "Reveal in sidebar")
			if seg.folderID == "" {
				tip = a.t("切换项目", "Switch project")
			}
			if a.textOverflows(c, seg.name, fsSmall, 0, 120) {
				tip = seg.name + "\n" + tip // 名称被截断时，提示里带上全名
			}
			sb := ui.ButtonBase(c.Key("seg" + itoa(i))).Height(22).PaddingX(4).Radius(radiusSm - 2).AlignItems(ui.Center).Shrink(1).MinWidth(0).Tooltip(tip)
			if sb.Hovered() {
				sb = sb.Background(p.hover)
			}
			col, w := p.muted, 0
			if seg.folderID != "" {
				col, w = p.fg2, 500
			}
			sb.Children(func() { txtE(c, seg.name, fsSmall, col, w).MaxWidth(120) })
			if sb.Clicked() {
				a.later(func() { a.clickCrumb(seg) })
			}
			icon(c, "arrowRight", 10, p.muted).Shrink(0)
		}
		a.requestName(c, t)
		ui.Box(c).Width(1).Height(14).Background(p.border).Shrink(0).MarginX(3)
		dirty := t.dirty
		col := p.muted
		if dirty {
			col = p.fg2
		}
		if a.button(c, btnOpts{h: 22, radius: radiusSm, padX: 8, gap: 5, disabled: !dirty, tip: a.t("保存", "Save") + " (⌘S)", key: "bc-save"}, func(bool) {
			icon(c, "save", 12, col)
			txt(c, a.t("保存", "Save"), fsSmall, col, 550)
			a.kbdHint(c, "⌘S")
		}) {
			a.later(func() { a.saveTab(t) })
		}
		noURL := strings.TrimSpace(t.draft.URL) == ""
		a.ibSm(c, "bc-curl", "copy", a.t("复制为 cURL", "Copy as cURL")+" (⌘⇧C)", noURL, func() { a.copyCurl(t) })
		a.ibSm(c, "bc-locate", "target", a.t("在侧栏中定位", "Reveal in sidebar"), t.unsaved, func() {
			a.showHome, a.side.tab = false, 0
			a.activate(a.tabIndex(t))
		})
	})
}

// requestName 是面包屑末尾的请求名：点击后原地编辑，回车或失焦提交，Esc 取消。
func (a *app) requestName(c *ui.Context, t *tab) {
	p := a.p
	if !t.editName {
		btn := ui.ButtonBase(c.Key("reqname")).Height(22).Radius(radiusSm - 2).PaddingX(4).AlignItems(ui.Center).Justify(ui.Start).Shrink(1).MinWidth(0).
			Tooltip(a.t("单击编辑请求名称", "Click to edit request name"))
		if btn.Hovered() {
			btn = btn.Background(p.hover)
		}
		btn.Children(func() { txtEFlat(c, t.title(), fsSmall, p.fg, 600).MaxWidth(180) })
		if btn.Clicked() {
			t.editName, t.nameBuf = true, t.draft.Name
		}
		return
	}
	in := ui.TextInputBase(c.Key("reqname-edit"), &t.nameBuf).FontSize(fsSmall).FontWeight(600).TextColor(p.fg).Height(22).PaddingX(6).Radius(radiusSm-2).
		Border(1, p.accent.Alpha(0.4)).MinWidth(140).AutoFocus()
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
}

// urlBar 是地址栏行（设计稿方案 C）：左边是全圆角的地址框（方法选择 + URL 输入），右边是独立的全圆角发送按钮组，
// 两者间距 8、高 sizeURLBar；地址框获得焦点时描边变强调色并带光环。
func (a *app) urlBar(c *ui.Context, t *tab) {
	p := a.p
	d := t.draft
	ui.Row(c).FillWidth().Shrink(0).Padding(10, 12, 0, 12).Gap(8).AlignItems(ui.Center).Children(func() {
		r := float32(sizeURLBar / 2)
		box := ui.Row(c.Key("urlbox")).Grow(1).Basis(0).Height(sizeURLBar).Radius(r).Background(p.surface).AlignItems(ui.Stretch)
		focus := box.FocusWithin()
		bc := p.btnBorder
		if focus {
			bc = p.accent
			box = box.Shadow(0, 0, 0, 3, p.acSoft).Shadow(0, 4, 14, -6, p.glow)
		}
		box.Border(1, bc).Clip().Children(func() {
			mc := p.method(d.Method)
			m := ui.ButtonBase(c.Key("method")).Width(88).Shrink(0).Padding(0, 8, 0, 16).AlignItems(ui.Center).Justify(ui.SpaceBetween).
				BorderWidth(0, 1, 0, 0).BorderColor(p.border).
				PopoverMenuButton(func(m *ui.Menu) {
					for _, mm := range client.Methods {
						mm := mm
						if m.Item(mm).Checked(d.Method == mm).Chosen() {
							a.later(func() { d.Method = mm })
						}
					}
				}, a.menuOpts())
			if m.Hovered() {
				m = m.Background(mc.Alpha(0.13))
			} else {
				m = m.Background(mc.Alpha(0.06))
			}
			m.Children(func() {
				txtMono(c, d.Method, fsBase, mc, 700)
				icon(c, "tbChevDown", 12, p.muted)
			})
			wrap := ui.Row(c.Key("urlwrap")).Grow(1).Basis(0).FillHeight()
			refs := client.FindTemplateRefs(d.URL, a.vars())
			hl := len(refs) > 0 && !wrap.FocusWithin()
			wrap.Children(func() {
				url := ui.TextInputBase(c.Key("url"), &d.URL).Placeholder(a.t("输入请求 URL...", "Enter request URL...")).PlaceholderColor(p.muted).
					FontSize(fsV2xs+0.5).Font(monoFont).TextColor(p.fg).Grow(1).Basis(0).FillHeight().Padding(0, 12, 2, 12)
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
		})
		a.sendGroup(c, t)
	})
}

// sendGroup 是地址栏右侧的发送按钮组（设计稿 .send）：渐变强调色的全圆角胶囊，“发送 ⌘↵”加下拉（发送并下载），
// 带强调色光晕；发送中变为“取消”（浅底 + 描边）。
func (a *app) sendGroup(c *ui.Context, t *tab) {
	p := a.p
	empty := strings.TrimSpace(t.draft.URL) == ""
	h := float32(sizeURLBar)
	if t.sending {
		cancel := ui.ButtonBase(c.Key("cancel")).Height(h).Radius(h/2).PaddingX(16).Gap(7).AlignItems(ui.Center).Justify(ui.Center).Shrink(0).
			Background(p.surface2).Border(1, p.btnBorder)
		if cancel.Hovered() {
			cancel = cancel.Background(p.surface2)
		}
		cancel.Children(func() {
			ui.Spinner(c).Size(13, 13)
			txt(c, a.t("取消", "Cancel"), fsBase, p.fg, 650)
		})
		if cancel.Clicked() {
			a.be.CancelHTTPRequest(t.sendID)
		}
		return
	}
	box := ui.Row(c.Key("sendgrp")).Height(h).Shrink(0).Radius(h/2).Clip().Shadow(0, 2, 4, 0, p.glow).Shadow(0, 8, 20, -6, p.glow)
	if empty {
		box = box.Opacity(0.5)
	}
	p.gradAccent(box, false).Children(func() {
		send := ui.ButtonBase(c.Key("send")).FillHeight().Padding(0, 14, 0, 16).AlignItems(ui.Center).Justify(ui.Center).Gap(7).Disabled(empty).Tooltip("⌘↩")
		if send.Hovered() && !empty {
			send = send.Background(ui.RGBA(255, 255, 255, 0.08))
		}
		send.Children(func() {
			icon(c, "send", 14, p.accentFg)
			txt(c, a.t("发送", "Send"), fsBase, p.accentFg, 650)
			ui.Row(c).Height(16).PaddingX(5).Radius(4).Background(p.accentFg.Alpha(0.16)).AlignItems(ui.Center).Shrink(0).Children(func() {
				txtMono(c, "⌘↵", fsTiny, p.accentFg, 500)
			})
		})
		if send.Clicked() {
			a.later(func() { a.sendTab(t) })
		}
		arrow := ui.ButtonBase(c.Key("sendmore")).Width(28).FillHeight().Shrink(0).AlignItems(ui.Center).Justify(ui.Center).Disabled(empty).
			BorderWidth(0, 0, 0, 1).BorderColor(p.accentFg.Alpha(0.28)).
			PopoverMenuButton(func(m *ui.Menu) {
				if m.Item("Send and Download").Chosen() {
					a.later(func() {
						t.downloadAfter = true
						a.sendTab(t)
					})
				}
			}, a.menuOpts())
		if arrow.Hovered() && !empty {
			arrow = arrow.Background(ui.RGBA(255, 255, 255, 0.08))
		}
		arrow.Children(func() { icon(c, "tbChevDown", 12, p.accentFg) })
	})
}

// requestPane 是请求参数区：Params / Headers / Body / Auth 子标签（右侧并入面包屑与保存按钮）。
func (a *app) requestPane(c *ui.Context, t *tab) {
	p := a.p
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
		// 设计稿：Body / Auth 已配置时在标签后显示一个强调色小圆点
		extras := make([]tabExtra, 4)
		extras[2].dot = d.Body.Type != client.BodyNone && d.Body.Type != ""
		extras[3].dot = d.Auth.Type != client.AuthNone && d.Auth.Type != ""
		ui.Row(c).FillWidth().Height(30).Shrink(0).PaddingX(12).Gap(2).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(p.border).
			Gradient(p.surface, p.surface.Mix(p.surface2, 0.35), 180).Children(func() {
			a.underlineTabs(c, "rq", &t.sub, labels, counts, extras, 30)
			a.crumbBar(c, t)
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
