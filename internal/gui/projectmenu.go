package gui

import (
	"regexp"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"minipost/internal/pkg/logger"
)

// projMenuState 是工具栏项目下拉面板的状态。
type projMenuState struct {
	open  bool
	query string
}

// projPanelW 是项目下拉面板的宽度（设计稿 .pop 的 260）。
const projPanelW = 260

// projectDropdown 绘制项目下拉面板：搜索/新建栏 + 项目列表（当前标记、悬停时的导出与删除）。
func (a *app) projectDropdown(c *ui.Context, anchor ui.Element, trigW float32) {
	p := a.p
	pm := &a.projMenu
	w := max(float32(projPanelW), trigW)
	open := true
	create := false
	ui.PopoverBase(c, anchor, &open, func(panel ui.Element) {
		a.popPanel(panel.AttachTo(anchor, ui.AnchorBottomLeft, ui.AnchorTopLeft).Margin(6, 0, 0, 0), w).Clip()
		ui.Column(c).FillWidth().Children(func() {
			// 搜索 / 新建（设计稿：输入框 + 26 高“新建”按钮）
			ui.Row(c).FillWidth().Padding(2, 2, 6, 2).Gap(6).AlignItems(ui.Center).Shrink(0).Children(func() {
				in := a.input(c, &pm.query, a.t("搜索或新建项目…", "Search or create project…"), fieldOpts{
					h: 26, radius: radiusR, fs: fsBase, icon: "search", key: "pm-q", focus: true, iconSize: 12, iconLeft: 8, iconGap: 6,
				})
				if in.Submitted() {
					create = true
				}
				bd := p.btnBorder
				if a.button(c, btnOpts{h: 26, radius: radiusR, padX: 11, border: &bd, bg: &p.surface, hoverBg: &p.surface2, key: "pm-new"}, func(bool) {
					txt(c, a.t("新建", "New"), fsBase, p.fg, 550)
				}) {
					create = true
				}
			})
			// 项目列表（行高 28：悬停时的操作按钮替换右侧时间，不会撑高行）
			ui.Scroll(c).FillWidth().MaxHeight(320).Scrollbars(a.scrollbarsVisible()).Children(func() {
				ui.Column(c).FillWidth().Children(func() {
					q := strings.ToLower(pm.query)
					now := time.Now()
					i := 0
					for _, pr := range a.projects {
						if q != "" && !strings.Contains(strings.ToLower(pr.Name), q) {
							continue
						}
						pr := pr
						selected := pr.ID == a.projectID
						row := a.popItem(c, "pm-sel:"+pr.ID, i, selected)
						i++
						hover := row.Hovered()
						row.Children(func() {
							a.projectAvatar(c, pr, 16, 9)
							txtE(c, pr.Name, fsBase, p.fg, 500).Grow(1).Basis(0)
							switch {
							case hover:
								ui.Row(c).Gap(2).Shrink(0).Children(func() {
									if a.button(c, btnOpts{w: 20, h: 20, radius: 5, hoverBg: &p.btnBg, tip: a.t("导出项目", "Export project"), key: "pm-ex:" + pr.ID}, func(bool) {
										icon(c, "download", 11, p.muted)
									}) {
										id, name := pr.ID, pr.Name
										a.later(func() { a.exportProjectByID(id, name) })
									}
									if a.button(c, btnOpts{w: 20, h: 20, radius: 5, hoverBg: &p.btnBg, tip: a.t("删除项目", "Delete project"), key: "pm-del:" + pr.ID}, func(bool) {
										icon(c, "delete", 11, p.muted)
									}) {
										id, name := pr.ID, pr.Name
										a.later(func() {
											pm.open = false
											a.deleteProject(id, name)
										})
									}
								})
							case selected:
								a.curBadge(c)
							default:
								txt(c, a.relTime(pr.UpdatedAt, now), fsSmall-0.5, p.muted, 0).Shrink(0)
							}
						})
						if row.Clicked() {
							id, name := pr.ID, pr.Name
							a.later(func() { a.chooseProject(id, name) })
						}
					}
				})
			})
			a.popSep(c)
			// 全部项目…（回到项目主页）
			all := a.popItem(c, "pm-all", len(a.projects), false)
			all.Children(func() {
				icon(c, "grid", 13, p.fg2)
				txt(c, a.t("全部项目…", "All projects…"), fsBase, p.fg, 0).Grow(1)
				a.kbdHint(c, "⌘⇧H")
			})
			if all.Clicked() {
				a.later(func() {
					pm.open = false
					if !a.showHome {
						a.persistSession()
						a.showHome = true
					}
				})
			}
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
