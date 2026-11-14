package gui

// Backend 是界面依赖的后端能力，由 main 包的 App 实现，测试中可用假对象替换。
type Backend interface {
	BaseDir() string
}
