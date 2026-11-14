package app

import (
	"strings"
	"time"

	"minipost/internal/model"
	appErrors "minipost/internal/pkg/errors"
	"minipost/internal/pkg/httputil"
	"minipost/internal/pkg/logger"
)

func normalizeCurlInput(input model.SendRequestInput) (model.SendRequestInput, error) {
	fields := strings.Fields(strings.TrimSpace(input.URL))
	if len(fields) == 0 || !strings.EqualFold(fields[0], "curl") {
		return input, nil
	}

	parsed, err := httputil.ParseCurlCommand(input.URL)
	if err != nil {
		return input, appErrors.Wrap("INVALID_CURL", "cURL 命令解析失败", err)
	}
	parsed.Options = input.Options
	parsed.SourceID = input.SourceID
	return *parsed, nil
}

// ---- HTTP 请求 ----

// SendRequestWithEnv 解析环境变量后发送请求并记录历史。
func (a *App) SendRequestWithEnv(input model.SendRequestInput, projectID, envID string) (*model.HttpResponse, error) {
	return a.sendRequestWithEnv(input, projectID, envID, nil)
}

// SendRequestWithEnvStream 以流式方式发送请求，每个数据块通过 emit 回调交给调用方（由界面层转为主线程更新）。
// 出错或连接关闭时，若已收到过数据块，会补发一条 error / connection_closed 终止块。
func (a *App) SendRequestWithEnvStream(input model.SendRequestInput, projectID, envID string, emit func(model.StreamChunk)) (*model.HttpResponse, error) {
	lastSequence := 0
	onChunk := func(chunk model.StreamChunk) {
		lastSequence = chunk.Sequence
		if emit != nil {
			emit(chunk)
		}
	}
	emitFinal := func(chunk model.StreamChunk) {
		if emit != nil {
			emit(chunk)
		}
	}

	resp, err := a.sendRequestWithEnv(input, projectID, envID, onChunk)
	if err != nil {
		if lastSequence > 0 {
			emitFinal(model.StreamChunk{
				Kind:      "error",
				Data:      err.Error(),
				Raw:       err.Error(),
				Timestamp: time.Now().Format(time.RFC3339Nano),
				Sequence:  lastSequence + 1,
			})
		}
		return nil, err
	}

	if lastSequence > 0 {
		emitFinal(model.StreamChunk{
			Kind:      "connection_closed",
			Data:      "Connection closed",
			Raw:       "Connection closed",
			Timestamp: time.Now().Format(time.RFC3339Nano),
			Sequence:  lastSequence + 1,
		})
	}
	return resp, nil
}

func (a *App) sendRequestWithEnv(input model.SendRequestInput, projectID, envID string, onChunk func(model.StreamChunk)) (*model.HttpResponse, error) {
	normalizedInput, err := normalizeCurlInput(input)
	if err != nil {
		logger.Warn("cURL 解析失败", "target", logger.RedactRequestTarget(input.URL), "error", err.Error())
		return nil, err
	}
	input = normalizedInput

	logger.Debug("SendRequestWithEnv 调用",
		"method", input.Method,
		"target", logger.RedactRequestTarget(input.URL),
		"projectID", projectID,
		"envID", envID,
		"streaming", onChunk != nil,
	)

	// 未选环境时也要解析：内置动态变量（{{$timestamp}} 等）不依赖环境。
	var envVars []model.Variable
	if envID != "" && projectID != "" {
		envs, err := a.envSvc.ListEnvironments(projectID)
		if err == nil {
			for _, env := range envs {
				if env.ID == envID {
					envVars = env.Variables
					logger.Debug("环境变量已解析", "envName", env.Name, "varCount", len(env.Variables))
					break
				}
			}
		}
	}
	input = httputil.ResolveRequestInput(input, envVars)

	var resp *model.HttpResponse
	if onChunk != nil {
		resp, err = a.httpSvc.SendRequestStreaming(input, onChunk)
	} else {
		resp, err = a.httpSvc.SendRequest(input)
	}
	if err != nil {
		logger.Warn("HTTP 请求失败", "method", input.Method, "target", logger.RedactRequestTarget(input.URL), "error", err.Error())
		return nil, err
	}

	logger.Info("HTTP 请求完成",
		"method", input.Method,
		"target", logger.RedactRequestTarget(input.URL),
		"status", resp.StatusCode,
		"duration_ms", resp.Duration,
		"size", resp.Size,
	)

	if projectID != "" {
		entry := &model.HistoryEntry{
			RequestID:  input.SourceID,
			Name:       input.URL,
			Method:     input.Method,
			URL:        input.URL,
			StatusCode: resp.StatusCode,
			Duration:   resp.Duration,
			Size:       resp.Size,
		}
		if err := a.historySvc.AddEntry(projectID, entry); err != nil {
			logger.Warn("历史记录保存失败", "projectID", projectID, "error", err.Error())
		}
	}

	return resp, nil
}

// CancelHTTPRequest 取消仍在进行中的 HTTP 请求。
func (a *App) CancelHTTPRequest(requestID string) bool {
	cancelled := a.httpSvc.CancelRequest(requestID)
	if cancelled {
		logger.Info("已取消 HTTP 请求", "requestID", requestID)
	}
	return cancelled
}

func (a *App) SendRequest(input model.SendRequestInput) (*model.HttpResponse, error) {
	normalizedInput, err := normalizeCurlInput(input)
	if err != nil {
		logger.Warn("cURL 解析失败", "target", logger.RedactRequestTarget(input.URL), "error", err.Error())
		return nil, err
	}
	input = normalizedInput

	logger.Debug("SendRequest 调用", "method", input.Method, "target", logger.RedactRequestTarget(input.URL))
	resp, err := a.httpSvc.SendRequest(input)
	if err != nil {
		logger.Warn("HTTP 请求失败", "method", input.Method, "target", logger.RedactRequestTarget(input.URL), "error", err.Error())
		return nil, err
	}
	logger.Info("HTTP 请求完成", "method", input.Method, "target", logger.RedactRequestTarget(input.URL), "status", resp.StatusCode)
	return resp, nil
}
