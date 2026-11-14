package app

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"minipost/internal/repository"
)

// ListBackups：只列常规备份（不含安全备份与无关文件），按时间从新到旧，并受 limit 限制。
func TestListBackups(t *testing.T) {
	store, err := repository.NewFileStoreWithDir(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := &App{store: store}

	// 目录不存在时返回空列表而不是报错
	if got, err := a.ListBackups(5); err != nil || len(got) != 0 {
		t.Fatalf("没有备份目录时应返回空列表，实际 %v err=%v", got, err)
	}

	dir := filepath.Join(store.BaseDir(), backupDirName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 7; i++ {
		name := fmt.Sprintf("%s202611%02d-120000.zip", backupFilePrefix, i)
		if err := os.WriteFile(filepath.Join(dir, name), make([]byte, i*10), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{safetyBackupPrefix + "20261120-000000.zip", "notes.txt", backupFilePrefix + "x.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	got, err := a.ListBackups(5)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 {
		t.Fatalf("应受 limit=5 限制，实际 %d 个", len(got))
	}
	if got[0].Name != backupFilePrefix+"20261107-120000.zip" || got[0].Size != 70 {
		t.Fatalf("第一个应是最新的备份，实际 %+v", got[0])
	}
	if got[4].Name != backupFilePrefix+"20261103-120000.zip" {
		t.Fatalf("顺序应从新到旧，实际末项 %s", got[4].Name)
	}
	if a.DataDir() != store.BaseDir() {
		t.Fatalf("DataDir 应为存储根目录，实际 %q", a.DataDir())
	}
}
