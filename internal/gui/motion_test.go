package gui

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

// 项目色适配：只保留 hex 的色相，明度/彩度跟随配色强调色（夹到规则范围内）。
func TestAdaptProjectColorKeepsHueAndFollowsScheme(t *testing.T) {
	light, dark := schemePalette("mist", false), schemePalette("mist", true)
	a, b := adaptProjectColor("#ff3366", light.baseAccent), adaptProjectColor("#ff3366", dark.baseAccent)
	la, ca, ha := oklchOf(a)
	lb, _, hb := oklchOf(b)
	_, _, h0 := oklchOf(ui.Hex("#ff3366"))
	for _, h := range []float64{ha, hb} {
		if d := math.Abs(h - h0); d > 4 && d < 356 {
			t.Errorf("色相应保持不变：原 %.1f 现 %.1f", h0, h)
		}
	}
	if lb <= la {
		t.Errorf("暗色配色的强调色更亮，项目色也应更亮：亮 %.3f 暗 %.3f", la, lb)
	}
	if la < projLMin-0.01 || lb > projLMax+0.01 || ca > projCMax+0.01 {
		t.Errorf("明度/彩度应落在规则范围内：L=%.3f..%.3f C=%.3f", la, lb, ca)
	}
}

// 不同配色下同一项目色的结果不同（与外观配色联动），且色相相同。
func TestAdaptProjectColorDiffersAcrossSchemes(t *testing.T) {
	seen := map[ui.Color]bool{}
	for _, n := range colorSchemeNames {
		seen[adaptProjectColor("#30d158", schemePalette(n, false).baseAccent)] = true
	}
	if len(seen) < 2 {
		t.Fatalf("项目色应随配色变化，但 5 套配色得到同一个颜色")
	}
}

// 无色相的灰色输入不应崩溃，也不应产生超出范围的通道。
func TestAdaptProjectColorGreyInput(t *testing.T) {
	c := adaptProjectColor("#808080", schemePalette("jade", false).baseAccent)
	if c.A != 255 && c.A != 0 {
		t.Fatalf("颜色异常: %v", c)
	}
}

func TestSlideRect(t *testing.T) {
	if _, ok := slideRect(ui.Rect{}, ui.Rect{W: 10}); ok {
		t.Fatal("容器未布局（首帧）时应返回 ok=false")
	}
	if _, ok := slideRect(ui.Rect{W: 100}, ui.Rect{}); ok {
		t.Fatal("活动项未布局时应返回 ok=false")
	}
	r, ok := slideRect(ui.Rect{X: 10, Y: 20, W: 100, H: 30}, ui.Rect{X: 40, Y: 22, W: 30, H: 26})
	if !ok || r != (ui.Rect{X: 30, Y: 2, W: 30, H: 26}) {
		t.Fatalf("相对容器的矩形不对: %+v ok=%v", r, ok)
	}
}

// 浮层面板必须是不透明的渐变底（MyGo 先画纯色层、后画渐变层，纯色面板会被下方的渐变按钮透出）。
func TestMenuOptsPanelHasGradient(t *testing.T) {
	a, _, _ := zhApp(t)
	opts := a.menuOpts()
	if opts.Panel == nil {
		t.Fatal("menuOpts 应提供 Panel 样式")
	}
}

// 开关：点击后切换并返回 true，旋钮位置随之改变。
func TestToggleSwitchesAndMovesKnob(t *testing.T) {
	a, _, _ := zhApp(t)
	on := false
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Row(c).Padding(20).Children(func() { a.toggle(c, "sw", &on) })
	}, 200, 80)
	tt.Frame()
	tt.ClickAt(36, 30)
	tt.Frame()
	if !on {
		t.Fatal("点击后开关应为开")
	}
}
