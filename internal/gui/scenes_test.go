package gui

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestScenes 生成与旧版逐屏对照的截图（s_<名称>.png），仅在设置了 MINIPOST_SNAPSHOT_DIR 时落盘。
// 旧版对应截图由 /tmp/ref/scenes.sh 以相同操作序列生成（o_<名称>.png）。
type scene struct {
	name string
	dark bool
	act  func(a *app, tt *ui.Tester)
}

func sceneClick(tt *ui.Tester, names ...string) {
	for _, n := range names {
		_ = tt.Click(n)
	}
	tt.Frame()
}

// sceneList 与 /tmp/ref/scenes.sh 中的旧版操作序列一一对应。
var sceneList = []scene{
	{"empty", false, func(a *app, tt *ui.Tester) {}},
	{"req", false, func(a *app, tt *ui.Tester) { sceneClick(tt, "用户服务", "获取用户列表") }},
	{"hdrs", false, func(a *app, tt *ui.Tester) { sceneClick(tt, "用户服务", "获取用户列表", "Headers") }},
	{"body", false, func(a *app, tt *ui.Tester) { sceneClick(tt, "用户服务", "获取用户列表", "Body") }},
	{"auth", false, func(a *app, tt *ui.Tester) { sceneClick(tt, "用户服务", "获取用户列表", "Auth") }},
	{"resp", false, func(a *app, tt *ui.Tester) { sceneClick(tt, "用户服务", "获取用户列表", "Send") }},
	{"hist", false, func(a *app, tt *ui.Tester) { sceneClick(tt, "历史") }},
	{"envtab", false, func(a *app, tt *ui.Tester) { sceneClick(tt, "环境") }},
	{"cookies", false, func(a *app, tt *ui.Tester) { sceneClick(tt, "Cookies") }},
	{"console", false, func(a *app, tt *ui.Tester) { sceneClick(tt, "Console") }},
	{"settings", false, func(a *app, tt *ui.Tester) { sceneClick(tt, "设置 (⌘,)") }},
	{"projmenu", false, func(a *app, tt *ui.Tester) {
		sceneClick(tt, "Demo")
		tt.Move(260, 120)
		tt.Frame()
	}},
	{"curl", false, func(a *app, tt *ui.Tester) { a.openCurlDialog(); tt.Frame() }},
	{"dreq", true, func(a *app, tt *ui.Tester) { sceneClick(tt, "用户服务", "获取用户列表") }},
	{"dresp", true, func(a *app, tt *ui.Tester) { sceneClick(tt, "用户服务", "获取用户列表", "Send") }},
}

// TestScenes 生成与旧版逐屏对照的截图（s_<名称>.png），仅在设置了 MINIPOST_SNAPSHOT_DIR 时落盘。
// 同时若设置了 MINIPOST_TEXTS_DIR，会输出每个场景的文本/控件矩形（n_<名称>.json）。
func TestScenes(t *testing.T) {
	dir := os.Getenv("MINIPOST_TEXTS_DIR")
	for _, s := range sceneList {
		a, tt, _ := visualApp(t, s.dark)
		s.act(a, tt)
		snapshot(t, tt, "s_"+s.name)
		if dir == "" {
			continue
		}
		var res []map[string]any
		for _, str := range tt.Texts() {
			if r, ok := tt.Find(str); ok {
				res = append(res, map[string]any{"t": str, "x": r.X, "y": r.Y, "w": r.W, "h": r.H})
			}
		}
		b, _ := json.Marshal(res)
		_ = os.WriteFile(dir+"/n_"+s.name+".json", b, 0o644)
	}
}
