package gui

// navState 记录标签的访问历史，供标题栏的前进/后退使用（行为同旧版 WorkspaceHeader）。
type navState struct {
	hist []string // 请求 ID
	idx  int
	skip bool // 正在由前进/后退触发的切换，不再记录
}

// navRecord 记录一次激活。
func (a *app) navRecord(id string) {
	n := &a.nav
	if n.skip || id == "" {
		return
	}
	if n.idx >= 0 && n.idx < len(n.hist) && n.hist[n.idx] == id {
		return
	}
	n.hist = append(n.hist[:min(n.idx+1, len(n.hist))], id)
	n.idx = len(n.hist) - 1
}

// navTarget 从 from 开始沿 dir 方向找到第一个仍然打开着的标签。
func (a *app) navTarget(dir int) (idx, tab int, ok bool) {
	n := &a.nav
	for i := n.idx + dir; i >= 0 && i < len(n.hist); i += dir {
		if t := a.findTab(n.hist[i]); t >= 0 {
			return i, t, true
		}
	}
	return 0, 0, false
}

func (a *app) canNav(dir int) bool {
	_, _, ok := a.navTarget(dir)
	return ok
}

func (a *app) navigate(dir int) {
	i, t, ok := a.navTarget(dir)
	if !ok {
		return
	}
	a.nav.skip = true
	a.nav.idx = i
	a.showHome = false
	a.activate(t)
	a.nav.skip = false
}
