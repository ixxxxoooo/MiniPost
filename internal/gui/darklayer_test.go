package gui

import (
	"image/color"
	"testing"

	"github.com/egoist/mygo/ui"
)

func lum(c color.RGBA) int { return int(c.R) + int(c.G) + int(c.B) }

func lumColor(c ui.Color) int { return int(c.R) + int(c.G) + int(c.B) }

// 深色调色板的层次（设计稿）：标题栏/标签栏/状态栏（chrome）最深 < 侧栏 < 内容区；选中标签与内容区同色。
func TestDarkPaletteLayers(t *testing.T) {
	for _, scheme := range colorSchemeNames {
		p := schemePalette(scheme, true)
		if p.tabStrip == p.sidebar || p.sidebar == p.surface || p.tabStrip == p.surface {
			t.Fatalf("[%s] 标签栏、侧栏、内容区应是三种不同的底色: tab=%v side=%v surface=%v", scheme, p.tabStrip, p.sidebar, p.surface)
		}
		if p.toolbar != p.tabStrip || p.bottom != p.tabStrip {
			t.Fatalf("[%s] 标题栏、标签栏和状态栏应同属 chrome 层", scheme)
		}
		if lumColor(p.tabStrip) >= lumColor(p.sidebar) || lumColor(p.sidebar) >= lumColor(p.surface) {
			t.Fatalf("[%s] 暗色亮度应为 chrome < 侧栏 < 内容区", scheme)
		}
		if p.tabActive != p.surface {
			t.Fatalf("[%s] 选中标签应与内容区同色，才能连成一体", scheme)
		}
		// 分隔线/边框在外围背景上要看得见
		if p.border == p.sidebar || p.border == p.tabStrip {
			t.Fatalf("[%s] 边框色不应与外围背景相同（会隐形）: %v", scheme, p.border)
		}
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
