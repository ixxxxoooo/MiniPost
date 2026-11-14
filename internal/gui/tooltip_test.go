package gui

import (
	"testing"
	"time"
)

const longName = "这是一个名字特别特别长的请求用来测试悬停提示是否会显示完整名称而不是被省略号截断"

func countText(texts []string, s string) int {
	n := 0
	for _, t := range texts {
		if t == s {
			n++
		}
	}
	return n
}

// 名称过长被截断的请求：悬停后出现显示全名的提示；名称很短时不出现提示。
func TestSidebarRowTooltipOnlyWhenTruncated(t *testing.T) {
	a, tt, be := zhApp(t)
	be.CreateRequestItem(a.projectID, "", longName)
	a.reloadCollection()
	tt.Frame()
	if countText(tt.Texts(), longName) != 1 {
		t.Fatalf("悬停前全名应只出现一次: %d", countText(tt.Texts(), longName))
	}
	r, ok := tt.Find(longName)
	if !ok {
		t.Fatal("找不到长名称行")
	}
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	tt.Advance(time.Second)
	tt.Frame()
	if got := countText(tt.Texts(), longName); got < 2 {
		t.Fatalf("悬停被截断的名称应出现全名提示（出现 %d 次）", got)
	}

	// 短名称：悬停也不出现额外提示
	r2, ok := tt.Find("alpha")
	if !ok {
		t.Fatal("找不到 alpha 行")
	}
	tt.Move(r2.X+r2.W/2, r2.Y+r2.H/2)
	tt.Advance(time.Second)
	tt.Frame()
	if got := countText(tt.Texts(), "alpha"); got != 1 {
		t.Fatalf("放得下的名称不应有提示（出现 %d 次）", got)
	}
}

// 过长的标签标题悬停时显示全名。
func TestTabTitleTooltipWhenTruncated(t *testing.T) {
	a, tt, be := zhApp(t)
	it, _ := be.CreateRequestItem(a.projectID, "", longName)
	a.reloadCollection()
	a.openRequest(it.ID)
	tt.Frame()
	base := countText(tt.Texts(), longName) // 标签标题 + 面包屑 + 侧栏行
	// 标签在标签栏：y 落在标签条内
	tabY := float32(sizeToolbar + sizeTabStrip/2)
	tt.Move(a.set.SidebarWidth+120, tabY)
	tt.Advance(time.Second)
	tt.Frame()
	if got := countText(tt.Texts(), longName); got <= base {
		t.Fatalf("悬停被截断的标签应多出一份全名提示: 之前 %d 之后 %d", base, got)
	}
}
