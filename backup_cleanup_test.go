package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestCleanupBackups_KeepsNewestN(t *testing.T) {
	dir := t.TempDir()

	// 创建 25 个常规备份与 3 个安全备份，文件名带递增时间戳以保证排序稳定。
	for i := 1; i <= 25; i++ {
		name := fmt.Sprintf("%s202601%02d-000000.zip", backupFilePrefix, i)
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0644); err != nil {
			t.Fatalf("写入备份文件失败: %v", err)
		}
	}
	for i := 1; i <= 3; i++ {
		name := fmt.Sprintf("%s202601%02d-000000.zip", safetyBackupPrefix, i)
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0644); err != nil {
			t.Fatalf("写入安全备份文件失败: %v", err)
		}
	}
	// 一个无关文件，不应被清理。
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0644); err != nil {
		t.Fatalf("写入无关文件失败: %v", err)
	}

	removed := cleanupBackups(dir, backupFilePrefix, maxRegularBackupRetention)
	if removed != 25-maxRegularBackupRetention {
		t.Fatalf("应清理 %d 个常规备份, 实际 %d", 25-maxRegularBackupRetention, removed)
	}

	remaining := countFilesWithPrefix(t, dir, backupFilePrefix)
	if remaining != maxRegularBackupRetention {
		t.Fatalf("常规备份应保留 %d 个, 实际 %d", maxRegularBackupRetention, remaining)
	}

	// 安全备份不受常规清理影响。
	if got := countFilesWithPrefix(t, dir, safetyBackupPrefix); got != 3 {
		t.Fatalf("安全备份不应被常规清理影响, 实际剩余 %d", got)
	}

	// 保留的应是最新（时间戳最大）的那批。
	newest := fmt.Sprintf("%s202601%02d-000000.zip", backupFilePrefix, 25)
	if _, err := os.Stat(filepath.Join(dir, newest)); err != nil {
		t.Fatalf("最新备份不应被删除: %v", err)
	}
	oldest := fmt.Sprintf("%s202601%02d-000000.zip", backupFilePrefix, 1)
	if _, err := os.Stat(filepath.Join(dir, oldest)); !os.IsNotExist(err) {
		t.Fatal("最旧备份应被删除")
	}

	// 无关文件保留。
	if _, err := os.Stat(filepath.Join(dir, "notes.txt")); err != nil {
		t.Fatalf("无关文件不应被删除: %v", err)
	}
}

func TestCleanupBackups_NoopWhenWithinLimit(t *testing.T) {
	dir := t.TempDir()
	for i := 1; i <= 3; i++ {
		name := fmt.Sprintf("%s202601%02d-000000.zip", backupFilePrefix, i)
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0644); err != nil {
			t.Fatalf("写入备份文件失败: %v", err)
		}
	}
	if removed := cleanupBackups(dir, backupFilePrefix, maxRegularBackupRetention); removed != 0 {
		t.Fatalf("未超过上限时不应清理, 实际清理 %d", removed)
	}
}

func countFilesWithPrefix(t *testing.T, dir, prefix string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("读取目录失败: %v", err)
	}
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() && len(entry.Name()) >= len(prefix) && entry.Name()[:len(prefix)] == prefix {
			count++
		}
	}
	return count
}
