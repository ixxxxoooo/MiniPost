package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"minipost/internal/repository"
)

// 备份含环境变量、Cookie 等私密数据：目录和文件只对当前用户可读写，
// 即使备份目录之前以更宽的权限创建过也会被收紧。
func TestCreateBackupUsesPrivatePermissions(t *testing.T) {
	store, err := repository.NewFileStoreWithDir(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a := &App{ctx: ctx, cancel: cancel, store: store}

	dir := filepath.Join(store.BaseDir(), backupDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path, err := a.CreateBackup()
	if err != nil {
		t.Fatal(err)
	}
	di, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if di.Mode().Perm() != 0o700 {
		t.Fatalf("备份目录权限应为 0700，实际 %o", di.Mode().Perm())
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("备份文件权限应为 0600，实际 %o", fi.Mode().Perm())
	}
}

// 导出到用户选择的位置时，文件只对当前用户可读写。
func TestWriteSelectedFilePrivate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.json")
	if err := writeSelectedFile(path, []byte("{}")); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("导出文件权限应为 0600，实际 %o", fi.Mode().Perm())
	}
}
