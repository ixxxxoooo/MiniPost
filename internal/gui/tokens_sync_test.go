package gui

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// 本文件把设计稿 docs/design/assets/tokens.css 当作唯一事实来源：
// 配色、语义色、动效时长、字号、间距与 Go 里的令牌必须逐项一致，改任何一边都要同步另一边。

// readTokensCSS 读取设计稿令牌文件。
func readTokensCSS(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("../../docs/design/assets/tokens.css")
	if err != nil {
		t.Fatalf("读取设计稿令牌失败: %v", err)
	}
	return string(b)
}

// cssBlock 返回选择器对应的 { … } 内容。
func cssBlock(t *testing.T, css, selector string) string {
	t.Helper()
	i := strings.Index(css, "\n"+selector+" {")
	if i < 0 {
		t.Fatalf("设计稿里找不到选择器 %s", selector)
	}
	j := strings.Index(css[i:], "}")
	return css[i : i+j]
}

// cssVar 取块内的变量值（形如 --bg: #fff;）。
func cssVar(t *testing.T, block, name string) string {
	t.Helper()
	m := regexp.MustCompile(`(?:^|[\s;{])` + regexp.QuoteMeta(name) + `:\s*([^;]+);`).FindStringSubmatch(block)
	if m == nil {
		t.Fatalf("块里找不到变量 %s", name)
	}
	return strings.TrimSpace(m[1])
}

func TestTokensSchemeColorsMatchDesign(t *testing.T) {
	css := readTokensCSS(t)
	for _, name := range colorSchemeNames {
		for i, mode := range []string{"light", "dark"} {
			blk := cssBlock(t, css, `[data-scheme="`+name+`"][data-mode="`+mode+`"]`)
			got := colorSchemes[name][i]
			pairs := map[string]string{
				"--chrome": got.chrome, "--side": got.side, "--bg": got.bg, "--bg2": got.bg2, "--bd": got.bd, "--bd2": got.bd2,
				"--fg": got.fg, "--fg2": got.fg2, "--mu": got.mu, "--ac": got.ac, "--ach": got.ach, "--acf": got.acf,
			}
			for k, v := range pairs {
				if want := cssVar(t, blk, k); !strings.EqualFold(want, v) {
					t.Errorf("%s/%s 的 %s：设计稿 %s，代码 %s", name, mode, k, want, v)
				}
			}
		}
	}
}

func TestTokensSemanticColorsMatchDesign(t *testing.T) {
	css := readTokensCSS(t)
	for i, mode := range []string{"light", "dark"} {
		blk := cssBlock(t, css, `[data-mode="`+mode+`"]`)
		s := []semanticTokens{semanticLight, semanticDark}[i]
		pairs := map[string]string{
			"--get": s.get, "--post": s.post, "--put": s.put, "--patch": s.pat, "--del": s.del, "--head": s.head, "--opt": s.opt,
			"--ok": s.ok, "--warn": s.warn, "--err": s.err,
			"--j-key": s.k, "--j-str": s.s, "--j-num": s.n, "--j-bool": s.b, "--j-punc": s.p,
		}
		for k, v := range pairs {
			if want := cssVar(t, blk, k); !strings.EqualFold(want, v) {
				t.Errorf("%s 的 %s：设计稿 %s，代码 %s", mode, k, want, v)
			}
		}
	}
}

func TestTokensMotionMatchDesign(t *testing.T) {
	root := cssBlock(t, readTokensCSS(t), ":root")
	durs := map[string]time.Duration{"--dur-fast": durFast, "--dur-base": durBase, "--dur-slow": durSlow, "--dur-xl": durXL}
	for k, v := range durs {
		ms, err := strconv.Atoi(strings.TrimSuffix(cssVar(t, root, k), "ms"))
		if err != nil || time.Duration(ms)*time.Millisecond != v {
			t.Errorf("%s：设计稿 %s，代码 %v", k, cssVar(t, root, k), v)
		}
	}
	// 曲线：用控制点的采样值比对（CubicBezier 是函数，不能直接比较）
	eases := map[string]func(float32) float32{
		"--ease-out": easeOut, "--ease-emph": easeEmph, "--ease-spring": easeSpring, "--ease-soft": easeSoft,
	}
	for k, f := range eases {
		m := regexp.MustCompile(`cubic-bezier\(([^)]+)\)`).FindStringSubmatch(cssVar(t, root, k))
		if m == nil {
			t.Fatalf("%s 不是 cubic-bezier", k)
		}
		var n [4]float64
		for i, s := range strings.Split(m[1], ",") {
			n[i], _ = strconv.ParseFloat(strings.TrimSpace(s), 64)
		}
		ref := ui.CubicBezier(float32(n[0]), float32(n[1]), float32(n[2]), float32(n[3]))
		for _, x := range []float32{0.1, 0.3, 0.5, 0.8} {
			if d := f(x) - ref(x); d > 0.002 || d < -0.002 {
				t.Errorf("%s 在 %.1f 处不一致：代码 %.4f 设计稿 %.4f", k, x, f(x), ref(x))
			}
		}
	}
}

func TestTokensFontSizeMatchDesign(t *testing.T) {
	root := cssBlock(t, readTokensCSS(t), ":root")
	sizes := map[string]float32{"--fs-xs": fsTiny, "--fs-sm": fsSmall, "--fs-base": fsBase, "--fs-md": fsLarge}
	for k, v := range sizes {
		n, err := strconv.Atoi(strings.TrimSuffix(cssVar(t, root, k), "px"))
		if err != nil || float32(n) != v {
			t.Errorf("%s：设计稿 %s，代码 %v", k, cssVar(t, root, k), v)
		}
	}
}
