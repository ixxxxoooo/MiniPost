// Package jsonutil 提供 JSON 文本处理的公共工具，供导入解析等场景复用。
package jsonutil

// StripComments 移除 JSON 文本中的 // 行注释与 /* */ 块注释，
// 同时保留字符串字面量内的注释样式字符。常用于兼容带注释的 JSONC 导入内容。
func StripComments(raw []byte) []byte {
	if len(raw) == 0 {
		return raw
	}

	result := make([]byte, 0, len(raw))
	inString := false
	escapeNext := false
	inLineComment := false
	inBlockComment := false

	for i := 0; i < len(raw); i++ {
		ch := raw[i]
		next := byte(0)
		hasNext := i+1 < len(raw)
		if hasNext {
			next = raw[i+1]
		}

		if inLineComment {
			if ch == '\n' {
				inLineComment = false
				result = append(result, ch)
			}
			continue
		}
		if inBlockComment {
			if ch == '*' && hasNext && next == '/' {
				inBlockComment = false
				i++
			}
			continue
		}

		if inString {
			result = append(result, ch)
			if escapeNext {
				escapeNext = false
				continue
			}
			if ch == '\\' {
				escapeNext = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		if ch == '"' {
			inString = true
			result = append(result, ch)
			continue
		}

		if ch == '/' && hasNext && next == '/' {
			inLineComment = true
			i++
			continue
		}
		if ch == '/' && hasNext && next == '*' {
			inBlockComment = true
			i++
			continue
		}

		result = append(result, ch)
	}

	return result
}
