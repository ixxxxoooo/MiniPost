package app

import (
	"context"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"regexp"
	"strings"
	"time"

	"minipost/internal/pkg/logger"
	"minipost/internal/service"
)

var swaggerUIURLPattern = regexp.MustCompile(`(?i)\burl\s*:\s*["']([^"']+)["']`)

const importPreviewMaxConflictSamples = 20

func (a *App) buildImportPreview(projectID, format, content, sourceURL string) (*service.ImportPreview, error) {
	kind, normalizedContent, err := resolveImportKind(format, content)
	if err != nil {
		return nil, err
	}

	var conflicts []service.ImportConflict
	switch kind {
	case "postman":
		conflicts, err = a.requestSvc.PreviewPostmanConflicts(projectID, []byte(normalizedContent))
	case "swagger":
		conflicts, err = a.requestSvc.PreviewSwaggerConflicts(projectID, []byte(normalizedContent), sourceURL)
	default:
		// 环境导入不涉及请求冲突
		conflicts = nil
	}
	if err != nil {
		logger.Error("导入冲突预检失败", "projectID", projectID, "kind", kind, "error", err.Error())
		return nil, err
	}

	preview := &service.ImportPreview{
		Kind:          kind,
		ResolvedURL:   sourceURL,
		ConflictCount: len(conflicts),
		Conflicts:     conflicts,
	}
	if len(preview.Conflicts) > importPreviewMaxConflictSamples {
		preview.Conflicts = preview.Conflicts[:importPreviewMaxConflictSamples]
	}
	logger.Info("导入冲突预检完成", "projectID", projectID, "kind", kind, "conflictCount", preview.ConflictCount)
	return preview, nil
}

// PreviewImportFromFile 预检文件导入内容与项目中已有请求的 URL 冲突
func (a *App) PreviewImportFromFile(projectID, format, content string) (*service.ImportPreview, error) {
	return a.buildImportPreview(projectID, format, content, "")
}

// PreviewImportFromURL 拉取远程导入内容并预检 URL 冲突；返回内容供后续导入复用，避免二次拉取
func (a *App) PreviewImportFromURL(projectID, format, sourceURL string) (*service.ImportPreview, error) {
	trimmedURL := strings.TrimSpace(sourceURL)
	if trimmedURL == "" {
		return nil, fmt.Errorf("导入地址不能为空")
	}

	parsedURL, err := neturl.Parse(trimmedURL)
	if err != nil {
		return nil, fmt.Errorf("导入地址无效: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("仅支持 http/https 导入地址")
	}

	content, resolvedURL, err := fetchRemoteImportContent(a.ctx, trimmedURL)
	if err != nil {
		return nil, err
	}

	preview, err := a.buildImportPreview(projectID, format, content, resolvedURL)
	if err != nil {
		return nil, err
	}
	preview.Content = content
	preview.ResolvedURL = resolvedURL
	return preview, nil
}

// ImportCollectionWithStrategy 按指定冲突策略导入集合内容（strategy: update / copy / overwrite）
func (a *App) ImportCollectionWithStrategy(projectID, format, content, sourceURL, strategy string) error {
	return a.importCollectionContentWithStrategy(projectID, format, content, sourceURL, strategy)
}

func fetchRemoteImportContent(ctx context.Context, sourceURL string) (string, string, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	fetch := func(targetURL string) (string, string, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
		if err != nil {
			return "", "", fmt.Errorf("创建导入请求失败: %w", err)
		}
		req.Header.Set("Accept", "application/json, application/yaml, text/yaml, text/plain, text/html, */*")
		req.Header.Set("User-Agent", "MiniPost/1.0")

		resp, err := client.Do(req)
		if err != nil {
			return "", "", fmt.Errorf("拉取导入地址失败: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "", "", fmt.Errorf("拉取导入地址失败: HTTP %d", resp.StatusCode)
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
		if err != nil {
			return "", "", fmt.Errorf("读取导入内容失败: %w", err)
		}

		finalURL := targetURL
		if resp.Request != nil && resp.Request.URL != nil {
			finalURL = resp.Request.URL.String()
		}
		content := strings.TrimSpace(string(body))
		if content == "" {
			return "", "", fmt.Errorf("导入地址返回内容为空")
		}
		return content, finalURL, nil
	}

	content, finalURL, err := fetch(sourceURL)
	if err != nil {
		return "", "", err
	}
	if strings.Contains(strings.ToLower(content), "swaggeruibundle") {
		if matches := swaggerUIURLPattern.FindStringSubmatch(content); len(matches) == 2 {
			baseURL, parseErr := neturl.Parse(finalURL)
			if parseErr != nil {
				return content, finalURL, nil
			}
			refURL, resolveErr := baseURL.Parse(strings.TrimSpace(matches[1]))
			if resolveErr != nil {
				return content, finalURL, nil
			}
			return fetch(refURL.String())
		}
	}

	return content, finalURL, nil
}

func (a *App) ImportFromFile(projectID, format, content string) error {
	return a.importCollectionContent(projectID, format, content, "")
}

func (a *App) ImportFromURL(projectID, format, sourceURL string) error {
	trimmedURL := strings.TrimSpace(sourceURL)
	if trimmedURL == "" {
		return fmt.Errorf("导入地址不能为空")
	}

	parsedURL, err := neturl.Parse(trimmedURL)
	if err != nil {
		return fmt.Errorf("导入地址无效: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("仅支持 http/https 导入地址")
	}

	content, resolvedURL, err := fetchRemoteImportContent(a.ctx, trimmedURL)
	if err != nil {
		return err
	}

	logger.Info("开始从 URL 导入集合", "projectID", projectID, "url", logger.RedactURL(trimmedURL), "resolvedURL", logger.RedactURL(resolvedURL), "size", len(content))
	return a.importCollectionContent(projectID, format, content, resolvedURL)
}
