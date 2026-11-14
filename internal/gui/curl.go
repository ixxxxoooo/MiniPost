package gui

import (
	"errors"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/google/uuid"

	"minipost/internal/client"
	"minipost/internal/pkg/logger"
)

// draftIDPrefix 标记尚未写入磁盘的草稿标签 ID（旧版 cURL 导入只生成未保存草稿）。
const draftIDPrefix = "draft:"

// curlState 是 ⌘I 的 cURL 导入弹窗状态。
type curlState struct {
	open bool
	text string
	err  string
	// enter 由文本框的按键回调置位（回调在两帧之间运行），下一帧据此导入。
	enter bool
}

// openCurlDialog 打开 cURL 导入弹窗（⌘I / 工具栏图标）。
func (a *app) openCurlDialog() {
	a.overlays.curl = curlState{open: true}
}

// importCurl 解析 cURL 命令，并把结果作为「未保存草稿」标签打开（与旧版一致，保存时才写入项目）。
func (a *app) importCurl(text, folderID string) error {
	if a.projectID == "" {
		return errors.New(a.t("请先选择项目后再导入", "Please select a project before importing"))
	}
	in, err := a.be.ImportCurl(text)
	if err != nil {
		logger.Warn("cURL 解析失败", "error", err.Error())
		return err
	}
	d := client.DraftFromSendInput(*in, "", a.projectID, folderID)
	d.ID = draftIDPrefix + uuid.NewString()
	base := client.NewDraft(a.projectID, folderID, "")
	base.ID = d.ID
	t := newTab(d, base)
	t.unsaved = true
	a.tabs = append(a.tabs, t)
	a.activate(len(a.tabs) - 1)
	logger.Info("cURL 已导入为未保存草稿", "method", d.Method, "hasURL", d.URL != "", "headers", len(d.Headers))
	return nil
}

// persistDraft 把未保存草稿标签真正写入项目（第一次保存时创建请求）。
func (a *app) persistDraft(t *tab) error {
	d := t.draft
	it, err := a.be.CreateRequestItem(a.projectID, d.FolderID, d.Name)
	if err != nil {
		return err
	}
	d.ID, d.SortOrder, d.CreatedAt, d.UpdatedAt = it.ID, it.SortOrder, it.CreatedAt, it.UpdatedAt
	t.unsaved = false
	logger.Info("草稿已创建为请求", "requestID", d.ID)
	return nil
}

// curlView 绘制 ⌘I 的 cURL 导入弹窗：500px，Enter 导入、Shift+Enter 换行（同旧版 Toolbar）。
func (a *app) curlView(c *ui.Context) {
	s := &a.overlays.curl
	if !s.open {
		return
	}
	p := a.p
	doImport := false
	empty := strings.TrimSpace(s.text) == ""
	a.dialogDim(c, &s.open, 500, 0, 12, 0.5, func() {
		ui.Column(c).FillWidth().Children(func() {
			// 标题行：px20 pt16 pb8
			ui.Row(c).FillWidth().Padding(16, 20, 8, 20).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Row(c).Size(28, 28).Radius(8).AlignItems(ui.Center).Justify(ui.Center).Shrink(0).Children(func() {
					icon(c, "fileImport", 15, p.accent)
				})
				ui.Column(c).Grow(1).Basis(0).Children(func() {
					txt(c, a.t("导入 cURL", "Import cURL"), 14, p.fg, 600)
					txt(c, a.t("粘贴 cURL 命令以快速创建请求", "Paste a cURL command to quickly create a request"), fsSmall, p.muted, 0).Margin(2, 0, 0, 0)
				})
				if a.button(c, btnOpts{w: 20, h: 20, radius: 6, hoverBg: &p.sideHover, tip: a.t("关闭", "Close"), key: "curl-close"}, func(bool) {
					icon(c, "clear", 11, p.muted)
				}) {
					s.open = false
				}
			})
			// 正文：px20 pb8
			ui.Column(c).FillWidth().Padding(0, 20, 8, 20).Children(func() {
				bd := p.border
				box := ui.Row(c).FillWidth().Height(180).Radius(8).Background(p.surface2)
				if box.FocusWithin() {
					bd = p.accent
				}
				box.Border(1, bd).Children(func() {
					ta := ui.TextAreaBase(c, &s.text).Placeholder("curl --request GET \\\n  --url https://api.example.com/data \\\n  --header 'Content-Type: application/json'").
						PlaceholderColor(p.muted.Alpha(0.6)).Font(monoFont).FontSize(fsBase).TextColor(p.fg).Grow(1).Basis(0).FillHeight().Padding(14).AutoFocus()
					// Enter 导入，Shift+Enter 换行
					ta.OnTextKey(func(k ui.TextKey) bool {
						if k.Key == ui.KeyEnter && k.Mods == 0 {
							s.enter = true
							return true
						}
						return false
					})
					if s.enter {
						s.enter, doImport = false, true
					}
				})
				if s.err != "" {
					ui.Row(c).Gap(6).AlignItems(ui.Center).Margin(8, 0, 0, 0).Children(func() {
						icon(c, "info", 12, p.danger)
						ui.Text(c, s.err).FontSize(fsSmall).TextColor(p.danger).Selectable()
					})
				}
				txt(c, a.t("Enter 导入，Shift + Enter 换行", "Press Enter to import, Shift + Enter for newline"), fsTiny, p.muted, 0).Margin(14, 0, 0, 0)
			})
			// 底部：快捷键提示 + 取消 / 导入
			ui.Row(c).FillWidth().Padding(2, 20, 16, 20).AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					a.kbdChip(c, "⌘I")
					txt(c, a.t("快速唤起", "Quick launch"), fsTiny, p.muted, 0)
				})
				ui.Row(c).Gap(6).Shrink(0).Children(func() {
					bd := p.border
					if a.button(c, btnOpts{h: 30, radius: 8, padX: 14, border: &bd, key: "curl-cancel"}, func(bool) {
						ui.Row(c).MinWidth(84 - 28).Justify(ui.Center).Children(func() { txt(c, a.t("取消", "Cancel"), 12, p.fg2, 500) })
					}) {
						s.open = false
					}
					acc := p.accent
					col := p.accentFg
					if empty {
						// 旧版整体 opacity 0.4：文字仍是白色，只有按钮底色变浅
						acc = acc.Alpha(0.4)
					}
					if a.button(c, btnOpts{h: 30, radius: 8, padX: 18, bg: &acc, hoverBg: &p.accentHover, disabled: empty, key: "curl-ok"}, func(bool) {
						ui.Row(c).MinWidth(84 - 36).Justify(ui.Center).Children(func() { txt(c, a.t("导入", "Import")+" (Enter)", 12, col, 500) })
					}) {
						doImport = true
					}
				})
			})
		})
	})
	if doImport && !empty {
		text := strings.TrimSpace(s.text)
		a.later(func() {
			if err := a.importCurl(text, ""); err != nil {
				s.err = err.Error()
				return
			}
			s.text, s.err, s.open = "", "", false
		})
	}
}
