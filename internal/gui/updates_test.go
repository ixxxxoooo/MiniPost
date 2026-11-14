package gui

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// fakeUpdater 是假的更新后端：记录检查次数，按设定返回结果。
type fakeUpdater struct {
	enabled    bool
	up         *pendingUpdate
	err        error
	checks     int
	installErr error
	installs   int
}

func (f *fakeUpdater) Enabled() bool { return f.enabled }

func (f *fakeUpdater) Check(context.Context) (*pendingUpdate, error) {
	f.checks++
	if f.err != nil || f.up == nil {
		return nil, f.err
	}
	up := *f.up
	up.Install = func(_ context.Context, progress func(done, total int64)) error {
		f.installs++
		progress(50, 100)
		progress(100, 100)
		return f.installErr
	}
	return &up, nil
}

// updApp 返回装了假更新后端的应用；relaunched 记录重启次数。
func updApp(t *testing.T, f *fakeUpdater) (*app, *ui.Tester, *int) {
	t.Helper()
	a, tt, _ := zhApp(t)
	a.updater = f
	n := 0
	a.relaunch = func() { n++ }
	return a, tt, &n
}

func newVersion() *pendingUpdate {
	return &pendingUpdate{Version: "9.9.9", Notes: "修复若干问题"}
}

// 更新进度按下载字节算百分比；总大小未知时是 0，不会除零。
func TestUpdatePercent(t *testing.T) {
	var u updateState
	if u.percent() != 0 {
		t.Fatal("总大小未知时应为 0")
	}
	u.total.Store(200)
	u.done.Store(50)
	if u.percent() != 25 {
		t.Fatalf("percent = %d", u.percent())
	}
	u.done.Store(999)
	if u.percent() != 100 {
		t.Fatal("不应超过 100")
	}
}

// 默认间隔就是每小时，且启动后很快做第一次检查。
func TestUpdateIntervalIsHourly(t *testing.T) {
	if updateCheckInterval.Hours() != 1 {
		t.Fatalf("间隔应为 1 小时，实际 %v", updateCheckInterval)
	}
	if updateFirstDelay <= 0 || updateFirstDelay >= updateCheckInterval {
		t.Fatalf("首次检查延迟不合理: %v", updateFirstDelay)
	}
}

// 不支持更新的构建（开发版、包管理器安装）既不启动后台检查，也不会进入检查状态。
func TestUpdateDisabledBuild(t *testing.T) {
	f := &fakeUpdater{enabled: false, up: newVersion()}
	a, _, _ := updApp(t, f)
	a.startUpdater()
	if a.stopUpdate != nil {
		t.Fatal("不支持更新时不应启动后台检查")
	}
	a.checkUpdates(true)
	if a.upd.phase != upIdle || f.checks != 0 {
		t.Fatalf("不应检查: phase=%v checks=%d", a.upd.phase, f.checks)
	}
}

// 支持更新的构建会启动后台检查，退出时可以停掉。
func TestUpdateStartsAndStopsWhenEnabled(t *testing.T) {
	a, _, _ := updApp(t, &fakeUpdater{enabled: true})
	a.startUpdater()
	if a.stopUpdate == nil {
		t.Fatal("应启动后台检查")
	}
	a.stopUpdater()
	if a.stopUpdate != nil {
		t.Fatal("停止后应清空")
	}
}

// 完整路径：检查到新版本 → 自动下载 → 弹出带「重启更新」按钮的提示 → 点击后重启。
func TestUpdateAutoDownloadThenRestart(t *testing.T) {
	f := &fakeUpdater{enabled: true, up: newVersion()}
	a, tt, relaunched := updApp(t, f)
	a.checkUpdates(false)

	if f.installs != 1 || a.upd.phase != upReady {
		t.Fatalf("应自动下载完成: installs=%d phase=%v", f.installs, a.upd.phase)
	}
	if a.upd.percent() != 100 {
		t.Fatalf("进度应为 100，实际 %d", a.upd.percent())
	}
	var act *toastAct
	for i := range a.toastActs {
		if a.toastActs[i].label == "重启更新" {
			act = &a.toastActs[i]
		}
	}
	if act == nil || !strings.Contains(act.msg, "9.9.9") {
		t.Fatalf("下载完成后应弹出带按钮的提示，实际 %+v", a.toastActs)
	}
	tt.Frame() // 提示交给界面显示后即被清空
	if len(a.toastActs) != 0 {
		t.Fatal("提示显示后应被清空")
	}
	if !tt.HasText("新版本 9.9.9 · 重启更新") {
		t.Fatalf("标题栏应显示重启更新，实际 %q", tt.Texts())
	}
	if err := tt.Click("新版本 9.9.9 · 重启更新"); err != nil {
		t.Fatal(err)
	}
	if *relaunched != 1 {
		t.Fatalf("点击后应重启一次，实际 %d", *relaunched)
	}
}

// 关闭自动下载：只提示一次，停在「有新版本」；之后每小时再查到同一版本不再重复提示。
func TestUpdateManualDownloadPromptsOnce(t *testing.T) {
	f := &fakeUpdater{enabled: true, up: newVersion()}
	a, _, _ := updApp(t, f)
	a.set.NoAutoUpdate = true
	a.checkUpdates(false)
	if a.upd.phase != upAvailable || f.installs != 0 {
		t.Fatalf("应停在有新版本: phase=%v installs=%d", a.upd.phase, f.installs)
	}
	if len(a.toastActs) != 1 || a.toastActs[0].label != "下载" {
		t.Fatalf("应提示一次，实际 %+v", a.toastActs)
	}
	a.toastActs = nil
	a.checkUpdates(false) // 下一个小时又查到同一版本
	if f.checks != 2 || len(a.toastActs) != 0 {
		t.Fatalf("同一版本不应重复提示: checks=%d toasts=%+v", f.checks, a.toastActs)
	}
	// 在提示里点「下载」开始下载
	a.toastActs = nil
	a.downloadUpdate()
	if a.upd.phase != upReady || f.installs != 1 {
		t.Fatalf("手动下载应完成: phase=%v installs=%d", a.upd.phase, f.installs)
	}
}

// 在设置里重新打开自动下载：已发现的新版本马上开始下载。
func TestUpdateToggleAutoDownloadStartsDownload(t *testing.T) {
	f := &fakeUpdater{enabled: true, up: newVersion()}
	a, tt, _ := updApp(t, f)
	a.set.NoAutoUpdate = true
	a.checkUpdates(false)
	a.overlays.prefs.open = true
	a.overlays.prefs.section = 4
	tt.Frame()
	snapshot(t, tt, "upd_about")
	if !tt.HasText("发现新版本 9.9.9") || !tt.HasText("修复若干问题") {
		t.Fatalf("关于页应显示新版本和说明，实际 %q", tt.Texts())
	}
	if err := tt.Click("下载更新"); err != nil {
		t.Fatal(err)
	}
	if a.upd.phase != upReady {
		t.Fatalf("点击下载更新后应下载完成，实际 %v", a.upd.phase)
	}
}

// 下载失败：后台状态变为失败并保留版本，可重试；重试成功后进入就绪。
func TestUpdateInstallFailureCanRetry(t *testing.T) {
	f := &fakeUpdater{enabled: true, up: newVersion(), installErr: errors.New("签名校验失败")}
	a, tt, _ := updApp(t, f)
	a.checkUpdates(false)
	if a.upd.phase != upFailed || a.upd.err != "签名校验失败" {
		t.Fatalf("应记录失败: phase=%v err=%q", a.upd.phase, a.upd.err)
	}
	tt.Frame()
	if !tt.HasText("更新下载失败 · 点击重试") {
		t.Fatalf("标题栏应提示重试，实际 %q", tt.Texts())
	}
	f.installErr = nil
	a.updateAction()
	if a.upd.phase != upReady || f.installs != 2 {
		t.Fatalf("重试后应成功: phase=%v installs=%d", a.upd.phase, f.installs)
	}
}

// 检查失败：后台检查只记日志、不打扰；手动检查要把错误显示出来。
func TestUpdateCheckFailure(t *testing.T) {
	f := &fakeUpdater{enabled: true, err: errors.New("网络不可用")}
	a, tt, _ := updApp(t, f)
	a.checkUpdates(false)
	if a.upd.phase != upIdle || a.upd.err != "" {
		t.Fatalf("后台检查失败不应显示: phase=%v err=%q", a.upd.phase, a.upd.err)
	}
	a.checkUpdates(true)
	if a.upd.phase != upFailed || a.upd.err != "网络不可用" {
		t.Fatalf("手动检查失败应显示: phase=%v err=%q", a.upd.phase, a.upd.err)
	}
	tt.Frame()
	if tt.HasText("点击重试") {
		t.Fatal("检查失败（没有可下载的版本）不应在标题栏打扰")
	}
}

// 已是最新：手动检查后显示最新状态，不弹提示。
func TestUpdateUpToDate(t *testing.T) {
	a, tt, _ := updApp(t, &fakeUpdater{enabled: true})
	a.checkUpdates(true)
	if a.upd.phase != upUpToDate || a.upd.checkedAt.IsZero() || len(a.toastActs) != 0 {
		t.Fatalf("应为最新: phase=%v toasts=%+v", a.upd.phase, a.toastActs)
	}
	a.overlays.prefs.open = true
	a.overlays.prefs.section = 4
	tt.Frame()
	if !tt.HasText("已是最新版本") || !tt.HasText("检查更新") {
		t.Fatalf("关于页应显示最新状态与检查按钮，实际 %q", tt.Texts())
	}
}

// 标题栏提示随阶段变化：下载中显示进度，下载好显示重启，没有更新什么也不画。
func TestUpdateChipFollowsPhase(t *testing.T) {
	a, tt, _ := updApp(t, &fakeUpdater{enabled: true})
	a.upd.version = "9.9.9"
	a.upd.total.Store(100)
	a.upd.done.Store(40)
	for phase, want := range map[updatePhase]string{
		upDownloading: "正在下载更新 40%", upReady: "新版本 9.9.9 · 重启更新", upAvailable: "新版本 9.9.9 · 点击下载",
	} {
		a.upd.phase = phase
		tt.Frame()
		if !tt.HasText(want) {
			t.Fatalf("阶段 %v 应显示 %q，实际 %q", phase, want, tt.Texts())
		}
	}
	a.upd.phase = upUpToDate
	tt.Frame()
	if tt.HasText("重启更新") {
		t.Fatal("已是最新时不应显示更新提示")
	}
}

// 没有下载好就不会重启。
func TestUpdateApplyRequiresReady(t *testing.T) {
	a, _, relaunched := updApp(t, &fakeUpdater{enabled: true})
	a.upd.phase = upAvailable
	a.applyUpdate()
	if *relaunched != 0 {
		t.Fatal("没下载好不应重启")
	}
}

func TestHeadLines(t *testing.T) {
	if got := headLines("a\nb\nc", 2); got != "a\nb\n…" {
		t.Fatalf("got %q", got)
	}
	if got := headLines("  a\nb  ", 5); got != "a\nb" {
		t.Fatalf("got %q", got)
	}
}
