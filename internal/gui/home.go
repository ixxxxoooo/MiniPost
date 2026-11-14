package gui

import (
	"github.com/egoist/mygo/ui"

	"minipost/internal/pkg/logger"
)

// 项目主题色可选项（后端不接受空值，所以没有“无”选项）。
var projectColors = []string{"#3b82f6", "#10b981", "#f59e0b", "#ef4444", "#8b5cf6", "#ec4899", "#14b8a6"}

// projectDialogState 是新建/编辑项目对话框的状态。
type projectDialogState struct {
	open  bool
	id    string // 为空表示新建
	name  string
	desc  string
	color string
}

func (a *app) openProjectDialog(id string) {
	d := projectDialogState{open: true, id: id}
	if id != "" {
		for _, p := range a.projects {
			if p.ID == id {
				d.name, d.desc, d.color = p.Name, p.Description, p.ThemeColor
			}
		}
	}
	a.overlays.projDlg = d
}

// homeView 是项目首页：列出所有项目，可新建、编辑、删除或打开。
func (a *app) homeView(c *ui.Context) {
	th := c.Theme()
	ui.Column(c).Grow(1).AlignItems(ui.Center).Padding(32).Children(func() {
		ui.Column(c).Width(640).MaxWidth(640).Gap(16).Grow(1).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Children(func() {
				ui.Text(c, a.t("项目", "Projects")).FontSize(th.FontSize + 9).Bold()
				ui.Spacer(c)
				if len(a.tabs) > 0 && a.projectID != "" && ui.Button(c, a.t("返回工作区", "Back to Workspace")).Clicked() {
					a.showHome = false
				}
				if ui.PrimaryButton(c, a.t("新建项目", "New Project")).Clicked() {
					a.openProjectDialog("")
				}
			})
			if len(a.projects) == 0 {
				ui.Text(c, a.t("还没有项目，先新建一个吧。", "No projects yet. Create one to get started.")).TextColor(th.TextMuted)
				return
			}
			ui.Scroll(c).Grow(1).Children(func() {
				ui.Column(c).Gap(8).Children(func() {
					for _, p := range a.projects {
						p := p
						ui.Row(c.Key(p.ID)).Gap(12).AlignItems(ui.Center).Padding(12).Radius(8).Background(th.Surface).Children(func() {
							dot := ui.Box(c).Size(12, 12).Radius(6).Shrink(0)
							if p.ThemeColor != "" {
								dot.Background(ui.Hex(p.ThemeColor))
							} else {
								dot.Background(th.Accent)
							}
							ui.Column(c).Grow(1).Gap(2).Children(func() {
								ui.Text(c, p.Name).Bold().SingleLine()
								if p.Description != "" {
									ui.Text(c, p.Description).TextColor(th.TextMuted).SingleLine()
								}
							})
							if ui.PrimaryButton(c, a.t("打开", "Open")).Clicked() {
								a.later(func() { a.selectProject(p.ID) })
							}
							if ui.Button(c, a.t("编辑", "Edit")).Clicked() {
								a.later(func() { a.openProjectDialog(p.ID) })
							}
							if ui.Button(c, a.t("删除", "Delete")).Clicked() {
								a.later(func() { a.deleteProject(p.ID, p.Name) })
							}
						})
					}
				})
			})
		})
	})
}

func (a *app) deleteProject(id, name string) {
	title := a.t("删除项目“", "Delete project “") + name + a.t("”？", "”?")
	msg := a.t("项目内的所有请求和环境都会被删除，此操作无法撤销。", "All requests and environments in it will be deleted. This can't be undone.")
	a.overlays.askConfirm(title, msg, a.t("删除", "Delete"), func() {
		if err := a.be.DeleteProject(id); err != nil {
			a.fail(a.t("删除项目", "Delete project"), err)
			return
		}
		if id == a.projectID {
			a.projectID = ""
			a.selectProject("")
		}
		a.loadProjects()
		logger.Info("已删除项目", "projectID", id)
	})
}

// projectDialogView 绘制新建/编辑项目对话框。
func (a *app) projectDialogView(c *ui.Context) {
	d := &a.overlays.projDlg
	th := c.Theme()
	ui.Modal(c, &d.open, func() {
		title := a.t("新建项目", "New Project")
		if d.id != "" {
			title = a.t("编辑项目", "Edit Project")
		}
		ui.Text(c, title).Bold()
		ui.Text(c, a.t("名称", "Name")).TextColor(th.TextMuted)
		submit := ui.TextInput(c, &d.name).AutoFocus().MinWidth(360).Submitted()
		ui.Text(c, a.t("描述", "Description")).TextColor(th.TextMuted)
		ui.TextInput(c, &d.desc).MinWidth(360)
		ui.Text(c, a.t("主题色", "Theme color")).TextColor(th.TextMuted)
		ui.Row(c).Gap(8).Children(func() {
			for _, col := range projectColors {
				col := col
				sw := ui.Box(c.Key("sw"+col)).Size(24, 24).Radius(12).Background(ui.Hex(col))
				if col == d.color {
					sw = sw.Border(2, th.Text)
				}
				if sw.Clicked() {
					d.color = col
				}
			}
		})
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			cancel := ui.Button(c, a.t("取消", "Cancel")).Clicked()
			ok := ui.PrimaryButton(c, a.t("保存", "Save")).Clicked()
			if cancel {
				d.open = false
			}
			if (ok || submit) && trimName(d.name) != "" {
				d.open = false
				saved := *d
				a.later(func() { a.saveProjectDialog(saved) })
			}
		})
	})
}

func (a *app) saveProjectDialog(d projectDialogState) {
	name := trimName(d.name)
	id := d.id
	if id == "" {
		p, err := a.be.CreateProject(name)
		if err != nil {
			a.fail(a.t("创建项目", "Create project"), err)
			return
		}
		id = p.ID
		logger.Info("已创建项目", "projectID", id)
	} else if _, err := a.be.RenameProject(id, name); err != nil {
		a.fail(a.t("重命名项目", "Rename project"), err)
		return
	}
	if _, err := a.be.UpdateProjectDescription(id, d.desc); err != nil {
		a.fail(a.t("保存项目描述", "Save description"), err)
	}
	if d.color != "" {
		if _, err := a.be.UpdateProjectTheme(id, d.color); err != nil {
			a.fail(a.t("保存主题色", "Save theme color"), err)
		}
	}
	a.loadProjects()
	a.applied.ready = false // 主题色可能已变化
	if d.id == "" {
		a.selectProject(id)
	}
}
