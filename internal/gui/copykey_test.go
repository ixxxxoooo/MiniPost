package gui

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// ⌘C 是系统的“复制文字”：侧栏里选中了节点、焦点不在输入框时，也不能再把节点复制出一份。
func TestCmdCDoesNotDuplicateNode(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.selectNode(a.coll.Requests[0].ID)
	tt.Frame()
	before := len(a.coll.Requests)
	tt.Key(ui.Cmd, ui.KeyC)
	tt.Frame()
	if len(a.coll.Requests) != before {
		t.Fatalf("⌘C 不应复制节点: %d → %d", before, len(a.coll.Requests))
	}
}

// ⌘D 仍然复制选中节点。
func TestCmdDDuplicatesNode(t *testing.T) {
	a, tt, _ := zhApp(t)
	a.selectNode(a.coll.Requests[0].ID)
	tt.Frame()
	before := len(a.coll.Requests)
	tt.Key(ui.Cmd, ui.KeyD)
	tt.Frame()
	if len(a.coll.Requests) != before+1 {
		t.Fatalf("⌘D 应复制出一份: %d → %d", before, len(a.coll.Requests))
	}
}

// 快捷键说明里不再提 ⌘C 复制节点。
func TestShortcutListMentionsOnlyCmdD(t *testing.T) {
	for _, s := range prefsShortcuts {
		if s.zh == "复制选中请求/文件夹" && s.keys != "⌘ + D" {
			t.Fatalf("快捷键应只写 ⌘ + D: %q", s.keys)
		}
	}
}
