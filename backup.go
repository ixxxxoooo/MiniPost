package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"minipost/internal/pkg/logger"
)

const (
	backupDirName      = "backups"
	backupProjectsDir  = "projects"
	backupConfigFile   = "config.json"
	backupFilenameFmt  = "minipost-backup-20060102-150405.zip"
	backupFilePrefix   = "minipost-backup-"
	safetyBackupPrefix = "minipost-safety-"

	// 备份保留上限：超过数量的最旧备份会在创建新备份后被清理，避免备份目录无限增长。
	maxRegularBackupRetention = 20
	maxSafetyBackupRetention  = 5
)

// cleanupBackups 仅保留指定前缀的最新 keep 个 .zip 备份，删除其余更旧的备份。
// 文件名带有时间戳，可按字典序升序即时间先后排序。清理失败不影响主流程。
func cleanupBackups(backupsDir, prefix string, keep int) int {
	if keep <= 0 {
		return 0
	}
	entries, err := os.ReadDir(backupsDir)
	if err != nil {
		return 0
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, ".zip") {
			names = append(names, name)
		}
	}
	if len(names) <= keep {
		return 0
	}

	sort.Strings(names)
	removable := names[:len(names)-keep]
	removed := 0
	for _, name := range removable {
		if err := os.Remove(filepath.Join(backupsDir, name)); err == nil {
			removed++
		}
	}
	if removed > 0 {
		logger.Info("已清理过期备份", "prefix", prefix, "removed", removed, "kept", keep)
	}
	return removed
}

func shouldIncludeInBackup(relPath string) bool {
	rel := filepath.ToSlash(strings.TrimSpace(relPath))
	if rel == "." || rel == "" {
		return false
	}
	if rel == backupProjectsDir || strings.HasPrefix(rel, backupProjectsDir+"/") {
		return true
	}
	return rel == backupConfigFile
}

func (a *App) CreateBackup() (string, error) {
	baseDir := a.store.BaseDir()
	backupsDir := filepath.Join(baseDir, backupDirName)
	if err := os.MkdirAll(backupsDir, 0o755); err != nil {
		return "", fmt.Errorf("创建备份目录失败: %w", err)
	}

	backupPath := filepath.Join(backupsDir, time.Now().Format(backupFilenameFmt))
	if err := createBackupArchive(baseDir, backupPath); err != nil {
		return "", err
	}

	// 创建成功后清理超出保留上限的旧备份，避免（尤其是自动备份场景下）备份目录无限增长。
	cleanupBackups(backupsDir, backupFilePrefix, maxRegularBackupRetention)

	logger.Info("创建备份成功", "path", backupPath)
	return backupPath, nil
}

func createBackupArchive(baseDir, backupPath string) error {
	output, err := os.OpenFile(backupPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("创建备份文件失败: %w", err)
	}
	defer output.Close()

	zipWriter := zip.NewWriter(output)

	err = filepath.WalkDir(baseDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)

		if relPath == "." {
			return nil
		}
		if relPath == backupDirName || strings.HasPrefix(relPath, backupDirName+"/") {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if relPath == "logs" || strings.HasPrefix(relPath, "logs/") {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if !shouldIncludeInBackup(relPath) {
			return nil
		}

		fileInfo, err := entry.Info()
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(fileInfo)
		if err != nil {
			return err
		}
		header.Name = relPath
		header.Method = zip.Deflate

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(writer, file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("写入备份失败: %w", err)
	}

	if err := zipWriter.Close(); err != nil {
		return fmt.Errorf("关闭备份文件失败: %w", err)
	}
	return nil
}

func isAllowedRestorePath(name string) bool {
	normalized := filepath.ToSlash(filepath.Clean(strings.TrimSpace(name)))
	if normalized == "." || normalized == "" {
		return false
	}
	if normalized == backupProjectsDir || strings.HasPrefix(normalized, backupProjectsDir+"/") {
		return true
	}
	return normalized == backupConfigFile
}

func extractBackupArchive(backupPath, targetDir string) error {
	reader, err := zip.OpenReader(backupPath)
	if err != nil {
		return fmt.Errorf("打开备份文件失败: %w", err)
	}
	defer reader.Close()

	cleanTargetDir := filepath.Clean(targetDir)
	prefix := cleanTargetDir + string(os.PathSeparator)

	for _, file := range reader.File {
		if !isAllowedRestorePath(file.Name) {
			continue
		}

		cleanName := filepath.Clean(file.Name)
		targetPath := filepath.Join(cleanTargetDir, cleanName)
		cleanTargetPath := filepath.Clean(targetPath)
		if cleanTargetPath != cleanTargetDir && !strings.HasPrefix(cleanTargetPath, prefix) {
			return fmt.Errorf("备份文件包含非法路径: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(cleanTargetPath, 0o755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(cleanTargetPath), 0o755); err != nil {
			return err
		}

		readerFile, err := file.Open()
		if err != nil {
			return err
		}

		mode := file.Mode().Perm()
		if mode == 0 {
			mode = 0o644
		}
		outputFile, err := os.OpenFile(cleanTargetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
		if err != nil {
			readerFile.Close()
			return err
		}

		_, copyErr := io.Copy(outputFile, readerFile)
		closeErr := outputFile.Close()
		readerCloseErr := readerFile.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if readerCloseErr != nil {
			return readerCloseErr
		}
	}

	return nil
}

func copyFile(srcPath, dstPath string, mode os.FileMode) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return err
	}

	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return err
	}

	return nil
}

func copyDir(srcDir, dstDir string) error {
	return filepath.WalkDir(srcDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dstDir, relPath)

		if entry.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}
		mode := info.Mode().Perm()
		if mode == 0 {
			mode = 0o644
		}
		return copyFile(path, targetPath, mode)
	})
}

func (a *App) createSafetyBackup() {
	path, err := a.CreateBackup()
	if err != nil {
		logger.Warn("恢复前安全备份失败", "error", err.Error())
		return
	}

	base := filepath.Base(path)
	if !strings.HasPrefix(base, backupFilePrefix) {
		return
	}

	safetyName := strings.Replace(base, backupFilePrefix, safetyBackupPrefix, 1)
	safetyPath := filepath.Join(filepath.Dir(path), safetyName)
	if renameErr := os.Rename(path, safetyPath); renameErr != nil {
		logger.Warn("重命名安全备份失败", "from", path, "to", safetyPath, "error", renameErr.Error())
		return
	}

	// 安全备份同样设保留上限，避免多次恢复后无限堆积。
	cleanupBackups(filepath.Dir(safetyPath), safetyBackupPrefix, maxSafetyBackupRetention)
}

func (a *App) RestoreBackup(backupPath string) error {
	resolvedPath := strings.TrimSpace(backupPath)
	if resolvedPath == "" {
		return fmt.Errorf("请选择备份文件")
	}

	absPath, err := filepath.Abs(resolvedPath)
	if err != nil {
		return fmt.Errorf("解析备份路径失败: %w", err)
	}
	if _, err = os.Stat(absPath); err != nil {
		return fmt.Errorf("备份文件不存在: %w", err)
	}

	tempDir, err := os.MkdirTemp("", "minipost-restore-*")
	if err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tempDir)

	if err = extractBackupArchive(absPath, tempDir); err != nil {
		return fmt.Errorf("解压备份失败: %w", err)
	}

	projectsSrc := filepath.Join(tempDir, backupProjectsDir)
	configSrc := filepath.Join(tempDir, backupConfigFile)
	projectsExists := false
	if _, statErr := os.Stat(projectsSrc); statErr == nil {
		projectsExists = true
	}
	configExists := false
	if _, statErr := os.Stat(configSrc); statErr == nil {
		configExists = true
	}
	if !projectsExists && !configExists {
		return fmt.Errorf("备份文件不包含可恢复的数据")
	}

	a.createSafetyBackup()

	baseDir := a.store.BaseDir()
	if projectsExists {
		projectsDst := filepath.Join(baseDir, backupProjectsDir)
		if err = os.RemoveAll(projectsDst); err != nil {
			return fmt.Errorf("清理旧项目数据失败: %w", err)
		}
		if err = copyDir(projectsSrc, projectsDst); err != nil {
			return fmt.Errorf("恢复项目数据失败: %w", err)
		}
	}
	if configExists {
		if err = copyFile(configSrc, filepath.Join(baseDir, backupConfigFile), 0o644); err != nil {
			return fmt.Errorf("恢复配置文件失败: %w", err)
		}
	}

	logger.Info("恢复备份成功", "backupPath", absPath)
	return nil
}
