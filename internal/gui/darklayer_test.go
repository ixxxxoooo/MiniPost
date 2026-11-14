package gui

import (
	"image/color"
	"testing"
)

func lum(c color.RGBA) int { return int(c.R) + int(c.G) + int(c.B) }

// 深色调色板的层次：标签栏最深 < 侧栏/标题栏/状态栏 < 内容区；选中标签与内容区同色。
func TestDarkPaletteLayers(t *testing.T) {
	p := darkPalette()
	if p.tabStrip == p.sidebar || p.sidebar == p.surface || p.tabStrip == p.surface {
		t.Fatalf("标签栏、侧栏、内容区应是三种不同的灰度: tab=%v side=%v surface=%v", p.tabStrip, p.sidebar, p.surface)
	}
	if p.toolbar != p.sidebar || p.bottom != p.sidebar {
		t.Fatalf("标题栏和状态栏应与侧栏同层")
	}
	if p.tabActive != p.surface {
		t.Fatalf("选中标签应与内容区同色，才能连成一体")
	}
	// 分隔线/边框在比内容区更深的外围背景上要看得见
	if p.border == p.sidebar || p.border == p.tabStrip {
		t.Fatalf("边框色不应与外围背景相同（会隐形）: %v", p.border)
	}
}

// 实际渲染：深色下标签栏、侧栏、内容区三处的像素亮度依次升高。
func TestDarkModeRenderedLayers(t *testing.T) {
	_, tt, _ := visualApp(t, true)
	_ = tt.Click("健康检查")
	tt.Frame()
	img := tt.Image()
	side := img.RGBAAt(20, 700)
	content := img.RGBAAt(900, 420)
	strip := img.RGBAAt(1100, int(sizeToolbar)+int(sizeTabStrip)/2+2)
	if !(lum(strip) < lum(side) && lum(side) < lum(content)) {
		t.Fatalf("亮度应为 标签栏(%v) < 侧栏(%v) < 内容区(%v)", strip, side, content)
	}
}
