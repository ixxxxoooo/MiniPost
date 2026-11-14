package gui

import (
	"github.com/egoist/mygo/ui"

	"minipost/internal/client"
)

// headersTab 绘制 Headers 子标签：可编辑的键值表；系统自动补充的 Header 作为只读行并入同一张表，
// 默认隐藏，可通过表头的“显示 N”按钮展开并逐项关闭。
func (a *app) headersTab(c *ui.Context, t *tab) {
	d := t.draft
	var visible, markers []*client.Row
	for _, h := range d.Headers {
		if client.IsAutoHeaderMarker(h.Key) {
			markers = append(markers, h)
		} else {
			visible = append(visible, h)
		}
	}
	a.kvTableOpts(c, &visible, kvOpts{
		id:    d.ID + ":headers",
		keyPH: a.t("Header 名", "Header Name"),
		valPH: a.t("Header 值", "Header Value"),
		t:     t,
	})
	d.Headers = append(visible, markers...)
}
