package gui

import (
	"fmt"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"

	"minipost/internal/client"
	"minipost/internal/pkg/logger"
)

// 表格列权重：取自旧版 table-fixed 在 1280 宽窗口下的实测列宽，随容器宽度等比伸缩。
const (
	colCheck = 37.4
	colKey   = 245.5
	colValue = 409.2
	colDesc  = 245.5
	colAct   = 85.4
)

// 键值表的行高、表头高度与单元格字号（对照原型：行更高、字更大，读起来更松）。
const (
	kvRowH  = 32
	kvHeadH = 30
	kvFont  = 13
)

// kvOpts 描述键值表的列与自动请求头行。
type kvOpts struct {
	id     string // 表格唯一标识，用于批量编辑状态与控件 key
	keyPH  string
	valPH  string
	noDesc bool
	t      *tab   // 提供自动请求头显隐状态；nil 表示没有自动请求头
	hint   string // 表格上方左侧的说明文字（没有自动请求头时显示）
}

// kvSubhead 绘制表格上方的一行：左侧是自动请求头折叠开关（或说明文字），右侧是“批量编辑”。
func (a *app) kvSubhead(c *ui.Context, rows *[]*client.Row, o kvOpts, auto []client.AutoHeader) {
	p := a.p
	ui.Row(c).FillWidth().Height(32).Shrink(0).PaddingX(12).AlignItems(ui.Center).Gap(6).Children(func() {
		if len(auto) > 0 && o.t != nil {
			name := "arrowRight"
			if o.t.autoOpen {
				name = "arrowDown"
			}
			names := make([]string, 0, 3)
			for i, h := range auto {
				if i == 3 {
					break
				}
				names = append(names, h.Key)
			}
			label := fmt.Sprintf(a.t("%d 个自动请求头（%s…）", "%d auto headers (%s…)"), len(auto), strings.Join(names, a.t("、", ", ")))
			if a.button(c, btnOpts{h: 22, padX: 2, gap: 4, noHover: true, key: "auto-toggle"}, func(hover bool) {
				col := p.muted
				if hover {
					col = p.fg2
				}
				icon(c, name, 10, col)
				txt(c, label, fsBase, col, 0)
			}) {
				o.t.autoOpen = !o.t.autoOpen
			}
		} else if o.hint != "" {
			txt(c, o.hint, fsBase, p.muted, 0)
		}
		ui.Spacer(c)
		if a.button(c, btnOpts{h: 22, padX: 4, gap: 4, noHover: true, key: "bulk-" + o.id}, func(hover bool) {
			col := p.accent
			if hover {
				col = p.accentHover
			}
			icon(c, "formatList", 12, col)
			txt(c, a.t("批量编辑", "Bulk Edit"), fsBase, col, 600)
		}) {
			a.bulk[o.id] = serializeBulk(*rows)
		}
	})
}

// tableCell 画一个带下分隔线和右分隔线（列间竖线）的表格单元。
func (a *app) tableCell(c *ui.Context, grow, h float32, children func()) {
	ui.Row(c).Grow(grow).Basis(0).Height(h).Shrink(1).AlignItems(ui.Center).
		BorderWidth(0, 1, 1, 0).BorderColor(a.p.border).Children(children)
}

// tableCellEnd 是表格最后一个单元（行末操作列）：只有下分隔线，没有右竖线。
func (a *app) tableCellEnd(c *ui.Context, grow, h float32, children func()) {
	ui.Row(c).Grow(grow).Basis(0).Height(h).Shrink(1).AlignItems(ui.Center).
		BorderWidth(0, 0, 1, 0).BorderColor(a.p.border).Children(children)
}

// tableFrame 是表格的滚动容器：铺满面板宽度，行与行、列与列之间是 1px 细线。
func (a *app) tableFrame(c *ui.Context, rows func()) {
	ui.Scroll(c).Grow(1).Basis(0).Scrollbars(a.scrollbarsVisible()).Children(func() {
		ui.Column(c).FillWidth().Children(rows)
	})
}

// kvCellInput 是表格单元里的无边框输入框（13px，键与值用等宽字体，占位符灰色）。
func (a *app) kvCellInput(c *ui.Context, key string, v *string, ph string, mono bool, col ui.Color, dim bool, disabled bool) ui.Element {
	p := a.p
	in := ui.TextInputBase(c.Key(key), v).Placeholder(ph).PlaceholderColor(p.muted).FontSize(kvFont).TextColor(col).
		Grow(1).Basis(0).Height(kvRowH-2).Padding(0, 8, 1, 8)
	if mono {
		in = in.Font(monoFont)
	}
	if dim {
		in = in.Opacity(0.4)
	}
	if disabled {
		in = in.Disabled(true)
	}
	return in
}

// kvHeaderCell 绘制表头单元：12px 半粗，次要色。
func (a *app) kvHeaderCell(c *ui.Context, grow float32, label string, extra func()) {
	a.tableCell(c, grow, kvHeadH, func() {
		ui.Row(c).FillWidth().PaddingX(8).AlignItems(ui.Center).Children(func() {
			txtE(c, label, fsBase, a.p.muted, 500).Grow(1)
			if extra != nil {
				extra()
			}
		})
	})
}

// kvHeaderCellEnd 是最后一列的表头单元：没有右竖线。
func (a *app) kvHeaderCellEnd(c *ui.Context, grow float32, label string) {
	a.tableCellEnd(c, grow, kvHeadH, func() {
		ui.Row(c).FillWidth().PaddingX(8).AlignItems(ui.Center).Children(func() {
			txtE(c, label, fsBase, a.p.muted, 500).Grow(1)
		})
	})
}

// kvTableOpts 绘制键值表；末尾始终保留一行空白占位行；返回本帧内容是否被修改。
// 处于“批量编辑”模式时改为文本框，退出时把文本解析回行。
func (a *app) kvTableOpts(c *ui.Context, rows *[]*client.Row, o kvOpts) bool {
	p := a.p
	if text, ok := a.bulk[o.id]; ok {
		return a.kvBulk(c, rows, o, text)
	}
	changed := false
	*rows = client.EnsureTrailingRow(*rows)
	remove := -1
	var auto []client.AutoHeader
	if o.t != nil {
		auto = client.AutoHeaders(o.t.draft, a.set, a.jar, a.vars(), a.t)
	}
	a.kvSubhead(c, rows, o, auto)
	a.tableFrame(c, func() {
		// 表头
		ui.Row(c).FillWidth().Children(func() {
			a.kvHeaderCell(c, colCheck, "", nil)
			a.kvHeaderCell(c, colKey, o.keyPH, nil)
			a.kvHeaderCell(c, colValue, o.valPH, nil)
			if !o.noDesc {
				a.kvHeaderCellEnd(c, colDesc, a.t("说明", "Description"))
			}
			a.tableCellEnd(c, colAct, kvHeadH, func() {})
		})
		// 自动请求头（只读，可单独关闭）
		if o.t != nil && o.t.autoOpen {
			for _, it := range auto {
				it := it
				ui.Row(c.Key("auto:" + it.Key)).FillWidth().Background(p.surface2.Alpha(0.4)).Children(func() {
					a.tableCell(c, colCheck, kvRowH, func() {
						ui.Row(c).FillWidth().Justify(ui.Center).Children(func() {
							on := it.Enabled
							if a.checkbox(c, "autochk:"+it.Key, &on, false) {
								client.SetAutoHeaderEnabled(o.t.draft, it.Key, on)
							}
						})
					})
					a.tableCell(c, colKey, kvRowH, func() {
						txtEMono(c, it.Key, fsSmall, p.fg2, 0).PaddingX(8).Grow(1)
					})
					a.tableCell(c, colValue, kvRowH, func() {
						txtEMono(c, it.Value, fsSmall, p.muted, 0).PaddingX(8).Grow(1)
					})
					if !o.noDesc {
						a.tableCellEnd(c, colDesc, kvRowH, func() {
							d := it.Description
							if d == "" {
								d = a.t("发送时自动生成", "Auto-generated when sending")
							}
							txtE(c, d, fsSmall, p.muted, 0).PaddingX(8).Grow(1)
						})
					}
					a.tableCellEnd(c, colAct, kvRowH, func() {})
				})
			}
		}
		for i, r := range *rows {
			i, r := i, r
			last := r.IsPlaceholder()
			rowEl := ui.Row(c.Key(r.ID)).FillWidth()
			hover := rowEl.Hovered()
			if hover && !last {
				rowEl = rowEl.Background(p.surface2)
			}
			rowEl.Children(func() {
				a.tableCell(c, colCheck, kvRowH, func() {
					ui.Row(c).FillWidth().Height(kvRowH - 2).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
						changed = a.kvCheck(c, r.ID, &r.Enabled, last) || changed
					})
				})
				dim := !r.Enabled && !last
				a.tableCell(c, colKey, kvRowH, func() {
					in := a.kvCellInput(c, "k:"+r.ID, &r.Key, o.keyPH, true, p.fg, dim, false)
					if dim {
						in = in.Strikethrough() // 已停用的行：键名加删除线
					}
					changed = in.Changed() || changed
				})
				a.tableCell(c, colValue, kvRowH, func() {
					changed = a.kvCellInput(c, "v:"+r.ID, &r.Value, o.valPH, true, p.fg, dim, last).Changed() || changed
				})
				if !o.noDesc {
					a.tableCellEnd(c, colDesc, kvRowH, func() {
						changed = a.kvCellInput(c, "d:"+r.ID, &r.Description, a.t("说明", "Description"), false, p.fg2, dim, last).Changed() || changed
					})
				}
				a.tableCellEnd(c, colAct, kvRowH, func() {
					ui.Row(c).FillWidth().Justify(ui.Center).Children(func() {
						if !last {
							if hover && a.miniDelete(c, "del:"+r.ID, nil) {
								remove = i
							}
						}
					})
				})
			})
		}
	})
	if remove >= 0 {
		*rows = append((*rows)[:remove], (*rows)[remove+1:]...)
		changed = true
	}
	*rows = client.EnsureTrailingRow(*rows)
	return changed
}

// miniDelete 是表格行末 16×16 的删除按钮。
func (a *app) miniDelete(c *ui.Context, key string, _ *bool) bool {
	p := a.p
	return a.button(c, btnOpts{w: 16, h: 16, radius: 2, key: key, tip: a.t("删除", "Delete")}, func(hover bool) {
		col := p.muted
		if hover {
			col = p.danger
		}
		icon(c, "delete", 10, col)
	})
}

// serializeBulk 把键值行序列化为“key: value”文本。
func serializeBulk(rows []*client.Row) string {
	var sb strings.Builder
	for _, r := range rows {
		if r.Key == "" {
			continue
		}
		if sb.Len() > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(r.Key + ": " + r.Value)
	}
	return sb.String()
}

// parseBulk 解析“key: value”文本；没有冒号的行只有键。
func parseBulk(text string) []*client.Row {
	var out []*client.Row
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		r := client.NewRow()
		if i := strings.Index(line, ":"); i >= 0 {
			r.Key, r.Value = strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+1:])
		} else {
			r.Key = strings.TrimSpace(line)
		}
		out = append(out, r)
	}
	return out
}

// kvBulk 绘制批量编辑模式：说明文字、退出按钮与多行文本框。
func (a *app) kvBulk(c *ui.Context, rows *[]*client.Row, o kvOpts, text string) bool {
	p := a.p
	ui.Column(c).Grow(1).Basis(0).Padding(8).Children(func() {
		ui.Row(c).FillWidth().Shrink(0).AlignItems(ui.Center).Padding(0, 0, 8, 0).Children(func() {
			txt(c, a.t("每行一条，格式：key: value（兼容 Chrome 复制的带空格格式）", "One item per line, format: key: value (compatible with Chrome copied format)."), fsTiny, p.muted, 0)
			ui.Spacer(c)
			if a.button(c, btnOpts{noHover: true, key: "bulk-exit-" + o.id}, func(hover bool) {
				col := p.accent
				if hover {
					col = p.accentHover
				}
				txt(c, a.t("退出批量编辑", "Exit bulk edit"), fsTiny, col, 0)
			}) {
				parsed := parseBulk(a.bulk[o.id])
				delete(a.bulk, o.id)
				*rows = client.EnsureTrailingRow(parsed)
			}
		})
		v := text
		in := ui.TextAreaBase(c, &v).Placeholder("Content-Type: application/json\nAuthorization: Bearer token").PlaceholderColor(p.muted).
			FontSize(fsSmall).Font(monoFont).TextColor(p.fg).FillWidth().MinHeight(200).Padding(12).Radius(radiusInput).Background(p.surface)
		if in.FocusWithin() {
			in = in.Border(1, p.accent)
		} else {
			in = in.Border(1, p.border)
		}
		if in.Changed() {
			a.bulk[o.id] = v
		}
	})
	return false
}

// formTable 绘制 multipart/form-data 表：每行可选文本或文件。
func (a *app) formTable(c *ui.Context, rows *[]*client.FormRow) bool {
	p := a.p
	changed := false
	*rows = client.EnsureTrailingFormRow(*rows)
	remove := -1
	a.tableFrame(c, func() {
		ui.Row(c).FillWidth().Children(func() {
			a.kvHeaderCell(c, 28, "", nil)
			a.kvHeaderCell(c, colKey, a.t("键", "Key"), nil)
			a.tableCell(c, 72, kvHeadH, func() {
				ui.Row(c).FillWidth().Justify(ui.Center).Children(func() {
					txt(c, a.t("类型", "Type"), fsSmall, p.fg2, 600)
				})
			})
			a.kvHeaderCell(c, colValue, a.t("值", "Value"), nil)
			a.kvHeaderCellEnd(c, colDesc, a.t("说明", "Description"))
			a.tableCellEnd(c, 32, kvHeadH, func() {})
		})
		for i, r := range *rows {
			i, r := i, r
			last := r.IsPlaceholder()
			rowEl := ui.Row(c.Key(r.ID)).FillWidth()
			hover := rowEl.Hovered()
			rowEl.Children(func() {
				a.tableCell(c, 28, kvRowH, func() {
					ui.Row(c).FillWidth().Height(kvRowH - 2).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
						changed = a.kvCheck(c, r.ID, &r.Enabled, last) || changed
					})
				})
				dim := !r.Enabled && !last
				a.tableCell(c, colKey, kvRowH, func() {
					in := a.kvCellInput(c, "k:"+r.ID, &r.Key, a.t("键", "Key"), true, p.fg, dim, false)
					if dim {
						in = in.Strikethrough()
					}
					changed = in.Changed() || changed
				})
				a.tableCell(c, 72, kvRowH, func() {
					ui.Row(c).FillWidth().Justify(ui.Center).Children(func() {
						label := a.t("文本", "Text")
						if r.Type == "file" {
							label = a.t("文件", "File")
						}
						ui.ButtonBase(c.Key("ty:" + r.ID)).Height(18).Radius(4).PaddingX(6).Gap(4).AlignItems(ui.Center).
							PopoverMenuButton(func(m *ui.Menu) {
								if m.Item(a.t("文本", "Text")).Checked(r.Type != "file").Chosen() {
									a.later(func() { r.Type, r.Value, r.FilePath, r.FileName = "text", "", "", "" })
								}
								if m.Item(a.t("文件", "File")).Checked(r.Type == "file").Chosen() {
									a.later(func() { r.Type, r.Value, r.FilePath, r.FileName = "file", "", "", "" })
								}
							}).Children(func() {
							txt(c, label, fsTiny, p.muted, 0)
							icon(c, "arrowDown", 8, p.muted)
						})
					})
				})
				a.tableCell(c, colValue, kvRowH, func() {
					if r.Type == "file" {
						label, col := r.FileName, p.fg
						if label == "" {
							label, col = a.t("选择文件…", "Choose file…"), p.accent
						}
						if a.button(c, btnOpts{h: 24, padX: 8, noHover: true, key: "file:" + r.ID}, func(bool) {
							txtE(c, label, fsSmall, col, 0)
						}) {
							a.pickFormFile(r)
						}
						return
					}
					changed = a.kvCellInput(c, "v:"+r.ID, &r.Value, a.t("值", "Value"), true, p.fg, dim, last).Changed() || changed
				})
				a.tableCellEnd(c, colDesc, kvRowH, func() {
					changed = a.kvCellInput(c, "d:"+r.ID, &r.Description, a.t("说明", "Description"), false, p.fg2, dim, last).Changed() || changed
				})
				a.tableCellEnd(c, 32, kvRowH, func() {
					ui.Row(c).FillWidth().Justify(ui.Center).Children(func() {
						if !last && hover && a.miniDelete(c, "del:"+r.ID, nil) {
							remove = i
						}
					})
				})
			})
		}
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
		// 后台任务出现 panic 时只记录日志，不让整个应用崩溃。
		defer func() {
			if r := recover(); r != nil {
				logger.Error("后台任务异常", "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
			}
		}()
		if done := work(); done != nil {
			a.win.Update(done)
		}
	}()
}

func itoa(n int) string { return strconv.Itoa(n) }

// kvCheck 绘制键值表每行的“启用”复选框。末尾的空占位行还不是一条真实数据，复选框列留空；
// 一旦在占位行里输入内容，它就变成真实行，复选框以勾选态正常显示。
func (a *app) kvCheck(c *ui.Context, id string, on *bool, placeholder bool) bool {
	if placeholder {
		return false // 原型里占位行的复选框列是空的
	}
	return a.checkbox(c, "chk:"+id, on, false)
}
