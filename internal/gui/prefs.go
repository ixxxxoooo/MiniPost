package gui

import (
	"time"

	"github.com/egoist/mygo/ui"

	"minipost/internal/pkg/logger"
)

// prefsState 是设置对话框的状态。
type prefsState struct {
	open    bool
	tab     int
	timeout float64
	maxSize float64
	backup  float64
}

func (a *app) openPrefs() {
	a.overlays.prefs = prefsState{
		open:    true,
		timeout: float64(a.set.RequestTimeoutMs),
		maxSize: float64(a.set.MaxResponseSizeMB),
		backup:  float64(a.set.AutoBackupIntervalMinutes),
	}
}

// 设置里下拉框的选项与对应的存储值。
var (
	localeValues = []string{"", localeZH, localeEN}
	themeValues  = []string{"system", "light", "dark"}
	httpValues   = []string{"auto", "http1", "http2"}
	detectValues = []string{"auto", "json"}
)

func indexOf(values []string, v string) int {
	for i, x := range values {
		if x == v {
			return i
		}
	}
	return 0
}

// choice 绘制把存储值映射到文案的下拉框，返回是否变化。
func choice(c *ui.Context, value *string, values, labels []string) bool {
	i := indexOf(values, *value)
	if selectIndex(c, &i, labels).Changed() {
		*value = values[i]
		return true
	}
	return false
}

// prefsView 绘制设置对话框；任何修改立即生效并保存。
func (a *app) prefsView(c *ui.Context) {
	s := &a.overlays.prefs
	ui.Modal(c, &s.open, func() {
		changed := false
		ui.Text(c, a.t("设置", "Settings")).Bold()
		ui.Tabs(c, &s.tab, a.t("通用", "General"), a.t("请求", "Requests"), a.t("备份", "Backup"))
		ui.Divider(c)
		ui.Column(c).Width(520).MinHeight(260).Children(func() {
			ui.Form(c, func() {
				switch s.tab {
				case 0:
					changed = a.prefsGeneral(c, s) || changed
				case 1:
					changed = a.prefsRequests(c, s) || changed
				default:
					changed = a.prefsBackup(c, s) || changed
				}
			})
		})
		ui.Row(c).Justify(ui.End).Children(func() {
			if ui.PrimaryButton(c, a.t("完成", "Done")).Clicked() {
				s.open = false
			}
		})
		if changed {
			a.later(a.applySettings)
		}
	})
}

func (a *app) prefsGeneral(c *ui.Context, s *prefsState) bool {
	changed := false
	ui.Field(c, a.t("语言", "Language"), func() {
		changed = choice(c, &a.set.Locale, localeValues, []string{a.t("跟随系统", "System"), "简体中文", "English"}) || changed
	})
	ui.Field(c, a.t("外观", "Appearance"), func() {
		changed = choice(c, &a.set.Theme, themeValues, []string{a.t("跟随系统", "System"), a.t("浅色", "Light"), a.t("深色", "Dark")}) || changed
	})
	ui.Field(c, a.t("字号", "Font size"), func() {
		size := float64(a.set.FontSize)
		if ui.NumberInput(c, &size, 11, 20, 1).Changed() {
			a.set.FontSize = int(size)
			changed = true
		}
	})
	ui.Field(c, a.t("请求/响应布局", "Layout"), func() {
		dir := 0
		if a.set.LayoutDirection == "horizontal" {
			dir = 1
		}
		if ui.Segmented(c, &dir, a.t("上下", "Stacked"), a.t("左右", "Side by side")).Changed() {
			a.set.LayoutDirection = []string{"vertical", "horizontal"}[dir]
			changed = true
		}
	})
	ui.Field(c, a.t("关闭未保存标签", "Unsaved tabs"), func() {
		changed = ui.Checkbox(c, &a.set.AlwaysSaveUnsaved, a.t("关闭时总是保存", "Always save on close")).Changed() || changed
		changed = ui.Checkbox(c, &a.set.AlwaysDiscardUnsaved, a.t("关闭时总是丢弃", "Always discard on close")).Changed() || changed
	})
	return changed
}

func (a *app) prefsRequests(c *ui.Context, s *prefsState) bool {
	changed := false
	ui.Field(c, a.t("网络", "Network"), func() {
		changed = ui.Checkbox(c, &a.set.FollowRedirects, a.t("自动跟随重定向", "Follow redirects")).Changed() || changed
		changed = ui.Checkbox(c, &a.set.SSLVerify, a.t("校验 SSL 证书", "Verify SSL certificates")).Changed() || changed
		changed = ui.Checkbox(c, &a.set.DisableCookies, a.t("禁用 Cookies", "Disable cookies")).Changed() || changed
	})
	ui.Field(c, "HTTP", func() {
		changed = choice(c, &a.set.HTTPVersion, httpValues, []string{a.t("自动", "Auto"), "HTTP/1.1", "HTTP/2"}) || changed
	})
	ui.Field(c, a.t("超时（毫秒，0 为不限）", "Timeout (ms, 0 = none)"), func() {
		if ui.NumberInput(c, &s.timeout, 0, 3_600_000, 1000).Changed() {
			a.set.RequestTimeoutMs = int(s.timeout)
			changed = true
		}
	})
	ui.Field(c, a.t("响应大小上限（MB，0 为不限）", "Max response size (MB, 0 = none)"), func() {
		if ui.NumberInput(c, &s.maxSize, 0, 2048, 10).Changed() {
			a.set.MaxResponseSizeMB = int(s.maxSize)
			changed = true
		}
	})
	ui.Field(c, a.t("请求头", "Headers"), func() {
		changed = ui.Checkbox(c, &a.set.SendNoCacheHeader, a.t("发送 no-cache 请求头", "Send no-cache header")).Changed() || changed
		changed = ui.Checkbox(c, &a.set.SendPostmanTokenHeader, a.t("发送 MiniPost-Token 请求头", "Send MiniPost-Token header")).Changed() || changed
	})
	ui.Field(c, a.t("JSON 识别", "JSON detection"), func() {
		changed = choice(c, &a.set.ResponseFormatDetection, detectValues, []string{a.t("自动", "Auto"), a.t("总是尝试", "Always try")}) || changed
	})
	return changed
}

func (a *app) prefsBackup(c *ui.Context, s *prefsState) bool {
	changed := false
	ui.Field(c, a.t("自动备份", "Auto backup"), func() {
		changed = ui.Checkbox(c, &a.set.AutoBackupEnabled, a.t("定时自动备份数据", "Back up data periodically")).Changed() || changed
	})
	ui.Field(c, a.t("备份间隔（分钟）", "Interval (minutes)"), func() {
		if ui.NumberInput(c, &s.backup, 5, 1440, 5).Changed() {
			a.set.AutoBackupIntervalMinutes = int(s.backup)
			changed = true
		}
	})
	ui.Field(c, a.t("手动", "Manual"), func() {
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, a.t("立即备份", "Back Up Now")).Clicked() {
				a.later(a.backupNow)
			}
			if ui.Button(c, a.t("从备份恢复…", "Restore…")).Clicked() {
				a.later(a.restoreBackup)
			}
		})
	})
	return changed
}

// applySettings 规整并保存设置，重建依赖设置的后台任务。
func (a *app) applySettings() {
	a.set.Normalize()
	a.saveSettings()
	a.restartBackupTimer()
	a.applied.ready = false
	logger.Info("设置已更新")
}

func (a *app) backupNow() {
	a.async(func() func() {
		path, err := a.be.CreateBackup()
		return func() {
			if err != nil {
				a.fail(a.t("创建备份", "Back up"), err)
				return
			}
			a.toast(a.t("备份已创建：", "Backup created: ") + path)
		}
	})
}

func (a *app) restoreBackup() {
	a.async(func() func() {
		path, err := a.be.OpenFileDialogAny(a.win)
		return func() {
			if err != nil {
				a.fail(a.t("选择备份", "Choose backup"), err)
				return
			}
			if path == "" {
				return
			}
			a.overlays.askConfirm(a.t("从备份恢复？", "Restore from backup?"),
				a.t("当前数据会被备份内容替换。", "Current data will be replaced by the backup."), a.t("恢复", "Restore"), func() {
					if err := a.be.RestoreBackup(path); err != nil {
						a.fail(a.t("恢复备份", "Restore backup"), err)
						return
					}
					a.reloadAll()
					a.toast(a.t("已从备份恢复", "Restored from backup"))
				})
		}
	})
}

// reloadAll 在数据被外部替换（恢复备份、导入）后重新加载全部内容。
func (a *app) reloadAll() {
	a.loadProjects()
	id := a.projectID
	a.projectID = ""
	for _, p := range a.projects {
		if p.ID == id {
			a.selectProject(id)
			return
		}
	}
	a.selectProject("")
}

// restartBackupTimer 按设置启停定时自动备份。
func (a *app) restartBackupTimer() {
	if a.backupStop != nil {
		close(a.backupStop)
		a.backupStop = nil
	}
	if !a.set.AutoBackupEnabled || !a.native {
		return
	}
	stop := make(chan struct{})
	a.backupStop = stop
	interval := time.Duration(a.set.AutoBackupIntervalMinutes) * time.Minute
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				if path, err := a.be.CreateBackup(); err != nil {
					logger.Warn("自动备份失败", "error", err.Error())
				} else {
					logger.Info("自动备份完成", "path", path)
				}
			}
		}
	}()
}
