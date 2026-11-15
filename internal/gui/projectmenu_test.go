package gui

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// 工具栏项目按钮应弹出下拉面板，列出所有项目并带「新建」入口。
func TestProjectDropdownListsProjectsAndCreates(t *testing.T) {
	a, tt, be := zhApp(t)
	other := be.seedProject("Other")
	a.loadProjects()
	tt.Frame()
	if err := tt.Click("Demo"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.projMenu.open || !tt.HasText("Other") || !tt.HasText("当前") || !tt.HasText("新建") {
		t.Fatalf("应显示项目下拉面板，实际文本 %q", tt.Texts())
	}
	// 输入名称后回车新建并切换
	tt.Type("Brand New")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	var created string
	for _, p := range be.projects {
		if p.Name == "Brand New" {
			created = p.ID
		}
	}
	if created == "" || a.projectID != created {
		t.Fatalf("应新建并切换到项目，created=%q 当前=%q", created, a.projectID)
	}
	_ = other
}

// 切换到其他项目：没有未保存标签时直接切换。
func TestProjectDropdownSwitch(t *testing.T) {
	a, tt, be := zhApp(t)
	other := be.seedProject("Other")
	a.loadProjects()
	tt.Frame()
	if err := tt.Click("Demo"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click("Other"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.projectID != other || a.projMenu.open {
		t.Fatalf("应切换到 Other 并收起面板，当前=%q open=%v", a.projectID, a.projMenu.open)
	}
}

// 当前工作区有未保存标签时，切换项目需要确认。
func TestProjectSwitchConfirmsDirtyTabs(t *testing.T) {
	a, tt, be := zhApp(t)
	other := be.seedProject("Other")
	a.loadProjects()
	a.openRequest(a.coll.Requests[0].ID)
	a.activeTab().draft.URL = "https://changed.example"
	a.activeTab().refreshDirty()
	a.chooseProject(other, "Other")
	tt.Frame()
	if a.projectID == other || !a.overlays.confirm.open {
		t.Fatalf("有未保存标签时应先弹确认，当前=%q confirm=%v", a.projectID, a.overlays.confirm.open)
	}
	a.overlays.confirm.onOK()
	if a.projectID != other {
		t.Fatalf("确认后应切换项目，当前=%q", a.projectID)
	}
}

// 下拉面板里的删除应走删除确认，并删掉项目。
func TestProjectDropdownDelete(t *testing.T) {
	a, tt, be := zhApp(t)
	be.seedProject("Other")
	a.loadProjects()
	tt.Frame()
	if err := tt.Click("Demo"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	// 操作按钮悬停才显示：先把指针移到 Other 行
	r, ok := tt.Find("Other")
	if !ok {
		t.Fatal("找不到 Other 行")
	}
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	tt.Frame()
	if err := tt.Click("删除项目"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.overlays.confirm.open {
		t.Fatal("应弹出删除确认")
	}
	a.overlays.confirm.onOK()
	for _, p := range be.projects {
		if p.Name == "Other" {
			t.Fatal("确认后项目应被删除")
		}
	}
}

// 界面字体只缩放旧版使用 --size-font-* 变量的文本档位（N、N+1、N+2）。
func TestSetUIFontSizeScalesVarTiers(t *testing.T) {
	defer setUIFontSize(12)
	setUIFontSize(14)
	if fsV2xs != 14 || fsVXs != 15 || fsVSm != 16 {
		t.Fatalf("N=14 时应为 14/15/16，实际 %v/%v/%v", fsV2xs, fsVXs, fsVSm)
	}
	setUIFontSize(99)
	if fsV2xs != 16 {
		t.Fatalf("上限应为 16，实际 %v", fsV2xs)
	}
	setUIFontSize(1)
	if fsV2xs != 10 || fsVXs != 11 || fsVSm != 12 {
		t.Fatalf("下限应为 10/11/12，实际 %v/%v/%v", fsV2xs, fsVXs, fsVSm)
	}
}

// 固定字号的文本（如底栏）不应随界面字体变化，随变量缩放的文本（如标签名）则应变化。
func TestFontSizeSettingDoesNotScaleWholeUI(t *testing.T) {
	defer setUIFontSize(12)
	a, tt, _ := zhApp(t)
	a.openRequest(a.coll.Requests[0].ID)
	tt.Frame()
	base, _ := tt.Find("控制台")
	a.set.FontSize = 16
	tt.Frame()
	big, _ := tt.Find("控制台")
	if base.W != big.W || base.H != big.H {
		t.Fatalf("底栏文字不应随界面字体缩放：%+v → %+v", base, big)
	}
	if fsV2xs != 16 {
		t.Fatalf("设置 16 后可缩放字号应为 16，实际 %v", fsV2xs)
	}
}

// 「滚动条自动隐藏」：开启时只在滚动后短暂显示；关闭时一直允许（悬停显示）。
func TestScrollbarAutoHideVisibility(t *testing.T) {
	a, _, _ := zhApp(t)
	a.set.ScrollbarAutoHide = true
	a.scrollAt = time.Time{}
	if a.scrollbarsVisible() {
		t.Fatal("开启自动隐藏且没有滚动时应隐藏滚动条")
	}
	a.scrollAt = time.Now()
	if !a.scrollbarsVisible() {
		t.Fatal("刚滚动过应显示滚动条")
	}
	a.scrollAt = time.Now().Add(-2 * scrollbarLinger)
	if a.scrollbarsVisible() {
		t.Fatal("停止滚动超过 750ms 应再次隐藏")
	}
	a.set.ScrollbarAutoHide = false
	a.scrollAt = time.Time{}
	if !a.scrollbarsVisible() {
		t.Fatal("关闭自动隐藏后滚动条应始终可显示")
	}
}

// 项目按钮的整个区域（图标、名称、右侧箭头，含内边距）都能点开/收起下拉面板，
// 而不只是名称文字本身；按钮之外（后面的空白标题栏）则不响应。
func TestProjectButtonWholeAreaToggles(t *testing.T) {
	a, tt, _ := zhApp(t)
	// 项目按钮自带“切换项目”标签，其区域就是整颗按钮（头像、名称、箭头与内边距）。
	btn, ok := tt.Find("切换项目")
	if !ok {
		t.Fatal("找不到项目按钮")
	}
	left, w := btn.X, btn.W
	cy := btn.Y + btn.H/2
	for label, x := range map[string]float32{
		"左内边距": left + 3,
		"头像":   left + 9 + 9,
		"名称":   left + w/2,
		"箭头":   left + w - 9 - 5,
		"右内边距": left + w - 3,
	} {
		tt.ClickAt(x, cy)
		tt.Frame()
		if !a.projMenu.open {
			t.Fatalf("点击%s（x=%.0f）应展开下拉面板", label, x)
		}
		tt.ClickAt(x, cy)
		tt.Frame()
		if a.projMenu.open {
			t.Fatalf("再次点击%s应收起下拉面板", label)
		}
	}
	// 按钮左侧（悬浮胶囊之外）的标题栏空白不应触发
	tt.ClickAt(left-60, cy)
	tt.Frame()
	if a.projMenu.open {
		t.Fatal("点按钮之外的标题栏空白不应展开面板")
	}
}
