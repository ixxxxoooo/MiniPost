package main

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"minipost/internal/model"
	"minipost/internal/repository"
)

func newBackupApp(t *testing.T) *App {
	t.Helper()
	store, err := repository.NewFileStoreWithDir(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return &App{ctx: ctx, cancel: cancel, store: store}
}

func projectName(t *testing.T, a *App, id string) string {
	t.Helper()
	p, err := a.store.GetProject(id)
	if err != nil {
		t.Fatal(err)
	}
	return p.Name
}

// 备份后修改数据，再恢复：数据回到备份时的样子，并留下一份恢复前的安全备份。
func TestBackupRestoreRoundTrip(t *testing.T) {
	a := newBackupApp(t)
	if err := a.store.SaveProject(&model.Project{ID: "p1", Name: "原始名称"}); err != nil {
		t.Fatal(err)
	}
	path, err := a.CreateBackup()
	if err != nil {
		t.Fatal(err)
	}
	if err := a.store.SaveProject(&model.Project{ID: "p1", Name: "改过的名称"}); err != nil {
		t.Fatal(err)
	}
	if err := a.RestoreBackup(path); err != nil {
		t.Fatal(err)
	}
	if got := projectName(t, a, "p1"); got != "原始名称" {
		t.Fatalf("恢复后应回到备份时的数据，实际 %q", got)
	}
	matches, _ := filepath.Glob(filepath.Join(a.store.BaseDir(), backupDirName, safetyBackupPrefix+"*.zip"))
	if len(matches) != 1 {
		t.Fatalf("恢复前应留下一份安全备份，实际 %v", matches)
	}
}

// 备份里不包含备份目录本身和日志等无关内容，只含项目数据与配置。
func TestBackupOnlyIncludesProjectsAndConfig(t *testing.T) {
	a := newBackupApp(t)
	if err := a.store.SaveProject(&model.Project{ID: "p1", Name: "x"}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(a.store.BaseDir(), "cookies.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	path, err := a.CreateBackup()
	if err != nil {
		t.Fatal(err)
	}
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	for _, f := range r.File {
		if !isAllowedRestorePath(f.Name) {
			t.Fatalf("备份里不应包含 %q", f.Name)
		}
	}
}

// 恢复时拒绝带 ../ 的压缩包条目，防止写出数据目录之外。
func TestRestoreRejectsPathTraversal(t *testing.T) {
	a := newBackupApp(t)
	zipPath := filepath.Join(t.TempDir(), "evil.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, _ := zw.Create("projects/../../escape.txt")
	_, _ = w.Write([]byte("x"))
	_ = zw.Close()
	_ = f.Close()

	err = a.RestoreBackup(zipPath)
	if err == nil {
		t.Fatal("带路径穿越的备份应被拒绝")
	}
	if _, statErr := os.Stat(filepath.Join(filepath.Dir(a.store.BaseDir()), "escape.txt")); statErr == nil {
		t.Fatal("不应在数据目录之外写出文件")
	}
}

func TestRestoreInputValidation(t *testing.T) {
	a := newBackupApp(t)
	if err := a.RestoreBackup("   "); err == nil || !strings.Contains(err.Error(), "请选择") {
		t.Fatalf("空路径应提示选择备份文件，实际 %v", err)
	}
	if err := a.RestoreBackup(filepath.Join(t.TempDir(), "missing.zip")); err == nil {
		t.Fatal("文件不存在应报错")
	}
}
