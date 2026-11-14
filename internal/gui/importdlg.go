package gui

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"minipost/internal/pkg/logger"
	"minipost/internal/service"
)

// importState 是导入预检对话框的状态。
type importState struct {
	open     bool
	preview  *service.ImportPreview
	strategy int
}

// 导入冲突策略，顺序与界面下拉框一致。
var importStrategies = []string{service.ImportStrategyCopy, service.ImportStrategyUpdate, service.ImportStrategyOverwrite}

// importFromFile 选择文件并预检。
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

// importFromURL 询问地址并预检（支持 Swagger UI 页面与 OpenAPI 文档地址）。
func (a *app) importFromURL() {
	pid := a.projectID
	a.overlays.askText(a.t("从 URL 导入", "Import from URL"), a.t("支持 Swagger / OpenAPI 文档或 Swagger UI 页面地址", "Swagger / OpenAPI document or Swagger UI page URL"), "", a.t("预检", "Preview"), func(u string) {
		a.async(func() func() {
			p, err := a.be.PreviewImportFromURL(pid, "", u)
			return func() { a.showImportPreview(p, err) }
		})
	})
}

func (a *app) showImportPreview(p *service.ImportPreview, err error) {
	if err != nil {
		a.fail(a.t("导入预检", "Import preview"), err)
		return
	}
	a.overlays.imp = importState{open: true, preview: p}
	logger.Info("导入预检完成", "kind", p.Kind, "conflicts", p.ConflictCount)
}

// importView 绘制导入预检对话框：显示类型与冲突，选择策略后执行导入。
func (a *app) importView(c *ui.Context) {
	s := &a.overlays.imp
	th := c.Theme()
	ui.Modal(c, &s.open, func() {
		ui.Text(c, a.t("导入集合", "Import")).Bold()
		if s.preview == nil {
			return
		}
		p := s.preview
		ui.Text(c, a.t("检测到的格式：", "Detected format: ")+p.Kind).TextColor(th.TextMuted)
		if p.ConflictCount > 0 {
			ui.Textf(c, a.t("发现 %d 个与现有请求（方法 + URL）相同的接口：", "%d requests match existing ones (method + URL):"), p.ConflictCount).TextColor(th.Warning)
			ui.Scroll(c).MaxHeight(140).Width(520).Children(func() {
				ui.Column(c).Gap(2).Children(func() {
					for _, cf := range p.Conflicts {
						ui.Text(c, fmt.Sprintf("%s %s  (%s)", cf.Method, cf.URL, cf.Name)).SingleLine()
					}
					if p.ConflictCount > len(p.Conflicts) {
						ui.Text(c, "…").TextColor(th.TextMuted)
					}
				})
			})
			ui.Text(c, a.t("冲突处理方式", "When a request already exists")).TextColor(th.TextMuted)
			selectIndex(c, &s.strategy, []string{
				a.t("导入副本（保留现有）", "Import as copies"),
				a.t("更新已有（保留我的修改）", "Update existing (keep my edits)"),
				a.t("覆盖已有", "Overwrite existing"),
			})
		} else {
			ui.Text(c, a.t("没有重复的请求。", "No duplicate requests.")).TextColor(th.TextMuted)
		}
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			cancel := ui.Button(c, a.t("取消", "Cancel")).Clicked()
			ok := ui.PrimaryButton(c, a.t("导入", "Import")).Clicked()
			if cancel {
				s.open = false
			}
			if ok {
				s.open = false
				strategy := importStrategies[s.strategy]
				a.later(func() { a.runImport(p, strategy) })
			}
		})
	})
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
	pid := a.projectID
	name := "minipost-export"
	if p := a.project(); p != nil {
		name = strings.ReplaceAll(p.Name, "/", "-")
	}
	a.async(func() func() {
		content, err := a.be.ExportProjectJSON(pid)
		if err == nil {
			err = a.be.SaveFileDialogJSON(a.win, name+".json", content)
		}
		return func() { a.fail(a.t("导出项目", "Export project"), err) }
	})
}
