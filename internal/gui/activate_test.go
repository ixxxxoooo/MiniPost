package gui

import "testing"

// fakeWindow 记录 restoreOnActivate 对窗口发出的调用。
type fakeWindow struct {
	visible, minimized bool
	calls              []string
}

func (f *fakeWindow) IsVisible() bool   { return f.visible }
func (f *fakeWindow) IsMinimized() bool { return f.minimized }
func (f *fakeWindow) Show()             { f.calls = append(f.calls, "show"); f.visible = true }
func (f *fakeWindow) Restore()          { f.calls = append(f.calls, "restore"); f.minimized = false }
func (f *fakeWindow) Focus()            { f.calls = append(f.calls, "focus") }

func equalCalls(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// 窗口被最小化（仍算"可见"）：必须还原，这是 Option+A 唤醒不出窗口的根因。
func TestRestoreOnActivateMinimized(t *testing.T) {
	w := &fakeWindow{visible: true, minimized: true}
	if got := restoreOnActivate(w); got != "restored" {
		t.Fatalf("最小化窗口应还原，实际 %q", got)
	}
	if !equalCalls(w.calls, []string{"restore", "focus"}) {
		t.Fatalf("调用序列不符: %v", w.calls)
	}
	if w.minimized {
		t.Fatal("还原后不应再处于最小化")
	}
}

// 窗口被隐藏：重新显示并聚焦。
func TestRestoreOnActivateHidden(t *testing.T) {
	w := &fakeWindow{visible: false}
	if got := restoreOnActivate(w); got != "shown" {
		t.Fatalf("隐藏窗口应重新显示，实际 %q", got)
	}
	if !equalCalls(w.calls, []string{"show", "focus"}) {
		t.Fatalf("调用序列不符: %v", w.calls)
	}
}

// 窗口本来就正常显示：只聚焦，不做还原或显示，避免打扰用户。
func TestRestoreOnActivateAlreadyVisible(t *testing.T) {
	w := &fakeWindow{visible: true}
	if got := restoreOnActivate(w); got != "focused" {
		t.Fatalf("正常窗口只应聚焦，实际 %q", got)
	}
	if !equalCalls(w.calls, []string{"focus"}) {
		t.Fatalf("调用序列不符: %v", w.calls)
	}
}

// 既被隐藏又被最小化：以最小化为准还原一次，且重复激活是幂等的（第二次只聚焦）。
func TestRestoreOnActivateIdempotent(t *testing.T) {
	w := &fakeWindow{visible: false, minimized: true}
	if got := restoreOnActivate(w); got != "restored" {
		t.Fatalf("首次应还原，实际 %q", got)
	}
	w.visible = true
	w.calls = nil
	if got := restoreOnActivate(w); got != "focused" {
		t.Fatalf("重复激活应只聚焦，实际 %q", got)
	}
	if !equalCalls(w.calls, []string{"focus"}) {
		t.Fatalf("调用序列不符: %v", w.calls)
	}
}
