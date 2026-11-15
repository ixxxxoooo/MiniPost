package gui

// 测试默认关闭面板切换动画：AfterExits 会让新面板在旧面板退场结束前不可见，
// 而绝大多数用例切换后要立刻读取新面板内容。motion_test.go 里的专用用例会自行打开它。
func init() { motionEnabled = false }
