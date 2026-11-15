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

// cookiesView 绘制 Cookie 管理器：顶部标题与统计，左栏域名列表，右栏当前域名的 Cookie 表格，右下角是使用提示。
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
	// 设计稿 .ck：900×540，左栏通高（标题 + 添加域名 + 搜索 + 域名列表），右栏顶部是域名、数量与操作按钮，下面是表格
	a.dialog(c, &s.open, 900, 540, radiusLg, func() {
		ui.Row(c).FillWidth().FillHeight().AlignItems(ui.Stretch).Children(func() {
			ui.Row(c).Width(230).Shrink(0).Background(p.sidebar).AlignItems(ui.Stretch).Children(func() {
				ui.Column(c).Grow(1).Basis(0).Padding(12, 10).Gap(8).Children(func() {
					ui.Row(c).Gap(8).AlignItems(ui.Center).PaddingX(2).Children(func() {
						icon(c, "cookie", 14, p.fg)
						txt(c, a.t("Cookie 管理器", "Cookie Jar"), fsBase+0.5, p.fg, 650)
					})
					ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Shrink(0).Children(func() {
						in := a.input(c, &s.domInput, a.t("输入域名", "Type a domain name"), fieldOpts{h: 28, radius: radiusInput, fs: fsSmall, icon: "plus", key: "ck-dom", bd: &bd, iconSize: 12, iconLeft: 8, iconGap: 8})
						add := a.btn(c, dbtn{key: "ck-dom-add", label: a.t("添加", "Add"), tip: a.t("新增域名", "Add domain")})
						if add || in.Submitted() {
							a.later(a.addCookieDomain)
						}
					})
					ui.Row(c).FillWidth().Shrink(0).Children(func() {
						a.input(c, &s.filter, a.t("搜索 Cookie", "Search cookies"), fieldOpts{h: 28, radius: radiusInput, fs: fsSmall, icon: "search", key: "ck-search", bd: &bd, iconSize: 12, iconLeft: 8, iconGap: 8})
					})
					ui.Scroll(c).Grow(1).Basis(0).Scrollbars(a.scrollbarsVisible()).Children(func() {
						ui.Column(c).FillWidth().Gap(2).Children(func() {
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
			ui.Column(c).Grow(1).Basis(0).Children(func() {
				ui.Row(c).Height(46).FillWidth().Shrink(0).Padding(0, 10, 0, 14).Gap(8).AlignItems(ui.Center).Children(func() {
					name := cur
					if name == "" {
						name = a.t("无域名", "No Domain")
					}
					icon(c, "globe", 15, p.muted)
					txt(c, name, 14, p.fg, 650)
					ui.Row(c).Height(18).MinWidth(18).PaddingX(6).Radius(9).Background(p.surface2).Border(1, p.border).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
						txt(c, itoa(len(shown)), fsTiny, p.fg2, 0)
					})
					ui.Spacer(c)
					if a.btn(c, dbtn{key: "ck-paste", label: a.t("粘贴识别", "Paste & Detect"), icon: "copy", sm: true, tip: a.t("复制浏览器里的 a=1; b=2 后点这里，可一次导入多个 Cookie", "Copy a browser's a=1; b=2, then click here to import several cookies at once")}) {
						a.later(func() { a.pasteCookieHeader(cur) })
					}
					if a.btn(c, dbtn{key: "ck-add", label: a.t("添加 Cookie", "Add Cookie"), icon: "plus", sm: true}) {
						a.later(func() { a.addBlankCookie(cur) })
					}
					if a.btn(c, dbtn{key: "ck-clear-dom", label: a.t("清空当前域名", "Clear Domain"), sm: true}) {
						d := cur
						a.later(func() { a.jar.ClearDomain(d) })
					}
					if a.btn(c, dbtn{key: "ck-clear-all", label: a.t("清空全部", "Clear All"), sm: true, danger: true}) {
						a.later(func() {
							a.overlays.askConfirm(a.t("清空所有 Cookie？", "Clear all cookies?"), a.t("此操作无法撤销。", "This can't be undone."), a.t("清空", "Clear"), a.jar.Clear)
						})
					}
					a.ibSm(c, "ck-close", "x", a.t("关闭", "Close"), false, func() { s.open = false })
				})
				a.cookieTable(c, shown, cur)
			})
		})
	})
}

// domainItem 是左栏的一个域名行：地球图标、域名与 Cookie 个数胶囊。
func (a *app) domainItem(c *ui.Context, g cookieGroup, selected bool) {
	p := a.p
	b := ui.ButtonBase(c.Key("dom:" + g.domain)).FillWidth().Height(30).Radius(7).PaddingX(10).AlignItems(ui.Center).Justify(ui.Start)
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
			txtE(c, name, fsSmall, p.fg, selected2weight(selected)).Grow(1).Basis(0)
			ui.Row(c).MinWidth(20).Height(16).PaddingX(6).Radius(8).Background(p.surface).Border(1, p.border).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
				txt(c, itoa(len(g.items)), fsTiny, p.fg2, 0)
			})
		})
	})
	if b.Clicked() {
		dom := g.domain
		a.later(func() { a.overlays.cookies.domain = dom })
	}
}

// selected2weight 返回域名行文字的字重：选中为 500，否则常规。
func selected2weight(selected bool) int {
	if selected {
		return 500
	}
	return 0
}

// cookieTable 绘制 Cookie 表格（设计稿 .ck-t）：表头 28 高带淡底，行 34 高、输入框带细边框，
// 列宽 40 / 1fr / 1.4fr / 70 / 100 / 60 / 66 / 30；表格下方的空白处是“勾选启用才会随请求发送”的提示。
func (a *app) cookieTable(c *ui.Context, shown []client.Cookie, domain string) {
	p := a.p
	type col struct {
		title string
		w     float32 // 0 为自适应
		grow  float32
	}
	cols := []col{{a.t("启用", "On"), 40, 0}, {a.t("名称", "Name"), 0, 1}, {a.t("值", "Value"), 0, 1.4}, {a.t("路径", "Path"), 70, 0}, {a.t("过期时间", "Expires"), 100, 0}, {"Secure", 60, 0}, {"HttpOnly", 66, 0}, {"", 30, 0}}
	cell := func(e ui.Element, i int) ui.Element {
		if cols[i].w > 0 {
			return e.Width(cols[i].w).Shrink(0)
		}
		return e.Grow(cols[i].grow).Basis(0).MinWidth(0)
	}
	bd := p.btnBorder
	ui.Scroll(c).Grow(1).Basis(0).Scrollbars(a.scrollbarsVisible()).Children(func() {
		ui.Column(c).FillWidth().FillHeight().Children(func() {
			ui.Row(c).FillWidth().Height(28).Shrink(0).Background(p.surface2).BorderWidth(0, 0, 1, 0).BorderColor(p.border).AlignItems(ui.Center).Children(func() {
				for i, h := range cols {
					r := cell(ui.Row(c.Key("h"+itoa(i))).PaddingX(8).AlignItems(ui.Center), i)
					if i == 0 || i >= 5 {
						r = r.Justify(ui.Center)
					}
					r.Children(func() { txt(c, h.title, fsSmall, p.muted, 600) })
				}
			})
			for _, ck := range shown {
				ck := ck
				ui.Row(c.Key(ck.ID)).FillWidth().Height(34).Shrink(0).BorderWidth(0, 0, 1, 0).BorderColor(p.border).AlignItems(ui.Center).Children(func() {
					cell(ui.Row(c).PaddingX(8).AlignItems(ui.Center).Justify(ui.Center), 0).Children(func() {
						on := ck.Enabled
						if a.checkbox(c, "en", &on, false) {
							a.jar.Update(ck.ID, func(x *client.Cookie) { x.Enabled = on })
						}
					})
					field := func(i int, key string, v string, ph string, set func(x *client.Cookie, s string)) {
						cell(ui.Row(c).PaddingX(8), i).Children(func() {
							val := v
							in := a.input(c, &val, ph, fieldOpts{h: 24, radius: radiusSm, fs: fsSmall, key: key, mono: true, bd: &bd})
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
						cell(ui.Row(c).PaddingX(8).AlignItems(ui.Center).Justify(ui.Center), 5+k).Children(func() {
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
					cell(ui.Row(c).AlignItems(ui.Center).Justify(ui.Center), 7).Children(func() {
						if a.iconButton(c, "delete", a.t("删除", "Delete"), 13, p.muted) {
							id := ck.ID
							a.later(func() { a.jar.Remove(id) })
						}
					})
				})
			}
			// 空白占位行：点击即新增一条 Cookie
			add := ui.ButtonBase(c.Key("ck-new-row")).FillWidth().Height(34).Shrink(0).BorderWidth(0, 0, 1, 0).BorderColor(p.border).AlignItems(ui.Center).Justify(ui.Start)
			if add.Hovered() {
				add = add.Background(p.sideHover)
			}
			add.Children(func() {
				for i, ph := range []string{"", a.t("名称", "Name"), a.t("值", "Value"), "/", a.t("会话", "Session"), "", "", ""} {
					cell(ui.Row(c.Key("nr"+itoa(i))).PaddingX(14).AlignItems(ui.Center), i).Children(func() {
						if ph != "" {
							txtMono(c, ph, fsSmall, p.muted, 0)
						}
					})
				}
			})
			if add.Clicked() {
				a.later(func() { a.addBlankCookie(domain) })
			}
			// 设计稿 .empty-r：表格下方的提示
			ui.Column(c).FillWidth().Grow(1).Basis(0).AlignItems(ui.Center).Justify(ui.Center).Gap(10).Padding(30).Children(func() {
				icon(c, "cookie", 22, p.muted.Alpha(0.75))
				txt(c, a.t("勾选「启用」的 Cookie 会随请求自动发送", "Enabled cookies are sent with requests automatically"), fsBase, p.muted, 0)
			})
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
		a.toastWarn(a.t("请先输入或选择域名", "Please choose or enter a domain first"))
		return
	}
	text := a.clipboard
	if a.native {
		text = mygo.Clipboard.ReadText()
	}
	if strings.TrimSpace(text) == "" {
		a.toastWarn(a.t("剪贴板为空或不可读取", "Clipboard is empty or unavailable"))
		return
	}
	n := a.jar.ImportHeader(text, domain, "/")
	if n > 0 {
		a.toastOK(a.t("已识别并导入 ", "Imported ") + itoa(n) + a.t(" 个 Cookie", " cookies"))
		logger.Info("已从剪贴板导入 Cookie", "domain", domain, "count", n)
	} else {
		a.toastWarn(a.t("未识别到可导入的 Cookie（请复制形如 a=1; b=2）", "No valid cookies found (expected format: a=1; b=2)"))
	}
}
