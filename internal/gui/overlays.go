package gui

import (
	"github.com/egoist/mygo/ui"
)

// overlays 汇总所有弹层（对话框）的状态。
type overlays struct {
	prompt  promptState
	confirm confirmState
	closing closeState
	projDlg projectDialogState
	curl    curlState
	prefs   prefsState
	imp     importState
	envMgr  envState
	cookies cookiesState
}

// promptState 是单行文本输入对话框。
type promptState struct {
	open  bool
	title string
	label string
	ok    string
	value string
	onOK  func(string)
}

// confirmState 是只有“取消 / 确认”的确认框。
type confirmState struct {
	open    bool
	title   string
	message string
	ok      string
	danger  bool
	onOK    func()
}

// closeState 是关闭未保存标签时的三选一提示。
type closeState struct {
	open  bool
	index int
	name  string
}

func (o *overlays) askText(title, label, initial, ok string, onOK func(string)) {
	o.prompt = promptState{open: true, title: title, label: label, ok: ok, value: initial, onOK: onOK}
}

func (o *overlays) askConfirm(title, message, ok string, onOK func()) {
	o.confirm = confirmState{open: true, title: title, message: message, ok: ok, onOK: onOK}
}

func (o *overlays) askClose(index int, name string) {
	o.closing = closeState{open: true, index: index, name: name}
}

// promptView 绘制文本输入对话框。
func (a *app) promptView(c *ui.Context) {
	p := &a.overlays.prompt
	ui.Modal(c, &p.open, func() {
		ui.Text(c, p.title).Bold()
		if p.label != "" {
			ui.Text(c, p.label).TextColor(c.Theme().TextMuted)
		}
		submit := ui.TextInput(c, &p.value).AutoFocus().MinWidth(320).Submitted()
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			cancel := ui.Button(c, a.t("取消", "Cancel")).Clicked()
			ok := ui.PrimaryButton(c, p.ok).Clicked()
			if cancel {
				p.open = false
			}
			if (ok || submit) && trimName(p.value) != "" {
				p.open = false
				if p.onOK != nil {
					value, fn := trimName(p.value), p.onOK
					a.later(func() { fn(value) })
				}
			}
		})
	})
}

// confirmView 绘制确认框。
func (a *app) confirmView(c *ui.Context) {
	cf := &a.overlays.confirm
	if ui.AlertDialog(c, &cf.open, cf.title, cf.message, a.t("取消", "Cancel"), cf.ok) == 1 && cf.onOK != nil {
		a.later(cf.onOK)
	}
}

// closeView 绘制关闭未保存标签时的提示：取消 / 不保存 / 保存。
func (a *app) closeView(c *ui.Context) {
	cl := &a.overlays.closing
	title := a.t("保存对“", "Save changes to “") + cl.name + a.t("”的修改？", "”?")
	msg := a.t("不保存的话，这些修改将会丢失。", "Your changes will be lost if you don't save them.")
	idx := cl.index
	switch ui.AlertDialog(c, &cl.open, title, msg, a.t("取消", "Cancel"), a.t("不保存", "Don't Save"), a.t("保存", "Save")) {
	case 1:
		a.later(func() { a.removeTab(idx) })
	case 2:
		a.later(func() {
			if idx < len(a.tabs) && a.saveTab(a.tabs[idx]) {
				a.removeTab(idx)
			}
		})
	}
}
