package gui

import (
	"encoding/json"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/google/uuid"

	"minipost/internal/client"
	"minipost/internal/model"
	"minipost/internal/pkg/logger"
)

// envEdit 是环境编辑页（以标签页形式打开）的状态：环境的可编辑副本，保存后才写回。
type envEdit struct {
	env  model.Environment
	base string // 最近一次保存（或载入）时的内容指纹，用于判断是否有未保存修改
}

// fingerprint 生成忽略末尾空白行后的内容指纹。
func (e *envEdit) fingerprint() string {
	type row struct {
		K, V   string
		On, Sc bool
	}
	rows := []row{}
	for _, v := range e.env.Variables {
		if strings.TrimSpace(v.Key) == "" && v.Value == "" {
			continue
		}
		rows = append(rows, row{strings.TrimSpace(v.Key), v.Value, v.Enabled, v.IsSecret})
	}
	b, _ := json.Marshal(struct {
		Name string
		Rows []row
	}{strings.TrimSpace(e.env.Name), rows})
	return string(b)
}

func (e *envEdit) dirty() bool { return e.fingerprint() != e.base }

// newEnvTab 为环境创建编辑标签；草稿只是占位，保证请求标签的通用逻辑不会空指针。
func newEnvTab(e model.Environment) *tab {
	e.Variables = append([]model.Variable(nil), e.Variables...)
	ensureBlankVar(&e)
	ed := &envEdit{env: e}
	ed.base = ed.fingerprint()
	d := &client.Draft{ID: "env:" + e.ID}
	t := newTab(d, d.Clone())
	t.env = ed
	return t
}

func (a *app) openEnvManager() { a.openEnvManagerFor(a.envID) }

// openEnvManagerFor 打开指定环境的编辑标签；id 为空时取当前或第一个环境，没有环境则先新建。
func (a *app) openEnvManagerFor(id string) {
	if id == "" && len(a.envs) > 0 {
		id = a.envs[0].ID
	}
	if id == "" {
		a.newEnvironment()
		return
	}
	for i, t := range a.tabs {
		if t.env != nil && t.env.env.ID == id {
			a.showHome = false
			a.activate(i)
			return
		}
	}
	for _, e := range a.envs {
		if e.ID == id {
			a.tabs = append(a.tabs, newEnvTab(e))
			a.showHome = false
			a.activate(len(a.tabs) - 1)
			logger.Info("已打开环境编辑页", "envID", id)
			return
		}
	}
}

// ensureBlankVar 保证变量表末尾有一行空白行供输入。
func ensureBlankVar(e *model.Environment) {
	n := len(e.Variables)
	if n == 0 || e.Variables[n-1].Key != "" || e.Variables[n-1].Value != "" {
		e.Variables = append(e.Variables, model.Variable{ID: uuid.NewString(), Enabled: true})
	}
}

// envPage 绘制环境编辑页：44px 标题行（环境名输入、保存、关闭）+ 变量表。
func (a *app) envPage(c *ui.Context, t *tab) {
	p := a.p
	ed := t.env
	ui.Column(c).Grow(1).Basis(0).Background(p.surface).Children(func() {
		ui.Row(c).Height(44).FillWidth().Shrink(0).PaddingX(16).AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				icon(c, "globe", 14, p.accent)
				bg := p.surface2
				bdc := p.border
				row := ui.Row(c).Height(28).MinWidth(220).MaxWidth(420).Radius(6).Background(bg).AlignItems(ui.Center)
				if row.FocusWithin() {
					bdc = p.accent
				}
				row.Border(1, bdc).Children(func() {
					ui.TextInputBase(c, &ed.env.Name).Placeholder("Environment name").PlaceholderColor(p.muted).
						FontSize(fsLarge).FontWeight(500).TextColor(p.fg).Grow(1).Basis(0).FillHeight().PaddingX(8)
				})
			})
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				dirty := ed.dirty()
				col := p.muted
				if dirty {
					col = p.fg2
				}
				if a.button(c, btnOpts{h: sizeBtn, radius: radiusBtn, padX: 12, gap: 4, disabled: !dirty, key: "env-save", tip: a.t("保存", "Save") + " (⌘S)"}, func(bool) {
					icon(c, "save", 12, col)
					txt(c, a.t("保存", "Save"), fsTiny, col, 500)
				}) {
					a.later(func() { a.saveTab(t) })
				}
				if a.iconButton(c, "clear", a.t("关闭", "Close"), 14, p.muted) {
					a.later(func() { a.closeTab(a.active) })
				}
			})
		})
		ui.Scroll(c).Grow(1).Basis(0).Scrollbars(a.scrollbarsVisible()).Children(func() {
			ui.Column(c).FillWidth().Padding(20, 16, 16, 16).Children(func() {
				a.envTable(c, ed)
			})
		})
	})
}

// envTable 是变量表：表头 34px，行高 32px，两列等宽加 44px 操作列。
func (a *app) envTable(c *ui.Context, ed *envEdit) {
	p := a.p
	ui.Column(c).FillWidth().Border(1, p.border).Background(p.surface).Children(func() {
		ui.Row(c).FillWidth().Height(34).Background(p.surface2).AlignItems(ui.Stretch).Children(func() {
			for _, h := range []string{"Variable", "Value"} {
				ui.Row(c.Key(h)).Grow(1).Basis(0).MinWidth(220).PaddingX(12).AlignItems(ui.Center).BorderWidth(0, 1, 0, 0).BorderColor(p.border).Children(func() {
					txt(c, h, fsBase, p.fg2, 600)
				})
			}
			ui.Row(c).Width(44).Shrink(0).AlignItems(ui.Center).Justify(ui.Center).Children(func() { icon(c, "more", 13, p.muted) })
		})
		hline(c, p.border)
		remove := -1
		vars := ed.env.Variables
		for i := range vars {
			v := &vars[i]
			last := i == len(vars)-1 && v.Key == "" && v.Value == ""
			row := ui.Row(c.Key(v.ID)).FillWidth().MinHeight(32).AlignItems(ui.Stretch)
			hover := row.Hovered() && !last
			if hover {
				row = row.Background(p.surface2)
			}
			row.Children(func() {
				ph := "Variable"
				if last {
					ph = "Add variable"
				}
				ui.Row(c).Grow(1).Basis(0).MinWidth(220).BorderWidth(0, 1, 0, 0).BorderColor(p.border).Children(func() {
					ui.TextInputBase(c.Key("k"), &v.Key).Placeholder(ph).PlaceholderColor(p.muted).FontSize(fsBase).TextColor(p.fg).Grow(1).Basis(0).Height(32).PaddingX(12)
				})
				ui.Row(c).Grow(1).Basis(0).MinWidth(220).BorderWidth(0, 1, 0, 0).BorderColor(p.border).Children(func() {
					vp := "Value"
					if last {
						vp = ""
					}
					in := ui.TextInputBase(c.Key("v"), &v.Value).Placeholder(vp).PlaceholderColor(p.muted).FontSize(fsBase).TextColor(p.fg).Grow(1).Basis(0).Height(32).PaddingX(12)
					if last {
						in.Disabled(true).Opacity(0.5)
					}
					if v.IsSecret {
						in.Password()
					}
				})
				ui.Row(c).Width(44).Shrink(0).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
					if hover && a.button(c, btnOpts{w: sizeBtn, h: sizeBtn, radius: 6, key: "del", tip: a.t("删除变量", "Delete variable")}, func(h bool) {
						col := p.muted
						if h {
							col = p.danger
						}
						icon(c, "delete", 12, col)
					}) {
						remove = i
					}
				})
			})
			if i < len(vars)-1 {
				hline(c, p.border)
			}
		}
		if remove >= 0 {
			ed.env.Variables = append(ed.env.Variables[:remove], ed.env.Variables[remove+1:]...)
		}
		ensureBlankVar(&ed.env)
	})
}

// saveEnvTab 去掉空白行后保存环境编辑页的内容。
func (a *app) saveEnvTab(t *tab) bool {
	ed := t.env
	d := ed.env
	d.Name = strings.TrimSpace(d.Name)
	if d.Name == "" {
		a.toastWarn(a.t("环境名称不能为空", "Environment name is required"))
		return false
	}
	vars := make([]model.Variable, 0, len(d.Variables))
	for _, v := range d.Variables {
		if v.Key == "" && v.Value == "" {
			continue
		}
		v.Key = strings.TrimSpace(v.Key)
		vars = append(vars, v)
	}
	d.Variables = vars
	if err := a.be.SaveEnvironment(d); err != nil {
		a.fail(a.t("保存环境", "Save environment"), err)
		return false
	}
	ed.env.Name = d.Name
	ed.base = ed.fingerprint()
	a.reloadEnvs()
	a.toastOK(a.t("环境已保存", "Environment saved"))
	logger.Info("环境已保存", "envID", d.ID, "variables", len(vars))
	return true
}

func (a *app) newEnvironment() {
	a.overlays.askText(a.t("新建环境", "New Environment"), "", a.t("新环境", "New Environment"), a.t("创建", "Create"), func(name string) {
		e, err := a.be.CreateEnvironment(a.projectID, name)
		if err != nil {
			a.fail(a.t("创建环境", "Create environment"), err)
			return
		}
		a.reloadEnvs()
		logger.Info("已创建环境", "envID", e.ID)
		a.openEnvManagerFor(e.ID)
	})
}

func (a *app) deleteEnvironment(id, name string) {
	a.overlays.askConfirm(a.t("删除环境“", "Delete environment “")+name+a.t("”？", "”?"), a.t("删除后可立即撤销。", "You can undo right after deleting."), a.t("删除", "Delete"), func() {
		var saved *model.Environment
		for i := range a.envs {
			if a.envs[i].ID == id {
				e := a.envs[i]
				saved = &e
			}
		}
		pid := a.projectID
		if err := a.be.DeleteEnvironment(a.projectID, id); err != nil {
			a.fail(a.t("删除环境", "Delete environment"), err)
			return
		}
		if a.envID == id {
			a.setEnv("")
		}
		for i := len(a.tabs) - 1; i >= 0; i-- {
			if a.tabs[i].env != nil && a.tabs[i].env.env.ID == id {
				a.removeTab(i)
			}
		}
		a.reloadEnvs()
		if saved != nil {
			a.pushUndo(a.t("环境“", "Environment “")+name+"”", func() {
				if err := a.be.SaveEnvironment(*saved); err != nil {
					a.fail(a.t("撤销删除", "Undo delete"), err)
					return
				}
				if pid == a.projectID {
					a.reloadEnvs()
				}
			})
		}
	})
}
