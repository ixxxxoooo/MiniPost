package client

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
)

// 响应体展示模式（与旧版格式下拉一致）。
const (
	ModeJSON       = "json"
	ModeXML        = "xml"
	ModeHTML       = "html"
	ModeJavaScript = "javascript"
	ModeRaw        = "raw"
	ModeHex        = "hex"
	ModeBase64     = "base64"
)

// BodyModes 按下拉菜单顺序列出全部展示模式：前四项是结构化格式，后三项是编码视图。
var BodyModes = []string{ModeJSON, ModeXML, ModeHTML, ModeJavaScript, ModeRaw, ModeHex, ModeBase64}

// 过滤失败的原因，界面按原因显示本地化提示。
var (
	ErrFilterPath  = errors.New("JSONPath 表达式无效")
	ErrFilterEmpty = errors.New("过滤结果为空")
	ErrFilterParse = errors.New("JSON 解析失败，无法按路径过滤")
)

var jsStartRe = regexp.MustCompile(`^(var|let|const|function|class|import|export|return)\s`)

func looksLikeJSON(text string) bool {
	t := strings.TrimLeft(text, " \t\r\n")
	if !strings.HasPrefix(t, "{") && !strings.HasPrefix(t, "[") {
		return false
	}
	return json.Valid([]byte(text))
}

func looksLikeHTML(text string) bool {
	t := strings.ToLower(strings.TrimLeft(text, " \t\r\n"))
	return strings.HasPrefix(t, "<!doctype html") || strings.HasPrefix(t, "<html")
}

func looksLikeXML(text string) bool {
	t := strings.TrimLeft(text, " \t\r\n")
	return strings.HasPrefix(t, "<?xml") || (strings.HasPrefix(t, "<") && strings.Contains(t, ">"))
}

func looksLikeJS(text string) bool {
	t := strings.TrimLeft(text, " \t\r\n")
	return jsStartRe.MatchString(t) || strings.HasPrefix(t, "(function") || strings.HasPrefix(t, "!function")
}

// DetectBodyMode 按 Content-Type 与内容推断默认展示模式；detection 为 "json" 时一律按 JSON。
func DetectBodyMode(contentType, body, detection string) string {
	if detection == "json" {
		return ModeJSON
	}
	ct := strings.ToLower(contentType)
	switch {
	case strings.Contains(ct, "json"):
		return ModeJSON
	case strings.Contains(ct, "html"):
		return ModeHTML
	case strings.Contains(ct, "xml"), strings.Contains(ct, "svg"):
		return ModeXML
	case strings.Contains(ct, "javascript"), strings.Contains(ct, "ecmascript"):
		return ModeJavaScript
	}
	switch {
	case looksLikeJSON(body):
		return ModeJSON
	case looksLikeHTML(body):
		return ModeHTML
	case looksLikeXML(body):
		return ModeXML
	case looksLikeJS(body):
		return ModeJavaScript
	}
	return ModeRaw
}

// BodyLanguageFile 返回编辑器按文件名选择高亮语言时用的文件名。
func BodyLanguageFile(mode string) string {
	switch mode {
	case ModeJSON:
		return "x.json"
	case ModeXML:
		return "x.xml"
	case ModeHTML:
		return "x.html"
	case ModeJavaScript:
		return "x.js"
	}
	return "x.txt"
}

// FormatBody 按展示模式整理响应体文本。
func FormatBody(mode, raw string) string {
	switch mode {
	case ModeJSON:
		return formatJSONText(raw)
	case ModeXML, ModeHTML:
		return formatXMLText(raw)
	case ModeJavaScript:
		return formatJSText(raw)
	case ModeHex:
		parts := make([]string, 0, len(raw))
		for _, b := range []byte(raw) {
			parts = append(parts, hex.EncodeToString([]byte{b}))
		}
		return strings.Join(parts, " ")
	case ModeBase64:
		return base64.StdEncoding.EncodeToString([]byte(raw))
	}
	return raw
}

// formatJSONText 缩进 JSON；整体不是合法 JSON 时按 NDJSON（每行一个 JSON）逐行缩进，仍失败则原样返回。
func formatJSONText(text string) string {
	if out, ok := indentJSON(text); ok {
		return out
	}
	var pretty []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out, ok := indentJSON(line)
		if !ok {
			return text
		}
		pretty = append(pretty, out)
	}
	if len(pretty) == 0 {
		return text
	}
	return strings.Join(pretty, "\n")
}

func indentJSON(text string) (string, bool) {
	if !json.Valid([]byte(text)) {
		return "", false
	}
	var out bytes.Buffer
	if err := json.Indent(&out, []byte(strings.TrimSpace(text)), "", "  "); err != nil {
		return "", false
	}
	return out.String(), true
}

// formatXMLText 是旧版的轻量 XML/HTML 缩进：标签之间换行，按层级缩进两个空格。
func formatXMLText(text string) string {
	text = regexp.MustCompile(`>\s*<`).ReplaceAllString(text, ">\n<")
	var b strings.Builder
	indent := 0
	for _, line := range strings.Split(text, "\n") {
		t := strings.TrimSpace(line)
		if t == "" {
			continue
		}
		if strings.HasPrefix(t, "</") {
			indent = max(0, indent-1)
		}
		b.WriteString(strings.Repeat("  ", indent))
		b.WriteString(t)
		b.WriteByte('\n')
		if strings.HasPrefix(t, "<") && !strings.HasPrefix(t, "</") && !strings.HasPrefix(t, "<?") &&
			!strings.HasSuffix(t, "/>") && !strings.Contains(t, "</") {
			indent++
		}
	}
	return strings.TrimRight(b.String(), " \t\r\n")
}

// formatJSText 是旧版的轻量 JS 排版：括号与分号后换行，字符串内容原样保留。
func formatJSText(text string) string {
	rs := []rune(text)
	var b strings.Builder
	indent := 0
	var inString rune
	pad := func() string { return strings.Repeat("  ", indent) }
	for i := 0; i < len(rs); {
		ch := rs[i]
		if inString != 0 {
			b.WriteRune(ch)
			if ch == '\\' && i+1 < len(rs) {
				b.WriteRune(rs[i+1])
				i += 2
				continue
			}
			if ch == inString {
				inString = 0
			}
			i++
			continue
		}
		switch ch {
		case '"', '\'', '`':
			inString = ch
			b.WriteRune(ch)
			i++
		case '{', '[', '(':
			indent++
			b.WriteRune(ch)
			b.WriteByte('\n')
			b.WriteString(pad())
			i++
		case '}', ']', ')':
			indent = max(0, indent-1)
			b.WriteByte('\n')
			b.WriteString(pad())
			b.WriteRune(ch)
			i++
		case ';', ',':
			b.WriteRune(ch)
			b.WriteByte('\n')
			b.WriteString(pad())
			i++
			for i < len(rs) && (rs[i] == ' ' || rs[i] == '\t') {
				i++
			}
		default:
			b.WriteRune(ch)
			i++
		}
	}
	return b.String()
}

// CanPreview 判断当前内容是否能进入预览视图。
func CanPreview(mode, contentType, body string) bool {
	switch mode {
	case ModeHTML, ModeXML, ModeJSON, ModeJavaScript:
		return true
	}
	ct := strings.ToLower(contentType)
	return strings.Contains(ct, "html") || strings.Contains(ct, "xml") || looksLikeHTML(body) || looksLikeXML(body)
}

// ---- 过滤 ----

// FilterBody 按表达式过滤响应体：JSON 模式按 JSONPath，其余模式按关键字过滤行。
// 表达式为空时原样返回 formatted；失败时返回 err，文本为未过滤内容。
func FilterBody(mode, raw, formatted, expr string) (string, error) {
	if strings.TrimSpace(expr) == "" {
		return formatted, nil
	}
	if mode == ModeJSON {
		return filterJSON(raw, expr)
	}
	query := strings.ToLower(strings.TrimSpace(expr))
	var kept []string
	for _, line := range strings.Split(formatted, "\n") {
		if strings.Contains(strings.ToLower(line), query) {
			kept = append(kept, line)
		}
	}
	if len(kept) == 0 {
		return formatted, ErrFilterEmpty
	}
	return strings.Join(kept, "\n"), nil
}

func filterJSON(raw, expr string) (string, error) {
	root, err := ParseJSONNode(raw)
	if err != nil {
		return raw, ErrFilterParse
	}
	tokens, ok := ParseJSONPath(expr)
	if !ok {
		return raw, ErrFilterPath
	}
	node, ok := root.Resolve(tokens)
	if !ok {
		return raw, ErrFilterEmpty
	}
	switch node.Kind {
	case JSONString:
		return node.Str, nil
	case JSONNumber, JSONBool, JSONNull:
		return node.Scalar(), nil
	}
	return node.Pretty(), nil
}

// ParseJSONPath 解析 `$.a.b[0]['c']` 形式的路径，返回的元素为 string（键）或 int（下标）。
// 空表达式与 "$" 表示根；不以 $ 开头或语法错误时 ok 为 false。
func ParseJSONPath(expr string) (tokens []any, ok bool) {
	in := strings.TrimSpace(expr)
	if in == "" || in == "$" {
		return nil, true
	}
	if !strings.HasPrefix(in, "$") {
		return nil, false
	}
	isKeyChar := func(c byte) bool {
		return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '$' || c == '-'
	}
	i := 1
	for i < len(in) {
		switch in[i] {
		case '.':
			i++
			start := i
			for i < len(in) && isKeyChar(in[i]) {
				i++
			}
			if start == i {
				return nil, false
			}
			tokens = append(tokens, in[start:i])
		case '[':
			i++
			if i >= len(in) {
				return nil, false
			}
			if in[i] == '\'' || in[i] == '"' {
				quote := in[i]
				i++
				start := i
				for i < len(in) && in[i] != quote {
					i++
				}
				if i >= len(in) {
					return nil, false
				}
				key := in[start:i]
				i++
				if i >= len(in) || in[i] != ']' {
					return nil, false
				}
				i++
				tokens = append(tokens, key)
				continue
			}
			start := i
			for i < len(in) && in[i] >= '0' && in[i] <= '9' {
				i++
			}
			if start == i || i >= len(in) || in[i] != ']' {
				return nil, false
			}
			n := 0
			for _, d := range in[start:i] {
				n = n*10 + int(d-'0')
			}
			i++
			tokens = append(tokens, n)
		default:
			return nil, false
		}
	}
	return tokens, true
}

// ---- 有序 JSON 树（保持键顺序，供预览与过滤使用）----

// JSONKind 是 JSON 节点类型。
type JSONKind int

const (
	JSONNull JSONKind = iota
	JSONBool
	JSONNumber
	JSONString
	JSONArray
	JSONObject
)

// JSONNode 是保持对象键顺序的 JSON 节点。
type JSONNode struct {
	Kind JSONKind
	Str  string // 字符串值；数字与布尔为其文本形式
	Keys []string
	Kids []*JSONNode
}

// ParseJSONNode 解析一段 JSON 文本（只接受单个值）。
func ParseJSONNode(raw string) (*JSONNode, error) {
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.UseNumber()
	n, err := decodeNode(dec)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, errors.New("JSON 末尾有多余内容")
	}
	return n, nil
}

func decodeNode(dec *json.Decoder) (*JSONNode, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch v := tok.(type) {
	case json.Delim:
		n := &JSONNode{Kind: JSONArray}
		if v == '{' {
			n.Kind = JSONObject
		}
		for dec.More() {
			if n.Kind == JSONObject {
				kt, err := dec.Token()
				if err != nil {
					return nil, err
				}
				n.Keys = append(n.Keys, kt.(string))
			}
			kid, err := decodeNode(dec)
			if err != nil {
				return nil, err
			}
			n.Kids = append(n.Kids, kid)
		}
		if _, err := dec.Token(); err != nil {
			return nil, err
		}
		return n, nil
	case string:
		return &JSONNode{Kind: JSONString, Str: v}, nil
	case json.Number:
		return &JSONNode{Kind: JSONNumber, Str: v.String()}, nil
	case bool:
		if v {
			return &JSONNode{Kind: JSONBool, Str: "true"}, nil
		}
		return &JSONNode{Kind: JSONBool, Str: "false"}, nil
	}
	return &JSONNode{Kind: JSONNull}, nil
}

// Scalar 返回标量节点的显示文本（null 为 "null"，字符串为原文）。
func (n *JSONNode) Scalar() string {
	switch n.Kind {
	case JSONNull:
		return "null"
	case JSONArray, JSONObject:
		return ""
	}
	return n.Str
}

// Child 按路径元素（string 键或 int 下标）取子节点。
func (n *JSONNode) Child(token any) (*JSONNode, bool) {
	switch t := token.(type) {
	case int:
		if n.Kind != JSONArray || t < 0 || t >= len(n.Kids) {
			return nil, false
		}
		return n.Kids[t], true
	case string:
		if n.Kind != JSONObject {
			return nil, false
		}
		// 与 JS 对象一致：重复键以最后一个为准
		for i := len(n.Keys) - 1; i >= 0; i-- {
			if n.Keys[i] == t {
				return n.Kids[i], true
			}
		}
	}
	return nil, false
}

// Resolve 沿路径向下取节点。
func (n *JSONNode) Resolve(tokens []any) (*JSONNode, bool) {
	cur := n
	for _, t := range tokens {
		next, ok := cur.Child(t)
		if !ok {
			return nil, false
		}
		cur = next
	}
	return cur, true
}

// Badge 是复合节点在预览表里的角标：数组 `[3]`，对象 `{2}`，空对象 `{}`。
func (n *JSONNode) Badge() string {
	switch n.Kind {
	case JSONArray:
		return "[" + strconv.Itoa(len(n.Kids)) + "]"
	case JSONObject:
		if len(n.Kids) == 0 {
			return "{}"
		}
		return "{" + strconv.Itoa(len(n.Kids)) + "}"
	}
	return ""
}

// Pretty 以两个空格缩进输出（与 JSON.stringify(value, null, 2) 一致，不转义 <>&）。
func (n *JSONNode) Pretty() string {
	var b strings.Builder
	n.write(&b, 0)
	return b.String()
}

func (n *JSONNode) write(b *strings.Builder, depth int) {
	switch n.Kind {
	case JSONNull:
		b.WriteString("null")
	case JSONBool, JSONNumber:
		b.WriteString(n.Str)
	case JSONString:
		b.WriteString(quoteJSON(n.Str))
	case JSONArray, JSONObject:
		open, closer := "[", "]"
		if n.Kind == JSONObject {
			open, closer = "{", "}"
		}
		if len(n.Kids) == 0 {
			b.WriteString(open + closer)
			return
		}
		b.WriteString(open + "\n")
		for i, k := range n.Kids {
			b.WriteString(strings.Repeat("  ", depth+1))
			if n.Kind == JSONObject {
				b.WriteString(quoteJSON(n.Keys[i]) + ": ")
			}
			k.write(b, depth+1)
			if i < len(n.Kids)-1 {
				b.WriteByte(',')
			}
			b.WriteByte('\n')
		}
		b.WriteString(strings.Repeat("  ", depth) + closer)
	}
}

func quoteJSON(s string) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	return strings.TrimRight(buf.String(), "\n")
}

// ---- 下载 ----

var textMimeHints = []string{"json", "xml", "yaml", "yml", "html", "javascript", "ecmascript", "x-www-form-urlencoded", "graphql", "csv", "plain", "svg"}

var binaryMimeExact = map[string]bool{
	"application/octet-stream": true, "application/zip": true, "application/x-zip-compressed": true,
	"application/x-7z-compressed": true, "application/x-rar-compressed": true, "application/gzip": true,
	"application/x-gzip": true, "application/pdf": true, "application/msword": true,
	"application/vnd.ms-excel": true, "application/vnd.ms-powerpoint": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         true,
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
	"application/x-binary": true, "application/x-protobuf": true, "application/protobuf": true,
	"application/wasm": true,
}

func headerValues(headers map[string][]string, name string) []string {
	for k, v := range headers {
		if strings.EqualFold(k, name) {
			return v
		}
	}
	return nil
}

func normalizeMime(contentType string) string {
	mime, _, _ := strings.Cut(contentType, ";")
	return strings.ToLower(strings.TrimSpace(mime))
}

// PreferDownload 判断响应更适合当作文件下载（二进制、attachment、非文本媒体类型）。
func PreferDownload(headers map[string][]string, contentType string, binary bool) bool {
	if binary {
		return true
	}
	if cd := headerValues(headers, "content-disposition"); len(cd) > 0 && strings.Contains(strings.ToLower(cd[0]), "attachment") {
		return true
	}
	mime := normalizeMime(contentType)
	if mime == "" || strings.HasPrefix(mime, "text/") {
		return false
	}
	for _, h := range textMimeHints {
		if strings.Contains(mime, h) {
			return false
		}
	}
	for _, p := range []string{"image/", "audio/", "video/", "font/"} {
		if strings.HasPrefix(mime, p) {
			return true
		}
	}
	return binaryMimeExact[mime] || strings.HasPrefix(mime, "application/")
}

var (
	filenameStarRe = regexp.MustCompile(`(?i)filename\*\s*=\s*([^;]+)`)
	filenameRe     = regexp.MustCompile(`(?i)filename\s*=\s*([^;]+)`)
	extRe          = regexp.MustCompile(`\.[A-Za-z0-9]{1,10}$`)
	unsafeNameRe   = regexp.MustCompile(`[/\\?%*:|"<>]`)
)

func stripQuotes(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 2 && strings.HasPrefix(v, `"`) && strings.HasSuffix(v, `"`) {
		return strings.ReplaceAll(v[1:len(v)-1], `\"`, `"`)
	}
	return v
}

// ParseContentDisposition 从 Content-Disposition 取文件名，优先 filename*（RFC 5987）。
func ParseContentDisposition(value string) string {
	if value == "" {
		return ""
	}
	if m := filenameStarRe.FindStringSubmatch(value); m != nil {
		raw := stripQuotes(m[1])
		if parts := strings.SplitN(raw, "''", 2); len(parts) == 2 && parts[1] != "" {
			if dec, err := url.PathUnescape(parts[1]); err == nil {
				return dec
			}
			return parts[1]
		}
	}
	if m := filenameRe.FindStringSubmatch(value); m != nil {
		return stripQuotes(m[1])
	}
	return ""
}

// SanitizeFilename 替换文件名里的非法字符并去掉控制字符；为空时返回 "response"。
func SanitizeFilename(in string) string {
	out := unsafeNameRe.ReplaceAllString(in, "_")
	out = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, out)
	out = strings.TrimSpace(out)
	if out == "" {
		return "response"
	}
	return out
}

// ExtensionFromMime 按媒体类型给出常见扩展名，未知返回空串。
func ExtensionFromMime(contentType string) string {
	mime := normalizeMime(contentType)
	for _, e := range []struct{ key, ext string }{
		{"json", ".json"}, {"xml", ".xml"}, {"html", ".html"}, {"csv", ".csv"}, {"pdf", ".pdf"},
		{"zip", ".zip"}, {"gzip", ".gz"}, {"jpeg", ".jpg"}, {"png", ".png"}, {"gif", ".gif"},
		{"webp", ".webp"}, {"svg", ".svg"}, {"audio/mpeg", ".mp3"}, {"video/mp4", ".mp4"},
		{"text/plain", ".txt"}, {"octet-stream", ".bin"},
	} {
		if strings.Contains(mime, e.key) {
			return e.ext
		}
	}
	return ""
}

func filenameFromURL(requestURL string) string {
	u, err := url.Parse(requestURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	last := path.Base(strings.TrimRight(u.Path, "/"))
	if last == "." || last == "/" {
		return ""
	}
	dec, err := url.PathUnescape(last)
	if err != nil {
		return last
	}
	return dec
}

// SuggestResponseFilename 给出保存响应时的默认文件名：
// Content-Disposition > URL 末段 > fallback，缺扩展名时按媒体类型补全。
func SuggestResponseFilename(headers map[string][]string, contentType, requestURL, fallback string) string {
	if fallback == "" {
		fallback = "response"
	}
	var fromHeader string
	if cd := headerValues(headers, "content-disposition"); len(cd) > 0 {
		fromHeader = ParseContentDisposition(cd[0])
	}
	base := fromHeader
	if base == "" {
		base = filenameFromURL(requestURL)
	}
	if base == "" {
		base = fallback
	}
	base = SanitizeFilename(base)
	if ext := ExtensionFromMime(contentType); ext != "" && !extRe.MatchString(base) {
		return base + ext
	}
	return base
}
