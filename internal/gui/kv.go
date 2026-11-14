package gui

import (
	"path/filepath"

	"github.com/egoist/mygo/ui"

	"minipost/internal/client"
)

// kvHeader 绘制键值表的列标题。
func (a *app) kvHeader(c *ui.Context, cols ...string) {
	th := c.Theme()
	ui.Row(c).Gap(6).PaddingX(4).Children(func() {
		ui.Box(c).Width(22).Shrink(0)
		for _, col := range cols {
			ui.Text(c, col).TextColor(th.TextMuted).FontSize(th.FontSize - 1).Grow(1).Basis(0)
		}
		ui.Box(c).Width(28).Shrink(0)
	})
}

// kvTable 绘制可增删的键值表（Params / Headers / x-www-form-urlencoded）。
// 末尾始终保留一行空白占位行；返回本帧内容是否被修改。
func (a *app) kvTable(c *ui.Context, rows *[]*client.Row, keyPH, valPH string) bool {
	changed := false
	*rows = client.EnsureTrailingRow(*rows)
	remove := -1
	ui.Scroll(c).Grow(1).Children(func() {
		ui.Column(c).Gap(4).Padding(8).Children(func() {
			a.kvHeader(c, keyPH, valPH, a.t("描述", "Description"))
			for i, r := range *rows {
				i, r := i, r
				ui.Row(c.Key(r.ID)).Gap(6).AlignItems(ui.Center).Children(func() {
					changed = ui.Checkbox(c, &r.Enabled, "").Width(22).Shrink(0).Changed() || changed
					changed = ui.TextInput(c, &r.Key).Placeholder(keyPH).Grow(1).Basis(0).Changed() || changed
					changed = ui.TextInput(c, &r.Value).Placeholder(valPH).Grow(1).Basis(0).Changed() || changed
					changed = ui.TextInput(c, &r.Description).Placeholder(a.t("描述", "Description")).Grow(1).Basis(0).Changed() || changed
					del := ui.Button(c, "×").Width(28).Shrink(0).Tooltip(a.t("删除", "Delete"))
					if r.IsPlaceholder() {
						del.Invisible()
					} else if del.Clicked() {
						remove = i
					}
				})
			}
		})
	})
	if remove >= 0 {
		*rows = append((*rows)[:remove], (*rows)[remove+1:]...)
		changed = true
	}
	*rows = client.EnsureTrailingRow(*rows)
	return changed
}

// formTable 绘制 multipart/form-data 表：每行可选文本或文件。
func (a *app) formTable(c *ui.Context, rows *[]*client.FormRow) bool {
	changed := false
	*rows = client.EnsureTrailingFormRow(*rows)
	remove := -1
	kinds := []string{"text", "file"}
	ui.Scroll(c).Grow(1).Children(func() {
		ui.Column(c).Gap(4).Padding(8).Children(func() {
			a.kvHeader(c, "Key", a.t("值", "Value"), a.t("描述", "Description"))
			for i, r := range *rows {
				i, r := i, r
				ui.Row(c.Key(r.ID)).Gap(6).AlignItems(ui.Center).Children(func() {
					changed = ui.Checkbox(c, &r.Enabled, "").Width(22).Shrink(0).Changed() || changed
					changed = ui.TextInput(c, &r.Key).Placeholder("Key").Grow(1).Basis(0).Changed() || changed
					kind := r.Type
					if ui.Select(c, &kind, kinds).Width(80).Shrink(0).Changed() {
						r.Type, r.Value, r.FilePath, r.FileName = kind, "", "", ""
						changed = true
					}
					if r.Type == "file" {
						label := r.FileName
						if label == "" {
							label = a.t("选择文件…", "Choose file…")
						}
						if ui.Button(c, label).Grow(1).Basis(0).Clicked() {
							a.pickFormFile(r)
						}
					} else {
						changed = ui.TextInput(c, &r.Value).Placeholder(a.t("值", "Value")).Grow(1).Basis(0).Changed() || changed
					}
					changed = ui.TextInput(c, &r.Description).Placeholder(a.t("描述", "Description")).Grow(1).Basis(0).Changed() || changed
					del := ui.Button(c, "×").Width(28).Shrink(0)
					if r.IsPlaceholder() {
						del.Invisible()
					} else if del.Clicked() {
						remove = i
					}
				})
			}
		})
	})
	if remove >= 0 {
		*rows = append((*rows)[:remove], (*rows)[remove+1:]...)
		changed = true
	}
	*rows = client.EnsureTrailingFormRow(*rows)
	return changed
}

// pickFormFile 在后台弹出文件选择，选中后写回表单行。
func (a *app) pickFormFile(r *client.FormRow) {
	a.async(func() func() {
		path, err := a.be.OpenFileDialogAny(a.win)
		return func() {
			if err != nil {
				a.fail(a.t("选择文件", "Choose file"), err)
				return
			}
			if path != "" {
				r.FilePath, r.FileName, r.Value = path, filepath.Base(path), path
			}
		}
	})
}

// post 把 fn 交回主线程执行；没有真实窗口（测试）时直接同步执行。
func (a *app) post(fn func()) {
	if a.win == nil {
		fn()
		return
	}
	a.win.Update(fn)
}

// async 在后台执行 work，并把它返回的收尾函数交回主线程执行；没有真实窗口（测试）时同步执行。
func (a *app) async(work func() func()) {
	if a.win == nil {
		if done := work(); done != nil {
			done()
		}
		return
	}
	go func() {
		if done := work(); done != nil {
			a.win.Update(done)
		}
	}()
}
