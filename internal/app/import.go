package app

import (
	"bytes"
	"encoding/json"
	"fmt"

	"minipost/internal/model"
	"minipost/internal/pkg/httputil"
	"minipost/internal/pkg/jsonutil"
	"minipost/internal/pkg/logger"
	"minipost/internal/service"

	"gopkg.in/yaml.v3"
)

// ---- 导入导出 ----

func (a *App) ExportProjectJSON(projectID string) (string, error) {
	data, err := a.requestSvc.ExportProjectJSON(projectID)
	if err != nil {
		logger.Error("导出项目失败", "projectID", projectID, "error", err.Error())
		return "", err
	}
	logger.Info("导出项目成功", "projectID", projectID, "size", len(data))
	return string(data), nil
}

// ExportNodeJSON 导出项目里的单个请求或文件夹为 Postman Collection JSON。
func (a *App) ExportNodeJSON(projectID, nodeID string) (string, error) {
	data, err := a.requestSvc.ExportNodeJSON(projectID, nodeID)
	if err != nil {
		logger.Error("导出节点失败", "projectID", projectID, "nodeID", nodeID, "error", err.Error())
		return "", err
	}
	logger.Info("导出节点成功", "projectID", projectID, "nodeID", nodeID, "size", len(data))
	return string(data), nil
}

func (a *App) ImportPostmanCollection(projectID, jsonStr string) error {
	if err := a.requestSvc.ImportPostmanCollection(projectID, []byte(jsonStr)); err != nil {
		logger.Error("导入 Postman 集合失败", "projectID", projectID, "error", err.Error())
		return err
	}
	logger.Info("导入 Postman 集合成功", "projectID", projectID)
	return nil
}

func (a *App) ImportPostmanEnvironment(projectID, jsonStr string) error {
	if err := a.requestSvc.ImportPostmanEnvironment(projectID, []byte(jsonStr)); err != nil {
		logger.Error("导入 Postman Environment 失败", "projectID", projectID, "error", err.Error())
		return err
	}
	logger.Info("导入 Postman Environment 成功", "projectID", projectID)
	return nil
}

func (a *App) ImportSwagger(projectID, jsonStr string) error {
	if err := a.requestSvc.ImportSwaggerWithSource(projectID, []byte(jsonStr), ""); err != nil {
		logger.Error("导入 Swagger 失败", "projectID", projectID, "error", err.Error())
		return err
	}
	logger.Info("导入 Swagger 成功", "projectID", projectID)
	return nil
}

func parseImportContentAsObject(raw string) (map[string]json.RawMessage, string, error) {
	var data map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &data); err == nil {
		return data, raw, nil
	}
	cleaned := jsonutil.StripComments([]byte(raw))
	if !bytes.Equal(cleaned, []byte(raw)) {
		if err := json.Unmarshal(cleaned, &data); err == nil {
			return data, raw, nil
		}
	}

	// fallback: YAML -> JSON
	var yamlObj map[string]interface{}
	if err := yaml.Unmarshal([]byte(raw), &yamlObj); err != nil {
		return nil, "", fmt.Errorf("无法解析 JSON/YAML: %w", err)
	}
	normalized, err := json.Marshal(yamlObj)
	if err != nil {
		return nil, "", fmt.Errorf("YAML 转换失败: %w", err)
	}
	if err := json.Unmarshal(normalized, &data); err != nil {
		return nil, "", fmt.Errorf("转换后的 JSON 解析失败: %w", err)
	}
	return data, string(normalized), nil
}

// resolveImportKind 解析导入内容的实际类型（postman / postman-environment / swagger），并返回规范化后的内容
func resolveImportKind(format, content string) (string, string, error) {
	switch format {
	case "postman":
		return "postman", content, nil
	case "postman-environment":
		return "postman-environment", content, nil
	case "swagger", "openapi":
		return "swagger", content, nil
	default:
		// 自动检测格式
		raw, normalizedContent, err := parseImportContentAsObject(content)
		if err != nil {
			return "", "", err
		}
		if _, ok := raw["info"]; ok {
			if _, ok2 := raw["item"]; ok2 {
				return "postman", normalizedContent, nil
			}
		}
		if _, ok := raw["values"]; ok {
			if _, hasItem := raw["item"]; !hasItem {
				return "postman-environment", normalizedContent, nil
			}
		}
		if _, ok := raw["paths"]; ok {
			return "swagger", normalizedContent, nil
		}
		return "", "", fmt.Errorf("无法识别文件格式，请选择正确的导入格式")
	}
}

func (a *App) importCollectionContent(projectID, format, content, sourceURL string) error {
	return a.importCollectionContentWithStrategy(projectID, format, content, sourceURL, service.ImportStrategyCopy)
}

func (a *App) importCollectionContentWithStrategy(projectID, format, content, sourceURL, strategy string) error {
	kind, normalizedContent, err := resolveImportKind(format, content)
	if err != nil {
		return err
	}

	switch kind {
	case "postman":
		summary, err := a.requestSvc.ImportPostmanCollectionWithStrategy(projectID, []byte(normalizedContent), strategy)
		if err != nil {
			logger.Error("导入 Postman 集合失败", "projectID", projectID, "strategy", strategy, "error", err.Error())
			return err
		}
		logger.Info("导入 Postman 集合成功", "projectID", projectID, "strategy", strategy,
			"created", summary.Created, "updated", summary.Updated, "overwritten", summary.Overwritten)
		return nil
	case "postman-environment":
		return a.ImportPostmanEnvironment(projectID, normalizedContent)
	case "swagger":
		summary, err := a.requestSvc.ImportSwaggerWithSourceStrategy(projectID, []byte(normalizedContent), sourceURL, strategy)
		if err != nil {
			logger.Error("导入 Swagger 失败", "projectID", projectID, "strategy", strategy, "error", err.Error())
			return err
		}
		logger.Info("导入 Swagger 成功", "projectID", projectID, "strategy", strategy,
			"created", summary.Created, "updated", summary.Updated, "overwritten", summary.Overwritten)
		return nil
	default:
		return fmt.Errorf("无法识别文件格式，请选择正确的导入格式")
	}
}

// ImportPreview 导入预检结果：包含与已有请求的 URL 冲突信息

// ---- 导入 ----

func (a *App) ImportCurl(curlCommand string) (*model.SendRequestInput, error) {
	logger.Debug("ImportCurl 调用", "cmdLength", len(curlCommand))
	result, err := httputil.ParseCurlCommand(curlCommand)
	if err != nil {
		logger.Error("cURL 解析失败", "error", err.Error())
		return nil, err
	}
	logger.Info("cURL 解析成功", "method", result.Method, "target", logger.RedactRequestTarget(result.URL))
	return result, nil
}
