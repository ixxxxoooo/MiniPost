package logger

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLevelFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		fallback Level
		want     Level
	}{
		{name: "empty uses fallback", value: "", fallback: INFO, want: INFO},
		{name: "debug", value: "debug", fallback: INFO, want: DEBUG},
		{name: "warning alias", value: "warning", fallback: DEBUG, want: WARN},
		{name: "invalid uses fallback", value: "verbose", fallback: ERROR, want: ERROR},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := levelFromEnv(tt.value, tt.fallback); got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestRedactURL(t *testing.T) {
	got := RedactURL("https://example.com/api?access_token=abc&name=alice&password=secret")
	want := "https://example.com/api?access_token=%5BREDACTED%5D&name=alice&password=%5BREDACTED%5D"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestRedactRequestTargetCurl(t *testing.T) {
	got := RedactRequestTarget("curl https://example.com -H 'Cookie: sid=abc'")
	if got != "<curl command: 45 chars>" {
		t.Fatalf("expected cURL command to be summarized, got %q", got)
	}
}

func TestCleanupOldLogs(t *testing.T) {
	dir := t.TempDir()
	oldLog := filepath.Join(dir, "minipost-2000-01-01.log")
	currentLog := filepath.Join(dir, "minipost-9999-01-01.log")
	nonLog := filepath.Join(dir, "notes.txt")

	for _, p := range []string{oldLog, currentLog, nonLog} {
		if err := os.WriteFile(p, []byte("x"), 0644); err != nil {
			t.Fatalf("写入测试文件失败: %v", err)
		}
	}

	staleTime := time.Now().AddDate(0, 0, -30)
	if err := os.Chtimes(oldLog, staleTime, staleTime); err != nil {
		t.Fatalf("修改旧日志时间失败: %v", err)
	}
	if err := os.Chtimes(nonLog, staleTime, staleTime); err != nil {
		t.Fatalf("修改非日志文件时间失败: %v", err)
	}

	// 当前日志文件不应被清理
	prevLogPath := logPath
	logPath = currentLog
	defer func() { logPath = prevLogPath }()

	removed := cleanupOldLogs(dir, 14)
	if removed != 1 {
		t.Fatalf("应仅清理 1 个旧日志, 实际 %d", removed)
	}
	if _, err := os.Stat(oldLog); !os.IsNotExist(err) {
		t.Fatal("旧日志应被删除")
	}
	if _, err := os.Stat(currentLog); err != nil {
		t.Fatal("当前日志不应被删除")
	}
	if _, err := os.Stat(nonLog); err != nil {
		t.Fatal("非日志文件不应被删除")
	}
}
