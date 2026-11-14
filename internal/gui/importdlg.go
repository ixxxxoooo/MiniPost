package gui

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"minipost/internal/pkg/logger"
	"minipost/internal/service"
)

// 导入对话框的三种方式。
const (
	importFile = iota
	importURL
	importCurlMode
)

// importState 是统一导入对话框（文件 / URL / cURL）的状态。
type importState struct {
	open     bool
	mode     int
	preview  *service.ImportPreview
	strategy int
	url      string
	curl     string
	enter    bool // cURL 文本框按下 Enter（由按键回调置位）
	busy     bool
	err      string
}

// 导入冲突策略，顺序与界面选项一致。
var importStrategies = []string{service.ImportStrategyCopy, service.ImportStrategyUpdate, service.ImportStrategyOverwrite}

// openImport 打开导入对话框并切到指定方式。
func (a *app) openImport(mode int) {
	if a.projectID == "" {
		a.toast(a.t("请先选择项目", "Select a project first"))
		return
	}
	a.overlays.imp = importState{open: true, mode: mode}
}

// importFromFile 选择文件并预检，结果显示在导入对话框里。
func (a *app) importFromFile() {
	pid := a.projectID
	a.async(func() func() {
		content, err := a.be.OpenFileDialogJSON(a.win)
		if err != nil || content == "" {
			return func() { a.fail(a.t("读取导入文件", "Read import file"), err) }
		}
		p, err := a.be.PreviewImportFromFile(pid, "", content)
		return func() { a.showImportPreview(p, err) }
	})
}

// importFromURL 按 URL 方式打开导入对话框。
func (a *app) importFromURL() { a.openImport(importURL) }

// previewImportURL 拉取并预检远程文档；没有冲突时直接导入。
func (a *app) previewImportURL(u string) {
	pid := a.projectID
	s := &a.overlays.imp
	s.busy, s.err = true, ""
	a.async(func() func() {
		p, err := a.be.PreviewImportFromURL(pid, "", u)
		return func() {
			s.busy = false
			if err != nil {
				s.err = err.Error()
				logger.Warn("URL 导入预检失败", "error", err.Error())
				return
			}
			a.showImportPreview(p, nil)
			if p.ConflictCount == 0 {
				a.runImport(p, importStrategies[0])
				s.open = false
			}
		}
	})
}

// showImportPreview 把预检结果放进导入对话框（对话框未打开时按文件方式打开）。
func (a *app) showImportPreview(p *service.ImportPreview, err error) {
	if err != nil {
		a.fail(a.t("导入预检", "Import preview"), err)
		return
	}
	s := &a.overlays.imp
	if !s.open {
		*s = importState{open: true, mode: importFile}
	}
	s.preview, s.strategy, s.err = p, 0, ""
	logger.Info("导入预检完成", "kind", p.Kind, "conflicts", p.ConflictCount)
}

// importKindTitle 把后端识别的类型名转成界面文案。
func (a *app) importKindTitle(kind string) string {
	switch kind {
	case "postman":
		return "Postman Collection"
	case "postman_env", "environment":
		return "Postman Environment"
	case "openapi":
		return "OpenAPI"
	case "swagger":
		return "Swagger"
	}
	return kind
}

// importView 绘制统一导入对话框：620px，标题、方式切换、内容与底部按钮。
func (a *app) importView(c *ui.Context) {
	s := &a.overlays.imp
	if !s.open {
		return
	}
	p := a.p
	doImport := false
	a.stdDialog(c, &s.open, 620, func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Row(c).Size(28, 28).Radius(8).Background(p.accent.Alpha(0.12)).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
				icon(c, "fileImport", 14, p.accent)
			})
			ui.Column(c).Children(func() {
				txt(c, a.t("导入", "Import"), fsLarge, p.fg, 600)
				txt(c, a.t("支持文件、URL 与 cURL 导入", "Supports file, URL, and cURL import"), fsSmall, p.muted, 0)
			})
		})
		if !s.busy && a.iconButton(c, "clear", a.t("关闭", "Close"), 12, p.muted) {
			s.open = false
		}
	}, func() {
		ui.Row(c).FillWidth().Padding(12, 16, 0, 16).Children(func() {
			ui.Row(c).Radius(radiusInput).Background(p.surface2).Padding(2).Children(func() {
				for i, l := range []string{a.t("文件导入", "File import"), a.t("URL 导入", "URL import"), a.t("cURL 导入", "cURL import")} {
					active := s.mode == i
					b := ui.ButtonBase(c.Key("im-mode" + itoa(i))).Height(28).PaddingX(12).Radius(7).AlignItems(ui.Center).Justify(ui.Center)
					col := p.fg2
					if active {
						b = b.Background(p.surface).Shadow(0, 1, 2, 0, ui.RGBA(0, 0, 0, 0.08))
						col = p.fg
					}
					b.Children(func() { txt(c, l, fsBase, col, 500) })
					if b.Clicked() && !s.busy && s.mode != i {
						s.mode, s.err = i, ""
					}
				}
			})
		})
		ui.Column(c).FillWidth().Padding(12, 16, 12, 16).Gap(12).Children(func() {
			switch s.mode {
			case importFile:
				a.importFileBody(c, s)
			case importURL:
				a.importURLBody(c, s, &doImport)
			default:
				a.importCurlBody(c, s, &doImport)
			}
			if s.busy {
				txt(c, a.t("正在导入，请稍候...", "Importing, please wait..."), fsTiny, p.muted, 0)
			}
			if s.err != "" {
				ui.Row(c).FillWidth().Radius(8).Border(1, p.danger.Alpha(0.25)).Background(p.danger.Alpha(0.08)).PaddingX(10).PaddingY(8).Children(func() {
					ui.Text(c, s.err).FontSize(fsSmall).TextColor(p.danger).Selectable()
				})
			}
		})
	}, func() {
		if a.dlgButton(c, "im-cancel", a.t("取消", "Cancel"), dlgNormal, s.busy) {
			s.open = false
		}
		label := a.t("导入文件", "Import file")
		disabled := s.busy
		switch s.mode {
		case importFile:
			disabled = disabled || s.preview == nil
		case importURL:
			label = a.t("导入地址", "Import URL")
			disabled = disabled || strings.TrimSpace(s.url) == ""
		default:
			label = a.t("导入 cURL", "Import cURL")
			disabled = disabled || strings.TrimSpace(s.curl) == ""
		}
		if a.dlgButton(c, "im-ok", label, dlgPrimary, disabled) {
			doImport = true
		}
	})
	if doImport && !s.busy {
		switch s.mode {
		case importFile:
			if s.preview != nil {
				pv, strategy := s.preview, importStrategies[s.strategy]
				s.open = false
				a.later(func() { a.runImport(pv, strategy) })
			}
		case importURL:
			if u := strings.TrimSpace(s.url); u != "" {
				a.later(func() { a.previewImportURL(u) })
			}
		default:
			if text := strings.TrimSpace(s.curl); text != "" {
				a.later(func() {
					if err := a.importCurl(text, ""); err != nil {
						s.err = err.Error()
						return
					}
					s.open, s.curl, s.err = false, "", ""
				})
			}
		}
	}
}

// kbdChip 是对话框里的小号快捷键标记。
func (a *app) kbdChip(c *ui.Context, s string) {
	p := a.p
	ui.Row(c).Radius(4).Background(p.surface2).Border(1, p.border).PaddingX(4).PaddingY(1).Children(func() {
		txtMono(c, s, 9, p.muted, 0)
	})
}

// infoCard 是对话框里灰底小卡片：小标题 + 加粗说明 + 次要说明。
func (a *app) infoCard(c *ui.Context, label, title, desc string) {
	p := a.p
	ui.Column(c).FillWidth().Radius(radiusCard).Border(1, p.border).Background(p.surface2).PaddingX(12).PaddingY(10).Gap(2).Children(func() {
		txt(c, label, fsSmall, p.muted, 0)
		ui.Text(c, title).FontSize(fsBase).FontWeight(500).TextColor(p.fg).Margin(2, 0, 0, 0)
		ui.Text(c, desc).FontSize(fsSmall).TextColor(p.fg2)
	})
}

func (a *app) importFileBody(c *ui.Context, s *importState) {
	p := a.p
	b := ui.ButtonBase(c.Key("im-pick")).FillWidth().Height(88).Radius(radiusCard).Border(1, p.border).AlignItems(ui.Center).Justify(ui.Center).Gap(6)
	if b.Hovered() {
		b = b.Background(p.btnBg)
	} else {
		b = b.Background(p.surface2)
	}
	b.Children(func() {
		ui.Column(c).AlignItems(ui.Center).Gap(6).Children(func() {
			icon(c, "upload", 16, p.fg2)
			txt(c, a.t("选择导入文件", "Select import file"), fsBase, p.fg, 0)
			txt(c, "Postman Collection / Environment / OpenAPI / Swagger（JSON/YAML）", fsTiny, p.muted, 0)
		})
	})
	if b.Clicked() {
		a.later(a.importFromFile)
	}
	title, desc := a.t("尚未选择文件", "No file selected"), a.t("选择文件后将自动识别类型。", "The type is detected after you pick a file.")
	if pv := s.preview; pv != nil {
		title = a.importKindTitle(pv.Kind)
		desc = a.t("没有重复的请求。", "No duplicate requests.")
		if pv.ConflictCount > 0 {
			desc = fmt.Sprintf(a.t("发现 %d 个与现有请求（方法 + URL）相同的接口。", "%d requests match existing ones (method + URL)."), pv.ConflictCount)
		}
	}
	a.infoCard(c, a.t("识别结果", "Detection result"), title, desc)
	if pv := s.preview; pv != nil && pv.ConflictCount > 0 {
		a.conflictList(c, s, pv)
	}
}

// conflictList 列出冲突的请求并提供处理方式单选。
func (a *app) conflictList(c *ui.Context, s *importState, pv *service.ImportPreview) {
	p := a.p
	ui.Scroll(c).FillWidth().MaxHeight(120).Scrollbars(a.scrollbarsVisible()).Children(func() {
		ui.Column(c).FillWidth().Gap(2).Children(func() {
			for _, cf := range pv.Conflicts {
				txtEMono(c, fmt.Sprintf("%s %s  (%s)", cf.Method, cf.URL, cf.Name), fsSmall, p.fg2, 0)
			}
			if pv.ConflictCount > len(pv.Conflicts) {
				txt(c, "…", fsSmall, p.muted, 0)
			}
		})
	})
	txt(c, a.t("冲突处理方式", "When a request already exists"), fsSmall, p.muted, 0)
	labels := []string{
		a.t("导入副本（保留现有）", "Import as copies"),
		a.t("更新已有（保留我的修改）", "Update existing (keep my edits)"),
		a.t("覆盖已有", "Overwrite existing"),
	}
	for i, l := range labels {
		on := s.strategy == i
		b := ui.ButtonBase(c.Key("im-strat" + itoa(i))).FillWidth().Height(26).Gap(8).AlignItems(ui.Center).Justify(ui.Start)
		b.Children(func() {
			ui.Row(c).Grow(1).Gap(8).AlignItems(ui.Center).Children(func() {
				ring := ui.Row(c).Size(12, 12).Radius(6).Shrink(0).AlignItems(ui.Center).Justify(ui.Center)
				if on {
					ring.Border(1, p.accent).Children(func() { ui.Box(c).Size(6, 6).Radius(3).Background(p.accent) })
				} else {
					ring.Border(1, p.btnBorder)
				}
				txt(c, l, fsBase, p.fg, 0)
			})
		})
		if b.Clicked() {
			s.strategy = i
		}
	}
}

func (a *app) importURLBody(c *ui.Context, s *importState, doImport *bool) {
	bg := a.p.surface2
	ui.Row(c).FillWidth().Children(func() {
		in := a.input(c, &s.url, a.t("粘贴 Swagger / OpenAPI / Postman 文档地址", "Paste a Swagger / OpenAPI / Postman document URL"), fieldOpts{h: 40, radius: radiusCard, fs: fsBase, padX: 12, bg: &bg, key: "im-url"})
		if in.Submitted() {
			*doImport = true
		}
	})
	a.infoCard(c, a.t("导入方式", "Import method"), a.t("自动拉取并识别远程文档", "Fetch and auto-detect remote document"),
		a.t("支持 http/https 地址，内容会按 Postman / OpenAPI / Swagger 自动识别。", "Supports http/https URLs and auto-detects Postman / OpenAPI / Swagger content."))
	if pv := s.preview; pv != nil && pv.ConflictCount > 0 {
		a.conflictList(c, s, pv)
	}
}

func (a *app) importCurlBody(c *ui.Context, s *importState, doImport *bool) {
	p := a.p
	bd := p.border
	box := ui.Row(c).FillWidth().Height(180).Radius(radiusCard).Background(p.surface2)
	if box.FocusWithin() {
		bd = p.accent
	}
	box.Border(1, bd).Children(func() {
		ta := ui.TextAreaBase(c, &s.curl).Placeholder(a.t("粘贴 cURL 命令，按 Enter 可直接导入", "Paste a cURL command, press Enter to import")).
			PlaceholderColor(p.muted).Font(monoFont).FontSize(fsBase).TextColor(p.fg).Grow(1).Basis(0).FillHeight().Padding(12).AutoFocus()
		// Enter 导入，Shift+Enter 换行
		ta.OnTextKey(func(k ui.TextKey) bool {
			if k.Key == ui.KeyEnter && k.Mods == 0 {
				s.enter = true
				return true
			}
			return false
		})
		if s.enter {
			s.enter, *doImport = false, true
		}
	})
	txt(c, a.t("Enter 导入，Shift+Enter 换行", "Enter to import, Shift+Enter for newline"), fsTiny, p.muted, 0)
}

func (a *app) runImport(p *service.ImportPreview, strategy string) {
	if err := a.be.ImportCollectionWithStrategy(a.projectID, p.Kind, p.Content, p.ResolvedURL, strategy); err != nil {
		a.fail(a.t("导入", "Import"), err)
		return
	}
	a.reloadCollection()
	a.reloadEnvs()
	a.toast(a.t("导入完成", "Import finished"))
	logger.Info("导入完成", "kind", p.Kind, "strategy", strategy)
}

// exportProject 把当前项目导出为 JSON 文件。
func (a *app) exportProject() {
	name := "minipost-export"
	if p := a.project(); p != nil {
		name = p.Name
	}
	a.exportProjectByID(a.projectID, name)
}
