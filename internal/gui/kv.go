package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"

	"minipost/internal/client"
	"minipost/internal/pkg/logger"
)

// 表格列：对照设计稿 grid-template-columns: 34px 1fr 1.5fr 1fr 28px。
// 正数是伸缩权重；负数表示固定像素宽度（取绝对值）。
const (
	colCheck = -34
	colKey   = 1
	colValue = 1.5
	colDesc  = 1
	colAct   = -28
	colType  = -92 // form-data 的类型列
)

// 键值表的行高、表头高度与单元格字号（对照原型：行更高、字更大，读起来更松）。
const (
	kvRowH  = 28
	kvHeadH = 28
	kvFont  = 12
	subBarH = 34 // 表格上方的子工具条高度
)

// kvOpts 描述键值表的列与自动请求头行。
type kvOpts struct {
	id     string // 表格唯一标识，用于批量编辑状态与控件 key
	keyPH  string
	valPH  string
	noDesc bool
	t      *tab   // 提供自动请求头显隐状态；nil 表示没有自动请求头
	hint   string // 表格上方左侧的说明文字（没有自动请求头时显示）
	lead   func() // 表格上方左侧的自定义内容（优先于 hint；Body 的类型分段控件）
}

// kvSubhead 绘制表格上方的一行：左侧是自动请求头折叠开关（或说明文字），右侧是“批量编辑”。
func (a *app) kvSubhead(c *ui.Context, rows *[]*client.Row, o kvOpts, auto []client.AutoHeader) {
	p := a.p
	a.subBar(c, func() {
		if o.lead != nil {
			o.lead()
		} else if len(auto) > 0 && o.t != nil {
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
			txt(c, o.hint, fsSmall, p.muted, 0)
		}
	}, func() {
		if a.button(c, btnOpts{h: 22, padX: 4, gap: 4, noHover: true, key: "bulk-" + o.id}, func(hover bool) {
			col := p.accent
			if hover {
				col = p.accentHover
			}
			icon(c, "formatList", 12, col)
			txt(c, a.t("批量编辑", "Bulk Edit"), fsSmall, col, 600)
		}) {
			a.bulk[o.id] = serializeBulk(*rows)
		}
	})
}

// subBar 是 Params / Headers / Body 顶部共用的 34 高条带：浅底 + 底部发丝线，左侧放说明或分段控件，右侧放动作
// （设计稿 .hint）。三个面板用同一个条带，切换子标签时表格的位置不会跳动。
func (a *app) subBar(c *ui.Context, left, right func()) {
	p := a.p
	ui.Row(c).FillWidth().Height(subBarH).Shrink(0).PaddingX(12).AlignItems(ui.Center).Gap(8).
		Background(p.surface2.Mix(p.surface, 0.62)).BorderWidth(0, 0, 1, 0).BorderColor(p.border.Alpha(0.7)).Children(func() {
		left()
		ui.Spacer(c)
		right()
	})
}

// tableCell 画一个带下分隔线和右分隔线（列间竖线）的表格单元。
func (a *app) tableCell(c *ui.Context, grow, h float32, children func()) {
	a.cellBox(c, grow, h).BorderWidth(0, 1, 1, 0).BorderColor(a.p.border.Alpha(0.7)).Children(children)
}

// cellBox 按列定义（见 colCheck 等）生成固定宽度或按权重伸缩的单元格容器。
func (a *app) cellBox(c *ui.Context, grow, h float32) ui.Element {
	r := ui.Row(c).Height(h).AlignItems(ui.Center)
	if grow < 0 {
		return r.Width(-grow).Shrink(0)
	}
	return r.Grow(grow).Basis(0).Shrink(1)
}

// tableCellEnd 是表格最后一个单元（行末操作列）：只有下分隔线，没有右竖线。
func (a *app) tableCellEnd(c *ui.Context, grow, h float32, children func()) {
	a.cellBox(c, grow, h).BorderWidth(0, 0, 1, 0).BorderColor(a.p.border.Alpha(0.7)).Children(children)
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

// kvHeaderRow 是表头行：柔和底色 + 底边实线（所有表格共用，对应设计稿 .kv .hd）。
func (a *app) kvHeaderRow(c *ui.Context, cells func()) {
	p := a.p
	ui.Row(c).FillWidth().Height(kvHeadH).Background(p.surface2.Mix(p.surface, 0.42)).BorderWidth(0, 0, 1, 0).BorderColor(p.border).Children(cells)
}

// kvHeaderCell 绘制表头单元：11px 半粗、次要色、左对齐；与数据行一致，右侧有一条列间发丝线。
func (a *app) kvHeaderCell(c *ui.Context, grow float32, label string, extra func()) {
	a.cellBox(c, grow, kvHeadH).BorderWidth(0, 1, 0, 0).BorderColor(a.p.border.Alpha(0.7)).Children(func() {
		ui.Row(c).FillWidth().PaddingX(8).AlignItems(ui.Center).Children(func() {
			txtE(c, label, fsSmall, a.p.muted, 600).Grow(1)
			if extra != nil {
				extra()
			}
		})
	})
}

// kvHeaderCellEnd 是最后一列的表头单元：没有右竖线。
func (a *app) kvHeaderCellEnd(c *ui.Context, grow float32, label string) {
	a.cellBox(c, grow, kvHeadH).Children(func() {
		ui.Row(c).FillWidth().PaddingX(8).AlignItems(ui.Center).Children(func() {
			txtE(c, label, fsSmall, a.p.muted, 600).Grow(1)
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
		a.kvHeaderRow(c, func() {
			a.kvHeaderCell(c, colCheck, "", nil)
			a.kvHeaderCell(c, colKey, o.keyPH, nil)
			a.kvHeaderCell(c, colValue, o.valPH, nil)
			if !o.noDesc {
				a.kvHeaderCellEnd(c, colDesc, a.t("说明", "Description"))
			}
			a.kvHeaderCellEnd(c, colAct, "")
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
		a.kvHeaderRow(c, func() {
			a.kvHeaderCell(c, colCheck, "", nil)
			a.kvHeaderCell(c, colKey, a.t("键", "Key"), nil)
			a.kvHeaderCell(c, colType, a.t("类型", "Type"), nil)
			a.kvHeaderCell(c, colValue, a.t("值", "Value"), nil)
			a.kvHeaderCellEnd(c, colDesc, a.t("说明", "Description"))
			a.kvHeaderCellEnd(c, colAct, "")
		})
		for i, r := range *rows {
			i, r := i, r
			last := r.IsPlaceholder()
			rowEl := ui.Row(c.Key(r.ID)).FillWidth()
			hover := rowEl.Hovered()
			rowEl.Children(func() {
				a.tableCell(c, colCheck, kvRowH, func() {
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
				a.tableCell(c, colType, kvRowH, func() {
					if last {
						return // 末尾的空白占位行不显示类型
					}
					ui.Row(c).FillWidth().PaddingX(8).Children(func() {
						label := a.t("文本", "Text")
						if r.Type == "file" {
							label = a.t("文件", "File")
						}
						// 设计稿 .tsel：20 高的小圆角下拉，浅底 + 0.5px 描边，文字 11px
						ty := ui.ButtonBase(c.Key("ty:"+r.ID)).Height(20).Radius(6).PaddingStart(8).PaddingEnd(4).Gap(3).AlignItems(ui.Center).Background(p.surface2).Border(0.5, p.border)
						if ty.Hovered() {
							ty = ty.Background(p.press)
						}
						ty.Transition(transColors).PopoverMenuButton(func(m *ui.Menu) {
							if m.Item(a.t("文本", "Text")).Checked(r.Type != "file").Chosen() {
								a.later(func() { r.Type, r.Value, r.FilePath, r.FileName = "text", "", "", "" })
							}
							if m.Item(a.t("文件", "File")).Checked(r.Type == "file").Chosen() {
								a.later(func() { r.Type, r.Value, r.FilePath, r.FileName = "file", "", "", "" })
							}
						}, a.menuOpts()).Children(func() {
							txt(c, label, fsSmall, p.fg2, 550)
							icon(c, "chevUD", 10, p.muted)
						})
					})
				})
				a.tableCell(c, colValue, kvRowH, func() {
					if r.Type == "file" {
						if r.FileName == "" {
							if a.button(c, btnOpts{h: 24, padX: 8, noHover: true, key: "file:" + r.ID}, func(bool) {
								txtE(c, a.t("选择文件…", "Choose file…"), fsSmall, p.accent, 0)
							}) {
								a.pickFormFile(r)
							}
							return
						}
						// 设计稿 .fchip：已选文件显示为淡色胶囊（图片图标 + 加粗文件名 + 大小），点击重新选择
						ui.Row(c).FillWidth().PaddingX(8).Children(func() {
							chip := ui.ButtonBase(c.Key("file:" + r.ID)).Height(20).Radius(10).PaddingX(8).Gap(5).AlignItems(ui.Center).Background(p.acSoft).Tooltip(r.FilePath)
							chip.Transition(transColors).Children(func() {
								icon(c, "image", 12, p.accent)
								txtE(c, r.FileName, fsSmall, p.accent, 650)
								if fi, err := os.Stat(r.FilePath); err == nil {
									txt(c, formatSizeShort(fi.Size()), fsTiny, p.muted, 0)
								}
							})
							if chip.Clicked() {
								a.pickFormFile(r)
							}
						})
						return
					}
					changed = a.kvCellInput(c, "v:"+r.ID, &r.Value, a.t("值", "Value"), true, p.fg, dim, last).Changed() || changed
				})
				a.tableCellEnd(c, colDesc, kvRowH, func() {
					changed = a.kvCellInput(c, "d:"+r.ID, &r.Description, a.t("说明", "Description"), false, p.fg2, dim, last).Changed() || changed
				})
				a.tableCellEnd(c, colAct, kvRowH, func() {
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
