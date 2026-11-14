package gui

import (
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"minipost/internal/model"
	"minipost/internal/pkg/logger"
)

// 项目主题色可选项（同旧版 PROJECT_THEME_COLORS）。
var projectColors = []string{"#0A84FF", "#30D158", "#FF9F0A", "#FF453A", "#BF5AF2", "#64D2FF", "#5E5CE6", "#FF375F", "#34C759", "#FFD60A", "#FF6B35", "#00C7BE"}

// homeView 是项目主页：标题与新建项目输入在上，项目卡片网格在下。
func (a *app) homeView(c *ui.Context) {
	p := a.p
	cw, _ := c.Size()
	ui.Scroll(c).Grow(1).Basis(0).Background(p.surface).Scrollbars(a.scrollbarsVisible()).Children(func() {
		ui.Column(c).FillWidth().PaddingX(32).PaddingY(24).AlignItems(ui.Center).Children(func() {
			ui.Column(c).FillWidth().MaxWidth(1080).Children(func() {
				a.homeHeader(c)
				projects := append([]model.Project(nil), a.projects...)
				sort.SliceStable(projects, func(i, j int) bool { return projects[i].UpdatedAt > projects[j].UpdatedAt })
				if len(projects) == 0 {
					a.homeEmpty(c)
					return
				}
				cols := 1
				switch {
				case cw >= 1280:
					cols = 3
				case cw >= 768:
					cols = 2
				}
				ui.Column(c).FillWidth().Gap(16).Children(func() {
					for r := 0; r < len(projects); r += cols {
						ui.Row(c.Key("hr" + itoa(r))).FillWidth().Gap(16).AlignItems(ui.Start).Children(func() {
							for k := 0; k < cols; k++ {
								if r+k < len(projects) {
									a.projectCard(c, projects[r+k])
								} else {
									ui.Box(c.Key("pad" + itoa(k))).Grow(1).Basis(0)
								}
							}
						})
					}
				})
			})
		})
	})
}

func (a *app) homeHeader(c *ui.Context) {
	p := a.p
	ui.Row(c).FillWidth().Margin(0, 0, 20, 0).AlignItems(ui.End).Justify(ui.SpaceBetween).Gap(16).Children(func() {
		ui.Column(c).Grow(1).Basis(0).Children(func() {
			txt(c, a.t("项目主页", "Project home"), 20, p.fg, 600)
			ui.Text(c, a.t("选择项目继续工作，并在这里配置项目主题色。", "Select a project to continue and configure its theme color here.")).FontSize(fsLarge).TextColor(p.fg2).Margin(4, 0, 0, 0)
			ui.Text(c, a.t("项目主题色会同步应用到选中态、高亮态以及发送按钮。", "Project theme color will be applied to selected states, highlights, and the send button.")).FontSize(fsBase).TextColor(p.muted).Margin(4, 0, 0, 0)
		})
		ui.Row(c).Width(320).Shrink(0).Gap(8).AlignItems(ui.Center).Children(func() {
			in := a.input(c, &a.homeName, a.t("输入项目名后回车新建", "Enter project name and press Enter"), fieldOpts{h: 28, radius: radiusInput, fs: fsVSm, padX: 12, key: "home-new"})
			bg := p.accent
			create := a.button(c, btnOpts{h: sizeBtn, radius: radiusBtn, padX: 12, bg: &bg, hoverBg: &p.accentHover, key: "home-create"}, func(bool) {
				txt(c, a.t("新建项目", "New project"), fsVSm, p.accentFg, 500)
			})
			if create || in.Submitted() {
				a.later(a.createProjectFromHome)
			}
		})
	})
}

func (a *app) homeEmpty(c *ui.Context) {
	p := a.p
	ui.Row(c).FillWidth().Height(300).Radius(12).Border(1, p.border).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
		ui.Column(c).AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Row(c).Size(40, 40).Radius(10).Background(p.surface2).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
				icon(c, "folderOpen", 18, p.muted)
			})
			txt(c, a.t("还没有项目，先创建一个开始吧", "No projects yet. Create one to get started."), 14, p.fg2, 0)
		})
	})
}

// projectCard 是一张项目卡片：颜色点、名称、“当前”标记、更新时间、描述输入与主题色选择。
func (a *app) projectCard(c *ui.Context, pr model.Project) {
	p := a.p
	active := pr.ID == a.projectID
	card := ui.Column(c.Key("pc:" + pr.ID)).Grow(1).Basis(0).Radius(12).Padding(16).Gap(0).
		ContextMenu(func(m *ui.Menu) { a.projectCardMenu(m, pr) })
	hover := card.Hovered()
	switch {
	case active:
		card = card.Background(p.selected).Border(1, p.accent)
	case hover:
		card = card.Background(p.surface2).Border(1, p.accent.Alpha(0.6))
	default:
		card = card.Background(p.surface).Border(1, p.border)
	}
	dot := p.accent
	if pr.ThemeColor != "" {
		dot = ui.Hex(pr.ThemeColor)
	}
	inner := false // 指针是否在输入框/按钮上（此时点击不应打开项目）
	card.Children(func() {
		ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Margin(0, 0, 12, 0).Children(func() {
			ui.Box(c).Size(10, 10).Radius(5).Background(dot).Shrink(0)
			txtE(c, pr.Name, 14, p.fg, 600).Grow(1).Basis(0)
			if active {
				ui.Row(c).Height(18).PaddingX(8).Radius(9).Background(p.accent.Alpha(0.14)).AlignItems(ui.Center).Shrink(0).Children(func() {
					txt(c, a.t("当前", "Current"), fsSmall, p.accent, 0)
				})
			}
		})
		txt(c, a.t("最近更新", "Updated")+": "+formatProjectTime(pr.UpdatedAt), fsBase, p.muted, 0).Margin(0, 0, 12, 0)
		draft, editing := a.descDraft[pr.ID]
		if !editing {
			draft = pr.Description
		}
		var in ui.Element
		wrap := ui.Row(c).FillWidth().Height(24).Radius(radiusInput).Background(p.surface).AlignItems(ui.Center).Margin(0, 0, 8, 0)
		bd := p.border
		if wrap.FocusWithin() {
			bd = p.accent
		}
		wrap.Border(1, bd).Children(func() {
			in = ui.TextInputBase(c.Key("desc"), &draft).Placeholder(a.t("输入项目描述（回车或失焦保存）", "Enter project description (save on Enter/blur)")).
				PlaceholderColor(p.muted).FontSize(fsSmall).TextColor(p.fg2).Grow(1).Basis(0).FillHeight().PaddingX(12)
		})
		if wrap.Hovered() {
			inner = true
		}
		if in.Changed() {
			a.descDraft[pr.ID] = draft
		}
		if in.Submitted() || (in.FocusLeft() && editing) {
			id, text := pr.ID, strings.TrimSpace(draft)
			a.later(func() { a.saveProjectDescription(id, text) })
		}
		// 外层 Row 让按钮保持内容宽度（旧版是 inline-flex），否则在卡片的 Column 里会被拉满整行。
		var tb ui.Element
		open := a.themePick == pr.ID
		ui.Row(c).FillWidth().Children(func() {
			tb = ui.Row(c.Key("theme:"+pr.ID)).Height(sizeBtn).PaddingX(12).Gap(6).Radius(radiusBtn).Border(1, p.border).Background(p.surface2).AlignItems(ui.Center).Shrink(0)
			tb.Children(func() {
				ui.Box(c).Size(10, 10).Radius(5).Background(dot)
				txt(c, a.t("主题色", "Theme color"), fsSmall, p.fg2, 0)
				icon(c, "arrowDown", 10, p.fg2)
			})
		})
		if tb.Hovered() {
			inner = true
		}
		if tb.Clicked() {
			if open {
				a.themePick = ""
			} else {
				a.themePick = pr.ID
			}
		}
		if open {
			a.themePicker(c, tb, pr)
		}
	})
	if card.Clicked() && !inner {
		id := pr.ID
		a.later(func() { a.openProjectFromHome(id) })
	}
}

// themePicker 是项目主题色弹层：228px 宽，12 个 20px 圆形色块。
func (a *app) themePicker(c *ui.Context, anchor ui.Element, pr model.Project) {
	p := a.p
	open := true
	ui.PopoverBase(c, anchor, &open, func(panel ui.Element) {
		panel.AttachTo(anchor, ui.AnchorBottomLeft, ui.AnchorTopLeft).Margin(6, 0, 0, 0).Width(228).Radius(10).Background(p.surface).Border(1, p.border).Padding(8).Shadow(0, 8, 24, 0, p.shadowLg())
		ui.Column(c).FillWidth().Gap(8).Children(func() {
			txt(c, a.t("选择项目主题色", "Pick a project theme color"), fsTiny, p.muted, 0)
			ui.Row(c).FillWidth().Wrap().Gap(6).Children(func() {
				for _, col := range projectColors {
					sel := strings.EqualFold(pr.ThemeColor, col)
					b := ui.ButtonBase(c.Key("col"+col)).Size(20, 20).Radius(10).Background(ui.Hex(col))
					if sel {
						b = b.Border(1, p.fg)
					} else {
						b = b.Border(1, ui.RGBA(255, 255, 255, 0.45))
					}
					if b.Clicked() {
						id, color := pr.ID, col
						a.later(func() { a.setProjectTheme(id, color) })
					}
				}
			})
		})
	})
	if !open {
		a.themePick = ""
	}
}

func (a *app) projectCardMenu(m *ui.Menu, pr model.Project) {
	if m.Item(a.t("打开", "Open")).Chosen() {
		id := pr.ID
		a.later(func() { a.openProjectFromHome(id) })
	}
	if m.Item(a.t("重命名…", "Rename…")).Chosen() {
		a.later(func() {
			a.overlays.askText(a.t("重命名项目", "Rename project"), "", pr.Name, a.t("保存", "Save"), func(name string) {
				if _, err := a.be.RenameProject(pr.ID, name); err != nil {
					a.fail(a.t("重命名项目", "Rename project"), err)
					return
				}
				a.loadProjects()
			})
		})
	}
	if m.Item(a.t("导出…", "Export…")).Chosen() {
		a.later(func() { a.exportProjectByID(pr.ID, pr.Name) })
	}
	m.Separator()
	if m.Item(a.t("删除", "Delete")).Danger().Chosen() {
		a.later(func() { a.deleteProject(pr.ID, pr.Name) })
	}
}

// formatProjectTime 把 RFC3339 时间格式化为“月-日 时:分”，无法解析时显示 --。
func formatProjectTime(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return "--"
	}
	return t.Local().Format("01-02 15:04")
}

// openProjectFromHome 打开项目：当前项目直接回到工作区，否则切换。
func (a *app) openProjectFromHome(id string) {
	a.themePick = ""
	if id == a.projectID {
		a.showHome = false
		return
	}
	a.selectProject(id)
}

// createProjectFromHome 按顶部输入框创建项目并进入。
func (a *app) createProjectFromHome() {
	name := trimName(a.homeName)
	if name == "" {
		return
	}
	pr, err := a.be.CreateProject(name)
	if err != nil {
		a.fail(a.t("创建项目", "Create project"), err)
		return
	}
	a.homeName = ""
	logger.Info("已创建项目", "projectID", pr.ID)
	a.loadProjects()
	a.selectProject(pr.ID)
}

// saveProjectDescription 保存项目描述；内容没变化时只清掉草稿。
func (a *app) saveProjectDescription(id, text string) {
	delete(a.descDraft, id)
	for _, pr := range a.projects {
		if pr.ID == id && strings.TrimSpace(pr.Description) == text {
			return
		}
	}
	if _, err := a.be.UpdateProjectDescription(id, text); err != nil {
		a.fail(a.t("保存项目描述", "Save description"), err)
		return
	}
	a.loadProjects()
}

// setProjectTheme 保存项目主题色并刷新调色板。
func (a *app) setProjectTheme(id, color string) {
	a.themePick = ""
	if _, err := a.be.UpdateProjectTheme(id, color); err != nil {
		a.fail(a.t("保存主题色", "Save theme color"), err)
		return
	}
	a.loadProjects()
	a.applied.ready = false
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
