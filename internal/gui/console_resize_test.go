package gui

import (
	"testing"

	"minipost/internal/client"
)

// 窗口高度对应的控制台高度范围：下限固定，上限给请求区留出空间，且不小于下限。
func TestConsoleLimits(t *testing.T) {
	lo, hi := consoleLimits(760)
	if lo != client.ConsoleMinHeight || hi != 760-sizeToolbar-sizeBottom-consoleMinMain {
		t.Fatalf("760 高窗口范围不对: %v %v", lo, hi)
	}
	if lo, hi = consoleLimits(100); hi != lo {
		t.Fatalf("极矮窗口上限不应小于下限: %v %v", lo, hi)
	}
	if _, hi = consoleLimits(5000); hi != client.ConsoleMaxHeight {
		t.Fatalf("上限不应超过 %d，实际 %v", client.ConsoleMaxHeight, hi)
	}
}

// consoleTop 是 1200x760 测试窗口里控制台面板上边缘的 y 坐标（内容卡片与窗口底边留有 shellGap）。
func consoleTop(a *app) float32 { return 760 - shellGap - sizeBottom - a.set.ConsoleHeight }

// 向上拖动上边缘：面板变高，跟手；松手后保存。
func TestConsoleResizeDragUp(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.set.ConsoleOpen, a.set.ConsoleHeight = true, 220
	tt.Frame()
	y := consoleTop(a)
	tt.Press(600, y+1)
	tt.Move(600, y-39)
	tt.Frame()
	if a.set.ConsoleHeight != 260 {
		t.Fatalf("向上拖 40 应为 260，实际 %v", a.set.ConsoleHeight)
	}
	if !a.con.resizing {
		t.Fatalf("拖动中应处于调整状态")
	}
	tt.Release(600, y-39)
	tt.Frame()
	if a.con.resizing || a.set.ConsoleHeight != 260 {
		t.Fatalf("松手后应结束调整并保持 260，实际 %v resizing=%v", a.set.ConsoleHeight, a.con.resizing)
	}
	snapshot(t, tt, "console-resized")
}

// 向下拖动：面板变矮，且不会低于下限；多次移动累加正确（不会抖动或翻倍）。
func TestConsoleResizeDragDownClampsAtMin(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.set.ConsoleOpen, a.set.ConsoleHeight = true, 220
	tt.Frame()
	y := consoleTop(a)
	tt.Press(600, y+1)
	tt.Move(600, y+21)
	tt.Frame()
	if a.set.ConsoleHeight != 200 {
		t.Fatalf("向下拖 20 应为 200，实际 %v", a.set.ConsoleHeight)
	}
	tt.Move(600, y+21+30)
	tt.Frame()
	if a.set.ConsoleHeight != 170 {
		t.Fatalf("再向下 30 应为 170，实际 %v", a.set.ConsoleHeight)
	}
	tt.Move(600, y+500)
	tt.Frame()
	if a.set.ConsoleHeight != client.ConsoleMinHeight {
		t.Fatalf("不应低于下限 %d，实际 %v", client.ConsoleMinHeight, a.set.ConsoleHeight)
	}
	// 指针越过下限后折返：要先抵消越过的距离，面板才重新跟手。
	tt.Move(600, y+480)
	tt.Frame()
	if a.set.ConsoleHeight != client.ConsoleMinHeight {
		t.Fatalf("折返未抵消越界距离前应保持下限，实际 %v", a.set.ConsoleHeight)
	}
	tt.Release(600, y+480)
}

// 向上拖到头：不超过上限，请求区至少保留 consoleMinMain。
func TestConsoleResizeClampsAtMax(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.set.ConsoleOpen, a.set.ConsoleHeight = true, 220
	tt.Frame()
	y := consoleTop(a)
	tt.Press(600, y+1)
	tt.Move(600, 60)
	tt.Frame()
	_, hi := consoleLimits(760)
	if a.set.ConsoleHeight != hi {
		t.Fatalf("应夹在上限 %v，实际 %v", hi, a.set.ConsoleHeight)
	}
	tt.Release(600, 60)
}

// 双击上边缘恢复默认高度。
func TestConsoleResizeDoubleClickResets(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.set.ConsoleOpen, a.set.ConsoleHeight = true, 330
	tt.Frame()
	y := consoleTop(a)
	tt.ClickAt(600, y+1)
	tt.ClickAt(600, y+1)
	tt.Frame()
	if a.set.ConsoleHeight != client.ConsoleDefaultHeight {
		t.Fatalf("双击应恢复 %d，实际 %v", client.ConsoleDefaultHeight, a.set.ConsoleHeight)
	}
}

// 窗口太矮时只是显示夹紧，设置里的高度保留，窗口变高后恢复。
func TestConsoleShownHeightClampedButSettingKept(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.set.ConsoleOpen, a.set.ConsoleHeight = true, 500
	tt.SetSize(1200, 500)
	tt.Frame()
	_, hi := consoleLimits(500)
	if a.set.ConsoleHeight != 500 {
		t.Fatalf("设置值应保留 500，实际 %v", a.set.ConsoleHeight)
	}
	if _, ok := tt.Find("控制台"); !ok {
		t.Fatalf("矮窗口里控制台仍应显示")
	}
	if hi >= 500 {
		t.Fatalf("测试前提不成立：矮窗口上限 %v 应小于 500", hi)
	}
	tt.SetSize(1200, 900)
	tt.Frame()
	if a.set.ConsoleHeight != 500 {
		t.Fatalf("窗口变高后设置值仍应是 500，实际 %v", a.set.ConsoleHeight)
	}
}

// 控制台关闭时没有拖动手柄，拖底部区域不会改高度。
func TestConsoleResizeNotWhenClosed(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.set.ConsoleOpen, a.set.ConsoleHeight = false, 220
	tt.Frame()
	tt.Press(600, 520)
	tt.Move(600, 450)
	tt.Frame()
	tt.Release(600, 450)
	if a.set.ConsoleHeight != 220 {
		t.Fatalf("关闭时不应改高度，实际 %v", a.set.ConsoleHeight)
	}
}

// 设置里的控制台高度合法范围是 [最小, 最大]，越界值在加载时被修正。
func TestConsoleHeightSettingNormalize(t *testing.T) {
	s := client.DefaultSettings()
	s.ConsoleHeight = 10
	s.Normalize()
	if s.ConsoleHeight != client.ConsoleMinHeight {
		t.Fatalf("过小应修正为下限，实际 %v", s.ConsoleHeight)
	}
	s.ConsoleHeight = 99999
	s.Normalize()
	if s.ConsoleHeight != client.ConsoleMaxHeight {
		t.Fatalf("过大应修正为上限，实际 %v", s.ConsoleHeight)
	}
	s.ConsoleHeight = 640
	s.Normalize()
	if s.ConsoleHeight != 640 {
		t.Fatalf("合法值不应被改动，实际 %v", s.ConsoleHeight)
	}
}
