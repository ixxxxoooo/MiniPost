package gui

import "testing"

// TestTrafficLightAlignedWithTitleBarIcons 校验红绿灯垂直中心与标题栏图标行中心一致。
// 系统红绿灯按钮高 14px，上边缘 trafficLightY，中心 = trafficLightY+7；
// 标题栏内容行高 sizeToolbar-1，图标行中心 = (sizeToolbar-1)/2。
//
// @author ixxxxoooo
func TestTrafficLightAlignedWithTitleBarIcons(t *testing.T) {
	const trafficLightHeight = 14
	lightCenter := float32(trafficLightY) + trafficLightHeight/2
	iconCenter := float32(sizeToolbar-1) / 2
	if lightCenter != iconCenter {
		t.Fatalf("红绿灯中心 %.1f 与标题栏图标中心 %.1f 不在同一水平线", lightCenter, iconCenter)
	}
	if trafficLightX != 20 {
		t.Fatalf("红绿灯左边距应保持 20，实际 %d", trafficLightX)
	}
}
