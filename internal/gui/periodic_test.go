package gui

import (
	"sync/atomic"
	"testing"
	"time"
)

// 第一次在 first 之后触发，之后按 interval 重复；stop 后不再触发，且可重复调用。
func TestStartPeriodic(t *testing.T) {
	var n atomic.Int32
	stop := startPeriodic(5*time.Millisecond, 5*time.Millisecond, func() { n.Add(1) })
	deadline := time.Now().Add(2 * time.Second)
	for n.Load() < 3 && time.Now().Before(deadline) {
		time.Sleep(2 * time.Millisecond)
	}
	if n.Load() < 3 {
		t.Fatalf("应周期触发，实际 %d 次", n.Load())
	}
	stop()
	stop()
	time.Sleep(20 * time.Millisecond) // 等待可能正在执行的最后一次结束
	at := n.Load()
	time.Sleep(40 * time.Millisecond)
	if n.Load() != at {
		t.Fatalf("stop 之后不应再触发: %d -> %d", at, n.Load())
	}
}

// 在第一次触发之前 stop，fn 一次都不会执行。
func TestStartPeriodicStopBeforeFirst(t *testing.T) {
	var n atomic.Int32
	stop := startPeriodic(50*time.Millisecond, 50*time.Millisecond, func() { n.Add(1) })
	stop()
	time.Sleep(120 * time.Millisecond)
	if n.Load() != 0 {
		t.Fatalf("不应触发，实际 %d 次", n.Load())
	}
}

// 周期任务 panic 后不会让进程退出，下一次仍会继续执行。
func TestStartPeriodicSurvivesPanic(t *testing.T) {
	var n atomic.Int32
	stop := startPeriodic(5*time.Millisecond, 5*time.Millisecond, func() {
		n.Add(1)
		panic("boom")
	})
	defer stop()
	deadline := time.Now().Add(2 * time.Second)
	for n.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if n.Load() < 2 {
		t.Fatalf("panic 之后周期任务应继续执行，只执行了 %d 次", n.Load())
	}
}
