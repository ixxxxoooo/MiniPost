package gui

import (
	"context"
	"strings"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"minipost/internal/pkg/logger"
)

// 自动更新：启动 10 秒后和之后每小时在后台检查；发现新版本就自动下载（设置里可以关），
// 下载完成后弹一条带「重启更新」按钮的提示，标题栏右侧也留着「新版本 x · 重启更新」，点一下重启进入新版本。
// 检查、下载全程都在后台，不弹窗。
// 更新用 mygo.Updater：下载后核对签名、原地替换应用，运行中的这一份不受影响，重启（或下次启动）才生效。

const (
	updateFirstDelay    = 10 * time.Second // 启动后多久做第一次检查
	updateCheckInterval = time.Hour        // 之后的检查间隔
	updateCheckTimeout  = 30 * time.Second
	updateInstallLimit  = 30 * time.Minute
)

type updatePhase int

const (
	upIdle        updatePhase = iota
	upChecking                // 正在查
	upAvailable               // 有新版本，还没下载（自动下载关闭时停在这里）
	upDownloading             // 正在下载
	upReady                   // 下载好了，等重启
	upUpToDate                // 查过了，已是最新
	upFailed                  // 检查或下载失败
)

// pendingUpdate 是检查到的一个新版本。Install 下载并替换磁盘上的应用，progress 报告已下载/总字节。
type pendingUpdate struct {
	Version string
	Notes   string
	Install func(ctx context.Context, progress func(done, total int64)) error
}

// updateBackend 抽象 mygo.Updater，测试里换成假的。
type updateBackend interface {
	Enabled() bool
	Check(ctx context.Context) (*pendingUpdate, error)
}

// mygoUpdater 是真实的更新后端。
type mygoUpdater struct{}

func (mygoUpdater) Enabled() bool { return mygo.Updater.Enabled() }

func (mygoUpdater) Check(ctx context.Context) (*pendingUpdate, error) {
	up, err := mygo.Updater.Check(ctx)
	if err != nil || up == nil {
		return nil, err
	}
	return &pendingUpdate{Version: up.Version, Notes: up.Notes, Install: up.Install}, nil
}

// updateState 是更新的状态。下载进度由下载线程写、界面读，所以用原子数。
type updateState struct {
	phase     updatePhase
	version   string
	notes     string
	err       string
	checkedAt time.Time
	up        *pendingUpdate
	done      atomic.Int64 // 已下载字节
	total     atomic.Int64
}

// percent 是下载进度，0～100；总大小未知时为 0。
func (u *updateState) percent() int {
	if t := u.total.Load(); t > 0 {
		return int(min(u.done.Load()*100/t, 100))
	}
	return 0
}

// startUpdater 开始后台检查：启动 10 秒后第一次，之后每小时一次。
func (a *app) startUpdater() {
	if !a.updater.Enabled() {
		logger.Info("[Updater] 当前构建不支持自动更新，跳过后台检查")
		return
	}
	a.stopUpdate = startPeriodic(updateFirstDelay, updateCheckInterval, func() {
		a.post(func() { a.checkUpdates(false) })
	})
	logger.Info("[Updater] 后台检查已启动", "interval", updateCheckInterval.String())
}

// stopUpdater 停止后台检查。
func (a *app) stopUpdater() {
	if a.stopUpdate != nil {
		a.stopUpdate()
		a.stopUpdate = nil
	}
}

// checkUpdates 检查更新。manual 为真是用户在「关于」里点的：失败要显示出来；后台检查失败只记日志。
func (a *app) checkUpdates(manual bool) {
	u := &a.upd
	switch u.phase {
	case upChecking, upDownloading, upReady:
		return
	}
	if !a.updater.Enabled() {
		return
	}
	u.phase, u.err = upChecking, ""
	var (
		up  *pendingUpdate
		err error
	)
	a.async(func() func() {
		ctx, cancel := context.WithTimeout(context.Background(), updateCheckTimeout)
		defer cancel()
		up, err = a.updater.Check(ctx)
		return func() { a.onChecked(up, err, manual) }
	})
}

// onChecked 在主线程处理检查结果。
func (a *app) onChecked(up *pendingUpdate, err error, manual bool) {
	u := &a.upd
	u.checkedAt = time.Now()
	switch {
	case err != nil:
		logger.Warn("[Updater] 检查更新失败", "error", err.Error())
		u.phase = upIdle
		if manual {
			u.phase, u.err = upFailed, err.Error()
		}
	case up == nil:
		u.phase = upUpToDate
		logger.Debug("[Updater] 已是最新版本")
	default:
		fresh := u.version != up.Version // 每小时都会查：同一个版本只提示一次
		u.phase, u.up, u.version, u.notes = upAvailable, up, up.Version, up.Notes
		logger.Info("[Updater] 发现新版本", "version", up.Version)
		switch {
		case !a.set.NoAutoUpdate:
			if fresh {
				a.toast(a.t("发现新版本 "+up.Version+"，正在后台下载…", "Version "+up.Version+" found, downloading in the background…"))
			}
			a.downloadUpdate()
		case fresh:
			a.toastAction(a.t("发现新版本 "+up.Version, "Version "+up.Version+" is available"), a.t("下载", "Download"), a.downloadUpdate)
		}
	}
}

// downloadUpdate 在后台下载并安装（替换磁盘上的应用）新版本，进度写进 u.done / u.total。
func (a *app) downloadUpdate() {
	u := &a.upd
	if u.up == nil || u.phase == upDownloading || u.phase == upReady {
		return
	}
	u.phase, u.err = upDownloading, ""
	u.done.Store(0)
	u.total.Store(0)
	up := u.up
	last := -1
	a.async(func() func() {
		ctx, cancel := context.WithTimeout(context.Background(), updateInstallLimit)
		defer cancel()
		err := up.Install(ctx, func(done, total int64) {
			u.done.Store(done)
			u.total.Store(total)
			if p := u.percent(); p != last { // 百分比变了才请求重绘
				last = p
				a.redraw()
			}
		})
		return func() { a.onInstalled(err) }
	})
}

// onInstalled 在主线程处理下载安装结果。
func (a *app) onInstalled(err error) {
	u := &a.upd
	if err != nil {
		logger.Warn("[Updater] 下载更新失败", "version", u.version, "error", err.Error())
		u.phase, u.err = upFailed, err.Error()
		return
	}
	logger.Info("[Updater] 新版本已下载，等待重启", "version", u.version)
	u.phase = upReady
	a.toastAction(a.t("新版本 "+u.version+" 已下载好", "Version "+u.version+" is ready"), a.t("重启更新", "Restart to update"), a.applyUpdate)
}

// applyUpdate 重启进入已经装好的新版本。
func (a *app) applyUpdate() {
	if a.upd.phase != upReady {
		return
	}
	logger.Info("[Updater] 重启以应用更新", "version", a.upd.version)
	a.persistSession() // 先保存标签会话（含未保存的修改），重启后恢复
	a.relaunch()
}

// updateAction 是点「更新提示」时该做的事：下载好了就重启，还没下载就下载，失败就重试下载或重新检查。
func (a *app) updateAction() {
	u := &a.upd
	switch u.phase {
	case upReady:
		a.applyUpdate()
	case upAvailable:
		a.downloadUpdate()
	case upFailed:
		if u.up != nil {
			a.downloadUpdate()
		} else {
			a.checkUpdates(true)
		}
	}
}

// redraw 请求重绘（后台线程可调用）；没有真实窗口（测试）时什么也不做。
func (a *app) redraw() {
	if a.win != nil {
		a.win.Invalidate()
	}
}

// updateChip 是标题栏右侧的更新提示：下载中显示进度，下载好了「重启更新」，点一下就更新。
// 没有更新、正在检查时什么也不画。
func (a *app) updateChip(c *ui.Context) {
	u := &a.upd
	p := a.p
	switch u.phase {
	case upDownloading:
		ui.Row(c.Key("upd-dl")).Gap(6).AlignItems(ui.Center).PaddingX(6).Children(func() {
			ui.Spinner(c).Size(11, 11)
			txt(c, a.t("正在下载更新 "+itoa(u.percent())+"%", "Downloading update "+itoa(u.percent())+"%"), fsTiny, p.fg2, 0)
		})
	case upReady, upAvailable, upFailed:
		label, ic := a.t("新版本 "+u.version+" · 重启更新", "Version "+u.version+" · Restart to update"), "download"
		switch u.phase {
		case upAvailable:
			label = a.t("新版本 "+u.version+" · 点击下载", "Version "+u.version+" · Click to download")
		case upFailed:
			if u.up == nil {
				return // 检查失败只在「关于」里说，不在主界面打扰
			}
			label = a.t("更新下载失败 · 点击重试", "Update failed · Click to retry")
		}
		bg := p.accent
		col := p.accentFg
		if u.phase == upFailed {
			bg, col = p.surface2, p.fg
		}
		if a.button(c, btnOpts{h: 24, radius: radiusBtn, padX: 10, gap: 6, bg: &bg, noHover: true, key: "upd-chip"}, func(bool) {
			icon(c, ic, 12, col)
			txt(c, label, fsSmall, col, 500)
		}) {
			a.later(a.updateAction)
		}
	}
}

// headLines 取文本的前 n 行，更新说明太长时截断。
func headLines(s string, n int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) <= n {
		return strings.Join(lines, "\n")
	}
	return strings.Join(lines[:n], "\n") + "\n…"
}

// appVersion 是当前版本号，来自打包信息（mygo.json）；开发运行或测试里没有时显示 dev。
func (a *app) appVersion() string {
	if !a.native {
		return "dev"
	}
	if v := mygo.App.Version(); v != "" {
		return v
	}
	return "dev"
}

// updateCard 是「关于」里的更新状态卡：当前状态、对应的操作，以及「自动下载更新」开关。
func (a *app) updateCard(c *ui.Context) {
	u := &a.upd
	p := a.p
	enabled := a.updater.Enabled()
	ui.Column(c.Key("upd-card")).FillWidth().Radius(11).Border(1, p.border).Padding(14).Gap(10).Children(func() {
		ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Children(func() {
			title, sub := a.t("已是最新版本", "You're up to date"), ""
			switch {
			case !enabled:
				title = a.t("此构建不支持自动更新", "Auto-update is unavailable in this build")
				sub = a.t("开发版或由包管理器安装的应用不会自己更新。", "Development builds and package-managed installs do not update themselves.")
			case u.phase == upChecking:
				title = a.t("正在检查更新…", "Checking for updates…")
			case u.phase == upAvailable:
				title = a.t("发现新版本 "+u.version, "Version "+u.version+" is available")
			case u.phase == upDownloading:
				title = a.t("正在下载 "+u.version+"（"+itoa(u.percent())+"%）", "Downloading "+u.version+" ("+itoa(u.percent())+"%)")
			case u.phase == upReady:
				title = a.t("新版本 "+u.version+" 已下载好", "Version "+u.version+" is ready")
				sub = a.t("重启后生效，当前打开的标签会自动恢复。", "Takes effect after restart; open tabs are restored automatically.")
			case u.phase == upFailed:
				title, sub = a.t("更新失败", "Update failed"), u.err
			case !u.checkedAt.IsZero():
				sub = a.t("上次检查 "+u.checkedAt.Format("15:04"), "Last checked "+u.checkedAt.Format("15:04"))
			}
			ui.Column(c).Grow(1).Basis(0).Gap(2).Children(func() {
				txt(c, title, fsLarge, p.fg, 500)
				if sub != "" {
					ui.Text(c, sub).FontSize(fsSmall).FixedLineHeight(18).TextColor(p.fg2).Selectable()
				}
			})
			switch {
			case !enabled, u.phase == upChecking, u.phase == upDownloading:
			case u.phase == upReady:
				if a.updateButton(c, "upd-restart", a.t("重启更新", "Restart to update"), true) {
					a.later(a.applyUpdate)
				}
			case u.phase == upAvailable:
				if a.updateButton(c, "upd-dl", a.t("下载更新", "Download"), true) {
					a.later(a.downloadUpdate)
				}
			case u.phase == upFailed && u.up != nil:
				if a.updateButton(c, "upd-retry", a.t("重试下载", "Retry"), false) {
					a.later(a.downloadUpdate)
				}
			default:
				if a.updateButton(c, "upd-check", a.t("检查更新", "Check for updates"), false) {
					a.later(func() { a.checkUpdates(true) })
				}
			}
		})
		if u.phase == upDownloading {
			ui.Progress(c, float64(u.percent())/100).FillWidth()
		}
		if u.notes != "" && (u.phase == upAvailable || u.phase == upDownloading || u.phase == upReady) {
			ui.Text(c, headLines(u.notes, 12)).FontSize(fsSmall).FixedLineHeight(18).TextColor(p.fg2).Selectable().FillWidth().
				Padding(8, 10).Radius(radiusBtn).Background(p.surface2)
		}
		if enabled {
			auto := !a.set.NoAutoUpdate
			ui.Row(c).FillWidth().Gap(14).AlignItems(ui.Center).Children(func() {
				ui.Column(c).Grow(1).Basis(0).Gap(2).Children(func() {
					txt(c, a.t("自动下载更新", "Download updates automatically"), fsBase, p.fg, 500)
					txt(c, a.t("每小时在后台检查一次，发现新版本自动下载，下载好后提示你重启。", "Checks hourly in the background and downloads new versions, then asks you to restart."), fsSmall, p.fg2, 0)
				})
				if a.toggle(c, "upd-auto", &auto) {
					a.set.NoAutoUpdate = !auto
					a.later(func() {
						a.saveSettings()
						if auto && a.upd.phase == upAvailable {
							a.downloadUpdate() // 刚打开自动下载：已经发现的新版本马上开始下载
						}
					})
				}
			})
		}
	})
}

// updateButton 是更新卡里的按钮；primary 为强调色填充。
func (a *app) updateButton(c *ui.Context, key, label string, primary bool) bool {
	p := a.p
	bg, col := p.surface2, p.fg
	if primary {
		bg, col = p.accent, p.accentFg
	}
	return a.button(c, btnOpts{h: 28, radius: radiusBtn, padX: 12, bg: &bg, noHover: primary, key: key}, func(bool) {
		txt(c, label, fsSmall, col, 500)
	})
}
