package gui

import (
	"testing"
	"time"

	"minipost/internal/model"
)

func TestFormatDuration(t *testing.T) {
	cases := map[float64]string{
		-5:      "0ms",
		0:       "0ms",
		0.4:     "0ms",
		128:     "128ms",
		999.4:   "999ms",
		1000:    "1.00s",
		3870:    "3.87s",
		59_990:  "59.99s",
		60_000:  "1m 00s",
		125_400: "2m 05s",
	}
	for in, want := range cases {
		if got := formatDuration(in); got != want {
			t.Errorf("formatDuration(%v)=%q，想要 %q", in, got, want)
		}
	}
}

func TestFormatPhaseMs(t *testing.T) {
	cases := map[float64]string{
		0:       "0.00 ms",
		0.1:     "0.10 ms",
		2.444:   "2.44 ms",
		99.994:  "99.99 ms",
		100:     "100 ms",
		402.7:   "403 ms",
		3458.3:  "3458 ms",
		-1:      "0.00 ms",
		12345.6: "12346 ms",
	}
	for in, want := range cases {
		if got := formatPhaseMs(in); got != want {
			t.Errorf("formatPhaseMs(%v)=%q，想要 %q", in, got, want)
		}
	}
}

// 时间轴色块：起点按累计耗时，极短阶段也有最小宽度，且不会越过右端。
func TestTimelineBar(t *testing.T) {
	off, w := timelineBar(0, 100, 1000, 231)
	if off != 0 || w < 23 || w > 23.2 {
		t.Fatalf("起点阶段: off=%v w=%v", off, w)
	}
	if _, w := timelineBar(10, 0, 1000, 231); w != 2.8 {
		t.Fatalf("极短阶段应有最小宽度 2.8，实际 %v", w)
	}
	// 累计已超过总时长：色块贴着右端，不越界
	off, w = timelineBar(1200, 50, 1000, 231)
	if off+w > 231.001 || off < 0 {
		t.Fatalf("不应越过时间轴: off=%v w=%v", off, w)
	}
}

// 悬停到耗时上弹出响应时间卡片：总耗时用秒，各阶段按量级取整，并带有虚线框的 TTFB 一行。
func TestTimingCardMatchesReference(t *testing.T) {
	_, tt, _, _ := sendWith(t, &model.HttpResponse{
		StatusCode: 200, StatusText: "OK", ContentType: "text/plain", Body: "ok",
		Duration: 3870, Size: 441,
		Timings: model.TimingBreakdown{
			Prepare: 0.10, SocketInitialization: 0.27, DNSLookup: 2.44, TCPHandshake: 0.27,
			SSLHandshake: 402.7, WaitingTTFB: 3458.3, Download: 0.63, Process: 0, Total: 3870,
		},
		SizeDetails: model.SizeBreakdown{ResponseHeaders: 300, ResponseBody: 141, ResponseTotal: 441},
	})
	if !tt.HasText("3.87s") {
		t.Fatalf("标签行耗时应显示 3.87s: %q", tt.Texts())
	}
	r, _ := tt.Find("3.87s")
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	tt.Advance(time.Second)
	tt.Frame()
	for _, want := range []string{"响应时间", "准备", "Socket 初始化", "DNS 查询", "TCP 握手", "SSL 握手", "等待首字节 (TTFB)", "下载", "处理",
		"0.10 ms", "0.27 ms", "2.44 ms", "403 ms", "3458 ms", "0.63 ms", "0.00 ms"} {
		if !tt.HasText(want) {
			t.Errorf("响应时间卡片缺少 %q: %q", want, tt.Texts())
		}
	}
	// 卡片标题行的总耗时也是 3.87s（与标签行各一处）
	n := 0
	for _, s := range tt.Texts() {
		if s == "3.87s" {
			n++
		}
	}
	if n < 2 {
		t.Fatalf("卡片顶部应再显示一次总耗时 3.87s，实际出现 %d 次", n)
	}
	if tt.HasText("3458.00 ms") || tt.HasText("403.00 ms") {
		t.Fatal("大于 100ms 的阶段不应带两位小数")
	}
}

// 响应体为空（如 204）时，响应大小卡片只算响应头，不能把总大小又当成响应体重复计一次。
func TestSizeCardEmptyBodyNotDoubleCounted(t *testing.T) {
	_, tt, _, _ := sendWith(t, &model.HttpResponse{
		StatusCode: 204, StatusText: "No Content", Duration: 12, Size: 120,
		SizeDetails: model.SizeBreakdown{ResponseHeaders: 120, ResponseBody: 0, ResponseTotal: 120, RequestHeaders: 90, RequestBody: 0, RequestTotal: 90},
	})
	r, ok := tt.Find("120 B")
	if !ok {
		t.Fatalf("标签行应显示响应大小 120 B: %q", tt.Texts())
	}
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	tt.Advance(time.Second)
	tt.Frame()
	count := func(s string) int {
		n := 0
		for _, x := range tt.Texts() {
			if x == s {
				n++
			}
		}
		return n
	}
	// 120 B：标签行 + 卡片标题总计 + 响应头一行 = 3；响应体一行是 0 B
	if got := count("120 B"); got != 3 {
		t.Fatalf("120 B 应出现 3 次（标签行、卡片总计、响应头），实际 %d: %q", got, tt.Texts())
	}
	if count("0 B") < 2 {
		t.Fatalf("响应体与请求体都应是 0 B: %q", tt.Texts())
	}
}

// 没有分项数据的旧记录：总大小算作响应体，不显示成 0。
func TestSizeCardLegacyWithoutDetails(t *testing.T) {
	_, tt, _, _ := sendWith(t, &model.HttpResponse{StatusCode: 200, StatusText: "OK", ContentType: "text/plain", Body: "hello", Duration: 5, Size: 500})
	r, ok := tt.Find("500 B")
	if !ok {
		t.Fatalf("应显示 500 B: %q", tt.Texts())
	}
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	tt.Advance(time.Second)
	tt.Frame()
	if !tt.HasText("响应体") {
		t.Fatalf("应弹出大小卡片: %q", tt.Texts())
	}
}

// 状态栏的耗时与响应标签行保持同一写法：超过 1 秒用秒。
func TestStatusBarUsesSecondsForSlowRequests(t *testing.T) {
	_, tt, _, _ := sendWith(t, &model.HttpResponse{StatusCode: 200, StatusText: "OK", ContentType: "text/plain", Body: "ok", Duration: 1731, Size: 2})
	if !tt.HasText("200 · 1.73s") {
		t.Fatalf("状态栏应显示 200 · 1.73s: %q", tt.Texts())
	}
}
