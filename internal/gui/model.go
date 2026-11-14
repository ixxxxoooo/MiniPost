package gui

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// model 保存界面状态，仅在主线程访问；后台任务通过 win.Update 回到主线程。
type model struct {
	be  Backend
	win *mygo.Window
}

func newModel(be Backend) *model { return &model{be: be} }

func (m *model) view(c *ui.Context) {
	ui.Column(c).Fill().Center().Children(func() {
		ui.Text(c, "MiniPost").FontSize(28).Bold()
	})
}
