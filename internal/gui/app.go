package gui

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"minipost/internal/client"
	"minipost/internal/model"
	"minipost/internal/pkg/logger"
)

// app 保存整个界面的状态，只在主线程访问；后台任务通过 win.Update 回到主线程再修改。
type app struct {
	be        Backend
	win       *mygo.Window
	native    bool   // 是否运行在真实窗口中；测试里为 false，此时不调用 mygo 的全局原生接口
	clipboard string // 最近一次写入剪贴板的文本（测试可读）
	sysLocale string

	set      client.Settings
	setStore *client.SettingsStore
	sessions *client.SessionStore
	jar      *client.Jar

	projects  []model.Project
	projectID string
	coll      model.CollectionData
	tree      treeIndex
	outline   ui.OutlineState[string]
	outRow    int
	envs      []model.Environment
	envID     string
	history   []model.HistoryEntry
	histList  ui.ListState
	histRow   int

	console     []consoleEntry
	consoleList ui.ListState
	consolePct  float32

	tabs   []*tab
	active int

	sideTab       int               // 0 请求，1 历史，2 环境
	sideQuery     string            // 侧栏搜索词
	histCollapsed map[string]bool   // 已折叠的历史日期分组
	cardHot       string            // 当前保持显示的悬停卡片
	homeName      string            // 项目主页“新建项目”输入框
	themePick     string            // 正在选择主题色的项目 ID
	descDraft     map[string]string // 项目描述的编辑草稿
	bulk          map[string]string // 处于批量编辑模式的键值表及其文本
	showHome      bool
	splitPct      float32
	syncSel       bool // 下一帧让侧栏选中与当前标签一致
	overlays      overlays
	deferred      []func() // 在本帧构建结束后执行的变更，避免在遍历数据时修改它
	toastMsgs     []string
	applied       appliedTheme
	p             *palette  // 当前亮/暗调色板
	th            *ui.Theme // 当前 MyGo 主题（每帧都要重新设置）
	nav           navState  // 标签访问历史
	menuLocale    string
	backupStop    chan struct{}
	projMenu      projMenuState // 工具栏项目下拉面板状态
	scrollAt      time.Time     // 最近一次滚动输入的时间（滚动条自动隐藏用）
}

func newApp(be Backend) *app {
	base := be.BaseDir()
	a := &app{
		be:         be,
		setStore:   client.NewSettingsStore(base),
		sessions:   client.NewSessionStore(base),
		jar:        client.NewJar(filepath.Join(base, "cookies.json")),
		outRow:     -1,
		nav:        navState{idx: -1},
		histRow:    -1,
		bulk:       map[string]string{},
		descDraft:  map[string]string{},
		splitPct:   55,
		consolePct: 72,
	}
	a.set = a.setStore.Load()
	a.loadProjects()
	a.restoreProject()
	return a
}

// later 登记一个在本帧构建结束后执行的变更。
func (a *app) later(fn func()) { a.deferred = append(a.deferred, fn) }

func (a *app) runDeferred() {
	for len(a.deferred) > 0 {
		fn := a.deferred[0]
		a.deferred = a.deferred[1:]
		fn()
	}
}

// toast 登记一条在下一帧显示的提示。
func (a *app) toast(msg string) { a.toastMsgs = append(a.toastMsgs, msg) }

// fail 记录错误日志并提示用户。
func (a *app) fail(what string, err error) {
	if err == nil {
		return
	}
	logger.Warn(what+"失败", "error", err.Error())
	a.toast(what + a.t("失败：", " failed: ") + err.Error())
}

func (a *app) saveSettings() {
	if err := a.setStore.Save(a.set); err != nil {
		logger.Error("保存设置失败", "error", err.Error())
	}
}

func (a *app) loadProjects() {
	list, err := a.be.ListProjects()
	if err != nil {
		a.fail(a.t("加载项目", "Load projects"), err)
		return
	}
	a.projects = list
}

func (a *app) project() *model.Project {
	for i := range a.projects {
		if a.projects[i].ID == a.projectID {
			return &a.projects[i]
		}
	}
	return nil
}

// restoreProject 启动时恢复上次打开的项目；没有可用项目时停留在首页。
func (a *app) restoreProject() {
	id := a.set.LastProjectID
	found := false
	for _, p := range a.projects {
		found = found || p.ID == id
	}
	if !found {
		a.showHome = true
		return
	}
	a.selectProject(id)
}

// selectProject 切换项目：保存旧项目的标签会话，加载新项目的集合、环境、历史并恢复标签。
func (a *app) selectProject(id string) {
	if a.projectID != "" && a.projectID != id {
		a.persistSession()
	}
	a.projectID = id
	a.showHome = id == ""
	a.tabs, a.active, a.outRow = nil, 0, -1
	a.outline = ui.OutlineState[string]{}
	a.set.LastProjectID = id
	a.saveSettings()
	if id == "" {
		a.coll, a.envs, a.history = model.CollectionData{}, nil, nil
		return
	}
	a.reloadCollection()
	a.reloadEnvs()
	a.reloadHistory()
	a.restoreSession()
	logger.Info("已切换项目", "projectID", id)
}

func (a *app) reloadCollection() {
	data, err := a.be.GetCollectionData(a.projectID)
	if err != nil {
		a.fail(a.t("加载集合", "Load collection"), err)
		return
	}
	a.coll = *data
	a.tree = buildTree(a.coll)
}

func (a *app) reloadEnvs() {
	envs, err := a.be.ListEnvironments(a.projectID)
	if err != nil {
		a.fail(a.t("加载环境", "Load environments"), err)
		return
	}
	a.envs = envs
	if a.envID == "" || a.env() == nil {
		a.envID = a.set.EnvByProject[a.projectID]
		if a.env() == nil {
			a.envID = ""
		}
	}
}

func (a *app) reloadHistory() {
	h, err := a.be.GetHistory(a.projectID)
	if err != nil {
		a.fail(a.t("加载历史", "Load history"), err)
		return
	}
	a.history = h
}

// env 返回当前激活的环境，未选择时为 nil。
func (a *app) env() *model.Environment {
	for i := range a.envs {
		if a.envs[i].ID == a.envID {
			return &a.envs[i]
		}
	}
	return nil
}

func (a *app) setEnv(id string) {
	a.envID = id
	a.set.EnvByProject[a.projectID] = id
	a.saveSettings()
}

func (a *app) vars() []client.Var { return client.ActiveVars(a.env()) }

// trimName 规整用户输入的名称。
func trimName(s string) string { return strings.TrimSpace(s) }
