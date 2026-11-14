package gui

import (
	"sync"
	"time"
)

// startPeriodic 在后台周期执行 fn：first 之后执行第一次，之后每隔 interval 一次。
// 返回的 stop 可以重复调用，停止后不会再触发 fn。自动备份和自动检查更新共用它，
// 避免各自起 goroutine 加 ticker。fn 在后台线程里被调用，要碰界面状态须自己切回主线程。
func startPeriodic(first, interval time.Duration, fn func()) (stop func()) {
	done := make(chan struct{})
	var once sync.Once
	go func() {
		timer := time.NewTimer(first)
		defer timer.Stop()
		for {
			select {
			case <-done:
				return
			case <-timer.C:
				fn()
				timer.Reset(interval)
			}
		}
	}()
	return func() { once.Do(func() { close(done) }) }
}
