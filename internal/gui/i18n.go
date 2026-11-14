package gui

import "strings"

// 语言标识。
const (
	localeZH = "zh-CN"
	localeEN = "en-US"
)

// locale 返回当前界面语言；设置为空时跟随系统。
func (a *app) locale() string {
	switch a.set.Locale {
	case localeZH, localeEN:
		return a.set.Locale
	}
	if strings.HasPrefix(strings.ToLower(a.sysLocale), "en") {
		return localeEN
	}
	return localeZH
}

// t 按当前语言选择中文或英文文案。
func (a *app) t(zh, en string) string {
	if a.locale() == localeEN {
		return en
	}
	return zh
}
