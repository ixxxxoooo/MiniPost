package gui

import (
	"minipost/internal/client"
	"minipost/internal/pkg/logger"
)

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
