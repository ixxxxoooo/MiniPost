package gui

import "testing"

// TestTrafficLightAlignedWithTitleBarIcons 校验红绿灯垂直中心与标题栏图标行中心基本一致。
// 系统红绿灯按钮高 14px，上边缘 trafficLightY，中心 = trafficLightY+7；
// 标题栏内容行高 sizeToolbar-1，图标行中心 = (sizeToolbar-1)/2。窗口坐标取整，允许相差不到 1px。
//
// @author ixxxxoooo
func TestTrafficLightAlignedWithTitleBarIcons(t *testing.T) {
	const trafficLightHeight = 14
	lightCenter := float32(trafficLightY) + trafficLightHeight/2
	iconCenter := float32(sizeToolbar-1) / 2
	if d := iconCenter - lightCenter; d < -0.5 || d > 0.5 {
		t.Fatalf("红绿灯中心 %.1f 与标题栏图标中心 %.1f 不在同一水平线", lightCenter, iconCenter)
	}
	if trafficLightX != 12 {
		t.Fatalf("红绿灯左边距应为设计稿的 12，实际 %d", trafficLightX)
	}
	// 侧栏开关必须落在三颗红绿灯（14×3 + 间距 6×2 = 54px）之后。
	if titleBarLeftMin < trafficLightX+54 {
		t.Fatalf("标题栏左边距 %d 不足以让出红绿灯", titleBarLeftMin)
	}
}
