package logger

import (
	"fmt"
	"net/url"
	"strings"
)

const redactedValue = "[REDACTED]"

var sensitiveKeyHints = []string{
	"access_token",
	"refresh_token",
	"id_token",
	"token",
	"secret",
	"password",
	"passwd",
	"pwd",
	"auth",
	"authorization",
	"cookie",
	"session",
	"apikey",
	"api_key",
	"key",
	"signature",
	"sign",
}

// RedactRequestTarget returns a log-safe request target. Full cURL commands can
// contain cookies and body data, so only their length is logged.
func RedactRequestTarget(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if isCurlCommand(trimmed) {
		return fmt.Sprintf("<curl command: %d chars>", len(trimmed))
	}
	return RedactURL(trimmed)
}

// RedactURL preserves the useful shape of a URL while masking sensitive query
// parameter values. It is intentionally conservative: suspicious key names are
// redacted even when they are not guaranteed secrets.
func RedactURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.RawQuery == "" {
		return trimmed
	}

	values := parsed.Query()
	for key := range values {
		if isSensitiveKey(key) {
			values[key] = []string{redactedValue}
		}
	}
	parsed.RawQuery = values.Encode()
	return parsed.String()
}

func isCurlCommand(value string) bool {
	fields := strings.Fields(value)
	return len(fields) > 0 && strings.EqualFold(fields[0], "curl")
}

func isSensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	if normalized == "" {
		return false
	}
	compact := strings.NewReplacer("-", "_", ".", "_").Replace(normalized)
	for _, hint := range sensitiveKeyHints {
		if compact == hint || strings.Contains(compact, hint) {
			return true
		}
	}
	return false
}
