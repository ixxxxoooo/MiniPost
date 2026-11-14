package gui

import (
	"encoding/json"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/google/uuid"

	"minipost/internal/client"
	"minipost/internal/model"
	"minipost/internal/pkg/httputil"
	"minipost/internal/pkg/logger"
)

// envEdit 是环境编辑页（以标签页形式打开）的状态：环境的可编辑副本，保存后才写回。
type envEdit struct {
	env     model.Environment
	base    string   // 最近一次保存（或载入）时的内容指纹，用于判断是否有未保存修改
	samples []string // 右侧“动态变量”面板里各变量的示例值（打开页面时求值一次，避免随机值逐帧跳动）
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
	for _, name := range httputil.DynamicVarNames {
		ed.samples = append(ed.samples, httputil.ResolveDynamicVars("{{$"+name+"}}"))
	}
	d := &client.Draft{ID: "env:" + e.ID}
	t := newTab(d, d.Clone())
	t.env = ed
	return t
}

func (a *app) openEnvManager() { a.openEnvManagerFor(a.envID) }

// envTabOpenActive 报告当前标签是否正是 id 对应环境的编辑页（侧栏里据此高亮该行）。
func (a *app) envTabOpenActive(id string) bool {
	t := a.activeTab()
	return t != nil && t.env != nil && t.env.env.ID == id
}

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

// envPage 绘制环境编辑页（设计稿）：左侧是标题（环境名 + 当前使用）、说明与操作按钮、变量表，
// 右侧是“动态变量”面板。
func (a *app) envPage(c *ui.Context, t *tab) {
	p := a.p
	ed := t.env
	ui.Row(c).Grow(1).Basis(0).Background(p.surface).Children(func() {
		ui.Column(c).Grow(1).Basis(0).FillHeight().Children(func() {
			a.envHeader(c, t)
			ui.Scroll(c).Grow(1).Basis(0).Scrollbars(a.scrollbarsVisible()).Children(func() {
				ui.Column(c).FillWidth().Children(func() {
					a.envTable(c, ed)
				})
			})
			ui.Row(c).FillWidth().Height(30).Shrink(0).PaddingX(16).Gap(8).AlignItems(ui.Center).BorderWidth(1, 0, 0, 0).BorderColor(p.border).Children(func() {
				icon(c, "info", 12, p.success)
				txt(c, a.t("带锁的变量会在界面上遮罩，且不会写入日志。", "Locked variables are masked in the UI and never written to logs."), fsSmall, p.muted, 0)
			})
		})
		a.envDynamicPanel(c, ed)
	})
}

// envHeader 是环境页顶部：环境名（16/600，可直接改）、“当前使用”标记、说明，以及删除 / 设为当前 / 保存。
func (a *app) envHeader(c *ui.Context, t *tab) {
	p := a.p
	ed := t.env
	active := a.envID == ed.env.ID
	ui.Row(c).FillWidth().Shrink(0).Padding(14, 16, 12, 20).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).Basis(0).Gap(4).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				row := ui.Row(c.Key("env-name")).MinWidth(120).MaxWidth(360).Height(28).Radius(6).AlignItems(ui.Center)
				bdc := ui.RGBA(0, 0, 0, 0)
				switch {
				case row.FocusWithin():
					bdc = p.accent
				case row.Hovered():
					bdc = p.border
				}
				row.Border(1, bdc).Children(func() {
					ui.TextInputBase(c, &ed.env.Name).Placeholder(a.t("环境名称", "Environment name")).PlaceholderColor(p.muted).
						FontSize(16).FontWeight(600).TextColor(p.fg).Grow(1).Basis(0).FillHeight().PaddingX(6)
				})
				if active {
					ui.Row(c).Height(18).PaddingX(7).Radius(9).Background(p.success.Alpha(0.14)).AlignItems(ui.Center).Shrink(0).Children(func() {
						txt(c, a.t("当前使用", "In use"), fsTiny, p.success, 600)
					})
				}
			})
			ui.Text(c, a.t("变量可在请求的任意位置写成 {{变量名}} 使用，切换环境后自动替换。", "Use variables anywhere in a request as {{name}}; they are replaced when you switch environments.")).
				FontSize(fsBase).TextColor(p.fg2).Margin(0, 0, 0, 6)
		})
		border := p.btnBorder
		if a.button(c, btnOpts{w: 26, h: 26, radius: radiusBtn, border: &border, key: "env-del", tip: a.t("删除环境", "Delete environment")}, func(bool) {
			icon(c, "delete", 12, p.danger)
		}) {
			id, name := ed.env.ID, ed.env.Name
			a.later(func() { a.deleteEnvironment(id, name) })
		}
		if a.button(c, btnOpts{h: 26, radius: radiusBtn, padX: 10, border: &border, disabled: active, key: "env-use"}, func(bool) {
			col := p.fg
			if active {
				col = p.muted
			}
			txt(c, a.t("设为当前环境", "Use this environment"), fsBase, col, 0)
		}) {
			id, name := ed.env.ID, ed.env.Name
			a.later(func() {
				a.setEnv(id)
				a.toastOK(a.t("已切换到环境「"+name+"」", "Switched to environment “"+name+"”"))
				logger.Info("已从环境页设为当前环境", "envID", id)
			})
		}
		dirty := ed.dirty()
		bg := p.accent
		if a.button(c, btnOpts{h: 26, radius: radiusBtn, padX: 12, gap: 6, bg: &bg, hoverBg: &p.accentHover, disabled: !dirty, key: "env-save", tip: a.t("保存", "Save") + " (⌘S)"}, func(bool) {
			fg := p.accentFg
			if !dirty {
				fg = fg.Alpha(0.6)
			}
			txt(c, a.t("保存", "Save"), fsBase, fg, 600)
			txtMono(c, "⌘S", 9, fg.Alpha(0.8), 0)
		}) {
			a.later(func() { a.saveTab(t) })
		}
	})
}

// envDynamicPanel 是右侧的“动态变量”面板：列出内置动态变量、含义与一个示例值（打开页面时求值一次）。
func (a *app) envDynamicPanel(c *ui.Context, ed *envEdit) {
	p := a.p
	desc := map[string][2]string{
		"timestamp":    {"当前时间，秒", "Current time, seconds"},
		"timestampMs":  {"当前时间，毫秒", "Current time, milliseconds"},
		"isoTimestamp": {"ISO 8601 时间", "ISO 8601 time"},
		"randomUUID":   {"随机 UUID v4", "Random UUID v4"},
		"randomInt":    {"0–1000 的随机整数", "Random integer 0–1000"},
	}
	ui.Column(c.Key("env-dyn")).Width(240).FillHeight().Shrink(0).Background(p.sidebar).BorderWidth(0, 0, 0, 1).BorderColor(p.border).Children(func() {
		ui.Row(c).FillWidth().Shrink(0).Padding(14, 14, 8, 14).Gap(6).AlignItems(ui.Center).Children(func() {
			icon(c, "commandLine", 12, p.accent)
			txt(c, a.t("动态变量", "Dynamic variables"), fsBase, p.fg, 600)
			txt(c, a.t("发送时求值", "evaluated on send"), fsSmall, p.muted, 0)
		})
		ui.Scroll(c).Grow(1).Basis(0).Scrollbars(a.scrollbarsVisible()).Children(func() {
			ui.Column(c).FillWidth().PaddingX(12).PaddingY(4).Gap(8).Children(func() {
				for i, name := range httputil.DynamicVarNames {
					d := desc[name]
					ui.Column(c.Key("dyn"+name)).FillWidth().Padding(8, 10).Gap(2).Radius(radiusInput).Background(p.surface).Border(1, p.border).Children(func() {
						txtMono(c, "{{$"+name+"}}", fsBase, p.accent, 600).Selectable()
						txt(c, a.t(d[0], d[1]), fsSmall, p.fg2, 0)
						if i < len(ed.samples) {
							txtEMono(c, ed.samples[i], fsSmall, p.muted, 0)
						}
					})
				}
			})
		})
	})
}

// envTable 是变量表（设计稿）：启用勾选 | 变量名 | 值 | 密文锁 | 删除；行与行之间只有细横线，没有竖线。
func (a *app) envTable(c *ui.Context, ed *envEdit) {
	p := a.p
	const colEnable, colAct = 44, 32
	ui.Column(c).FillWidth().Children(func() {
		ui.Row(c).FillWidth().Height(28).AlignItems(ui.Center).BorderWidth(1, 0, 1, 0).BorderColor(p.border).Children(func() {
			ui.Box(c).Width(colEnable).Shrink(0)
			ui.Row(c.Key("h-key")).Grow(1).Basis(0).Children(func() { txt(c, a.t("变量名", "Variable"), fsSmall, p.muted, 600) })
			ui.Row(c.Key("h-val")).Grow(1.4).Basis(0).Children(func() { txt(c, a.t("值", "Value"), fsSmall, p.muted, 600) })
			ui.Box(c).Width(colAct * 2).Shrink(0)
		})
		remove := -1
		vars := ed.env.Variables
		for i := range vars {
			v := &vars[i]
			last := i == len(vars)-1 && v.Key == "" && v.Value == ""
			row := ui.Row(c.Key(v.ID)).FillWidth().Height(32).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(p.border)
			hover := row.Hovered() && !last
			if hover {
				row = row.Background(p.surface2)
			}
			row.Children(func() {
				ui.Row(c).Width(colEnable).Shrink(0).Justify(ui.Center).Children(func() {
					if !last {
						a.checkbox(c, "en", &v.Enabled, false)
					}
				})
				ph := a.t("变量名", "Variable")
				if last {
					ph = a.t("添加变量", "Add variable")
				}
				col := p.fg
				if !v.Enabled && !last {
					col = p.muted
				}
				ui.TextInputBase(c.Key("k"), &v.Key).Placeholder(ph).PlaceholderColor(p.muted).Font(monoFont).FontSize(fsBase).TextColor(col).Grow(1).Basis(0).Height(32).PaddingX(4)
				vp := a.t("值", "Value")
				if last {
					vp = ""
				}
				in := ui.TextInputBase(c.Key("v"), &v.Value).Placeholder(vp).PlaceholderColor(p.muted).Font(monoFont).FontSize(fsBase).TextColor(col).Grow(1.4).Basis(0).Height(32).PaddingX(4)
				if last {
					in.Disabled(true).Opacity(0.5)
				}
				if v.IsSecret {
					in.Password()
				}
				ui.Row(c).Width(colAct).Shrink(0).Justify(ui.Center).Children(func() {
					if last {
						return
					}
					tip := a.t("标为密文（遮罩显示）", "Mask this value")
					lc := p.muted.Alpha(0.6)
					if v.IsSecret {
						tip, lc = a.t("已标为密文，点击取消", "Masked; click to unmask"), p.success
					}
					if a.button(c, btnOpts{w: sizeBtn, h: sizeBtn, radius: 6, key: "lock", tip: tip}, func(bool) {
						icon(c, "lock", 12, lc)
					}) {
						v.IsSecret = !v.IsSecret
					}
				})
				ui.Row(c).Width(colAct).Shrink(0).Justify(ui.Center).Children(func() {
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
