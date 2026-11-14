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

// projStats 是项目卡片上的统计：请求数、文件夹数与环境数。
type projStats struct{ reqs, folders, envs int }

// projectStats 返回项目的统计。当前项目直接取内存里的实时数据；其他项目读后端并按更新时间缓存，
// 避免主页每帧都读文件。
func (a *app) projectStats(pr model.Project) projStats {
	if pr.ID == a.projectID {
		reqs, folders := a.treeCounts()
		return projStats{reqs, folders, len(a.envs)}
	}
	if st, ok := a.homeStats[pr.ID]; ok && a.homeStatsKey[pr.ID] == pr.UpdatedAt {
		return st
	}
	var st projStats
	if cd, err := a.be.GetCollectionData(pr.ID); err == nil && cd != nil {
		st.reqs, st.folders = len(cd.Requests), len(cd.Folders)
	}
	if envs, err := a.be.ListEnvironments(pr.ID); err == nil {
		st.envs = len(envs)
	}
	if a.homeStats == nil {
		a.homeStats, a.homeStatsKey = map[string]projStats{}, map[string]string{}
	}
	a.homeStats[pr.ID], a.homeStatsKey[pr.ID] = st, pr.UpdatedAt
	return st
}

// projectColor 是项目的主题色，未设置时用当前配色的强调色。
func (a *app) projectColor(pr model.Project) ui.Color {
	if pr.ThemeColor != "" {
		return ui.Hex(pr.ThemeColor)
	}
	return a.p.accent
}

// projectInitial 是头像方块里显示的项目名首字。
func projectInitial(name string) string {
	for _, r := range strings.TrimSpace(name) {
		return strings.ToUpper(string(r))
	}
	return "?"
}

// relTime 把 RFC3339 时间格式化为相对时间：刚刚 / N 分钟前 / N 小时前 / 昨天 / N 天前 / 上周 / 月-日。
func (a *app) relTime(s string, now time.Time) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return "--"
	}
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return a.t("刚刚", "just now")
	case d < time.Hour:
		n := int(d / time.Minute)
		return a.t(itoa(n)+" 分钟前", itoa(n)+" min ago")
	case d < 24*time.Hour:
		n := int(d / time.Hour)
		return a.t(itoa(n)+" 小时前", itoa(n)+" h ago")
	case d < 48*time.Hour:
		return a.t("昨天", "yesterday")
	case d < 7*24*time.Hour:
		n := int(d / (24 * time.Hour))
		return a.t(itoa(n)+" 天前", itoa(n)+" days ago")
	case d < 14*24*time.Hour:
		return a.t("上周", "last week")
	}
	return t.Local().Format("01-02")
}

// homeProjects 返回按搜索词过滤、按排序方式排好的项目列表。
func (a *app) homeProjects() []model.Project {
	q := strings.ToLower(strings.TrimSpace(a.homeQuery))
	var list []model.Project
	for _, pr := range a.projects {
		if q == "" || strings.Contains(strings.ToLower(pr.Name), q) {
			list = append(list, pr)
		}
	}
	if a.homeSort == "name" {
		sort.SliceStable(list, func(i, j int) bool { return strings.ToLower(list[i].Name) < strings.ToLower(list[j].Name) })
	} else {
		sort.SliceStable(list, func(i, j int) bool { return list[i].UpdatedAt > list[j].UpdatedAt })
	}
	return list
}

// homeSelected 返回主页右侧详情面板对应的项目：优先用户点选的，其次当前项目，最后第一个。
func (a *app) homeSelected() (model.Project, bool) {
	for _, pr := range a.projects {
		if pr.ID == a.homeSel {
			return pr, true
		}
	}
	for _, pr := range a.projects {
		if pr.ID == a.projectID {
			return pr, true
		}
	}
	if len(a.projects) > 0 {
		return a.projects[0], true
	}
	return model.Project{}, false
}

// homeDetailW 是主页右侧详情面板的宽度。
const homeDetailW = 300

// homeView 是项目主页（设计稿）：左侧标题、搜索排序与项目卡片网格，右侧是所选项目的详情面板。
func (a *app) homeView(c *ui.Context) {
	p := a.p
	cw, _ := c.Size()
	_, hasSel := a.homeSelected()
	left := cw
	if hasSel {
		left -= homeDetailW
	}
	ui.Row(c).Grow(1).Basis(0).Background(p.surface).Children(func() {
		ui.Scroll(c.Key("home-scroll")).Grow(1).Basis(0).FillHeight().Scrollbars(a.scrollbarsVisible()).Children(func() {
			ui.Column(c).FillWidth().PaddingX(24).PaddingY(24).Children(func() {
				a.homeHeader(c)
				if len(a.projects) == 0 {
					a.homeEmpty(c)
					return
				}
				projects := a.homeProjects()
				a.homeFilterRow(c, len(projects))
				cols := 1
				switch {
				case left >= 1500:
					cols = 3
				case left >= 700:
					cols = 2
				}
				if len(projects) == 0 {
					a.sideHint(c, a.t("没有匹配的项目", "No matching projects"))
					return
				}
				ui.Column(c).FillWidth().Gap(12).Children(func() {
					for r := 0; r < len(projects); r += cols {
						ui.Row(c.Key("hr" + itoa(r))).FillWidth().Gap(12).AlignItems(ui.Start).Children(func() {
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
		if sel, ok := a.homeSelected(); ok {
			a.homeDetail(c, sel)
		}
	})
}

// homeHeader 是主页顶部：标题与说明在左，新建项目输入框与按钮在右。
func (a *app) homeHeader(c *ui.Context) {
	p := a.p
	ui.Row(c).FillWidth().Margin(0, 0, 16, 0).AlignItems(ui.Start).Justify(ui.SpaceBetween).Gap(16).Children(func() {
		ui.Column(c).Grow(1).Basis(0).Gap(4).Children(func() {
			txt(c, a.t("项目主页", "Project home"), 20, p.fg, 600)
			ui.Text(c, a.t("选择项目继续工作。每个项目有自己的主题色，会应用到选中态、高亮和发送按钮。",
				"Pick a project to continue. Each project has its own theme color, applied to selection, highlights and the send button.")).FontSize(fsBase).TextColor(p.fg2)
		})
		ui.Row(c).Width(320).Shrink(0).Gap(8).AlignItems(ui.Center).Children(func() {
			in := a.input(c, &a.homeName, a.t("输入项目名，回车新建", "Project name, Enter to create"), fieldOpts{h: 28, radius: radiusInput, fs: fsBase, icon: "add", key: "home-new"})
			bg := p.accent
			create := a.button(c, btnOpts{h: 28, radius: radiusBtn, padX: 12, bg: &bg, hoverBg: &p.accentHover, key: "home-create"}, func(bool) {
				txt(c, a.t("新建项目", "New project"), fsBase, p.accentFg, 600)
			})
			if create || in.Submitted() {
				a.later(a.createProjectFromHome)
			}
		})
	})
}

// homeFilterRow 是搜索框、项目总数与排序方式。
func (a *app) homeFilterRow(c *ui.Context, n int) {
	p := a.p
	ui.Row(c).FillWidth().Margin(0, 0, 12, 0).Gap(12).AlignItems(ui.Center).Children(func() {
		a.input(c, &a.homeQuery, a.t("搜索项目…", "Search projects…"), fieldOpts{w: 240, h: 26, radius: radiusInput, icon: "search", key: "home-search", bg: &p.surface2})
		txt(c, a.t("共 "+itoa(n)+" 个项目", itoa(n)+" projects"), fsSmall, p.muted, 0)
		ui.Spacer(c)
		label := a.t("最近更新", "Recently updated")
		if a.homeSort == "name" {
			label = a.t("名称", "Name")
		}
		sortBtn := ui.ButtonBase(c.Key("home-sort")).Height(24).PaddingX(8).Gap(4).Radius(radiusBtn).AlignItems(ui.Center).
			PopoverMenuButton(func(m *ui.Menu) {
				if m.Item(a.t("最近更新", "Recently updated")).Checked(a.homeSort != "name").Chosen() {
					a.later(func() { a.homeSort = "updated" })
				}
				if m.Item(a.t("名称", "Name")).Checked(a.homeSort == "name").Chosen() {
					a.later(func() { a.homeSort = "name" })
				}
			})
		if sortBtn.Hovered() {
			sortBtn = sortBtn.Background(p.hover)
		}
		sortBtn.Children(func() {
			txt(c, a.t("排序", "Sort"), fsSmall, p.muted, 0)
			txt(c, label, fsSmall, p.fg, 500)
			icon(c, "arrowDown", 10, p.muted)
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

// projectAvatar 是项目的彩色首字方块。
func (a *app) projectAvatar(c *ui.Context, pr model.Project, size, fs float32) {
	ui.Row(c).Size(size, size).Radius(size / 4).Background(a.projectColor(pr)).AlignItems(ui.Center).Justify(ui.Center).Shrink(0).Children(func() {
		txt(c, projectInitial(pr.Name), fs, ui.Hex("#ffffff"), 600)
	})
}

// projectCard 是一张项目卡片：首字头像、名称与请求/文件夹数、“当前”标记、更多菜单，底部是更新时间与环境数。
// 单击选中（右侧显示详情），双击打开项目。
func (a *app) projectCard(c *ui.Context, pr model.Project) {
	p := a.p
	active := pr.ID == a.projectID
	sel, _ := a.homeSelected()
	selected := sel.ID == pr.ID
	st := a.projectStats(pr)
	card := ui.ButtonBase(c.Key("pc:"+pr.ID)).Grow(1).Basis(0).Radius(radiusCard).Padding(12, 14).
		AlignItems(ui.Stretch).Justify(ui.Start).ContextMenu(func(m *ui.Menu) { a.projectCardMenu(m, pr) })
	hover := card.Hovered()
	switch {
	case selected:
		card = card.Background(p.surface).Border(1.5, p.accent)
	case hover:
		card = card.Background(p.surface2).Border(1, p.btnBorder)
	default:
		card = card.Background(p.surface).Border(1, p.border)
	}
	card.Children(func() {
		ui.Column(c).FillWidth().Gap(10).Children(func() {
			ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Children(func() {
				a.projectAvatar(c, pr, 30, 13)
				ui.Column(c).Grow(1).Basis(0).Gap(1).Children(func() {
					txtE(c, pr.Name, fsLarge, p.fg, 600)
					txtE(c, a.t(itoa(st.reqs)+" 个请求 · "+itoa(st.folders)+" 个文件夹", itoa(st.reqs)+" requests · "+itoa(st.folders)+" folders"), fsSmall, p.muted, 0)
				})
				if active {
					ui.Row(c).Height(18).PaddingX(7).Radius(9).Background(p.hover).AlignItems(ui.Center).Shrink(0).Children(func() {
						txt(c, a.t("当前", "Current"), fsTiny, p.fg2, 500)
					})
				}
				ui.ButtonBase(c.Key("pcm:"+pr.ID)).Size(22, 22).Radius(6).AlignItems(ui.Center).Justify(ui.Center).Shrink(0).
					Tooltip(a.t("更多操作", "More actions")).PopoverMenuButton(func(m *ui.Menu) { a.projectCardMenu(m, pr) }).
					Children(func() { icon(c, "more", 13, p.muted) })
			})
			ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
				txt(c, a.t("更新于 ", "Updated ")+a.relTime(pr.UpdatedAt, time.Now()), fsSmall, p.muted, 0)
				ui.Spacer(c)
				icon(c, "globe", 11, p.success)
				txt(c, a.t(itoa(st.envs)+" 个环境", itoa(st.envs)+" environments"), fsSmall, p.fg2, 0)
			})
		})
	})
	switch {
	case card.DoubleClicked():
		id := pr.ID
		a.later(func() { a.openProjectFromHome(id) })
	case card.Clicked():
		a.homeSel = pr.ID
	}
}

// homeDetail 是主页右侧详情面板：头像与名称、改名与描述、主题色、实时预览、打开/重命名/导出/删除。
func (a *app) homeDetail(c *ui.Context, pr model.Project) {
	p := a.p
	st := a.projectStats(pr)
	col := a.projectColor(pr)
	label := func(s string) {
		txt(c, s, fsTiny, p.muted, 600).Margin(12, 0, 0, 0)
	}
	ui.Column(c.Key("home-detail")).Width(homeDetailW).FillHeight().Shrink(0).Background(p.sidebar).BorderWidth(0, 0, 0, 1).BorderColor(p.border).
		Padding(16).Gap(6).Children(func() {
		ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Children(func() {
			a.projectAvatar(c, pr, 40, 18)
			ui.Column(c).Grow(1).Basis(0).Gap(2).Children(func() {
				txtE(c, pr.Name, 16, p.fg, 600)
				txt(c, a.t(itoa(st.reqs)+" 个请求 · "+itoa(st.folders)+" 个文件夹", itoa(st.reqs)+" requests · "+itoa(st.folders)+" folders"), fsSmall, p.muted, 0)
			})
		})

		label(a.t("项目名称", "NAME"))
		draftName, editing := a.nameDraft[pr.ID]
		if !editing {
			draftName = pr.Name
		}
		var nameIn ui.Element
		ui.Row(c).FillWidth().Children(func() {
			nameIn = a.input(c, &draftName, "", fieldOpts{h: 26, radius: radiusInput, fs: fsBase, key: "home-name:" + pr.ID})
		})
		if nameIn.Changed() {
			if a.nameDraft == nil {
				a.nameDraft = map[string]string{}
			}
			a.nameDraft[pr.ID] = draftName
		}
		if nameIn.Submitted() || (nameIn.FocusLeft() && editing) {
			id, name := pr.ID, trimName(draftName)
			a.later(func() { a.commitProjectName(id, name) })
		}

		label(a.t("描述", "DESCRIPTION"))
		draft, editingDesc := a.descDraft[pr.ID]
		if !editingDesc {
			draft = pr.Description
		}
		var in ui.Element
		ui.Row(c).FillWidth().Children(func() {
			in = a.input(c, &draft, a.t("输入项目描述（回车或失焦保存）", "Description (save on Enter / blur)"), fieldOpts{h: 26, radius: radiusInput, fs: fsBase, key: "home-desc:" + pr.ID})
		})
		if in.Changed() {
			a.descDraft[pr.ID] = draft
		}
		if in.Submitted() || (in.FocusLeft() && editingDesc) {
			id, text := pr.ID, strings.TrimSpace(draft)
			a.later(func() { a.saveProjectDescription(id, text) })
		}

		label(a.t("主题色", "THEME COLOR"))
		ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
			for _, hex := range projectColors {
				on := strings.EqualFold(pr.ThemeColor, hex)
				b := ui.ButtonBase(c.Key("col"+hex)).Size(24, 24).Radius(12).Background(ui.Hex(hex)).AlignItems(ui.Center).Justify(ui.Center)
				if on {
					b = b.Border(2, p.fg)
				}
				b.Children(func() {
					if on {
						icon(c, "tick", 12, ui.Hex("#ffffff"))
					}
				})
				if b.Clicked() {
					id, color := pr.ID, hex
					a.later(func() { a.setProjectTheme(id, color) })
				}
			}
		})

		label(a.t("实时预览", "LIVE PREVIEW"))
		ui.Column(c).FillWidth().Radius(radiusCard).Background(p.surface).Border(1, p.border).Padding(8, 10).Gap(8).Children(func() {
			ui.Row(c).FillWidth().Gap(14).Children(func() {
				ui.Column(c).Gap(2).Children(func() {
					txt(c, "Params", fsSmall, col, 600)
					ui.Box(c).Height(2).FillWidth().Background(col)
				})
				txt(c, "Headers", fsSmall, p.muted, 0)
			})
			ui.Row(c).FillWidth().Height(24).PaddingX(8).Gap(10).Radius(6).Background(col.Alpha(0.14)).AlignItems(ui.Center).Children(func() {
				txtMono(c, "GET", 9, p.get, 700)
				txt(c, a.t("示例请求", "Sample request"), fsBase, p.fg, 0)
			})
			ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
				txt(c, a.t("发送按钮", "Send button"), fsSmall, p.muted, 0)
				ui.Spacer(c)
				ui.Row(c).Height(22).PaddingX(12).Radius(6).Background(col).AlignItems(ui.Center).Children(func() {
					txt(c, a.t("发送", "Send"), fsSmall, ui.Hex("#ffffff"), 600)
				})
			})
		})

		ui.Spacer(c)
		bg := p.accent
		if a.button(c, btnOpts{h: 30, radius: radiusInput, bg: &bg, hoverBg: &p.accentHover, key: "home-open"}, func(bool) {
			txt(c, a.t("打开项目", "Open project"), fsBase, p.accentFg, 600)
		}) {
			id := pr.ID
			a.later(func() { a.openProjectFromHome(id) })
		}
		ui.Row(c).FillWidth().Gap(6).Children(func() {
			border := p.btnBorder
			if a.button(c, btnOpts{h: 26, radius: radiusBtn, padX: 10, gap: 5, border: &border, key: "home-rename"}, func(bool) {
				icon(c, "pencil", 12, p.fg2)
				txt(c, a.t("重命名", "Rename"), fsSmall, p.fg, 0)
			}) {
				a.later(func() { a.renameProject(pr) })
			}
			if a.button(c, btnOpts{h: 26, radius: radiusBtn, padX: 10, gap: 5, border: &border, key: "home-export"}, func(bool) {
				icon(c, "upload", 12, p.fg2)
				txt(c, a.t("导出", "Export"), fsSmall, p.fg, 0)
			}) {
				a.later(func() { a.exportProjectByID(pr.ID, pr.Name) })
			}
			if a.button(c, btnOpts{w: 26, h: 26, radius: radiusBtn, border: &border, tip: a.t("删除项目", "Delete project"), key: "home-delete"}, func(bool) {
				icon(c, "delete", 12, p.danger)
			}) {
				a.later(func() { a.deleteProject(pr.ID, pr.Name) })
			}
		})
	})
}

// commitProjectName 提交详情面板里改好的项目名；为空或没变化时只清掉草稿。
func (a *app) commitProjectName(id, name string) {
	delete(a.nameDraft, id)
	for _, pr := range a.projects {
		if pr.ID == id && (name == "" || name == pr.Name) {
			return
		}
	}
	if _, err := a.be.RenameProject(id, name); err != nil {
		a.fail(a.t("重命名项目", "Rename project"), err)
		return
	}
	a.loadProjects()
	logger.Info("已重命名项目", "projectID", id)
}

// renameProject 弹出重命名对话框。
func (a *app) renameProject(pr model.Project) {
	a.overlays.askText(a.t("重命名项目", "Rename project"), "", pr.Name, a.t("保存", "Save"), func(name string) {
		a.commitProjectName(pr.ID, name)
	})
}

func (a *app) projectCardMenu(m *ui.Menu, pr model.Project) {
	if m.Item(a.t("打开", "Open")).Chosen() {
		id := pr.ID
		a.later(func() { a.openProjectFromHome(id) })
	}
	if m.Item(a.t("重命名…", "Rename…")).Chosen() {
		a.later(func() { a.renameProject(pr) })
	}
	if m.Item(a.t("导出…", "Export…")).Chosen() {
		a.later(func() { a.exportProjectByID(pr.ID, pr.Name) })
	}
	m.Separator()
	if m.Item(a.t("删除", "Delete")).Danger().Chosen() {
		a.later(func() { a.deleteProject(pr.ID, pr.Name) })
	}
}

// openProjectFromHome 打开项目：当前项目直接回到工作区，否则切换。
func (a *app) openProjectFromHome(id string) {
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
