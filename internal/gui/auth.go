package gui

import (
	"github.com/egoist/mygo/ui"

	"minipost/internal/client"
)

var authKinds = []string{client.AuthNone, client.AuthBasic, client.AuthBearer, client.AuthAPIKey}

// authField 绘制带标题的输入框：标题 10.4px 次要色，输入框 28px 高、圆角 8、等宽字体。
func (a *app) authField(c *ui.Context, label string, value *string, password bool) {
	p := a.p
	ui.Column(c).FillWidth().MaxWidth(448).Gap(4).Shrink(0).Children(func() {
		txt(c, label, 10.4, p.fg2, 0)
		in := a.input(c, value, "", fieldOpts{h: 28, radius: radiusInput, padX: 12, fs: fsVXs, key: "af:" + label})
		if password {
			in.Password()
		}
	})
}

// authTab 绘制 Auth 子标签：类型标签行与对应表单。
func (a *app) authTab(c *ui.Context, t *tab) {
	p := a.p
	au := &t.draft.Auth
	kind := 0
	for i, k := range authKinds {
		if k == au.Type {
			kind = i
		}
	}
	ui.Column(c).Grow(1).Basis(0).Padding(8).Children(func() {
		ui.Row(c).FillWidth().Shrink(0).Gap(4).AlignItems(ui.Center).Padding(0, 0, 12, 0).Children(func() {
			if a.kindPills(c, "auth", &kind, []string{"None", "Basic", "Bearer", "API Key"}, 8) {
				au.Type = authKinds[kind]
			}
		})
		ui.Column(c).FillWidth().PaddingStart(8).Gap(8).Children(func() {
			switch au.Type {
			case client.AuthBasic:
				a.authField(c, a.t("用户名", "Username"), &au.Username, false)
				a.authField(c, a.t("密码", "Password"), &au.Password, true)
			case client.AuthBearer:
				a.authField(c, "Token", &au.Token, false)
			case client.AuthAPIKey:
				a.authField(c, "Key", &au.APIKeyName, false)
				a.authField(c, "Value", &au.APIKeyValue, false)
				ui.Column(c).Gap(4).Shrink(0).Children(func() {
					txt(c, a.t("添加位置", "Add to"), 10.4, p.fg2, 0)
					loc := 0
					if au.APIKeyLocation == "query" {
						loc = 1
					}
					ui.Row(c).Gap(8).Children(func() {
						if a.kindPills(c, "akloc", &loc, []string{"Header", "Query Param"}, 8) {
							au.APIKeyLocation = []string{"header", "query"}[loc]
						}
					})
				})
			default:
				ui.Row(c).FillWidth().Padding(24, 0).Justify(ui.Center).Children(func() {
					txt(c, a.t("此请求不需要认证", "This request does not require authentication"), 10.4, p.muted, 0)
				})
			}
		})
	})
}
