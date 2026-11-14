package gui

import (
	"net/url"
	"sort"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"minipost/internal/client"
	"minipost/internal/pkg/logger"
)

// cookiesState 是 Cookie 管理器（大弹窗）的状态。
type cookiesState struct {
	open     bool
	domain   string // 当前选中的域名
	domInput string // 顶部“输入域名”框
	filter   string // 搜索框
	hint     string // 导入结果提示
}

func (a *app) openCookies() {
	a.overlays.cookies = cookiesState{open: true}
}

// activeHost 是当前请求标签 URL 的主机名，用作默认域名。
func (a *app) activeHost() string {
	t := a.activeTab()
	if t == nil || t.env != nil {
		return ""
	}
	raw := strings.TrimSpace(t.draft.URL)
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	if u, err := url.Parse(raw); err == nil {
		return client.NormalizeDomain(u.Hostname())
	}
	return ""
}

type cookieGroup struct {
	domain string
	items  []client.Cookie
}

// cookieGroups 按域名分组（域名排序），当前请求的域名即使没有 Cookie 也会列出。
func (a *app) cookieGroups(all []client.Cookie, host string) []cookieGroup {
	m := map[string][]client.Cookie{}
	for _, ck := range all {
		d := client.NormalizeDomain(ck.Domain)
		if d == "" {
			d = host
		}
		m[d] = append(m[d], ck)
	}
	if host != "" {
		if _, ok := m[host]; !ok {
			m[host] = nil
		}
	}
	out := make([]cookieGroup, 0, len(m))
	for d, items := range m {
		out = append(out, cookieGroup{d, items})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].domain < out[j].domain })
	return out
}

// cookiesView 绘制 Cookie 管理器：左侧 300px 域名列表，右侧当前域名的 Cookie 表格。
func (a *app) cookiesView(c *ui.Context) {
	s := &a.overlays.cookies
	if !s.open {
		return
	}
	p := a.p
	all := a.jar.All()
	host := a.activeHost()
	groups := a.cookieGroups(all, host)
	cur := s.domain
	if cur == "" {
		cur = host
	}
	if cur == "" && len(groups) > 0 {
		cur = groups[0].domain
	}
	var curCookies []client.Cookie
	for _, g := range groups {
		if g.domain == cur {
			curCookies = g.items
		}
	}
	q := strings.ToLower(strings.TrimSpace(s.filter))
	shown := curCookies
	if q != "" {
		shown = nil
		for _, ck := range curCookies {
			if strings.Contains(strings.ToLower(ck.Name), q) || strings.Contains(strings.ToLower(ck.Value), q) || strings.Contains(strings.ToLower(ck.Path), q) {
				shown = append(shown, ck)
			}
		}
	}
	bd := p.btnBorder
	a.dialog(c, &s.open, 1280, 760, 14, func() {
		ui.Row(c).FillWidth().FillHeight().AlignItems(ui.Stretch).Children(func() {
			// 左栏
			ui.Row(c).Width(300).Shrink(0).Background(p.surface2).AlignItems(ui.Stretch).Children(func() {
				ui.Column(c).Grow(1).Basis(0).Children(func() {
					ui.Row(c).Height(44).FillWidth().Shrink(0).PaddingX(12).Gap(8).AlignItems(ui.Center).Children(func() {
						icon(c, "cookie", 13, p.fg2)
						txt(c, a.t("Cookie 管理器", "Cookie Jar"), fsBase, p.fg, 600)
					})
					hline(c, p.border)
					ui.Column(c).FillWidth().Padding(10).Gap(8).Shrink(0).Children(func() {
						ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Children(func() {
							in := a.input(c, &s.domInput, a.t("输入域名", "Type a domain name"), fieldOpts{h: 28, radius: 7, fs: fsSmall, key: "ck-dom", bd: &bd})
							add := a.button(c, btnOpts{h: 28, radius: 7, padX: 10, border: &bd, key: "ck-dom-add", tip: a.t("新增域名", "Add domain")}, func(bool) {
								txt(c, a.t("添加", "Add"), fsSmall, p.fg2, 0)
							})
							if add || in.Submitted() {
								a.later(a.addCookieDomain)
							}
						})
						ui.Row(c).FillWidth().Children(func() {
							a.input(c, &s.filter, a.t("搜索 Cookie", "Search cookies"), fieldOpts{h: 28, radius: 7, fs: fsSmall, icon: "search", key: "ck-search", bd: &bd, iconSize: 11, iconLeft: 7, iconGap: 10})
						})
					})
					ui.Scroll(c).Grow(1).Basis(0).Scrollbars(a.scrollbarsVisible()).Children(func() {
						ui.Column(c).FillWidth().Padding(0, 8, 8, 8).Gap(2).Children(func() {
							if len(groups) == 0 {
								ui.Row(c).FillWidth().Justify(ui.Center).Padding(24, 8).Children(func() {
									txt(c, a.t("暂无域名", "No domains"), fsSmall, p.muted, 0)
								})
							}
							for _, g := range groups {
								a.domainItem(c, g, g.domain == cur)
							}
						})
					})
				})
				vline(c, p.border)
			})
			// 右栏
			ui.Column(c).Grow(1).Basis(0).Children(func() {
				ui.Row(c).Height(44).FillWidth().Shrink(0).PaddingX(14).AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						icon(c, "globe", 12, p.fg2)
						name := cur
						if name == "" {
							name = a.t("无域名", "No Domain")
						}
						txt(c, name, fsBase, p.fg, 600)
						txt(c, "("+itoa(len(shown))+")", fsSmall, p.muted, 0)
					})
					ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
						small := func(key, ic, label string, col ui.Color, disabled bool) bool {
							return a.button(c, btnOpts{h: 24, radius: 6, padX: 8, gap: 4, border: &bd, key: key, disabled: disabled}, func(bool) {
								if ic != "" {
									icon(c, ic, 11, col)
								}
								txt(c, label, fsSmall, col, 0)
							})
						}
						if small("ck-paste", "copy", a.t("粘贴识别", "Paste & Detect"), p.fg, false) {
							a.later(func() { a.pasteCookieHeader(cur) })
						}
						if small("ck-add", "add", a.t("添加 Cookie", "Add Cookie"), p.fg, false) {
							a.later(func() { a.addBlankCookie(cur) })
						}
						if small("ck-clear-dom", "", a.t("清空当前域名", "Clear Domain"), p.fg2, false) {
							d := cur
							a.later(func() { a.jar.ClearDomain(d) })
						}
						if small("ck-clear-all", "", a.t("清空全部", "Clear All"), p.fg2, false) {
							a.later(func() {
								a.overlays.askConfirm(a.t("清空所有 Cookie？", "Clear all cookies?"), a.t("此操作无法撤销。", "This can't be undone."), a.t("清空", "Clear"), a.jar.Clear)
							})
						}
						ui.Box(c).Width(1).Shrink(0)
						if a.iconButton(c, "clear", a.t("关闭", "Close"), 11, p.muted) {
							s.open = false
						}
					})
				})
				hline(c, p.border)
				if s.hint != "" {
					ui.Row(c).FillWidth().Shrink(0).PaddingX(14).PaddingY(4).BorderWidth(0, 0, 1, 0).BorderColor(p.border).Children(func() {
						txt(c, s.hint, fsSmall, p.muted, 0)
					})
				}
				a.cookieTable(c, shown)
			})
		})
	})
}

// domainItem 是左栏的一个域名行：地球图标、域名与 Cookie 个数。
func (a *app) domainItem(c *ui.Context, g cookieGroup, selected bool) {
	p := a.p
	b := ui.ButtonBase(c.Key("dom:" + g.domain)).FillWidth().Height(28).Radius(7).PaddingX(10).AlignItems(ui.Center).Justify(ui.Start)
	switch {
	case selected:
		b = b.Background(p.selected)
	case b.Hovered():
		b = b.Background(p.sideHover)
	}
	col := p.muted
	if selected {
		col = p.accent
	}
	b.Children(func() {
		ui.Row(c).Grow(1).Gap(8).AlignItems(ui.Center).Children(func() {
			icon(c, "globe", 11, col)
			name := g.domain
			if name == "" {
				name = a.t("无域名", "No Domain")
			}
			txtE(c, name, fsSmall, p.fg, 0).Grow(1).Basis(0)
			txt(c, itoa(len(g.items)), fsTiny, p.muted, 0)
		})
	})
	if b.Clicked() {
		dom := g.domain
		a.later(func() { a.overlays.cookies.domain = dom })
	}
}

// cookieTable 绘制 Cookie 表格：启用、名称、值、路径、过期时间、Secure、HttpOnly、删除。
func (a *app) cookieTable(c *ui.Context, shown []client.Cookie) {
	p := a.p
	if len(shown) == 0 {
		ui.Row(c).Grow(1).Basis(0).FillWidth().AlignItems(ui.Center).Justify(ui.Center).Children(func() {
			txt(c, a.t("当前域名暂无 Cookies，点击右上角 添加 Cookie", "No cookies for this domain. Click Add Cookie in the top-right."), fsSmall, p.muted, 0)
		})
		return
	}
	type col struct {
		title string
		w     float32 // 0 为自适应
	}
	cols := []col{{a.t("启用", "On"), 32}, {a.t("名称", "Name"), 0}, {a.t("值", "Value"), 0}, {a.t("路径", "Path"), 90}, {a.t("过期时间", "Expires"), 150}, {"Secure", 60}, {"HttpOnly", 70}, {"", 32}}
	cell := func(e ui.Element, w float32) ui.Element {
		if w > 0 {
			return e.Width(w).Shrink(0)
		}
		return e.Grow(1).Basis(0)
	}
	ui.Scroll(c).Grow(1).Basis(0).Scrollbars(a.scrollbarsVisible()).Children(func() {
		ui.Column(c).FillWidth().Children(func() {
			ui.Row(c).FillWidth().Height(28).Background(p.surface2).BorderWidth(0, 0, 1, 0).BorderColor(p.border).AlignItems(ui.Center).Children(func() {
				for i, h := range cols {
					r := cell(ui.Row(c.Key("h"+itoa(i))).PaddingX(8).AlignItems(ui.Center), h.w)
					if i == 0 || i >= 5 {
						r = r.Justify(ui.Center)
					}
					r.Children(func() { txt(c, h.title, fsSmall, p.fg2, 600) })
				}
			})
			for _, ck := range shown {
				ck := ck
				row := ui.Row(c.Key(ck.ID)).FillWidth().MinHeight(32).BorderWidth(0, 0, 1, 0).BorderColor(p.border).AlignItems(ui.Center)
				hover := row.Hovered()
				row.Children(func() {
					cell(ui.Row(c).PaddingX(8).AlignItems(ui.Center).Justify(ui.Center), cols[0].w).Children(func() {
						on := ck.Enabled
						if a.checkbox(c, "en", &on, false) {
							a.jar.Update(ck.ID, func(x *client.Cookie) { x.Enabled = on })
						}
					})
					field := func(i int, key string, v string, ph string, set func(x *client.Cookie, s string)) {
						cell(ui.Row(c).PaddingX(8).PaddingY(4), cols[i].w).Children(func() {
							val := v
							in := a.input(c, &val, ph, fieldOpts{h: 24, radius: 5, fs: fsSmall, key: key})
							if in.Changed() {
								a.jar.Update(ck.ID, func(x *client.Cookie) { set(x, val) })
							}
						})
					}
					field(1, "n", ck.Name, a.t("名称", "Name"), func(x *client.Cookie, s string) { x.Name = s })
					field(2, "v", ck.Value, a.t("值", "Value"), func(x *client.Cookie, s string) { x.Value = s })
					field(3, "p", ck.Path, "/", func(x *client.Cookie, s string) { x.Path = s })
					field(4, "e", ck.Expires, a.t("会话", "Session"), func(x *client.Cookie, s string) { x.Expires = s })
					for k, flag := range []bool{ck.Secure, ck.HTTPOnly} {
						on := flag
						cell(ui.Row(c).PaddingX(8).AlignItems(ui.Center).Justify(ui.Center), cols[5+k].w).Children(func() {
							if a.checkbox(c, "f"+itoa(k), &on, false) {
								a.jar.Update(ck.ID, func(x *client.Cookie) {
									if k == 0 {
										x.Secure = on
									} else {
										x.HTTPOnly = on
									}
								})
							}
						})
					}
					cell(ui.Row(c).AlignItems(ui.Center).Justify(ui.Center), cols[7].w).Children(func() {
						if hover && a.button(c, btnOpts{w: 20, h: 20, radius: 4, key: "del", tip: a.t("删除", "Delete")}, func(h bool) {
							col := p.muted
							if h {
								col = p.danger
							}
							icon(c, "delete", 10, col)
						}) {
							id := ck.ID
							a.later(func() { a.jar.Remove(id) })
						}
					})
				})
			}
		})
	})
}

// addCookieDomain 把顶部输入框里的域名加入列表并选中；该域名还没有 Cookie 时顺手建一条空白的。
func (a *app) addCookieDomain() {
	s := &a.overlays.cookies
	d := client.NormalizeDomain(s.domInput)
	if d == "" {
		return
	}
	s.domain, s.domInput = d, ""
	for _, ck := range a.jar.All() {
		if client.NormalizeDomain(ck.Domain) == d {
			return
		}
	}
	a.addBlankCookie(d)
}

// addBlankCookie 在域名下新增一条空白 Cookie 供编辑。
func (a *app) addBlankCookie(domain string) {
	if domain == "" {
		a.toastWarn(a.t("请先输入或选择域名", "Please choose or enter a domain first"))
		return
	}
	a.jar.Add(client.Cookie{Domain: domain, Path: "/", Enabled: true})
	logger.Info("已添加 Cookie 行", "domain", domain)
}

// pasteCookieHeader 读取剪贴板里的 Cookie 请求头文本并导入到域名下。
func (a *app) pasteCookieHeader(domain string) {
	s := &a.overlays.cookies
	if domain == "" {
		domain = client.NormalizeDomain(s.domInput)
	}
	if domain == "" {
		s.hint = a.t("请先输入或选择域名", "Please choose or enter a domain first")
		return
	}
	text := a.clipboard
	if a.native {
		text = mygo.Clipboard.ReadText()
	}
	if strings.TrimSpace(text) == "" {
		s.hint = a.t("剪贴板为空或不可读取", "Clipboard is empty or unavailable")
		return
	}
	n := a.jar.ImportHeader(text, domain, "/")
	if n > 0 {
		s.hint = a.t("已识别并导入 ", "Imported ") + itoa(n) + a.t(" 个 Cookie", " cookies")
		logger.Info("已从剪贴板导入 Cookie", "domain", domain, "count", n)
	} else {
		s.hint = a.t("未识别到可导入的 Cookie（请复制形如 a=1; b=2）", "No valid cookies found (expected format: a=1; b=2)")
	}
}
