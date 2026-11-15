package gui

import (
	"github.com/egoist/mygo/ui"

	"minipost/internal/client"
)

var authKinds = []string{client.AuthNone, client.AuthBasic, client.AuthBearer, client.AuthAPIKey}

// authLabelW 是 Auth 页左侧标签列宽度。
const authLabelW = 84

// authField 绘制一行“标签 + 输入框”：标签固定列宽，输入框等宽字体；secret 为 true 时带显示/隐藏开关。
func (a *app) authField(c *ui.Context, label string, value *string, secret bool) {
	p := a.p
	ui.Row(c).FillWidth().Shrink(0).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Row(c).Width(authLabelW).Shrink(0).Children(func() { txt(c, label, fsBase, p.fg2, 500) })
		opts := fieldOpts{h: 28, radius: radiusInput, padX: 10, fs: fsBase, key: "af:" + label, mono: true}
		if secret {
			opts.trail = func() {
				name, tip := "eye", a.t("显示", "Show")
				if a.authReveal {
					name, tip = "viewOff", a.t("隐藏", "Hide")
				}
				if a.iconButton(c, name, tip, 13, p.muted) {
					a.authReveal = !a.authReveal
				}
			}
		}
		in := a.input(c, value, "", opts)
		if secret && !a.authReveal {
			in.Password()
		}
	})
}

// authTypeLabel 返回认证类型在下拉框里显示的名字。
func (a *app) authTypeLabel(typ string) string {
	switch typ {
	case client.AuthBasic:
		return "Basic Auth"
	case client.AuthBearer:
		return "Bearer Token"
	case client.AuthAPIKey:
		return "API Key"
	}
	return "None"
}

// authPreview 返回“将自动添加请求头”提示里的预览文本（密钥以圆点遮盖）；不需要提示时返回空串。
func authPreview(au *client.AuthDraft) string {
	const mask = "•••••"
	switch au.Type {
	case client.AuthBasic:
		return "Authorization: Basic " + mask
	case client.AuthBearer:
		return "Authorization: Bearer " + mask
	case client.AuthAPIKey:
		name := au.APIKeyName
		if name == "" {
			name = "API-Key"
		}
		if au.APIKeyLocation == "query" {
			return "?" + name + "=" + mask
		}
		return name + ": " + mask
	}
	return ""
}

// authTab 绘制 Auth 子标签：类型下拉 + 各类型表单 + 将生成的请求头提示。
func (a *app) authTab(c *ui.Context, t *tab) {
	p := a.p
	au := &t.draft.Auth
	ui.Column(c).Grow(1).Basis(0).Padding(16, 16).Gap(12).Children(func() {
		ui.Row(c).FillWidth().Shrink(0).Gap(12).AlignItems(ui.Center).Children(func() {
			ui.Row(c).Width(authLabelW).Shrink(0).Children(func() { txt(c, a.t("认证类型", "Auth type"), fsBase, p.fg2, 500) })
			ui.ButtonBase(c.Key("authtype")).Width(200).Height(28).Radius(radiusInput).PaddingX(10).Gap(8).AlignItems(ui.Center).
				Border(1, p.btnBorder).Background(p.surface).
				PopoverMenuButton(func(m *ui.Menu) {
					for _, k := range authKinds {
						if m.Item(a.authTypeLabel(k)).Checked(au.Type == k || (k == client.AuthNone && au.Type == "")).Chosen() {
							k := k
							a.later(func() { au.Type = k })
						}
					}
				}, a.menuOpts()).Children(func() {
				txt(c, a.authTypeLabel(au.Type), fsBase, p.fg, 0)
				ui.Spacer(c)
				icon(c, "arrowDown", 9, p.muted)
			})
			txt(c, "None / Basic / Bearer / API Key", fsSmall, p.muted, 0)
		})
		switch au.Type {
		case client.AuthBasic:
			a.authField(c, a.t("用户名", "Username"), &au.Username, false)
			a.authField(c, a.t("密码", "Password"), &au.Password, true)
		case client.AuthBearer:
			a.authField(c, "Token", &au.Token, true)
		case client.AuthAPIKey:
			a.authField(c, "Key", &au.APIKeyName, false)
			a.authField(c, "Value", &au.APIKeyValue, true)
			ui.Row(c).FillWidth().Shrink(0).Gap(12).AlignItems(ui.Center).Children(func() {
				ui.Row(c).Width(authLabelW).Shrink(0).Children(func() { txt(c, a.t("添加位置", "Add to"), fsBase, p.fg2, 500) })
				loc := 0
				if au.APIKeyLocation == "query" {
					loc = 1
				}
				if a.pillSeg(c, "akloc", &loc, []string{"Header", "Query Param"}, nil, segOpts{h: 22, padX: 11, fs: fsSmall + 0.5}) {
					au.APIKeyLocation = []string{"header", "query"}[loc]
				}
			})
		default:
			ui.Row(c).FillWidth().Padding(24, 0).Justify(ui.Center).Children(func() {
				txt(c, a.t("此请求不需要认证", "This request does not require authentication"), fsSmall, p.muted, 0)
			})
		}
		if pv := authPreview(au); pv != "" {
			// 预览行（设计稿 .prev）：标签列 + 内容底圆角条，锁图标 + 等宽预览文本 + 右侧“请求头/查询参数”小标
			where := a.t("请求头", "Header")
			if au.Type == client.AuthAPIKey && au.APIKeyLocation == "query" {
				where = a.t("查询参数", "Query")
			}
			ui.Row(c).FillWidth().Shrink(0).Gap(12).AlignItems(ui.Center).Children(func() {
				ui.Row(c).Width(authLabelW).Shrink(0).Children(func() { txt(c, a.t("预览", "Preview"), fsBase, p.fg2, 500) })
				ui.Row(c).Grow(1).Basis(0).MinWidth(0).Padding(8, 10).Gap(8).Radius(radiusR).Background(p.surface2).Border(0.5, p.hair).AlignItems(ui.Center).Children(func() {
					icon(c, "lock", 12, p.muted)
					txtMono(c, pv, fsSmall+0.5, p.fg2, 0).Grow(1).Basis(0)
					ui.Row(c).Height(16).PaddingX(6).Radius(4).Background(p.surface).Border(0.5, p.border).AlignItems(ui.Center).Children(func() {
						txt(c, where, fsTiny, p.muted, 600)
					})
				})
			})
		}
		if au.Type == client.AuthBearer {
			// 提示框（设计稿 .tipbox）
			ui.Row(c).FillWidth().Shrink(0).Padding(9, 11).Gap(8).Radius(radiusLg).Background(p.surface2).Border(0.5, p.hair).AlignItems(ui.Start).Children(func() {
				icon(c, "info", 14, p.accent).Margin(1, 0, 0, 0)
				ui.Text(c, a.t("支持在 Token 中使用 {{token}} 等环境变量；敏感值不会写入日志。", "Environment variables such as {{token}} work in the token; secrets are never written to logs.")).
					FontSize(fsSmall + 0.5).TextColor(p.fg2).FixedLineHeight((fsSmall + 0.5) * 1.5).Grow(1).Basis(0).MinWidth(0)
			})
		}
	})
}
