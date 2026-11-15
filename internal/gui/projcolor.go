package gui

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// 项目主题色适配：项目里存的是色相基准（hex），显示时只取它的色相，再套用当前配色强调色的
// 明度与彩度（OKLCH），所以无论换成雾蓝、松石、琥珀还是玫瑰，项目色块都和外观配色协调；
// 暗色模式下强调色更亮，项目色也随之变亮。规则与设计稿 app.js 的 pjc() 完全一致：
//
//	oklch(from var(--ac-base) clamp(.56, l, .8) clamp(.11, c, .17) <色相>)
const (
	projLMin, projLMax = 0.56, 0.80
	projCMin, projCMax = 0.11, 0.17
)

// srgbToLinear / linearToSrgb 是 sRGB 与线性光之间的转换。
func srgbToLinear(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

func linearToSrgb(v float64) float64 {
	if v <= 0.0031308 {
		return 12.92 * v
	}
	return 1.055*math.Pow(v, 1/2.4) - 0.055
}

// oklchOf 把颜色转成 OKLCH（L 0~1，C，H 度数 0~360）。
func oklchOf(col ui.Color) (l, c, h float64) {
	r, g, b := srgbToLinear(float64(col.R)/255), srgbToLinear(float64(col.G)/255), srgbToLinear(float64(col.B)/255)
	lc := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*b)
	mc := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*b)
	sc := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*b)
	l = 0.2104542553*lc + 0.7936177850*mc - 0.0040720468*sc
	a := 1.9779984951*lc - 2.4285922050*mc + 0.4505937099*sc
	bb := 0.0259040371*lc + 0.7827717662*mc - 0.8086757660*sc
	c = math.Hypot(a, bb)
	h = math.Mod(math.Atan2(bb, a)*180/math.Pi+360, 360)
	return
}

// oklchRGB 把 OKLCH 转回 sRGB；ok 表示在色域内（不在时 r,g,b 仍是裁剪后的值）。
func oklchRGB(l, c, h float64) (r, g, b float64, ok bool) {
	a, bb := c*math.Cos(h*math.Pi/180), c*math.Sin(h*math.Pi/180)
	lc := l + 0.3963377774*a + 0.2158037573*bb
	mc := l - 0.1055613458*a - 0.0638541728*bb
	sc := l - 0.0894841775*a - 1.2914855480*bb
	lc, mc, sc = lc*lc*lc, mc*mc*mc, sc*sc*sc
	rl := 4.0767416621*lc - 3.3077115913*mc + 0.2309699292*sc
	gl := -1.2684380046*lc + 2.6097574011*mc - 0.3413193965*sc
	bl := -0.0041960863*lc - 0.7034186147*mc + 1.7076147010*sc
	const eps = 1e-4
	ok = rl >= -eps && rl <= 1+eps && gl >= -eps && gl <= 1+eps && bl >= -eps && bl <= 1+eps
	clip := func(v float64) float64 { return math.Min(math.Max(linearToSrgb(math.Min(math.Max(v, 0), 1)), 0), 1) }
	return clip(rl), clip(gl), clip(bl), ok
}

// oklchColor 生成颜色；超出 sRGB 色域时逐步降低彩度（与 CSS 的色域映射一致：保持明度与色相）。
func oklchColor(l, c, h float64) ui.Color {
	lo, hi := 0.0, c
	if _, _, _, ok := oklchRGB(l, c, h); !ok {
		for i := 0; i < 24; i++ {
			mid := (lo + hi) / 2
			if _, _, _, ok := oklchRGB(l, mid, h); ok {
				lo = mid
			} else {
				hi = mid
			}
		}
		c = lo
	}
	r, g, b, _ := oklchRGB(l, c, h)
	return ui.RGB(uint8(math.Round(r*255)), uint8(math.Round(g*255)), uint8(math.Round(b*255)))
}

// adaptProjectColor 把项目存的 hex 适配到当前配色：取 hex 的色相，明度/彩度取自配色强调色 base 并夹到合理范围。
func adaptProjectColor(hex string, base ui.Color) ui.Color {
	_, _, h := oklchOf(ui.Hex(hex))
	bl, bc, _ := oklchOf(base)
	return oklchColor(math.Min(math.Max(bl, projLMin), projLMax), math.Min(math.Max(bc, projCMin), projCMax), h)
}
