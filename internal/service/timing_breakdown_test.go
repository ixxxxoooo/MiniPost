package service

import (
	"math"
	"testing"
	"time"

	"minipost/internal/model"
)

func phaseSum(tm model.TimingBreakdown) float64 {
	return tm.Prepare + tm.SocketInitialization + tm.DNSLookup + tm.TCPHandshake +
		tm.SSLHandshake + tm.WaitingTTFB + tm.Download + tm.Process
}

// 新建 HTTPS 连接：各阶段之间故意留出空隙（如 DNS 结束到 TCP 开始），
// 阶段之和仍须等于总耗时，且每个阶段耗时符合预期。
func TestBuildTimingBreakdownTilesTotal(t *testing.T) {
	t0 := time.Now()
	at := func(ms float64) time.Time { return t0.Add(time.Duration(ms * float64(time.Millisecond))) }
	trace := requestTrace{
		requestStart: at(0),
		getConn:      at(1),
		dnsStart:     at(3),
		dnsDone:      at(10), // DNS 7ms
		connectStart: at(12), // 2ms 空隙，归入 TCP
		connectDone:  at(40),
		tlsStart:     at(41), // 1ms 空隙，归入 SSL
		tlsDone:      at(100),
		gotConn:      at(100),
		wroteRequest: at(102), // 2ms 空隙，归入等待
		firstByte:    at(500),
	}
	tm := buildTimingBreakdown(trace, at(520), at(525))

	want := map[string]float64{
		"Prepare": 1, "Socket": 2, "DNS": 7, "TCP": 30, "SSL": 60, "TTFB": 400, "Download": 20, "Process": 5,
	}
	got := map[string]float64{
		"Prepare": tm.Prepare, "Socket": tm.SocketInitialization, "DNS": tm.DNSLookup, "TCP": tm.TCPHandshake,
		"SSL": tm.SSLHandshake, "TTFB": tm.WaitingTTFB, "Download": tm.Download, "Process": tm.Process,
	}
	for k, w := range want {
		if math.Abs(got[k]-w) > 0.01 {
			t.Errorf("%s = %.3f，想要 %.3f", k, got[k], w)
		}
	}
	if math.Abs(tm.Total-525) > 0.01 {
		t.Fatalf("总耗时应为 525，实际 %.3f", tm.Total)
	}
	if math.Abs(phaseSum(tm)-tm.Total) > 0.01 {
		t.Fatalf("阶段之和 %.3f 应等于总耗时 %.3f", phaseSum(tm), tm.Total)
	}
}

// 复用连接：没有 DNS / TCP / SSL，这三项为 0，其余阶段依然首尾相接。
func TestBuildTimingBreakdownReusedConnection(t *testing.T) {
	t0 := time.Now()
	at := func(ms float64) time.Time { return t0.Add(time.Duration(ms * float64(time.Millisecond))) }
	trace := requestTrace{
		requestStart: at(0), getConn: at(0.5), gotConn: at(0.8), wroteRequest: at(1.2), firstByte: at(80),
	}
	tm := buildTimingBreakdown(trace, at(90), at(91))
	if tm.DNSLookup != 0 || tm.TCPHandshake != 0 || tm.SSLHandshake != 0 {
		t.Fatalf("复用连接不应有 DNS/TCP/SSL: %+v", tm)
	}
	if math.Abs(phaseSum(tm)-tm.Total) > 0.01 || math.Abs(tm.Total-91) > 0.01 {
		t.Fatalf("阶段之和 %.3f 应等于总耗时 %.3f(91)", phaseSum(tm), tm.Total)
	}
}

// 缺少事件（如出错提前结束）也不能出现负数，阶段之和仍等于总耗时。
func TestBuildTimingBreakdownMissingEvents(t *testing.T) {
	t0 := time.Now()
	trace := requestTrace{requestStart: t0}
	tm := buildTimingBreakdown(trace, t0.Add(30*time.Millisecond), t0.Add(31*time.Millisecond))
	for _, v := range []float64{tm.Prepare, tm.SocketInitialization, tm.DNSLookup, tm.TCPHandshake, tm.SSLHandshake, tm.WaitingTTFB, tm.Download, tm.Process} {
		if v < 0 {
			t.Fatalf("阶段耗时不应为负: %+v", tm)
		}
	}
	if math.Abs(phaseSum(tm)-tm.Total) > 0.01 {
		t.Fatalf("阶段之和 %.3f 应等于总耗时 %.3f", phaseSum(tm), tm.Total)
	}
}
