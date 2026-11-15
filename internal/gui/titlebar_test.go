package gui

import (
	"image/color"
	"testing"

	"github.com/egoist/mygo/ui"

	"minipost/internal/model"
)

// 取标题栏正中偏左一点的空白处像素，应接近 chrome 底色（流光方案：标题栏透明，露出极光）。
func titleBarBlank() (x, y int) { return 600, 6 }

// nearToolbar 报告标题栏像素是否接近 chrome 底色：标题栏透明，上面铺着极光，所以允许一定色差。
func nearToolbar(got color.RGBA, want ui.Color) bool {
	d := func(a, b uint8) int {
		if a > b {
			return int(a - b)
		}
		return int(b - a)
	}
	return d(got.R, want.R) <= 48 && d(got.G, want.G) <= 48 && d(got.B, want.B) <= 48
}

// 设计稿令牌：雾蓝亮色/暗色的关键色值与设计稿一致。
func TestSchemePaletteMatchesDesignTokens(t *testing.T) {
	l := schemePalette("mist", false)
	if rgb(l.accent) != [3]uint8{0x4f, 0x52, 0xd9} {
		t.Fatalf("mist-l 强调色应为 #4f52d9，实际 %v", l.accent)
	}
	if rgb(l.toolbar) != [3]uint8{0xf1, 0xf2, 0xf6} || rgb(l.sidebar) != [3]uint8{0xf5, 0xf6, 0xf9} {
		t.Fatalf("mist-l 的 chrome/side 与设计稿不符: %v %v", l.toolbar, l.sidebar)
	}
	d := schemePalette("mist", true)
	if !d.dark || rgb(d.accent) != [3]uint8{0x8c, 0x8f, 0xff} || rgb(d.surface) != [3]uint8{0x1f, 0x20, 0x27} {
		t.Fatalf("mist-d 与设计稿不符: accent=%v surface=%v", d.accent, d.surface)
	}
	ink := schemePalette("ink", false)
	if rgb(ink.accent) != [3]uint8{0x17, 0x17, 0x1a} {
		t.Fatalf("ink-l 强调色应为 #17171a，实际 %v", ink.accent)
	}
	jade := schemePalette("jade", true)
	if rgb(jade.accent) != [3]uint8{0x3f, 0xd0, 0xb8} {
		t.Fatalf("jade-d 强调色应为 #3fd0b8，实际 %v", jade.accent)
	}
	// 请求方法色按设计稿：POST 橙、PUT 蓝；暗色用提亮的同色系。
	if rgb(l.post) != [3]uint8{0xb8, 0x52, 0x0a} || rgb(l.put) != [3]uint8{0x22, 0x60, 0xc4} {
		t.Fatalf("方法色与设计稿不符: post=%v put=%v", l.post, l.put)
	}
}

// 三套配色 × 明暗共 6 套调色板互不相同；未知方案回落为雾蓝。
func TestSchemePalettesDistinct(t *testing.T) {
	seen := map[[3]uint8]string{}
	for _, name := range colorSchemeNames {
		for _, dark := range []bool{false, true} {
			p := schemePalette(name, dark)
			key := rgb(p.toolbar)
			if prev, dup := seen[key]; dup {
				t.Fatalf("%s(dark=%v) 与 %s 的标题栏底色相同", name, dark, prev)
			}
			seen[key] = name
			if p.dark != dark {
				t.Fatalf("%s 的 dark 标记应为 %v", name, dark)
			}
		}
	}
	if rgb(schemePalette("nope", false).accent) != rgb(schemePalette("mist", false).accent) {
		t.Fatal("未知配色方案应回落为雾蓝")
	}
}

// 标题栏按设计稿（流光）绘制：透明背景透出极光，高度为 sizeToolbar，侧栏开关/后退/前进/主页/项目/环境/主题/设置齐全。
func TestTitleBarControls(t *testing.T) {
	a, tt, _ := zhApp(t)
	for _, label := range []string{
		"收起侧边栏 (⌘B)", "后退", "前进", "打开主页", "切换项目", "切换环境",
		"切换为深色主题", "设置 (⌘,)",
	} {
		r, ok := tt.Find(label)
		if !ok {
			t.Fatalf("标题栏缺少按钮 %q，实际文本 %q", label, tt.Texts())
		}
		if r.Y+r.H > sizeToolbar {
			t.Fatalf("按钮 %q 应在标题栏内，Y=%.0f H=%.0f", label, r.Y, r.H)
		}
	}
	x, y := titleBarBlank()
	if got := tt.Image().RGBAAt(x, y); !nearToolbar(got, a.p.toolbar) {
		t.Fatalf("标题栏背景应是透出极光的 chrome 色 %v，实际 %v", a.p.toolbar, got)
	}
}

// 侧栏开关：点击后收起/展开侧栏，提示文字随之切换。
func TestTitleBarSidebarToggle(t *testing.T) {
	a, tt, _ := zhApp(t)
	if a.set.SidebarCollapsed {
		t.Fatal("默认应展开侧栏")
	}
	if err := tt.Click("收起侧边栏 (⌘B)"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.set.SidebarCollapsed {
		t.Fatal("点击侧栏开关后应收起侧栏")
	}
	if err := tt.Click("展开侧边栏 (⌘B)"); err != nil {
		t.Fatalf("收起后提示应变为“展开侧边栏”: %v", err)
	}
	tt.Frame()
	if a.set.SidebarCollapsed {
		t.Fatal("再次点击应展开侧栏")
	}
}

// 环境胶囊在标题栏：无环境时显示“无环境”，点开列出环境，选择后更新并持久化；标签栏里不再有环境选择器。
func TestTitleBarEnvPill(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.envs = []model.Environment{{ID: "e1", Name: "预发环境"}, {ID: "e2", Name: "生产环境"}}
	a.envID = ""
	tt.Frame()
	r, ok := tt.Find("切换环境")
	if !ok || r.Y >= sizeToolbar {
		t.Fatalf("环境胶囊应位于标题栏（Y<%d），实际 %+v ok=%v", sizeToolbar, r, ok)
	}
	if err := tt.Click("切换环境"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	menu := tt.Menu()
	for _, want := range []string{"无环境", "预发环境", "生产环境", "管理环境…"} {
		found := false
		for _, m := range menu {
			if m == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("环境菜单缺少 %q，实际 %q", want, menu)
		}
	}
	if err := tt.ChooseMenuItem("生产环境"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.envID != "e2" {
		t.Fatalf("选择后当前环境应为 e2，实际 %q", a.envID)
	}
	if !tt.HasText("生产环境") {
		t.Fatalf("胶囊应显示新环境名: %q", tt.Texts())
	}
}

// 环境胶囊的圆点（在名称右侧）：有环境为成功色（绿），无环境为粗边框灰（--bd2）。
func TestTitleBarEnvPillDotColor(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.envs = []model.Environment{{ID: "e1", Name: "预发环境"}}
	for _, tc := range []struct {
		id   string
		want [3]uint8
	}{{"", rgb(a.p.btnBorder)}, {"e1", rgb(a.p.success)}} {
		a.envID = tc.id
		tt.Frame()
		name, ok := tt.Find(a.envLabel())
		if !ok || name.Y >= sizeToolbar {
			t.Fatalf("找不到标题栏里的环境名 %q", a.envLabel())
		}
		// 圆点在名称右侧：间距 6 + 半径 4
		px := tt.Image().RGBAAt(int(name.X+name.W)+6+4, int(name.Y+name.H/2))
		if got := [3]uint8{px.R, px.G, px.B}; got != tc.want {
			t.Fatalf("环境 %q 的圆点色应为 %v，实际 %v", tc.id, tc.want, got)
		}
	}
}

// 标签栏不再放环境选择器：打开请求标签后，标题栏之下不应再出现“切换环境”按钮。
func TestTabBarHasNoEnvSelector(t *testing.T) {
	a, tt, _ := visualApp(t, false)
	_ = tt.Click("用户服务")
	_ = tt.Click("获取用户列表")
	tt.Frame()
	if len(a.tabs) == 0 {
		t.Fatal("应已打开请求标签")
	}
	if r, ok := tt.Find("切换环境"); !ok || r.Y >= sizeToolbar {
		t.Fatalf("环境选择只应在标题栏: %+v ok=%v", r, ok)
	}
}

// 项目选择器左侧的圆点取项目主题色。
func TestTitleBarProjectDotUsesThemeColor(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.setProjectTheme(a.projectID, "#ff3366")
	tt.Frame()
	name, ok := tt.Find("Demo")
	if !ok || name.Y >= sizeToolbar {
		t.Fatalf("找不到标题栏里的项目名: %+v", name)
	}
	// 圆点在名称左侧：间距 7 + 半径 4
	px := tt.Image().RGBAAt(int(name.X)-7-4, int(name.Y+name.H/2))
	if px.R < 0xe0 || px.G > 0x60 || px.B < 0x40 {
		t.Fatalf("项目圆点应为项目主题色 #ff3366，实际 %v", px)
	}
}

// 明暗切换：点击后保存主题，调色板随之变为暗色；图标提示切换为“浅色”。
func TestTitleBarThemeToggle(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.set.Theme = "light"
	tt.Frame()
	if err := tt.Click("切换为深色主题"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.set.Theme != "dark" || !a.p.dark {
		t.Fatalf("应切换为深色: theme=%q dark=%v", a.set.Theme, a.p.dark)
	}
	if err := tt.Click("切换为浅色主题"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.set.Theme != "light" || a.p.dark {
		t.Fatalf("应切回浅色: theme=%q dark=%v", a.set.Theme, a.p.dark)
	}
}

// 后退/前进在没有历史时不可用；点击不应改变状态。
func TestTitleBarNavDisabledWithoutHistory(t *testing.T) {
	a, tt, _ := zhApp(t)
	if a.canNav(-1) || a.canNav(1) {
		t.Fatal("刚启动时不应有可后退/前进的历史")
	}
	_ = tt.Click("后退")
	_ = tt.Click("前进")
	tt.Frame()
	if a.showHome {
		t.Fatal("禁用的后退/前进不应有任何效果")
	}
}

// 主页按钮：点击进入主页，再点击回到项目。
func TestTitleBarHomeToggle(t *testing.T) {
	a, tt, _ := zhApp(t)
	if err := tt.Click("打开主页"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.showHome {
		t.Fatal("点击主页按钮应显示项目主页")
	}
	if err := tt.Click("关闭主页"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.showHome {
		t.Fatal("再次点击应关闭项目主页")
	}
}

// 切换配色方案：调色板、标题栏实际像素与设置同步变化；明暗各自适配。
func TestColorSchemeAppliesToTitleBar(t *testing.T) {
	a, tt, _ := zhApp(t)
	for _, scheme := range colorSchemeNames {
		for _, theme := range []string{"light", "dark"} {
			a.set.ColorScheme, a.set.Theme = scheme, theme
			tt.Frame()
			want := schemePalette(scheme, theme == "dark").toolbar
			if a.p.toolbar != want {
				t.Fatalf("%s/%s 调色板未生效: %v != %v", scheme, theme, a.p.toolbar, want)
			}
			x, y := titleBarBlank()
			got := tt.Image().RGBAAt(x, y)
			if !nearToolbar(got, want) {
				t.Fatalf("%s/%s 标题栏像素应接近 %v（极光会在其上轻微着色），实际 %v", scheme, theme, want, got)
			}
		}
	}
}

// 设置页「外观」里选择配色方案后立即生效并保存到设置。
func TestPrefsColorSchemePicker(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openPrefs()
	tt.Frame()
	if err := tt.Click("外观"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("配色") {
		t.Fatalf("外观页应显示配色方案卡片，实际 %q", tt.Texts())
	}
	if err := tt.Click("松石"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.set.ColorScheme != "jade" {
		t.Fatalf("选择松石后配色应为 jade，实际 %q", a.set.ColorScheme)
	}
	if rgb(a.p.accent) == rgb(schemePalette("mist", a.p.dark).accent) {
		t.Fatal("配色切换后强调色应变化")
	}
}
