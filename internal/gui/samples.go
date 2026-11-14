package gui

import (
	"fmt"

	"minipost/internal/pkg/logger"
)

// installSamples 创建内置的测试示例项目；新建后切换到第一个新项目，已存在时只提示。
func (a *app) installSamples() {
	a.async(func() func() {
		created, err := a.be.InstallSampleProjects()
		return func() {
			a.fail(a.t("添加示例项目", "Add sample projects"), err)
			a.loadProjects()
			if len(created) == 0 {
				if err == nil {
					a.toast(a.t("示例项目已存在", "Sample projects already exist"))
				}
				return
			}
			logger.Info("已添加示例项目", "count", len(created))
			a.toast(a.t(fmt.Sprintf("已添加 %d 个示例项目", len(created)), fmt.Sprintf("Added %d sample projects", len(created))))
			for _, p := range a.projects {
				if p.Name == created[0] {
					a.showHome = false
					a.selectProject(p.ID)
					return
				}
			}
		}
	})
}
