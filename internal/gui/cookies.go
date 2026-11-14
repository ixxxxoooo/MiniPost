package gui

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"minipost/internal/client"
)

// cookiesState 是 Cookie 管理对话框的状态。
type cookiesState struct {
	open   bool
	filter string
	adding client.Cookie
	header string // 要导入的 Cookie 请求头文本
}

func (a *app) openCookies() {
	a.overlays.cookies = cookiesState{open: true, adding: client.Cookie{Enabled: true, Path: "/"}}
}

// cookiesView 绘制 Cookie 管理对话框：列表、编辑、添加与导入。
func (a *app) cookiesView(c *ui.Context) {
	s := &a.overlays.cookies
	th := c.Theme()
	ui.Modal(c, &s.open, func() {
		ui.Text(c, "Cookies").Bold()
		ui.Row(c).Gap(8).Children(func() {
			ui.SearchField(c, &s.filter).Placeholder(a.t("按域名或名称过滤", "Filter by domain or name")).Grow(1)
			if ui.Button(c, a.t("清空全部", "Clear All")).Clicked() {
				a.later(func() {
					a.overlays.askConfirm(a.t("清空所有 Cookie？", "Clear all cookies?"), a.t("此操作无法撤销。", "This can't be undone."), a.t("清空", "Clear"), a.jar.Clear)
				})
			}
		})
		all := a.jar.All()
		var shown []client.Cookie
		q := strings.ToLower(strings.TrimSpace(s.filter))
		for _, ck := range all {
			if q == "" || strings.Contains(strings.ToLower(ck.Domain), q) || strings.Contains(strings.ToLower(ck.Name), q) {
				shown = append(shown, ck)
			}
		}
		ui.Scroll(c).Height(260).Width(760).Children(func() {
			ui.Column(c).Gap(4).Children(func() {
				if len(shown) == 0 {
					ui.Text(c, a.t("没有 Cookie", "No cookies")).TextColor(th.TextMuted)
				}
				for _, ck := range shown {
					ck := ck
					ui.Row(c.Key(ck.ID)).Gap(6).AlignItems(ui.Center).Children(func() {
						on := ck.Enabled
						if ui.Checkbox(c, &on, "").Width(22).Shrink(0).Changed() {
							a.jar.Update(ck.ID, func(x *client.Cookie) { x.Enabled = on })
						}
						ui.Text(c, ck.Domain).SingleLine().Width(170).Shrink(0)
						ui.Text(c, ck.Name).SingleLine().Width(130).Shrink(0).Bold()
						val := ck.Value
						if ui.TextInput(c, &val).Grow(1).Basis(0).Changed() {
							a.jar.Update(ck.ID, func(x *client.Cookie) { x.Value = val })
						}
						ui.Text(c, ck.Path).SingleLine().Width(60).Shrink(0).TextColor(th.TextMuted)
						if ui.Button(c, "×").Width(28).Shrink(0).Clicked() {
							id := ck.ID
							a.later(func() { a.jar.Remove(id) })
						}
					})
				}
			})
		})
		ui.Divider(c)
		ui.Text(c, a.t("添加 Cookie", "Add cookie")).TextColor(th.TextMuted)
		ui.Row(c).Gap(6).Children(func() {
			ui.TextInput(c, &s.adding.Domain).Placeholder(a.t("域名", "Domain")).Grow(1).Basis(0)
			ui.TextInput(c, &s.adding.Name).Placeholder(a.t("名称", "Name")).Grow(1).Basis(0)
			ui.TextInput(c, &s.adding.Value).Placeholder(a.t("值", "Value")).Grow(1).Basis(0)
			if ui.Button(c, a.t("添加", "Add")).Clicked() {
				a.later(a.addCookie)
			}
		})
		ui.Text(c, a.t("导入 Cookie 请求头（形如 a=1; b=2，使用上面的域名）", "Import Cookie header (a=1; b=2, uses the domain above)")).TextColor(th.TextMuted)
		ui.Row(c).Gap(6).Children(func() {
			ui.TextInput(c, &s.header).Grow(1)
			if ui.Button(c, a.t("导入", "Import")).Clicked() {
				a.later(a.importCookieHeader)
			}
		})
		ui.Row(c).Justify(ui.End).Children(func() {
			if ui.PrimaryButton(c, a.t("完成", "Done")).Clicked() {
				s.open = false
			}
		})
	})
}

func (a *app) addCookie() {
	s := &a.overlays.cookies
	ck := s.adding
	ck.Domain = client.NormalizeDomain(ck.Domain)
	if ck.Domain == "" || strings.TrimSpace(ck.Name) == "" {
		a.toast(a.t("域名和名称不能为空", "Domain and name are required"))
		return
	}
	if ck.Path == "" {
		ck.Path = "/"
	}
	ck.Enabled = true
	a.jar.Add(ck)
	s.adding = client.Cookie{Enabled: true, Path: "/", Domain: s.adding.Domain}
}

func (a *app) importCookieHeader() {
	s := &a.overlays.cookies
	domain := client.NormalizeDomain(s.adding.Domain)
	if domain == "" || strings.TrimSpace(s.header) == "" {
		a.toast(a.t("请填写域名和 Cookie 文本", "Enter a domain and a Cookie header"))
		return
	}
	n := a.jar.ImportHeader(s.header, domain, "/")
	s.header = ""
	a.toast(a.t("已导入 ", "Imported ") + itoa(n) + a.t(" 条 Cookie", " cookies"))
}
