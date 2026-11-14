package gui

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/google/uuid"

	"minipost/internal/model"
	"minipost/internal/pkg/logger"
)

// envState 是环境管理对话框的状态。
type envState struct {
	open  bool
	sel   int
	list  ui.ListState
	draft *model.Environment // 正在编辑的环境副本，点“保存”后才写回
	index int                // draft 对应的环境下标，用于发现选择变化
}

func (a *app) openEnvManager() {
	s := &a.overlays.envMgr
	*s = envState{open: true, index: -1}
	for i, e := range a.envs {
		if e.ID == a.envID {
			s.sel = i
		}
	}
	a.loadEnvDraft(s)
}

// loadEnvDraft 把选中的环境复制为可编辑草稿。
func (a *app) loadEnvDraft(s *envState) {
	s.index = s.sel
	s.draft = nil
	if s.sel < 0 || s.sel >= len(a.envs) {
		return
	}
	d := a.envs[s.sel]
	d.Variables = append([]model.Variable(nil), d.Variables...)
	ensureBlankVar(&d)
	s.draft = &d
}

// ensureBlankVar 保证变量表末尾有一行空白行供输入。
func ensureBlankVar(e *model.Environment) {
	n := len(e.Variables)
	if n == 0 || e.Variables[n-1].Key != "" || e.Variables[n-1].Value != "" {
		e.Variables = append(e.Variables, model.Variable{ID: uuid.NewString(), Enabled: true})
	}
}

// envDialogView 绘制环境管理对话框。
func (a *app) envDialogView(c *ui.Context) {
	s := &a.overlays.envMgr
	th := c.Theme()
	ui.Modal(c, &s.open, func() {
		ui.Text(c, a.t("环境管理", "Environments")).Bold()
		ui.Row(c).Gap(14).Children(func() {
			ui.Column(c).Width(190).Shrink(0).Gap(8).Children(func() {
				s.list.Selected = &s.sel
				ui.List(c, &s.list, len(a.envs), func(i int) { ui.Text(c, a.envs[i].Name).SingleLine() }).Height(300)
				if ui.Button(c, a.t("新建环境", "New Environment")).Clicked() {
					a.later(a.newEnvironment)
				}
			})
			ui.Column(c).Width(560).Gap(8).Children(func() {
				if s.index != s.sel {
					a.loadEnvDraft(s)
				}
				if s.draft == nil {
					ui.Text(c, a.t("没有环境，先新建一个。", "No environments yet — create one.")).TextColor(th.TextMuted)
					return
				}
				a.envEditor(c, s)
			})
		})
		ui.Row(c).Justify(ui.End).Children(func() {
			if ui.Button(c, a.t("关闭", "Close")).Clicked() {
				s.open = false
			}
		})
	})
}

func (a *app) envEditor(c *ui.Context, s *envState) {
	d := s.draft
	th := c.Theme()
	ui.TextInput(c, &d.Name).Placeholder(a.t("环境名称", "Environment name"))
	ui.Row(c).Gap(6).PaddingX(2).Children(func() {
		ui.Box(c).Width(22).Shrink(0)
		for _, h := range []string{a.t("变量", "Variable"), a.t("值", "Value")} {
			ui.Text(c, h).TextColor(th.TextMuted).Grow(1).Basis(0)
		}
		ui.Box(c).Width(70).Shrink(0)
		ui.Box(c).Width(28).Shrink(0)
	})
	remove := -1
	ui.Scroll(c).Height(220).Children(func() {
		ui.Column(c).Gap(4).Children(func() {
			for i := range d.Variables {
				i := i
				v := &d.Variables[i]
				ui.Row(c.Key(v.ID)).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Checkbox(c, &v.Enabled, "").Width(22).Shrink(0)
					ui.TextInput(c, &v.Key).Placeholder("name").Grow(1).Basis(0)
					val := ui.TextInput(c, &v.Value).Placeholder("value").Grow(1).Basis(0)
					if v.IsSecret {
						val.Password()
					}
					ui.Checkbox(c, &v.IsSecret, a.t("密钥", "Secret")).Width(70).Shrink(0)
					del := ui.Button(c, "×").Width(28).Shrink(0)
					if v.Key == "" && v.Value == "" {
						del.Invisible()
					} else if del.Clicked() {
						remove = i
					}
				})
			}
		})
	})
	if remove >= 0 {
		d.Variables = append(d.Variables[:remove], d.Variables[remove+1:]...)
	}
	ensureBlankVar(d)
	ui.Row(c).Gap(8).Children(func() {
		if ui.PrimaryButton(c, a.t("保存", "Save")).Clicked() {
			a.later(func() { a.saveEnvDraft(s) })
		}
		if ui.Button(c, a.t("删除环境", "Delete")).Clicked() {
			id, name := d.ID, d.Name
			a.later(func() { a.deleteEnvironment(id, name) })
		}
	})
}

func (a *app) newEnvironment() {
	a.overlays.askText(a.t("新建环境", "New Environment"), "", a.t("新环境", "New Environment"), a.t("创建", "Create"), func(name string) {
		e, err := a.be.CreateEnvironment(a.projectID, name)
		if err != nil {
			a.fail(a.t("创建环境", "Create environment"), err)
			return
		}
		a.reloadEnvs()
		s := &a.overlays.envMgr
		for i, x := range a.envs {
			if x.ID == e.ID {
				s.sel = i
			}
		}
		s.index = -1
		logger.Info("已创建环境", "envID", e.ID)
	})
}

// saveEnvDraft 去掉空白行后保存当前编辑的环境。
func (a *app) saveEnvDraft(s *envState) {
	d := *s.draft
	d.Name = strings.TrimSpace(d.Name)
	if d.Name == "" {
		a.toast(a.t("环境名称不能为空", "Environment name is required"))
		return
	}
	vars := make([]model.Variable, 0, len(d.Variables))
	for _, v := range d.Variables {
		if v.Key == "" && v.Value == "" {
			continue
		}
		vars = append(vars, v)
	}
	d.Variables = vars
	if err := a.be.SaveEnvironment(d); err != nil {
		a.fail(a.t("保存环境", "Save environment"), err)
		return
	}
	a.reloadEnvs()
	s.index = -1 // 重新载入草稿
	a.toast(a.t("环境已保存", "Environment saved"))
	logger.Info("环境已保存", "envID", d.ID, "variables", len(vars))
}

func (a *app) deleteEnvironment(id, name string) {
	a.overlays.askConfirm(a.t("删除环境“", "Delete environment “")+name+a.t("”？", "”?"), a.t("此操作无法撤销。", "This can't be undone."), a.t("删除", "Delete"), func() {
		if err := a.be.DeleteEnvironment(a.projectID, id); err != nil {
			a.fail(a.t("删除环境", "Delete environment"), err)
			return
		}
		if a.envID == id {
			a.setEnv("")
		}
		a.reloadEnvs()
		s := &a.overlays.envMgr
		s.sel = max(0, min(s.sel, len(a.envs)-1))
		s.index = -1
	})
}
