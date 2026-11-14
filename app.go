package main

import (
	"context"
	"path/filepath"

	"minipost/internal/pkg/logger"
	"minipost/internal/repository"
	"minipost/internal/service"
)

// App 后端门面：聚合存储与各业务服务，供原生 UI 直接调用。
// 导出方法按职责拆分在 app_http.go、app_collection.go、app_import*.go、app_dialog.go、backup.go 中。
type App struct {
	// ctx 为应用生命周期上下文，用于远程导入等需要上下文的调用
	ctx        context.Context
	cancel     context.CancelFunc
	httpSvc    *service.HttpService
	projectSvc *service.ProjectService
	requestSvc *service.RequestService
	envSvc     *service.EnvironmentService
	historySvc *service.HistoryService
	store      *repository.FileStore
}

// NewApp 初始化存储与各服务，存储初始化失败时直接终止进程。
func NewApp() *App {
	logger.Info("正在初始化存储...")
	store, err := repository.NewFileStore()
	if err != nil {
		logger.Fatal("初始化存储失败", "error", err.Error())
	}
	logger.Info("存储初始化完成", "baseDir", store.BaseDir())

	logger.Info("正在初始化各服务...")
	ctx, cancel := context.WithCancel(context.Background())
	app := &App{
		ctx:        ctx,
		cancel:     cancel,
		httpSvc:    service.NewHttpService(),
		projectSvc: service.NewProjectService(store),
		requestSvc: service.NewRequestService(store),
		envSvc:     service.NewEnvironmentService(store),
		historySvc: service.NewHistoryService(store),
		store:      store,
	}
	app.httpSvc.SetKeyLogPath(filepath.Join(store.BaseDir(), "logs", "sslkeys.log"))
	logger.Info("所有服务初始化完成")

	return app
}

// BaseDir 返回数据目录（默认 ~/.minipost）。
func (a *App) BaseDir() string {
	return a.store.BaseDir()
}

// Shutdown 在应用退出时调用，取消仍在进行的后台操作。
func (a *App) Shutdown() {
	a.cancel()
	logger.Info("应用正在关闭")
}
