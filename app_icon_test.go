package main

import (
	"image/png"
	"os"
	"testing"
)

// 应用图标必须是 1024x1024 的 PNG：四角透明（圆角方块）、中心不透明，
// 并且不是纯色图（防止误提交空白或占位图标）。
func TestAppIconAsset(t *testing.T) {
	f, err := os.Open("resources/icon.png")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatalf("图标不是合法 PNG: %v", err)
	}
	if b := img.Bounds(); b.Dx() != 1024 || b.Dy() != 1024 {
		t.Fatalf("图标应为 1024x1024，实际 %dx%d", b.Dx(), b.Dy())
	}
	alpha := func(x, y int) uint32 { _, _, _, a := img.At(x, y).RGBA(); return a >> 8 }
	for _, p := range [][2]int{{0, 0}, {1023, 0}, {0, 1023}, {1023, 1023}} {
		if a := alpha(p[0], p[1]); a != 0 {
			t.Errorf("四角应透明，(%d,%d) alpha=%d", p[0], p[1], a)
		}
	}
	if a := alpha(512, 512); a != 255 {
		t.Errorf("中心应不透明，alpha=%d", a)
	}
	// 取样多个点的颜色，至少应有明显差异（渐变底 + 白色邮票）
	r1, g1, b1, _ := img.At(160, 200).RGBA()
	r2, g2, b2, _ := img.At(400, 600).RGBA()
	if r1 == r2 && g1 == g2 && b1 == b2 {
		t.Error("图标看起来是纯色图")
	}
}
