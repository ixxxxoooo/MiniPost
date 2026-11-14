package gui

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"minipost/internal/client"
	"minipost/internal/pkg/logger"
)

// curlState 是“导入 cURL”对话框的状态。
type curlState struct {
	open bool
	text string
}

// curlDialogView 绘制导入 cURL 对话框。
func (a *app) curlDialogView(c *ui.Context) {
	s := &a.overlays.curl
	ui.Modal(c, &s.open, func() {
		ui.Text(c, a.t("导入 cURL", "Import cURL")).Bold()
		ui.Text(c, a.t("粘贴 cURL 命令，将创建一个新请求。", "Paste a cURL command to create a new request.")).TextColor(c.Theme().TextMuted)
		ui.TextArea(c, &s.text).AutoFocus().Width(560).Height(180)
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			cancel := ui.Button(c, a.t("取消", "Cancel")).Clicked()
			ok := ui.PrimaryButton(c, a.t("导入", "Import")).Clicked()
			if cancel {
				s.open = false
			}
			if ok && strings.TrimSpace(s.text) != "" {
				s.open = false
				text := s.text
				s.text = ""
				a.later(func() { a.importCurl(text, "") })
			}
		})
	})
}

// importCurl 解析 cURL 命令，在指定文件夹下创建请求并打开。
func (a *app) importCurl(text, folderID string) {
	in, err := a.be.ImportCurl(text)
	if err != nil {
		a.fail(a.t("导入 cURL", "Import cURL"), err)
		return
	}
	d := client.DraftFromSendInput(*in, "", a.projectID, folderID)
	it, err := a.be.CreateRequestItem(a.projectID, folderID, d.Name)
	if err != nil {
		a.fail(a.t("创建请求", "Create request"), err)
		return
	}
	d.ID, d.SortOrder, d.CreatedAt, d.UpdatedAt = it.ID, it.SortOrder, it.CreatedAt, it.UpdatedAt
	if err := a.be.SaveRequestItem(d.ToItem()); err != nil {
		a.fail(a.t("保存请求", "Save request"), err)
		return
	}
	a.reloadCollection()
	a.openRequest(it.ID)
	logger.Info("已导入 cURL 请求", "requestID", it.ID)
}
