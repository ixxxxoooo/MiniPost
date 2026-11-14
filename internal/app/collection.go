package app

import (
	"minipost/internal/model"
	"minipost/internal/pkg/logger"
)

// ---- 项目管理 ----

func (a *App) ListProjects() ([]model.Project, error) {
	projects, err := a.projectSvc.ListProjects()
	if err != nil {
		logger.Error("列出项目失败", "error", err.Error())
		return nil, err
	}
	logger.Debug("列出项目", "count", len(projects))
	return projects, nil
}

func (a *App) CreateProject(name string) (*model.Project, error) {
	project, err := a.projectSvc.CreateProject(name)
	if err != nil {
		logger.Error("创建项目失败", "name", name, "error", err.Error())
		return nil, err
	}
	logger.Info("创建项目成功", "id", project.ID, "name", project.Name)
	return project, nil
}

func (a *App) RenameProject(id, name string) (*model.Project, error) {
	project, err := a.projectSvc.RenameProject(id, name)
	if err != nil {
		logger.Error("重命名项目失败", "id", id, "name", name, "error", err.Error())
		return nil, err
	}
	logger.Info("重命名项目成功", "id", id, "name", name)
	return project, nil
}

func (a *App) UpdateProjectTheme(id, color string) (*model.Project, error) {
	project, err := a.projectSvc.UpdateProjectTheme(id, color)
	if err != nil {
		logger.Error("更新项目主题色失败", "id", id, "color", color, "error", err.Error())
		return nil, err
	}
	logger.Info("更新项目主题色成功", "id", id, "themeColor", project.ThemeColor)
	return project, nil
}

func (a *App) UpdateProjectDescription(id, description string) (*model.Project, error) {
	project, err := a.projectSvc.UpdateProjectDescription(id, description)
	if err != nil {
		logger.Error("更新项目描述失败", "id", id, "error", err.Error())
		return nil, err
	}
	logger.Info("更新项目描述成功", "id", id)
	return project, nil
}

func (a *App) DeleteProject(id string) error {
	if err := a.projectSvc.DeleteProject(id); err != nil {
		logger.Error("删除项目失败", "id", id, "error", err.Error())
		return err
	}
	logger.Info("删除项目成功", "id", id)
	return nil
}

// ---- 集合管理 ----

func (a *App) GetCollectionData(projectID string) (*model.CollectionData, error) {
	data, err := a.requestSvc.GetCollectionData(projectID)
	if err != nil {
		logger.Error("获取集合数据失败", "projectID", projectID, "error", err.Error())
		return nil, err
	}
	return data, nil
}

func (a *App) MoveCollectionNode(projectID, nodeID string, nodeType model.CollectionNodeType, targetParentFolderID string, targetIndex int) error {
	if err := a.requestSvc.MoveCollectionNode(projectID, nodeID, nodeType, targetParentFolderID, targetIndex); err != nil {
		logger.Error("移动集合节点失败", "nodeID", nodeID, "nodeType", nodeType, "targetParentFolderID", targetParentFolderID, "targetIndex", targetIndex, "error", err.Error())
		return err
	}
	logger.Info("移动集合节点成功", "nodeID", nodeID, "nodeType", nodeType, "targetParentFolderID", targetParentFolderID, "targetIndex", targetIndex)
	return nil
}

// RestoreCollectionSubtree 撤销删除：把文件夹/请求子树按原样放回项目。
func (a *App) RestoreCollectionSubtree(projectID string, sub model.CollectionData) error {
	if err := a.requestSvc.RestoreCollectionSubtree(projectID, sub); err != nil {
		logger.Error("恢复集合子树失败", "projectID", projectID, "error", err.Error())
		return err
	}
	logger.Info("恢复集合子树成功", "projectID", projectID, "folders", len(sub.Folders), "requests", len(sub.Requests))
	return nil
}

// ---- 文件夹管理 ----

func (a *App) ListFolders(projectID string) ([]model.Folder, error) {
	folders, err := a.requestSvc.ListFolders(projectID)
	if err != nil {
		logger.Error("列出文件夹失败", "projectID", projectID, "error", err.Error())
		return nil, err
	}
	logger.Debug("列出文件夹", "projectID", projectID, "count", len(folders))
	return folders, nil
}

func (a *App) CreateFolder(projectID, parentID, name string) (*model.Folder, error) {
	folder, err := a.requestSvc.CreateFolder(projectID, parentID, name)
	if err != nil {
		logger.Error("创建文件夹失败", "projectID", projectID, "name", name, "error", err.Error())
		return nil, err
	}
	logger.Info("创建文件夹成功", "id", folder.ID, "name", folder.Name)
	return folder, nil
}

func (a *App) RenameFolder(projectID, folderID, name string) error {
	if err := a.requestSvc.RenameFolder(projectID, folderID, name); err != nil {
		logger.Error("重命名文件夹失败", "folderID", folderID, "name", name, "error", err.Error())
		return err
	}
	logger.Info("重命名文件夹成功", "folderID", folderID, "name", name)
	return nil
}

func (a *App) DeleteFolder(projectID, folderID string) error {
	if err := a.requestSvc.DeleteFolder(projectID, folderID); err != nil {
		logger.Error("删除文件夹失败", "folderID", folderID, "error", err.Error())
		return err
	}
	logger.Info("删除文件夹成功", "folderID", folderID)
	return nil
}

func (a *App) MoveFolder(projectID, folderID, targetParentID string, targetIndex int) error {
	if err := a.requestSvc.MoveFolder(projectID, folderID, targetParentID, targetIndex); err != nil {
		logger.Error("移动文件夹失败", "folderID", folderID, "targetParentID", targetParentID, "targetIndex", targetIndex, "error", err.Error())
		return err
	}
	logger.Info("移动文件夹成功", "folderID", folderID, "targetParentID", targetParentID, "targetIndex", targetIndex)
	return nil
}

// ---- 请求管理 ----

func (a *App) ListRequests(projectID string) ([]model.RequestItem, error) {
	requests, err := a.requestSvc.ListRequests(projectID)
	if err != nil {
		logger.Error("列出请求失败", "projectID", projectID, "error", err.Error())
		return nil, err
	}
	logger.Debug("列出请求", "projectID", projectID, "count", len(requests))
	return requests, nil
}

func (a *App) CreateRequestItem(projectID, folderID, name string) (*model.RequestItem, error) {
	req, err := a.requestSvc.CreateRequest(projectID, folderID, name)
	if err != nil {
		logger.Error("创建请求失败", "projectID", projectID, "name", name, "error", err.Error())
		return nil, err
	}
	logger.Info("创建请求成功", "id", req.ID, "name", req.Name)
	return req, nil
}

func (a *App) SaveRequestItem(request model.RequestItem) error {
	if err := a.requestSvc.SaveRequest(&request); err != nil {
		logger.Error("保存请求失败", "id", request.ID, "error", err.Error())
		return err
	}
	logger.Debug("保存请求成功", "id", request.ID, "name", request.Name)
	return nil
}

func (a *App) DeleteRequestItem(projectID, requestID string) error {
	if err := a.requestSvc.DeleteRequest(projectID, requestID); err != nil {
		logger.Error("删除请求失败", "requestID", requestID, "error", err.Error())
		return err
	}
	logger.Info("删除请求成功", "requestID", requestID)
	return nil
}

func (a *App) MoveRequestItem(projectID, requestID, targetFolderID string, targetIndex int) error {
	if err := a.requestSvc.MoveRequest(projectID, requestID, targetFolderID, targetIndex); err != nil {
		logger.Error("移动请求失败", "requestID", requestID, "targetFolderID", targetFolderID, "targetIndex", targetIndex, "error", err.Error())
		return err
	}
	logger.Info("移动请求成功", "requestID", requestID, "targetFolderID", targetFolderID, "targetIndex", targetIndex)
	return nil
}

// ---- 环境变量 ----

func (a *App) ListEnvironments(projectID string) ([]model.Environment, error) {
	envs, err := a.envSvc.ListEnvironments(projectID)
	if err != nil {
		logger.Error("列出环境变量失败", "projectID", projectID, "error", err.Error())
		return nil, err
	}
	logger.Debug("列出环境变量", "projectID", projectID, "count", len(envs))
	return envs, nil
}

func (a *App) CreateEnvironment(projectID, name string) (*model.Environment, error) {
	env, err := a.envSvc.CreateEnvironment(projectID, name)
	if err != nil {
		logger.Error("创建环境变量失败", "projectID", projectID, "name", name, "error", err.Error())
		return nil, err
	}
	logger.Info("创建环境变量成功", "id", env.ID, "name", env.Name)
	return env, nil
}

func (a *App) SaveEnvironment(env model.Environment) error {
	if err := a.envSvc.SaveEnvironment(&env); err != nil {
		logger.Error("保存环境变量失败", "id", env.ID, "error", err.Error())
		return err
	}
	logger.Debug("保存环境变量成功", "id", env.ID, "name", env.Name)
	return nil
}

func (a *App) DeleteEnvironment(projectID, envID string) error {
	if err := a.envSvc.DeleteEnvironment(projectID, envID); err != nil {
		logger.Error("删除环境变量失败", "envID", envID, "error", err.Error())
		return err
	}
	logger.Info("删除环境变量成功", "envID", envID)
	return nil
}

// ---- 重命名请求 ----

func (a *App) RenameRequest(projectID, requestID, name string) error {
	if err := a.requestSvc.RenameRequest(projectID, requestID, name); err != nil {
		logger.Error("重命名请求失败", "requestID", requestID, "name", name, "error", err.Error())
		return err
	}
	logger.Info("重命名请求成功", "requestID", requestID, "name", name)
	return nil
}

// ---- 复制 ----

func (a *App) DuplicateRequest(projectID, requestID string) (*model.RequestItem, error) {
	req, err := a.requestSvc.DuplicateRequest(projectID, requestID)
	if err != nil {
		logger.Error("复制请求失败", "requestID", requestID, "error", err.Error())
		return nil, err
	}
	logger.Info("复制请求成功", "sourceID", requestID, "newID", req.ID)
	return req, nil
}

func (a *App) DuplicateFolder(projectID, folderID string) (*model.Folder, error) {
	folder, err := a.requestSvc.DuplicateFolder(projectID, folderID)
	if err != nil {
		logger.Error("复制文件夹失败", "folderID", folderID, "error", err.Error())
		return nil, err
	}
	logger.Info("复制文件夹成功", "sourceID", folderID, "newID", folder.ID)
	return folder, nil
}

// ---- 历史记录 ----

func (a *App) GetHistory(projectID string) ([]model.HistoryEntry, error) {
	entries, err := a.historySvc.GetHistory(projectID)
	if err != nil {
		logger.Error("获取历史记录失败", "projectID", projectID, "error", err.Error())
		return nil, err
	}
	logger.Debug("获取历史记录", "projectID", projectID, "count", len(entries))
	return entries, nil
}

func (a *App) DeleteHistoryEntry(projectID, entryID string) error {
	if err := a.historySvc.DeleteEntry(projectID, entryID); err != nil {
		logger.Error("删除历史记录失败", "projectID", projectID, "entryID", entryID, "error", err.Error())
		return err
	}
	logger.Info("删除历史记录成功", "projectID", projectID, "entryID", entryID)
	return nil
}

func (a *App) ClearHistory(projectID string) error {
	if err := a.historySvc.ClearHistory(projectID); err != nil {
		logger.Error("清除历史记录失败", "projectID", projectID, "error", err.Error())
		return err
	}
	logger.Info("清除历史记录成功", "projectID", projectID)
	return nil
}
