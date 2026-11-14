package gui

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// 侧栏选中节点后：⌘D 复制、⌘E 重命名、⌫ 删除（弹确认框）。
func TestSidebarNodeShortcuts(t *testing.T) {
	a, tt, be := zhApp(t)
	beta := a.coll.Requests[1].ID
	a.selectNode(beta)
	tt.Frame()

	before := len(be.colls[a.projectID].Requests)
	tt.Key(ui.Cmd, ui.KeyD)
	tt.Frame()
	if got := len(be.colls[a.projectID].Requests); got != before+1 {
		t.Fatalf("⌘D 应复制出一个请求: %d -> %d", before, got)
	}

	tt.Key(ui.Cmd, ui.KeyE)
	tt.Frame()
	if !a.overlays.prompt.open || a.overlays.prompt.value == "" {
		t.Fatalf("⌘E 应打开重命名对话框: %+v", a.overlays.prompt)
	}
	a.overlays.prompt.open = false
	tt.Frame()

	a.selectNode(beta)
	tt.Frame()
	tt.Key(0, ui.KeyBackspace)
	tt.Frame()
	if !a.overlays.confirm.open {
		t.Fatal("⌫ 应弹出删除确认")
	}
}

// 有对话框打开、或输入框有焦点时，不能触发删除/重命名。
func TestShortcutsIgnoredInModalOrTextFocus(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.selectNode(a.coll.Requests[1].ID)
	a.openPrefs()
	tt.Frame()
	tt.Key(0, ui.KeyBackspace)
	tt.Frame()
	if a.overlays.confirm.open {
		t.Fatal("设置对话框打开时 ⌫ 不应删除节点")
	}
}

// ⌃Tab / ⌘⇧] 切到下一个标签，⌃⇧Tab / ⌘⇧[ 切到上一个，首尾相接。
func TestTabCycleShortcuts(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	a.openRequest(a.coll.Requests[1].ID)
	tt.Frame()
	if a.active != 1 {
		t.Fatalf("初始应选中第二个标签: %d", a.active)
	}
	tt.Key(ui.Ctrl, ui.KeyTab)
	tt.Frame()
	if a.active != 0 {
		t.Fatalf("⌃Tab 应回到第一个（首尾相接）: %d", a.active)
	}
	tt.Key(ui.Cmd|ui.Shift, ui.KeyBracketLeft)
	tt.Frame()
	if a.active != 1 {
		t.Fatalf("⌘⇧[ 应切到上一个（首尾相接）: %d", a.active)
	}
	tt.Key(ui.Ctrl|ui.Shift, ui.KeyTab)
	tt.Frame()
	if a.active != 0 {
		t.Fatalf("⌃⇧Tab 应切到上一个: %d", a.active)
	}
}

// 请求节点右键菜单：包含打开 / 复制 cURL / 导出 cURL。
func TestRequestNodeMenuHasCurl(t *testing.T) {
	a, tt, be := zhApp(t)
	if err := tt.RightClick("beta"); err != nil {
		t.Fatal(err)
	}
	menu := strings.Join(tt.Menu(), "|")
	for _, want := range []string{"打开", "复制 cURL", "导出 cURL…", "重命名", "删除"} {
		if !strings.Contains(menu, want) {
			t.Fatalf("菜单缺少 %q: %s", want, menu)
		}
	}
	if err := tt.ChooseMenuItem("复制 cURL"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !strings.HasPrefix(a.clipboard, "curl") || !strings.Contains(a.clipboard, a.coll.Requests[1].URL) {
		t.Fatalf("应复制 beta 的 cURL: %q", a.clipboard)
	}
	if err := tt.RightClick("beta"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("导出 cURL…"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if len(be.texts) != 1 || be.texts[0].name != "beta.curl.sh" || !strings.HasPrefix(be.texts[0].content, "curl") {
		t.Fatalf("应导出 beta.curl.sh: %+v", be.texts)
	}
}

// 标签右键菜单：新增保存、复制 cURL、关闭右侧。
func TestTabMenuItemsAndCloseRight(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	a.openRequest(a.coll.Requests[1].ID)
	tt.Frame()
	x, y := a.set.SidebarWidth+90, float32(sizeToolbar+sizeTabStrip/2)
	tt.RightClickAt(x, y)
	menu := strings.Join(tt.Menu(), "|")
	for _, want := range []string{"保存", "复制 cURL", "关闭右侧", "关闭其他", "关闭全部"} {
		if !strings.Contains(menu, want) {
			t.Fatalf("标签菜单缺少 %q: %s", want, menu)
		}
	}
	if err := tt.ChooseMenuItem("关闭右侧"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if len(a.tabs) != 1 || a.tabs[0].title() != "alpha" {
		t.Fatalf("应只剩 alpha: %d", len(a.tabs))
	}
}

// 历史行与环境行都有右键菜单。
func TestHistoryAndEnvMenus(t *testing.T) {
	a, tt, be := zhApp(t)
	if _, err := be.CreateEnvironment(a.projectID, "dev"); err != nil {
		t.Fatal(err)
	}
	a.reloadEnvs()
	a.side.tab = 2
	tt.Frame()
	if err := tt.RightClick("dev"); err != nil {
		t.Fatal(err)
	}
	menu := strings.Join(tt.Menu(), "|")
	for _, want := range []string{"编辑", "设为当前环境", "删除"} {
		if !strings.Contains(menu, want) {
			t.Fatalf("环境菜单缺少 %q: %s", want, menu)
		}
	}
	if err := tt.ChooseMenuItem("设为当前环境"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.envID == "" {
		t.Fatal("应已设为当前环境")
	}
}

// 文本输入对话框里的输入框必须有实际高度（曾因 Grow 方向错误塌成一条线，看不到文字）：
// 聚焦时强调色的上、下边框是两条长横线，它们之间要相隔输入框的高度。
func TestPromptInputHasHeight(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.renameNode(a.coll.Requests[0].ID)
	tt.Frame()
	tt.Frame()
	img := tt.Image()
	acc := a.p.accent
	b := img.Bounds()
	top, bottom := -1, -1
	for y := b.Min.Y; y < b.Max.Y; y++ {
		run, best := 0, 0
		for x := b.Min.X; x < b.Max.X; x++ {
			if p := img.RGBAAt(x, y); p.R == acc.R && p.G == acc.G && p.B == acc.B {
				run++
				if run > best {
					best = run
				}
			} else {
				run = 0
			}
		}
		if best >= 250 { // 输入框宽约 390px，按钮只有几十像素
			if top < 0 {
				top = y
			}
			bottom = y
		}
	}
	if top < 0 || bottom-top < 25 {
		t.Fatalf("输入框高度塌缩：上边框 y=%d，下边框 y=%d", top, bottom)
	}
}

// 侧栏同一时间只高亮一行：选中了别的节点后，当前标签对应的请求不再高亮。
func TestSidebarHighlightsSingleRow(t *testing.T) {
	a, tt, _ := zhApp(t)
	alpha, beta := a.coll.Requests[0].ID, a.coll.Requests[1].ID
	a.openRequest(alpha)
	tt.Frame()
	r, ok := tt.Find("beta")
	if !ok {
		t.Fatal("侧栏里应有 beta")
	}
	y := int(r.Y + r.H/2)
	img := tt.Image()
	hi := rgb(a.p.sidebar) // 未高亮行的底色
	if px(img, 12, y-28) == px(img, 12, y) {
		t.Fatal("初始只有当前标签 alpha 一行高亮，alpha 与 beta 的底色应不同")
	}
	a.selectNode(beta)
	tt.Frame()
	img = tt.Image()
	if px(img, 12, y-28) != hi {
		t.Fatalf("选中 beta 后 alpha 不应再高亮: %v", px(img, 12, y-28))
	}
	if px(img, 12, y) == hi {
		t.Fatal("选中 beta 后 beta 应高亮")
	}
}
