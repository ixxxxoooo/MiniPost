package gui

import (
	"image"
	"testing"
)

// bluePixels 数出矩形内偏蓝（强调色及其半透明混合）的像素个数。
func bluePixels(img *image.RGBA, x0, y0, x1, y1 int) int {
	n := 0
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			c := img.RGBAAt(x, y)
			if int(c.B)-int(c.R) > 50 {
				n++
			}
		}
	}
	return n
}

// Params 末尾的空占位行：复选框是空心不可点的，不能画成发灰的“已勾选”框（旧版像被禁用的勾选项）。
// 在占位行里输入后它变成真实行，复选框以强调色勾选态显示。
func TestParamsPlaceholderCheckboxNotChecked(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	tt.Frame()
	hdr, ok := tt.Find("参数名")
	if !ok {
		t.Fatal("找不到 Params 表头")
	}
	// 复选框列在“参数名”列左侧；扫描第一行（表头下面）该列的区域。
	x0, x1 := int(hdr.X)-40, int(hdr.X)-2
	y0, y1 := int(hdr.Y)+14, int(hdr.Y)+50
	if got := bluePixels(tt.Image(), x0, y0, x1, y1); got != 0 {
		t.Fatalf("占位行复选框不应出现蓝色勾选底（%d 个蓝色像素）", got)
	}

	// 在占位行的参数名里输入后变成真实行，复选框为勾选态（蓝色）。
	tb := a.activeTab()
	tb.draft.Params[len(tb.draft.Params)-1].Key = "q"
	tt.Frame()
	if got := bluePixels(tt.Image(), x0, y0, x1, y1); got == 0 {
		t.Fatal("真实行的复选框应是强调色勾选态")
	}
}
