package httputil

import (
	"errors"
	"minipost/internal/model"
	"path/filepath"
	"strings"
	"unicode"
)

// ParseCurlCommand 将 cURL 命令解析为 SendRequestInput（基础版本）
// 后续迭代可增强为完整的 cURL 解析器
func ParseCurlCommand(curlCmd string) (*model.SendRequestInput, error) {
	curlCmd = strings.TrimSpace(curlCmd)
	if curlCmd == "" {
		return nil, errors.New("cURL 命令为空")
	}

	input := &model.SendRequestInput{
		Method:  "GET",
		Headers: []model.KeyValue{},
		Params:  []model.KeyValue{},
		Body:    model.RequestBody{Type: "none"},
		Auth:    model.AuthConfig{Type: "none"},
	}

	appendLiteralCookieHeader := func(value string) {
		cookieValue := strings.TrimSpace(value)
		if cookieValue == "" || !strings.Contains(cookieValue, "=") {
			return
		}
		input.Headers = append(input.Headers, model.KeyValue{
			Key:   "Cookie",
			Value: cookieValue,
		})
	}

	parts := tokenize(curlCmd)
	if len(parts) == 0 {
		return nil, errors.New("cURL 命令为空")
	}
	if strings.EqualFold(parts[0], "curl") {
		parts = parts[1:]
	}
	if len(parts) == 0 {
		return nil, errors.New("cURL 命令缺少请求信息")
	}

	for i := 0; i < len(parts); i++ {
		part := parts[i]
		if strings.HasPrefix(part, "--cookie=") {
			appendLiteralCookieHeader(strings.TrimPrefix(part, "--cookie="))
			continue
		}
		switch part {
		case "-X", "--request":
			if i+1 < len(parts) {
				i++
				input.Method = strings.ToUpper(parts[i])
			}
		case "--url":
			if i+1 < len(parts) {
				i++
				input.URL = strings.Trim(parts[i], "'\"")
			}
		case "-I", "--head":
			input.Method = "HEAD"
		case "-H", "--header":
			if i+1 < len(parts) {
				i++
				headerStr := strings.TrimSpace(parts[i])
				if idx := strings.Index(headerStr, ":"); idx > 0 {
					input.Headers = append(input.Headers, model.KeyValue{
						Key:   strings.TrimSpace(headerStr[:idx]),
						Value: strings.TrimSpace(headerStr[idx+1:]),
					})
				}
			}
		case "-b", "--cookie":
			if i+1 < len(parts) {
				i++
				appendLiteralCookieHeader(parts[i])
			}
		case "-d", "--data", "--data-raw", "--data-binary", "--data-urlencode":
			if i+1 < len(parts) {
				i++
				data := parts[i]
				if strings.HasPrefix(strings.TrimSpace(data), "{") {
					input.Body = model.RequestBody{Type: "json", JSON: data}
				} else {
					input.Body = model.RequestBody{Type: "raw", Raw: data}
				}
				if input.Method == "GET" {
					input.Method = "POST"
				}
			}
		case "-F", "--form":
			if i+1 < len(parts) {
				i++
				formArg := strings.TrimSpace(parts[i])
				key, value, ok := strings.Cut(formArg, "=")
				if ok {
					key = strings.TrimSpace(key)
					value = strings.TrimSpace(value)
					if key != "" {
						if input.Body.Type != "form-data" {
							input.Body = model.RequestBody{
								Type:     "form-data",
								FormData: []model.FormData{},
							}
						}
						if strings.HasPrefix(value, "@") {
							filePath := strings.Trim(strings.TrimPrefix(value, "@"), "'\"")
							input.Body.FormData = append(input.Body.FormData, model.FormData{
								Key:      key,
								Type:     "file",
								Value:    filePath,
								FilePath: filePath,
								FileName: filepath.Base(filePath),
							})
						} else {
							input.Body.FormData = append(input.Body.FormData, model.FormData{
								Key:   key,
								Type:  "text",
								Value: strings.Trim(value, "'\""),
							})
						}
					}
				}
				if input.Method == "GET" {
					input.Method = "POST"
				}
			}
		case "-u", "--user":
			if i+1 < len(parts) {
				i++
				userPass := parts[i]
				if idx := strings.Index(userPass, ":"); idx > 0 {
					input.Auth = model.AuthConfig{
						Type: "basic",
						Basic: model.BasicAuth{
							Username: userPass[:idx],
							Password: userPass[idx+1:],
						},
					}
				}
			}
		default:
			if !strings.HasPrefix(part, "-") && input.URL == "" {
				input.URL = strings.Trim(part, "'\"")
			}
		}
	}
	if strings.TrimSpace(input.URL) == "" {
		return nil, errors.New("cURL 命令缺少 URL")
	}

	return input, nil
}

// tokenize performs a small shell-like split for cURL commands copied from browsers.
// It supports single quotes, double quotes, line continuations and Bash ANSI-C
// quoted strings such as $'{"key":"value with \'quotes\'"}'.
func tokenize(input string) []string {
	var tokens []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false

	for i := 0; i < len(input); i++ {
		ch := input[i]
		switch {
		case ch == '$' && !inSingleQuote && !inDoubleQuote && i+1 < len(input) && input[i+1] == '\'':
			value, next := readAnsiCQuotedString(input, i+2)
			current.WriteString(value)
			i = next
		case ch == '\'' && !inDoubleQuote:
			inSingleQuote = !inSingleQuote
		case ch == '"' && !inSingleQuote:
			inDoubleQuote = !inDoubleQuote
		case ch == '\\' && i+1 < len(input) && !inSingleQuote:
			next := input[i+1]
			// cURL 多行命令中的反斜杠换行
			if next == '\n' || next == '\r' {
				i++
				if next == '\r' && i+1 < len(input) && input[i+1] == '\n' {
					i++
				}
				continue
			}
			i++
			current.WriteByte(input[i])
		case unicode.IsSpace(rune(ch)) && !inSingleQuote && !inDoubleQuote:
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(ch)
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

func readAnsiCQuotedString(input string, start int) (string, int) {
	var current strings.Builder
	i := start
	for ; i < len(input); i++ {
		ch := input[i]
		if ch == '\'' {
			return current.String(), i
		}
		if ch == '\\' && i+1 < len(input) {
			i++
			current.WriteString(decodeAnsiCEscape(input[i]))
			continue
		}
		current.WriteByte(ch)
	}
	return current.String(), i
}

func decodeAnsiCEscape(ch byte) string {
	switch ch {
	case 'a':
		return "\a"
	case 'b':
		return "\b"
	case 'e', 'E':
		return "\x1b"
	case 'f':
		return "\f"
	case 'n':
		return "\n"
	case 'r':
		return "\r"
	case 't':
		return "\t"
	case 'v':
		return "\v"
	case '\\':
		return "\\"
	case '\'':
		return "'"
	case '"':
		return "\""
	case '?':
		return "?"
	default:
		return string(ch)
	}
}
