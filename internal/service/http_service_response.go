// 响应体读取：普通读取、SSE 流式读取，以及二进制/文本判定。

package service

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"minipost/internal/model"
	appErrors "minipost/internal/pkg/errors"
)

func (s *HttpService) readResponseBody(body io.Reader, maxResponseBytes int64, contentType string, onChunk func(model.StreamChunk)) ([]byte, error) {
	if onChunk != nil {
		if isSSEContentType(contentType) {
			return s.readSSEStreamBody(body, maxResponseBytes, onChunk)
		}
	}

	if maxResponseBytes <= 0 {
		bodyBytes, err := io.ReadAll(body)
		if err != nil {
			return nil, appErrors.Wrap("READ_BODY_FAILED", "读取响应体失败", err)
		}
		return bodyBytes, nil
	}

	limited := io.LimitReader(body, maxResponseBytes+1)
	bodyBytes, err := io.ReadAll(limited)
	if err != nil {
		return nil, appErrors.Wrap("READ_BODY_FAILED", "读取响应体失败", err)
	}
	if int64(len(bodyBytes)) > maxResponseBytes {
		return nil, newMaxBodySizeExceededError(maxResponseBytes)
	}
	return bodyBytes, nil
}

func (s *HttpService) readSSEStreamBody(body io.Reader, maxResponseBytes int64, onChunk func(model.StreamChunk)) ([]byte, error) {
	reader := bufio.NewReader(body)
	buffer := bytes.NewBuffer(nil)
	eventLines := make([]string, 0, 8)
	sequence := 0

	emit := func(kind string, data string, raw string) {
		sequence++
		onChunk(model.StreamChunk{
			Kind:       kind,
			Data:       data,
			Raw:        raw,
			Timestamp:  time.Now().Format(time.RFC3339Nano),
			Sequence:   sequence,
			BytesTotal: int64(buffer.Len()),
		})
	}

	flushEvent := func() {
		if len(eventLines) == 0 {
			return
		}
		raw := strings.Join(eventLines, "\n")
		dataLines := make([]string, 0, len(eventLines))
		for _, line := range eventLines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "data:") {
				dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(trimmed, "data:")))
			}
		}
		if len(dataLines) > 0 {
			emit("data", strings.Join(dataLines, "\n"), raw)
		} else {
			emit("event", raw, raw)
		}
		eventLines = eventLines[:0]
	}

	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			if _, writeErr := buffer.WriteString(line); writeErr != nil {
				return nil, appErrors.Wrap("READ_BODY_FAILED", "读取响应体失败", writeErr)
			}
			if maxResponseBytes > 0 && int64(buffer.Len()) > maxResponseBytes {
				return nil, newMaxBodySizeExceededError(maxResponseBytes)
			}

			trimmedLine := strings.TrimRight(line, "\r\n")
			if trimmedLine == "" {
				flushEvent()
			} else {
				eventLines = append(eventLines, trimmedLine)
			}
		}

		if err == nil {
			continue
		}
		if errors.Is(err, io.EOF) {
			break
		}
		return nil, appErrors.Wrap("READ_BODY_FAILED", "读取响应体失败", err)
	}

	flushEvent()
	return buffer.Bytes(), nil
}

func isSSEContentType(contentType string) bool {
	return strings.Contains(strings.ToLower(contentType), "text/event-stream")
}

func shouldTreatResponseAsBinary(contentType string, contentDisposition string, body []byte) bool {
	if strings.Contains(strings.ToLower(contentDisposition), "attachment") {
		return true
	}

	mime := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if mime == "" && len(body) > 0 {
		mime = strings.ToLower(strings.TrimSpace(strings.Split(http.DetectContentType(body), ";")[0]))
	}

	if mime == "" {
		return !utf8.Valid(body)
	}
	if isLikelyTextContentType(mime) {
		return false
	}
	if strings.HasPrefix(mime, "image/") || strings.HasPrefix(mime, "audio/") || strings.HasPrefix(mime, "video/") || strings.HasPrefix(mime, "font/") {
		return true
	}
	if strings.HasPrefix(mime, "application/") || strings.HasPrefix(mime, "multipart/") {
		return true
	}
	return !utf8.Valid(body)
}

func isLikelyTextContentType(mime string) bool {
	if strings.HasPrefix(mime, "text/") {
		return true
	}
	textHints := []string{
		"json",
		"xml",
		"html",
		"yaml",
		"yml",
		"javascript",
		"ecmascript",
		"x-www-form-urlencoded",
		"graphql",
		"csv",
		"svg",
	}
	for _, hint := range textHints {
		if strings.Contains(mime, hint) {
			return true
		}
	}
	return false
}

func newMaxBodySizeExceededError(maxResponseBytes int64) error {
	limitMB := maxResponseBytes / (1024 * 1024)
	return appErrors.New("MAX_RESPONSE_SIZE_EXCEEDED", fmt.Sprintf("响应体超过最大大小限制（%d MB）", limitMB))
}
