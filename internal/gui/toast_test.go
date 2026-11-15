package gui

import (
	"testing"
	"time"
)

// showToast 登记提示并让界面跑完进场动画。
func showToast(a *app, tt interface {
	Frame()
	Advance(time.Duration)
}, fn func()) {
	fn()
	tt.Frame()
	tt.Advance(300 * time.Millisecond)
}

// 提示是玻璃底的提示卡，显示在窗口右上角、标题栏下方，且不超出窗口。
func TestToastShowsTopRight(t *testing.T) {
	a, tt, _ := zhApp(t)
	showToast(a, tt, func() { a.toastOK("已复制响应体") })
	r, ok := tt.Find("已复制响应体")
	if !ok {
		t.Fatalf("应显示提示，实际 %q", tt.Texts())
	}
	if r.Y < sizeToolbar {
		t.Fatalf("提示不应盖住标题栏，y=%.0f", r.Y)
	}
	if r.Y > sizeToolbar+80 {
		t.Fatalf("提示应在窗口上部（标题栏下方），y=%.0f", r.Y)
	}
	if right := r.X + r.W; right > 1200 || right < 1200-toastMaxW {
		t.Fatalf("提示应靠右，右边缘 %.0f", right)
	}
	if r.X < 600 {
		t.Fatalf("提示应在窗口右半边，x=%.0f", r.X)
	}
	snapshot(t, tt, "toast-success")
}

// 多条提示竖向排列，最新的在最下面。
func TestToastNewestAtBottom(t *testing.T) {
	a, tt, _ := zhApp(t)
	showToast(a, tt, func() {
		a.toast("第一条")
		a.toastWarn("第二条")
		a.toastErr("第三条")
	})
	r1, ok1 := tt.Find("第一条")
	r2, ok2 := tt.Find("第二条")
	r3, ok3 := tt.Find("第三条")
	if !ok1 || !ok2 || !ok3 {
		t.Fatalf("三条提示都应显示，实际 %q", tt.Texts())
	}
	if !(r1.Y < r2.Y && r2.Y < r3.Y) {
		t.Fatalf("应最新在下: y1=%.0f y2=%.0f y3=%.0f", r1.Y, r2.Y, r3.Y)
	}
	snapshot(t, tt, "toast-stack")
}

// 提示到时间后自动消失，错误比成功停留更久。
func TestToastExpires(t *testing.T) {
	a, tt, _ := zhApp(t)
	showToast(a, tt, func() {
		a.toastOK("成功提示")
		a.toastErr("失败提示")
	})
	tt.Advance(toastTimeout(toastSuccess))
	tt.Frame()
	if tt.HasText("成功提示") {
		t.Fatalf("成功提示应已消失")
	}
	if !tt.HasText("失败提示") {
		t.Fatalf("错误提示应还在")
	}
	tt.Advance(toastTimeout(toastError))
	tt.Frame()
	if tt.HasText("失败提示") {
		t.Fatalf("错误提示也应消失")
	}
	if toastTimeout(toastError) <= toastTimeout(toastSuccess) {
		t.Fatalf("错误提示应比成功提示停留更久")
	}
}

// 点关闭按钮立即关闭提示。
func TestToastCloseButton(t *testing.T) {
	a, tt, _ := zhApp(t)
	showToast(a, tt, func() { a.toastWarn("请先输入请求地址") })
	r, ok := tt.Find("请先输入请求地址")
	if !ok {
		t.Fatal("应显示提示")
	}
	tt.Move(r.X+r.W/2, r.Y+r.H/2) // 鼠标停在提示条上，关闭按钮才出现
	tt.Frame()
	tt.Frame()
	if err := tt.Click("关闭"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(300 * time.Millisecond)
	if tt.HasText("请先输入请求地址") {
		t.Fatalf("点关闭后提示应消失")
	}
}

// 带按钮的提示：点按钮执行动作并关闭。
func TestToastActionButton(t *testing.T) {
	a, tt, _ := zhApp(t)
	done := 0
	showToast(a, tt, func() { a.toastAction("请求 已删除", "撤销", func() { done++ }) })
	snapshot(t, tt, "toast-action")
	if err := tt.Click("撤销"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if done != 1 {
		t.Fatalf("应执行一次撤销，实际 %d", done)
	}
	tt.Advance(300 * time.Millisecond)
	if tt.HasText("请求 已删除") {
		t.Fatalf("点按钮后提示应关闭")
	}
}

// 很长的错误信息会换行，卡片不超过最大宽度。
func TestToastLongMessageWraps(t *testing.T) {
	a, tt, _ := zhApp(t)
	long := "导入失败：" + string(make([]rune, 0))
	for i := 0; i < 40; i++ {
		long += "很长的错误信息"
	}
	showToast(a, tt, func() { a.toastErr(long) })
	r, ok := tt.Find(long)
	if !ok {
		t.Fatalf("应显示长提示")
	}
	if r.W > toastMaxW {
		t.Fatalf("文字宽度 %.0f 不应超过卡片最大宽度 %d", r.W, toastMaxW)
	}
	if r.H < 40 {
		t.Fatalf("长文字应换行，高度 %.0f", r.H)
	}
	snapshot(t, tt, "toast-long")
}

// 非法类型按普通提示处理，不会丢失。
func TestPushToastUnknownKindFallsBack(t *testing.T) {
	a, _, _ := zhApp(t)
	a.pushToast("x", "bogus")
	if len(a.toastMsgs) != 1 || a.toastMsgs[0].kind != toastInfo {
		t.Fatalf("应退回普通提示，实际 %+v", a.toastMsgs)
	}
}
