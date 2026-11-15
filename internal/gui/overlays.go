package gui

import (
	"github.com/egoist/mygo/ui"
)

// overlays 汇总所有弹层（对话框）的状态。
type overlays struct {
	prompt  promptState
	confirm confirmState
	closing closeState
	prefs   prefsState
	imp     importState
	curl    curlState
	cookies cookiesState
	quick   quickState
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
	open bool
	tab  *tab
	name string
}

func (o *overlays) askText(title, label, initial, ok string, onOK func(string)) {
	o.prompt = promptState{open: true, title: title, label: label, ok: ok, value: initial, onOK: onOK}
}

func (o *overlays) askConfirm(title, message, ok string, onOK func()) {
	o.confirm = confirmState{open: true, title: title, message: message, ok: ok, onOK: onOK}
}

func (o *overlays) askClose(t *tab) {
	o.closing = closeState{open: true, tab: t, name: t.title()}
}

// promptView 绘制文本输入对话框。
func (a *app) promptView(c *ui.Context) {
	pr := &a.overlays.prompt
	if !pr.open {
		return
	}
	p := a.p
	ok := false
	a.stdDialog(c, &pr.open, 420, func() { a.dlgTitle(c, &pr.open, pr.title, true) }, func() {
		ui.Column(c).FillWidth().Padding(14).Gap(6).Children(func() {
			if pr.label != "" {
				ui.Text(c, pr.label).FontSize(fsSmall).TextColor(p.muted)
			}
			// 输入框在 Row 里横向撑满；直接放进 Column 时它的 Grow 是纵向的，高度会塌成 0。
			ui.Row(c).FillWidth().Children(func() {
				in := a.input(c, &pr.value, "", fieldOpts{h: 30, radius: radiusR, fs: fsBase, key: "prompt-in", focus: true})
				if in.Submitted() {
					ok = true
				}
			})
		})
	}, func() {
		if a.dlgButton(c, "prompt-cancel", a.t("取消", "Cancel"), dlgNormal, false) {
			pr.open = false
		}
		if a.dlgButton(c, "prompt-ok", pr.ok, dlgPrimary, trimName(pr.value) == "") {
			ok = true
		}
	})
	if ok && trimName(pr.value) != "" {
		pr.open = false
		if pr.onOK != nil {
			value, fn := trimName(pr.value), pr.onOK
			a.later(func() { fn(value) })
		}
	}
}

// dangerLabels 是会让确认按钮变成红色的按钮文案。
var dangerLabels = map[string]bool{"删除": true, "Delete": true, "清空": true, "Clear": true}

// confirmView 绘制确认框：标题、说明与“取消 / 确认”；删除类操作的确认按钮为危险色。
func (a *app) confirmView(c *ui.Context) {
	cf := &a.overlays.confirm
	if !cf.open {
		return
	}
	run := false
	danger := cf.danger || dangerLabels[cf.ok]
	a.compactDialog(c, &cf.open, cf.title, func() {
		kind := toastInfo
		if danger {
			kind = toastError
		}
		a.dlgMsg(c, kind, cf.message, "")
	}, func() {
		if a.dlgButton(c, "confirm-cancel", a.t("取消", "Cancel"), dlgNormal, false) {
			cf.open = false
		}
		kind := dlgPrimary
		if danger {
			kind = dlgDanger
		}
		if a.dlgButton(c, "confirm-ok", cf.ok, kind, false) {
			run = true
		}
	})
	if run {
		cf.open = false
		if cf.onOK != nil {
			a.later(cf.onOK)
		}
	}
}

// closeView 绘制关闭未保存标签时的提示：不保存 / 取消 / 保存并关闭，并可勾选“默认保存/默认丢弃”。
func (a *app) closeView(c *ui.Context) {
	cl := &a.overlays.closing
	if !cl.open {
		return
	}
	p := a.p
	changed := false
	action := 0
	prevSave := a.set.AlwaysSaveUnsaved
	a.compactDialog(c, &cl.open, a.t("保存更改", "Save changes"), func() {
		a.dlgMsg(c, toastWarn, a.t("请求「"+cl.name+"」有未保存的修改", "“"+cl.name+"” has unsaved changes"),
			a.t("关闭后这些修改将会丢失，是否先保存？", "These changes will be lost if you close without saving."))
		check := func(key string, on *bool, label string) {
			ui.Row(c.Key(key)).Gap(8).AlignItems(ui.Center).Children(func() {
				if a.checkbox(c, key, on, false) {
					changed = true
				}
				txt(c, label, fsBase, p.fg, 0)
			})
		}
		check("always-save", &a.set.AlwaysSaveUnsaved, a.t("关闭 Tab 时默认保存未保存修改", "Save unsaved changes by default when closing tabs"))
		check("always-discard", &a.set.AlwaysDiscardUnsaved, a.t("关闭 Tab 时默认丢弃未保存修改", "Discard unsaved changes by default when closing tabs"))
	}, func() {
		// 设计稿底栏靠右：不保存 / 取消 / 保存并关闭
		if a.dlgButton(c, "close-discard", a.t("不保存", "Don't save"), dlgNormal, false) {
			action = 1
		}
		if a.dlgButton(c, "close-cancel", a.t("取消", "Cancel"), dlgNormal, false) {
			cl.open = false
			a.closeQueue = nil // 取消会放弃批量关闭中剩余的标签
		}
		if a.dlgButton(c, "close-save", a.t("保存并关闭", "Save & close"), dlgPrimary, false) {
			action = 2
		}
	})
	if changed {
		// 两个“默认”选项互斥。
		if a.set.AlwaysSaveUnsaved && a.set.AlwaysDiscardUnsaved {
			if prevSave { // 刚勾选的是“丢弃”
				a.set.AlwaysSaveUnsaved = false
			} else {
				a.set.AlwaysDiscardUnsaved = false
			}
		}
		a.later(a.applySettings)
	}
	switch action {
	case 1:
		cl.open = false
		t := cl.tab
		a.later(func() {
			if i := a.tabIndex(t); i >= 0 {
				a.removeTab(i)
			}
			a.drainCloseQueue()
		})
	case 2:
		cl.open = false
		t := cl.tab
		a.later(func() {
			if i := a.tabIndex(t); i >= 0 {
				if !a.saveTab(t) {
					a.closeQueue = nil
					return
				}
				a.removeTab(i)
			}
			a.drainCloseQueue()
		})
	}
}
