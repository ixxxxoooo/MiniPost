package httputil

import (
	"errors"
	"minipost/internal/model"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
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
	appendHeader := func(value string) {
		headerStr := strings.TrimSpace(value)
		if idx := strings.Index(headerStr, ":"); idx > 0 {
			input.Headers = append(input.Headers, model.KeyValue{
				Key:   strings.TrimSpace(headerStr[:idx]),
				Value: strings.TrimSpace(headerStr[idx+1:]),
			})
		}
	}
	appendData := func(data string) {
		var body string
		switch input.Body.Type {
		case "json":
			body = input.Body.JSON
		case "raw":
			body = input.Body.Raw
		}
		if body != "" {
			body += "&" + data
		} else {
			body = data
		}
		if looksLikeJSONBody(body) {
			input.Body = model.RequestBody{Type: "json", JSON: body}
		} else {
			input.Body = model.RequestBody{Type: "raw", Raw: body}
		}
		if input.Method == "GET" {
			input.Method = "POST"
		}
	}
	appendForm := func(value string) {
		formArg := strings.TrimSpace(value)
		key, formValue, ok := strings.Cut(formArg, "=")
		if !ok {
			return
		}
		key = strings.TrimSpace(key)
		formValue = strings.TrimSpace(formValue)
		if key == "" {
			return
		}
		if input.Body.Type != "form-data" {
			input.Body = model.RequestBody{
				Type:     "form-data",
				FormData: []model.FormData{},
			}
		}
		if strings.HasPrefix(formValue, "@") {
			filePath := strings.Trim(strings.TrimPrefix(formValue, "@"), "'\"")
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
				Value: strings.Trim(formValue, "'\""),
			})
		}
		if input.Method == "GET" {
			input.Method = "POST"
		}
	}
	setBasicAuth := func(value string) {
		if idx := strings.Index(value, ":"); idx >= 0 {
			input.Auth = model.AuthConfig{
				Type: "basic",
				Basic: model.BasicAuth{
					Username: value[:idx],
					Password: value[idx+1:],
				},
			}
		}
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
		longOption, inlineValue, hasInlineValue := splitLongOption(part)
		if longOption != "" {
			switch longOption {
			case "--request":
				if value, ok := consumeOptionValue(parts, &i, inlineValue, hasInlineValue); ok {
					input.Method = strings.ToUpper(value)
				}
				continue
			case "--url":
				if value, ok := consumeOptionValue(parts, &i, inlineValue, hasInlineValue); ok {
					input.URL = strings.Trim(value, "'\"")
				}
				continue
			case "--head":
				input.Method = "HEAD"
				continue
			case "--header":
				if value, ok := consumeHeaderOptionValue(parts, &i, inlineValue, hasInlineValue); ok {
					appendHeader(value)
				}
				continue
			case "--cookie":
				if value, ok := consumeOptionValue(parts, &i, inlineValue, hasInlineValue); ok {
					appendLiteralCookieHeader(value)
				}
				continue
			case "--data", "--data-raw", "--data-binary", "--data-urlencode":
				if value, ok := consumeOptionValue(parts, &i, inlineValue, hasInlineValue); ok {
					appendData(value)
				}
				continue
			case "--form":
				if value, ok := consumeOptionValue(parts, &i, inlineValue, hasInlineValue); ok {
					appendForm(value)
				}
				continue
			case "--user":
				if value, ok := consumeOptionValue(parts, &i, inlineValue, hasInlineValue); ok {
					setBasicAuth(value)
				}
				continue
			case "--compressed", "--insecure", "--location", "--globoff", "--http1.1", "--http2":
				continue
			}
		}
		switch part {
		case "-X":
			if i+1 < len(parts) {
				i++
				input.Method = strings.ToUpper(parts[i])
			}
		case "-I", "--head":
			input.Method = "HEAD"
		case "-H":
			if i+1 < len(parts) {
				i++
				value := parts[i]
				if joined, ok := maybeJoinHeaderValue(parts, &i, value); ok {
					value = joined
				}
				appendHeader(value)
			}
		case "-b":
			if i+1 < len(parts) {
				i++
				appendLiteralCookieHeader(parts[i])
			}
		case "-d":
			if i+1 < len(parts) {
				i++
				appendData(parts[i])
			}
		case "-F":
			if i+1 < len(parts) {
				i++
				appendForm(parts[i])
			}
		case "-u":
			if i+1 < len(parts) {
				i++
				setBasicAuth(parts[i])
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

func looksLikeJSONBody(data string) bool {
	trimmed := strings.TrimSpace(data)
	return strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")
}

func splitLongOption(part string) (option string, value string, hasValue bool) {
	if !strings.HasPrefix(part, "--") {
		return "", "", false
	}
	if idx := strings.Index(part, "="); idx > 2 {
		return part[:idx], part[idx+1:], true
	}
	return part, "", false
}

func consumeOptionValue(parts []string, index *int, inlineValue string, hasInlineValue bool) (string, bool) {
	if hasInlineValue {
		return inlineValue, true
	}
	if *index+1 >= len(parts) {
		return "", false
	}
	(*index)++
	return parts[*index], true
}

func consumeHeaderOptionValue(parts []string, index *int, inlineValue string, hasInlineValue bool) (string, bool) {
	value, ok := consumeOptionValue(parts, index, inlineValue, hasInlineValue)
	if !ok {
		return "", false
	}
	return maybeJoinHeaderValue(parts, index, value)
}

func maybeJoinHeaderValue(parts []string, index *int, value string) (string, bool) {
	_, after, found := strings.Cut(value, ":")
	if !found || strings.TrimSpace(after) != "" || *index+1 >= len(parts) || strings.HasPrefix(parts[*index+1], "-") {
		return value, true
	}
	(*index)++
	return value + " " + parts[*index], true
}

// tokenize performs a small shell-like split for cURL commands copied from browsers.
// It supports single quotes, double quotes, Bash/cmd line continuations and Bash
// ANSI-C quoted strings such as $'{"key":"value with \'quotes\'"}'.
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
		case ch == '^' && i+1 < len(input) && !inSingleQuote:
			next := input[i+1]
			if next == '\n' || next == '\r' {
				i++
				if next == '\r' && i+1 < len(input) && input[i+1] == '\n' {
					i++
				}
				continue
			}
			if !inDoubleQuote {
				i++
				current.WriteByte(input[i])
				continue
			}
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
		if ch == '\\' {
			value, next := decodeAnsiCEscape(input, i+1)
			current.WriteString(value)
			i = next - 1
			continue
		}
		current.WriteByte(ch)
	}
	return current.String(), i
}

func decodeAnsiCEscape(input string, start int) (string, int) {
	if start >= len(input) {
		return "\\", start
	}
	ch := input[start]
	switch ch {
	case 'a':
		return "\a", start + 1
	case 'b':
		return "\b", start + 1
	case 'e', 'E':
		return "\x1b", start + 1
	case 'f':
		return "\f", start + 1
	case 'n':
		return "\n", start + 1
	case 'r':
		return "\r", start + 1
	case 't':
		return "\t", start + 1
	case 'v':
		return "\v", start + 1
	case '\\':
		return "\\", start + 1
	case '\'':
		return "'", start + 1
	case '"':
		return "\"", start + 1
	case '?':
		return "?", start + 1
	case 'x':
		value, next, ok := parseVariableHexEscape(input, start+1, 2)
		if !ok {
			return literalEscape(input, start, next), next
		}
		return string(byte(value)), next
	case 'u':
		value, next, ok := parseExactHexEscape(input, start+1, 4)
		if !ok {
			return literalEscape(input, start, next), next
		}
		return stringFromUnicodeEscape(value, input, start, next), next
	case 'U':
		value, next, ok := parseExactHexEscape(input, start+1, 8)
		if !ok {
			return literalEscape(input, start, next), next
		}
		return stringFromUnicodeEscape(value, input, start, next), next
	default:
		if isOctalDigit(ch) {
			value, next := parseOctalEscape(input, start, 3)
			return string(byte(value)), next
		}
		return "\\" + string(ch), start + 1
	}
}

func literalEscape(input string, start int, next int) string {
	return "\\" + input[start:next]
}

func parseVariableHexEscape(input string, start int, maxDigits int) (int64, int, bool) {
	next := start
	for next < len(input) && next-start < maxDigits && isHexDigit(input[next]) {
		next++
	}
	if next == start {
		return 0, start, false
	}
	value, err := strconv.ParseInt(input[start:next], 16, 32)
	return value, next, err == nil
}

func parseExactHexEscape(input string, start int, digits int) (int64, int, bool) {
	next := start
	for next < len(input) && next-start < digits && isHexDigit(input[next]) {
		next++
	}
	if next-start != digits {
		return 0, next, false
	}
	value, err := strconv.ParseInt(input[start:next], 16, 32)
	return value, next, err == nil
}

func parseOctalEscape(input string, start int, maxDigits int) (int64, int) {
	next := start
	for next < len(input) && next-start < maxDigits && isOctalDigit(input[next]) {
		next++
	}
	value, err := strconv.ParseInt(input[start:next], 8, 32)
	if err != nil {
		return 0, next
	}
	return value, next
}

func stringFromUnicodeEscape(value int64, input string, start int, next int) string {
	r := rune(value)
	if !utf8.ValidRune(r) {
		return literalEscape(input, start, next)
	}
	return string(r)
}

func isHexDigit(ch byte) bool {
	return ('0' <= ch && ch <= '9') || ('a' <= ch && ch <= 'f') || ('A' <= ch && ch <= 'F')
}

func isOctalDigit(ch byte) bool {
	return '0' <= ch && ch <= '7'
}
