package gui

import (
	"github.com/egoist/mygo/ui"

	"minipost/internal/client"
)

var authKinds = []string{client.AuthNone, client.AuthBasic, client.AuthBearer, client.AuthAPIKey}

// field 绘制带标题的单行输入。
func (a *app) field(c *ui.Context, label string, value *string, password bool) {
	ui.Text(c, label).TextColor(c.Theme().TextMuted)
	in := ui.TextInput(c, value).MaxWidth(520)
	if password {
		in.Password()
	}
}

// authTab 绘制 Auth 子标签。
func (a *app) authTab(c *ui.Context, t *tab) {
	au := &t.draft.Auth
	kind := 0
	for i, k := range authKinds {
		if k == au.Type {
			kind = i
		}
	}
	ui.Column(c).Grow(1).Gap(8).Padding(12).Children(func() {
		if ui.Segmented(c, &kind, "None", "Basic", "Bearer", "API Key").Changed() {
			au.Type = authKinds[kind]
		}
		switch au.Type {
		case client.AuthBasic:
			a.field(c, a.t("用户名", "Username"), &au.Username, false)
			a.field(c, a.t("密码", "Password"), &au.Password, true)
		case client.AuthBearer:
			a.field(c, "Token", &au.Token, false)
		case client.AuthAPIKey:
			a.field(c, "Key", &au.APIKeyName, false)
			a.field(c, a.t("值", "Value"), &au.APIKeyValue, false)
			ui.Text(c, a.t("添加到", "Add to")).TextColor(c.Theme().TextMuted)
			loc := 0
			if au.APIKeyLocation == "query" {
				loc = 1
			}
			if ui.Segmented(c, &loc, "Header", "Query Params").Changed() {
				au.APIKeyLocation = []string{"header", "query"}[loc]
			}
		default:
			ui.Text(c, a.t("此请求不使用认证", "This request uses no authentication")).TextColor(c.Theme().TextMuted)
		}
	})
}
