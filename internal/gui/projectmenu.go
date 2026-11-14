package gui

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"

	"minipost/internal/pkg/logger"
)

// projMenuState 是工具栏项目下拉面板的状态。
type projMenuState struct {
	open  bool
	query string
}

// 旧版面板尺寸常量：宽度 = 84 + 最长名称长度×11，限制在 220~320。
const (
	projPanelMin  = 220
	projPanelMax  = 320
	projPanelBase = 84
	projCharWidth = 11
)

// projectDropdown 绘制项目下拉面板：搜索/新建栏 + 项目列表（当前标记、悬停时的导出与删除）。
func (a *app) projectDropdown(c *ui.Context, anchor ui.Element, trigW float32) {
	p := a.p
	pm := &a.projMenu
	longest := 0
	if pr := a.project(); pr != nil {
		longest = utf8.RuneCountInString(pr.Name)
	}
	for _, pr := range a.projects {
		longest = max(longest, utf8.RuneCountInString(pr.Name))
	}
	w := min(float32(projPanelMax), max(float32(projPanelMin), float32(projPanelBase+longest*projCharWidth)))
	w = max(w, trigW)
	open := true
	create := false
	ui.PopoverBase(c, anchor, &open, func(panel ui.Element) {
		panel.AttachTo(anchor, ui.AnchorBottomLeft, ui.AnchorTopLeft).Margin(6, 0, 0, 0).Width(w).Radius(9).Background(p.elevated()).
			Border(1, p.btnBorder).Clip().Shadow(0, 8, 24, 0, p.shadowLg())
		ui.Column(c).FillWidth().Children(func() {
			// 搜索 / 新建
			ui.Row(c).FillWidth().Padding(6).Gap(6).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(p.border).Children(func() {
				bd := p.btnBorder
				in := a.input(c, &pm.query, a.t("搜索或新建项目...", "Search or create project..."), fieldOpts{
					h: 28, radius: 7, fs: 10, icon: "search", key: "pm-q", bd: &bd, focus: true, iconSize: 10, iconLeft: 10, iconGap: 8,
				})
				if in.Submitted() {
					create = true
				}
				if a.button(c, btnOpts{h: 28, radius: 7, padX: 10, border: &bd, bg: &p.surface, hoverBg: &p.surface2, key: "pm-new"}, func(bool) {
					txt(c, a.t("新建", "New"), 10, p.fg, 500)
				}) {
					create = true
				}
			})
			// 项目列表
			ui.Scroll(c).FillWidth().MaxHeight(320).Scrollbars(a.scrollbarsVisible()).Children(func() {
				ui.Column(c).FillWidth().Padding(4).Children(func() {
					q := strings.ToLower(pm.query)
					for _, pr := range a.projects {
						if q != "" && !strings.Contains(strings.ToLower(pr.Name), q) {
							continue
						}
						pr := pr
						selected := pr.ID == a.projectID
						row := ui.Row(c.Key("pm:" + pr.ID)).FillWidth().Radius(7).PaddingX(8).PaddingY(6).Gap(8).AlignItems(ui.Center)
						hover := row.Hovered()
						switch {
						case selected:
							row = row.Background(p.selected)
						case hover:
							row = row.Background(p.surface2)
						}
						row.Children(func() {
							dot := p.accent
							if pr.ThemeColor != "" {
								dot = ui.Hex(pr.ThemeColor)
							}
							main := ui.ButtonBase(c.Key("pm-sel:" + pr.ID)).Grow(1).Basis(0).MinWidth(0).Gap(8).AlignItems(ui.Center).Justify(ui.Start)
							main.Children(func() {
								ui.Box(c).Size(10, 10).Radius(5).Background(dot).Shrink(0)
								txtE(c, pr.Name, fsBase, p.fg, 500).Grow(1).Basis(0)
							})
							if main.Clicked() {
								id, name := pr.ID, pr.Name
								a.later(func() { a.chooseProject(id, name) })
							}
							if selected {
								txt(c, a.t("当前", "Current"), 9, p.muted, 500).Shrink(0)
							}
							if hover {
								ui.Row(c).Gap(2).Shrink(0).Children(func() {
									if a.button(c, btnOpts{w: 20, h: 20, radius: 4, hoverBg: &p.btnBg, tip: a.t("导出项目", "Export project"), key: "pm-ex:" + pr.ID}, func(bool) {
										icon(c, "download", 11, p.muted)
									}) {
										id, name := pr.ID, pr.Name
										a.later(func() { a.exportProjectByID(id, name) })
									}
									if a.button(c, btnOpts{w: 20, h: 20, radius: 4, hoverBg: &p.btnBg, tip: a.t("删除项目", "Delete project"), key: "pm-del:" + pr.ID}, func(bool) {
										icon(c, "delete", 11, p.muted)
									}) {
										id, name := pr.ID, pr.Name
										a.later(func() {
											pm.open = false
											a.deleteProject(id, name)
										})
									}
								})
							}
						})
					}
				})
			})
		})
	})
	if !open {
		pm.open = false
	}
	if create {
		name := trimName(pm.query)
		if name != "" {
			a.later(func() { a.createProjectFromMenu(name) })
		}
	}
}

// chooseProject 在下拉面板里选择项目：当前项目只回到工作区；当前工作区有未保存标签时先确认。
func (a *app) chooseProject(id, name string) {
	a.projMenu.open = false
	if id == a.projectID {
		a.showHome = false
		return
	}
	dirty := false
	for _, t := range a.tabs {
		t.refreshDirty()
		if t.dirty {
			dirty = true
			break
		}
	}
	if !dirty {
		a.selectProject(id)
		return
	}
	a.overlays.askConfirm(
		a.t("切换项目确认", "Confirm project switch"),
		a.t("当前工作区存在未保存的标签。切换到「"+name+"」后，这些修改仍会保留在原项目标签中，但你将离开当前编辑上下文。确定继续吗？",
			"There are unsaved tabs in current workspace. When switching to \""+name+"\", those changes remain in original project tabs, but you'll leave the current editing context. Continue?"),
		a.t("继续切换", "Continue"),
		func() { a.selectProject(id) },
	)
	logger.Info("切换项目需要确认未保存标签", "projectID", id)
}

// createProjectFromMenu 按下拉面板里输入的名称新建项目并切换过去。
func (a *app) createProjectFromMenu(name string) {
	pr, err := a.be.CreateProject(name)
	if err != nil {
		a.fail(a.t("创建项目", "Create project"), err)
		return
	}
	a.projMenu = projMenuState{}
	logger.Info("已从工具栏创建项目", "projectID", pr.ID)
	a.loadProjects()
	a.selectProject(pr.ID)
}

var unsafeFileChars = regexp.MustCompile(`[\\/:*?"<>|]`)

// exportProjectByID 导出指定项目；文件名沿用旧版的 <项目名>.postman_collection.json。
func (a *app) exportProjectByID(pid, name string) {
	a.projMenu.open = false
	safe := strings.TrimSpace(unsafeFileChars.ReplaceAllString(name, "-"))
	if safe == "" {
		safe = pid
	}
	a.async(func() func() {
		content, err := a.be.ExportProjectJSON(pid)
		if err == nil {
			err = a.be.SaveFileDialogJSON(a.win, safe+".postman_collection.json", content)
		}
		return func() { a.fail(a.t("导出项目", "Export project"), err) }
	})
}
