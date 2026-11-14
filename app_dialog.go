package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/egoist/mygo"

	appErrors "minipost/internal/pkg/errors"
	"minipost/internal/pkg/logger"
)

// 文件对话框过滤器，文案与原 Wails 版本保持一致。
var (
	filterJSONYAML = []mygo.FileFilter{
		{Name: "JSON 文件 (*.json)", Extensions: []string{"json"}},
		{Name: "YAML 文件 (*.yaml;*.yml)", Extensions: []string{"yaml", "yml"}},
		{Name: "所有文件", Extensions: []string{"*"}},
	}
	filterJSON = []mygo.FileFilter{
		{Name: "JSON 文件 (*.json)", Extensions: []string{"json"}},
	}
	filterText = []mygo.FileFilter{
		{Name: "Shell 脚本 (*.sh)", Extensions: []string{"sh"}},
		{Name: "Text 文件 (*.txt)", Extensions: []string{"txt"}},
		{Name: "所有文件", Extensions: []string{"*"}},
	}
	filterAll = []mygo.FileFilter{{Name: "所有文件", Extensions: []string{"*"}}}
)

// pickOpenFile 弹出打开文件对话框，用户取消时返回空串且不报错。
func pickOpenFile(parent *mygo.Window, title string, filters []mygo.FileFilter) (string, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: parent, Title: title, Filters: filters})
	if err != nil {
		logger.Error("打开文件对话框失败", "title", title, "error", err.Error())
		return "", err
	}
	if len(paths) == 0 {
		logger.Debug("用户取消了打开文件对话框", "title", title)
		return "", nil
	}
	return paths[0], nil
}

// pickSaveFile 弹出保存文件对话框，用户取消时返回空串且不报错。
func pickSaveFile(parent *mygo.Window, title, defaultName string, filters []mygo.FileFilter) (string, error) {
	path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
		Parent: parent, Title: title, DefaultPath: defaultName, Filters: filters,
	})
	if err != nil {
		logger.Error("保存文件对话框失败", "title", title, "error", err.Error())
		return "", err
	}
	if path == "" {
		logger.Debug("用户取消了保存文件对话框", "title", title)
	}
	return path, nil
}

// OpenFileDialogJSON 打开文件选择对话框（JSON / YAML），返回文件内容；取消返回空串。
func (a *App) OpenFileDialogJSON(parent *mygo.Window) (string, error) {
	selection, err := pickOpenFile(parent, "选择文件", filterJSONYAML)
	if err != nil || selection == "" {
		return "", err
	}
	data, err := os.ReadFile(selection)
	if err != nil {
		logger.Error("读取导入文件失败", "error", err.Error())
		return "", fmt.Errorf("读取文件失败: %w", err)
	}
	logger.Info("已读取导入文件", "size", len(data))
	return string(data), nil
}

// SaveFileDialogJSON 打开保存对话框，将 JSON 内容写入所选文件。
func (a *App) SaveFileDialogJSON(parent *mygo.Window, defaultFilename, content string) error {
	return a.saveTextTo(parent, "保存文件", defaultFilename, filterJSON, content)
}

// SaveTextFile 保存文本内容（用于 cURL 等文本导出）。
func (a *App) SaveTextFile(parent *mygo.Window, defaultFilename, content string) error {
	return a.saveTextTo(parent, "保存文件", defaultFilename, filterText, content)
}

// OpenFileDialogAny 打开通用文件选择对话框，返回所选路径；取消返回空串。
func (a *App) OpenFileDialogAny(parent *mygo.Window) (string, error) {
	return pickOpenFile(parent, "选择文件", filterAll)
}

// SaveResponseToFile 保存响应体到用户选择的文件，支持 base64 二进制前缀。
func (a *App) SaveResponseToFile(parent *mygo.Window, defaultFilename, content string) error {
	selection, err := pickSaveFile(parent, "保存响应", defaultFilename, nil)
	if err != nil || selection == "" {
		return err
	}
	data := []byte(content)
	if strings.HasPrefix(content, binarySavePrefix) {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(content, binarySavePrefix))
		if err != nil {
			return appErrors.Wrap("INVALID_BINARY_PAYLOAD", "下载内容解析失败", err)
		}
		data = decoded
	}
	return writeSelectedFile(selection, data)
}

func (a *App) saveTextTo(parent *mygo.Window, title, defaultName string, filters []mygo.FileFilter, content string) error {
	selection, err := pickSaveFile(parent, title, defaultName, filters)
	if err != nil || selection == "" {
		return err
	}
	return writeSelectedFile(selection, []byte(content))
}

func writeSelectedFile(path string, data []byte) error {
	if err := os.WriteFile(path, data, 0o644); err != nil {
		logger.Error("写入文件失败", "error", err.Error())
		return err
	}
	logger.Info("文件已保存", "size", len(data))
	return nil
}
