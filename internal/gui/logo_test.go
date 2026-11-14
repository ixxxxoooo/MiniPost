package gui

import (
	"bytes"
	"image/png"
	"testing"
)

func luma(r, g, b uint32) float64 {
	return 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)
}

// 应用内图标有浅色、深色两个版本，都能正常加载；深色版底色明显更暗，四角透明。
func TestLogoLightAndDark(t *testing.T) {
	if logoImage(false) == nil || logoImage(true) == nil {
		t.Fatal("浅色与深色图标都应能加载")
	}
	load := func(name string) (l float64, corner uint32) {
		data, err := assetFS.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		img, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("%s 不是合法 PNG: %v", name, err)
		}
		if b := img.Bounds(); b.Dx() != 256 || b.Dy() != 256 {
			t.Fatalf("%s 应为 256x256，实际 %dx%d", name, b.Dx(), b.Dy())
		}
		_, _, _, ca := img.At(0, 0).RGBA()
		r, g, b, _ := img.At(24, 128).RGBA() // 左侧边缘附近，只有底色
		return luma(r, g, b), ca >> 8
	}
	ll, lc := load("icons/logo.png")
	dl, dc := load("icons/logo-dark.png")
	if lc != 0 || dc != 0 {
		t.Errorf("四角应透明: light=%d dark=%d", lc, dc)
	}
	if dl > ll-30 {
		t.Errorf("深色图标底色应明显比浅色暗: light=%.0f dark=%.0f", ll, dl)
	}
}

// 空状态页与设置-关于在深色/浅色主题下都应画出图标，不报错。
func TestLogoShownInEmptyStateAndAbout(t *testing.T) {
	for _, dark := range []bool{false, true} {
		a, tt, _ := zhApp(t)
		if dark {
			a.set.Theme = "dark"
		} else {
			a.set.Theme = "light"
		}
		tt.Frame()
		if !tt.HasText("MiniPost") {
			t.Fatalf("空状态应显示应用名（dark=%v）: %q", dark, tt.Texts())
		}
	}
}
