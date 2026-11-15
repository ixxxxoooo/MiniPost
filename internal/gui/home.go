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

// homeGridCols 返回项目卡片网格的列数：repeat(auto-fill, minmax(236px, 1fr))，列间距 10，
// 内容区左右各留 26 的内边距。参数 w 是主页卡片左侧部分（不含详情面板、侧栏与外边距）的宽度。
func homeGridCols(w float32) int {
	return max(1, int((w-52+10)/(236+10)))
}

// homeDetailW 是主页右侧详情面板的宽度（设计稿 .home-side）。
const homeDetailW = 270

// homeView 是项目主页（设计稿）：左侧标题、搜索与卡片网格 / 列表，右侧是所选项目的详情面板。
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
			ui.Column(c).FillWidth().Padding(22, 26, 16, 26).Gap(14).Children(func() {
				a.homeHeader(c)
				if len(a.projects) == 0 {
					a.homeEmpty(c)
					return
				}
				projects := a.homeProjects()
				a.homeFilterRow(c, len(projects))
				if len(projects) == 0 {
					a.sideHint(c, a.t("没有匹配的项目", "No matching projects"))
					return
				}
				if a.homeList {
					a.homeListView(c, projects)
					return
				}
				// 卡片网格：repeat(auto-fill, minmax(236px, 1fr))，间距 10
				side := a.sidebarShown()
				if a.set.SidebarCollapsed {
					side = 0
				}
				cols := homeGridCols(left - side - shellGap - 8)
				ui.Column(c).FillWidth().Gap(10).Children(func() {
					for r := 0; r < len(projects); r += cols {
						ui.Row(c.Key("hr" + itoa(r))).FillWidth().Gap(10).AlignItems(ui.Start).Children(func() {
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

// homeHeader 是主页顶部（设计稿 .home-h）：标题与说明在左，新建项目输入框与按钮在右，底对齐。
func (a *app) homeHeader(c *ui.Context) {
	p := a.p
	sub := a.t("选择项目继续工作。每个项目有自己的主题色，会应用到选中态、高亮和发送按钮。",
		"Pick a project to continue. Each project has its own theme color, applied to selection, highlights and the send button.")
	if len(a.projects) == 0 {
		sub = a.t("还没有项目。新建一个项目，把相关请求与环境放在一起。", "No projects yet. Create one to keep related requests and environments together.")
	}
	ui.Row(c).FillWidth().Gap(14).AlignItems(ui.End).Children(func() {
		ui.Column(c).Grow(1).Basis(0).Gap(3).Children(func() {
			txt(c, a.t("项目", "Projects"), 20, p.fg, 700)
			ui.Text(c, sub).FontSize(fsBase).TextColor(p.muted)
		})
		if len(a.projects) == 0 {
			return
		}
		ui.Row(c).Width(210).Shrink(0).Children(func() {
			in := a.input(c, &a.homeName, a.t("输入项目名，回车新建", "Project name, Enter to create"), fieldOpts{h: 30, radius: radiusR, fs: fsBase, icon: "plus", key: "home-new"})
			if in.Submitted() {
				a.later(a.createProjectFromHome)
			}
		})
		if a.btn(c, dbtn{key: "home-create", label: a.t("新建项目", "New project"), kind: dlgPrimary, h: 30}) {
			a.later(a.createProjectFromHome)
		}
	})
}

// homeFilterRow 是搜索框、项目总数、卡片/列表切换与排序方式（设计稿 .home-bar）。
func (a *app) homeFilterRow(c *ui.Context, n int) {
	p := a.p
	ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Row(c).Width(200).Shrink(0).Children(func() {
			a.input(c, &a.homeQuery, a.t("搜索项目…", "Search projects…"), fieldOpts{h: 26, radius: radiusR, icon: "search", key: "home-search"})
		})
		txt(c, a.t("共 "+itoa(n)+" 个项目", itoa(n)+" projects"), fsBase, p.muted, 0)
		ui.Spacer(c)
		mode := 0
		if a.homeList {
			mode = 1
		}
		if a.pillSeg(c, "home-mode", &mode, []string{"", ""}, []string{"grid", "list"}, segOpts{h: 22, padX: 10}) {
			a.homeList = mode == 1
		}
		label := a.t("最近更新", "Recently updated")
		if a.homeSort == "name" {
			label = a.t("名称", "Name")
		}
		sortBtn := ui.ButtonBase(c.Key("home-sort")).Height(26).MinWidth(110).PaddingStart(10).PaddingEnd(7).Gap(8).Radius(radiusR).AlignItems(ui.Center).Justify(ui.SpaceBetween).
			Background(p.surface).Border(1, p.btnBorder).
			PopoverMenuButton(func(m *ui.Menu) {
				if m.Item(a.t("最近更新", "Recently updated")).Checked(a.homeSort != "name").Chosen() {
					a.later(func() { a.homeSort = "updated" })
				}
				if m.Item(a.t("名称", "Name")).Checked(a.homeSort == "name").Chosen() {
					a.later(func() { a.homeSort = "name" })
				}
			})
		sortBtn.Children(func() {
			txt(c, a.t("排序：", "Sort: ")+label, fsBase, p.fg, 500)
			icon(c, "chevUD", 12, p.muted)
		})
	})
}

// homeEmpty 是零项目时的引导（设计稿 .home-empty）：大图标、标题、说明、新建输入与导入入口。
func (a *app) homeEmpty(c *ui.Context) {
	p := a.p
	ui.Column(c).FillWidth().Padding(40, 20, 90, 20).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Row(c).Size(84, 84).Radius(26).Background(p.acSoft).Shadow(0, 12, 32, -10, p.glow).Border(1, p.acSoft2).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
			if img := logoImage(p.dark); img != nil {
				ui.Image(c, img).Size(44, 44).Fit(ui.Contain)
			}
		})
		txt(c, a.t("创建你的第一个项目", "Create your first project"), 16, p.fg, 650).Margin(8, 0, 0, 0)
		ui.Text(c, a.t("项目会保存请求、环境与历史记录，并拥有独立的主题色。", "A project keeps requests, environments and history, with its own theme color.")).
			FontSize(fsBase).TextColor(p.fg2).MaxWidth(340)
		ui.Row(c).Gap(8).Margin(10, 0, 2, 0).AlignItems(ui.Center).Children(func() {
			ui.Row(c).Width(220).Shrink(0).Children(func() {
				in := a.input(c, &a.homeName, a.t("输入项目名，回车新建", "Project name, Enter to create"), fieldOpts{h: 30, radius: radiusR, fs: fsBase, icon: "plus", key: "home-new-empty"})
				if in.Submitted() {
					a.later(a.createProjectFromHome)
				}
			})
			if a.btn(c, dbtn{key: "home-create-empty", label: a.t("新建项目", "New project"), kind: dlgPrimary, h: 30}) {
				a.later(a.createProjectFromHome)
			}
		})
		if a.btn(c, dbtn{key: "home-import-empty", icon: "upload", label: a.t("或从 Postman / OpenAPI 导入", "Or import from Postman / OpenAPI"), ghost: true, sm: true}) {
			a.later(func() { a.openImport(importFile) })
		}
	})
}

// projectAvatar 是项目的彩色首字方块（设计稿 .pav）：左上到右下的高光渐变与柔和投影。
func (a *app) projectAvatar(c *ui.Context, pr model.Project, size, fs float32) {
	col := a.projectColor(pr)
	ui.Row(c).Size(size, size).Radius(size*0.3).Gradient(col.Mix(ui.RGB(255, 255, 255), 0.22), col, 155).Shadow(0, 2, 6, -1, ui.RGBA(0, 0, 0, 0.28)).
		AlignItems(ui.Center).Justify(ui.Center).Shrink(0).Children(func() {
		txt(c, projectInitial(pr.Name), fs, ui.Hex("#ffffff"), 700)
	})
}

// projectCard 是一张项目卡片（设计稿 .pcard）：首字头像、名称与请求/文件夹数、“当前”标记，底部是更新时间与环境数。
// 单击选中（右侧显示详情），双击打开项目；选中时是强调色描边 + 3px 淡色光环，悬停时浮起。
func (a *app) projectCard(c *ui.Context, pr model.Project) {
	p := a.p
	active := pr.ID == a.projectID
	sel, _ := a.homeSelected()
	selected := sel.ID == pr.ID
	st := a.projectStats(pr)
	card := ui.ButtonBase(c.Key("pc:" + pr.ID)).Grow(1).Basis(0).Radius(radiusLg).Padding(12).Gap(10).
		AlignItems(ui.Stretch).Justify(ui.Start).ContextMenu(func(m *ui.Menu) { a.projectCardMenu(m, pr) })
	hover := card.Hovered()
	switch {
	case selected:
		card = card.Background(p.surface).Border(1, p.accent).Shadow(0, 0, 0, 3, p.acSoft)
	case hover:
		card = p.shadowMd(card.Background(p.surface).Border(1, p.acLine.Mix(p.btnBorder, 0.4)))
	default:
		card = card.Background(p.surface.Alpha(0.92)).Border(1, p.border).Shadow(0, 1, 2, 0, ui.RGBA(20, 24, 40, 0.04))
	}
	card.Children(func() {
		ui.Column(c).FillWidth().Gap(10).Children(func() {
			ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Children(func() {
				a.projectAvatar(c, pr, 30, 13)
				ui.Column(c).Grow(1).Basis(0).MinWidth(0).Children(func() {
					txtE(c, pr.Name, fsBase+1, p.fg, 650)
					txtE(c, a.t(itoa(st.reqs)+" 个请求 · "+itoa(st.folders)+" 个文件夹", itoa(st.reqs)+" requests · "+itoa(st.folders)+" folders"), fsSmall, p.muted, 0)
				})
				if active {
					ui.Row(c).Height(20).PaddingX(8).Radius(10).Background(p.acSoft).Border(1, p.acSoft2).AlignItems(ui.Center).Shrink(0).Children(func() {
						txt(c, a.t("当前", "Current"), fsSmall, p.accent, 600)
					})
				}
				if hover {
					ui.ButtonBase(c.Key("pcm:"+pr.ID)).Size(22, 22).Radius(radiusSm).AlignItems(ui.Center).Justify(ui.Center).Shrink(0).
						Tooltip(a.t("更多操作", "More actions")).PopoverMenuButton(func(m *ui.Menu) { a.projectCardMenu(m, pr) }).
						Children(func() { icon(c, "more", 13, p.muted) })
				}
			})
			ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
				txt(c, a.relTime(pr.UpdatedAt, time.Now()), fsSmall, p.muted, 0)
				ui.Spacer(c)
				icon(c, "globe", 12, p.muted)
				txt(c, a.t(itoa(st.envs)+" 个环境", itoa(st.envs)+" environments"), fsSmall, p.muted, 0)
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

// homeListView 是列表模式（设计稿 .plist）：圆角外框内一行一个项目，列为头像、名称、描述、请求、环境、更新。
func (a *app) homeListView(c *ui.Context, projects []model.Project) {
	p := a.p
	cell := func(grow float32, f func()) {
		ui.Row(c).Grow(grow).Basis(0).MinWidth(0).AlignItems(ui.Center).Children(f)
	}
	ui.Column(c).FillWidth().Radius(radiusLg).Border(1, p.border).Clip().Children(func() {
		ui.Row(c.Key("plist-hd")).FillWidth().Height(28).PaddingX(12).Gap(10).AlignItems(ui.Center).Background(p.surface2).Children(func() {
			ui.Box(c).Width(24).Shrink(0)
			cell(1.4, func() { txt(c, a.t("名称", "Name"), fsSmall, p.muted, 600) })
			cell(1.4, func() { txt(c, a.t("描述", "Description"), fsSmall, p.muted, 600) })
			ui.Row(c).Width(70).Shrink(0).Children(func() { txt(c, a.t("请求", "Requests"), fsSmall, p.muted, 600) })
			ui.Row(c).Width(70).Shrink(0).Children(func() { txt(c, a.t("环境", "Envs"), fsSmall, p.muted, 600) })
			ui.Row(c).Width(90).Shrink(0).Children(func() { txt(c, a.t("更新", "Updated"), fsSmall, p.muted, 600) })
			ui.Box(c).Width(28).Shrink(0)
		})
		sel, _ := a.homeSelected()
		for _, pr := range projects {
			pr := pr
			st := a.projectStats(pr)
			row := ui.ButtonBase(c.Key("pr:"+pr.ID)).FillWidth().Height(38).PaddingX(12).Gap(10).AlignItems(ui.Center).Justify(ui.Start).
				BorderWidth(1, 0, 0, 0).BorderColor(p.border).ContextMenu(func(m *ui.Menu) { a.projectCardMenu(m, pr) })
			switch {
			case sel.ID == pr.ID:
				row = row.Background(p.acSoft)
			case row.Hovered():
				row = row.Background(p.hover)
			}
			row.Children(func() {
				a.projectAvatar(c, pr, 22, 11)
				cell(1.4, func() {
					txtE(c, pr.Name, fsBase, p.fg, 600)
					if pr.ID == a.projectID {
						ui.Row(c).Height(16).PaddingX(6).Radius(8).Background(p.acSoft).Margin(0, 0, 0, 6).AlignItems(ui.Center).Shrink(0).Children(func() {
							txt(c, a.t("当前", "Current"), fsTiny, p.accent, 600)
						})
					}
				})
				cell(1.4, func() { txtE(c, pr.Description, fsBase, p.fg2, 0) })
				ui.Row(c).Width(70).Shrink(0).Children(func() { txtMono(c, itoa(st.reqs), fsSmall, p.fg2, 0) })
				ui.Row(c).Width(70).Shrink(0).Children(func() { txtMono(c, itoa(st.envs), fsSmall, p.fg2, 0) })
				ui.Row(c).Width(90).Shrink(0).Children(func() { txt(c, a.relTime(pr.UpdatedAt, time.Now()), fsSmall, p.muted, 0) })
				ui.Row(c).Width(28).Shrink(0).Justify(ui.Center).Children(func() { icon(c, "chevR", 12, p.muted) })
			})
			switch {
			case row.DoubleClicked():
				id := pr.ID
				a.later(func() { a.openProjectFromHome(id) })
			case row.Clicked():
				a.homeSel = pr.ID
			}
		}
	})
}

// homeDetail 是主页右侧详情面板（设计稿 .home-side）：头像与名称、改名与描述、主题色、实时预览、打开/重命名/导出/删除。
func (a *app) homeDetail(c *ui.Context, pr model.Project) {
	p := a.p
	st := a.projectStats(pr)
	col := a.projectColor(pr)
	label := func(s string) {
		txt(c, s, fsSmall, p.muted, 600).Margin(0, 0, 5, 0)
	}
	ui.Column(c.Key("home-detail")).Width(homeDetailW).FillHeight().Shrink(0).Background(p.sidebar).BorderWidth(0, 0, 0, 1).BorderColor(p.border).
		Padding(16).Gap(12).Children(func() {
		ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Children(func() {
			a.projectAvatar(c, pr, 38, 17)
			ui.Column(c).Grow(1).Basis(0).MinWidth(0).Children(func() {
				txtE(c, pr.Name, fsBase+2, p.fg, 700)
				txt(c, a.t(itoa(st.reqs)+" 个请求 · "+itoa(st.folders)+" 个文件夹", itoa(st.reqs)+" requests · "+itoa(st.folders)+" folders"), fsSmall, p.muted, 0)
			})
		})

		ui.Column(c).FillWidth().Children(func() {
			label(a.t("项目名称", "Name"))
			draftName, editing := a.nameDraft[pr.ID]
			if !editing {
				draftName = pr.Name
			}
			var nameIn ui.Element
			ui.Row(c).FillWidth().Children(func() {
				nameIn = a.input(c, &draftName, "", fieldOpts{h: 28, radius: radiusR, fs: fsBase, key: "home-name:" + pr.ID})
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
		})

		ui.Column(c).FillWidth().Children(func() {
			label(a.t("描述", "Description"))
			draft, editingDesc := a.descDraft[pr.ID]
			if !editingDesc {
				draft = pr.Description
			}
			var in ui.Element
			ui.Row(c).FillWidth().Children(func() {
				in = a.input(c, &draft, a.t("添加描述", "Add a description"), fieldOpts{h: 28, radius: radiusR, fs: fsBase, key: "home-desc:" + pr.ID})
			})
			if in.Changed() {
				a.descDraft[pr.ID] = draft
			}
			if in.Submitted() || (in.FocusLeft() && editingDesc) {
				id, text := pr.ID, strings.TrimSpace(draft)
				a.later(func() { a.saveProjectDescription(id, text) })
			}
		})

		ui.Column(c).FillWidth().Children(func() {
			label(a.t("主题色", "Theme color"))
			ui.Row(c).FillWidth().Wrap().Gap(7).Children(func() {
				for _, hex := range projectColors {
					on := strings.EqualFold(pr.ThemeColor, hex)
					b := ui.ButtonBase(c.Key("col"+hex)).Size(20, 20).Radius(10).Background(ui.Hex(hex)).AlignItems(ui.Center).Justify(ui.Center)
					if on {
						b = b.Border(2, p.fg).Shadow(0, 0, 0, 2, p.surface)
					}
					b.Children(func() {})
					if b.Clicked() {
						id, color := pr.ID, hex
						a.later(func() { a.setProjectTheme(id, color) })
					}
				}
			})
		})

		ui.Column(c).FillWidth().Children(func() {
			label(a.t("实时预览", "Live preview"))
			ui.Column(c).FillWidth().Radius(radiusR).Background(p.surface).Border(1, p.border).Padding(10).Gap(8).Children(func() {
				ui.Row(c).FillWidth().Gap(12).BorderWidth(0, 0, 1, 0).BorderColor(p.border).Children(func() {
					ui.Column(c).Height(24).Justify(ui.End).Children(func() {
						txt(c, "Params", fsBase, col, 650).Margin(0, 0, 4, 0)
						ui.Box(c).Height(2).FillWidth().Radius(2, 2, 0, 0).Background(col)
					})
					ui.Row(c).Height(24).AlignItems(ui.Center).Children(func() { txt(c, "Headers", fsBase, p.fg2, 550) })
				})
				ui.Row(c).FillWidth().Height(24).PaddingX(8).Gap(8).Radius(6).Background(col.Alpha(0.14)).AlignItems(ui.Center).Children(func() {
					txtMono(c, "GET", fsTiny-0.5, p.get, 700)
					txt(c, a.t("示例请求", "Sample request"), fsBase, col, 600)
				})
				ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
					txt(c, a.t("发送按钮", "Send button"), fsBase, p.muted, 0)
					ui.Spacer(c)
					ui.Row(c).Height(22).PaddingX(8).Radius(10).Gradient(col.Mix(ui.RGB(255, 255, 255), 0.16), col, 180).AlignItems(ui.Center).Children(func() {
						txt(c, a.t("发送", "Send"), fsSmall, ui.Hex("#ffffff"), 550)
					})
				})
			})
		})

		ui.Spacer(c)
		if a.btn(c, dbtn{key: "home-open", label: a.t("打开项目", "Open project"), kind: dlgPrimary, h: 30}) {
			id := pr.ID
			a.later(func() { a.openProjectFromHome(id) })
		}
		ui.Row(c).FillWidth().Gap(6).Children(func() {
			if a.btn(c, dbtn{key: "home-rename", icon: "pencil", label: a.t("重命名", "Rename"), grow: true}) {
				a.later(func() { a.renameProject(pr) })
			}
			if a.btn(c, dbtn{key: "home-export", icon: "upload", label: a.t("导出", "Export"), grow: true}) {
				a.later(func() { a.exportProjectByID(pr.ID, pr.Name) })
			}
			if a.btn(c, dbtn{key: "home-delete", icon: "trash", danger: true, w: 36, tip: a.t("删除项目", "Delete project")}) {
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
