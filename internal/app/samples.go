package app

import (
	"minipost/internal/pkg/logger"
	"minipost/internal/service"
)

// InstallSampleProjects 创建内置的测试示例项目（已存在的同名项目会跳过），返回本次新建的项目名。
func (a *App) InstallSampleProjects() ([]string, error) {
	created, err := service.SampleInstaller{Projects: a.projectSvc, Requests: a.requestSvc, Envs: a.envSvc}.InstallSamples()
	if err != nil {
		logger.Error("安装示例项目失败", "error", err.Error())
		return created, err
	}
	logger.Info("安装示例项目完成", "created", len(created))
	return created, nil
}
