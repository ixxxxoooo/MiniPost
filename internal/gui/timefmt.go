package gui

import (
	"fmt"
	"math"
)

// formatDuration 把毫秒数格式化为易读的耗时：不足 1 秒用“128ms”，
// 1 秒以上用“3.87s”，超过 1 分钟用“1m 05s”。
func formatDuration(ms float64) string {
	switch {
	case ms < 0 || math.IsNaN(ms):
		return "0ms"
	case ms < 1000:
		return fmt.Sprintf("%.0fms", ms)
	case ms < 60_000:
		return fmt.Sprintf("%.2fs", ms/1000)
	}
	s := int(math.Round(ms / 1000))
	return fmt.Sprintf("%dm %02ds", s/60, s%60)
}

// formatPhaseMs 是响应时间卡片里每个阶段右侧的耗时：不足 100ms 保留两位小数（0.27 ms），
// 100ms 以上取整（403 ms、3458 ms），避免出现“3458.00 ms”这样又长又没意义的小数。
func formatPhaseMs(ms float64) string {
	if ms < 0 || math.IsNaN(ms) {
		ms = 0
	}
	if ms >= 100 {
		return fmt.Sprintf("%.0f ms", ms)
	}
	return fmt.Sprintf("%.2f ms", ms)
}
