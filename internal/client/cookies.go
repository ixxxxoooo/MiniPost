package client

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"minipost/internal/pkg/logger"
)

// Cookie Cookie Jar 中的一条记录。
type Cookie struct {
	ID       string `json:"id"`
	Domain   string `json:"domain"`
	Name     string `json:"name"`
	Value    string `json:"value"`
	Path     string `json:"path"`
	Expires  string `json:"expires"`
	Secure   bool   `json:"secure"`
	HTTPOnly bool   `json:"httpOnly"`
	Enabled  bool   `json:"enabled"`
}

// Jar 自动吸收响应 Set-Cookie 并在后续请求中携带的 Cookie 容器，变更时落盘。
type Jar struct {
	mu      sync.Mutex
	path    string
	cookies []Cookie
}

// NewJar 从 path 加载 Cookie（过期项被丢弃）；path 为空则只保存在内存中。
func NewJar(path string) *Jar {
	j := &Jar{path: path}
	if path == "" {
		return j
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Warn("读取 Cookie 文件失败", "error", err.Error())
		}
		return j
	}
	var list []Cookie
	if err := json.Unmarshal(data, &list); err != nil {
		logger.Warn("解析 Cookie 文件失败", "error", err.Error())
		return j
	}
	for _, c := range list {
		if !isExpired(c.Expires) && c.Name != "" {
			if c.ID == "" {
				c.ID = uuid.NewString()
			}
			if c.Path == "" {
				c.Path = "/"
			}
			j.cookies = append(j.cookies, c)
		}
	}
	return j
}

func isExpired(expires string) bool {
	if expires == "" {
		return false
	}
	t, ok := parseCookieTime(expires)
	return ok && !t.After(time.Now())
}

func parseCookieTime(s string) (time.Time, bool) {
	for _, layout := range []string{time.RFC1123, time.RFC1123Z, time.RFC3339, "Mon, 02-Jan-2006 15:04:05 MST", "Monday, 02-Jan-06 15:04:05 MST"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// NormalizeDomain 把输入（可含协议、端口、路径）规整为小写主机名。
func NormalizeDomain(domain string) string {
	v := strings.ToLower(strings.TrimSpace(domain))
	if strings.Contains(v, "://") {
		if u, err := url.Parse(v); err == nil && u.Hostname() != "" {
			v = u.Hostname()
		}
	}
	v = strings.TrimPrefix(v, ".")
	v, _, _ = strings.Cut(v, "/")
	v, _, _ = strings.Cut(v, ":")
	return v
}

func parseRequestURL(raw string) *url.URL {
	v := strings.TrimSpace(raw)
	if v == "" {
		return nil
	}
	if u, err := url.Parse(v); err == nil && u.Host != "" {
		return u
	}
	if u, err := url.Parse("http://" + v); err == nil && u.Host != "" {
		return u
	}
	return nil
}

func sameKey(a, b Cookie) bool {
	return NormalizeDomain(a.Domain) == NormalizeDomain(b.Domain) && a.Path == b.Path && a.Name == b.Name
}

func (j *Jar) upsertLocked(c Cookie) {
	for i := range j.cookies {
		if sameKey(j.cookies[i], c) {
			c.ID = j.cookies[i].ID
			j.cookies[i] = c
			return
		}
	}
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	j.cookies = append(j.cookies, c)
}

// saveLocked 原子写入文件；调用方需持有锁。
func (j *Jar) saveLocked() {
	if j.path == "" {
		return
	}
	data, err := json.MarshalIndent(j.cookies, "", "  ")
	if err == nil {
		err = os.MkdirAll(filepath.Dir(j.path), 0o755)
	}
	if err == nil {
		tmp := j.path + ".tmp"
		if err = os.WriteFile(tmp, data, 0o600); err == nil {
			err = os.Rename(tmp, j.path)
		}
	}
	if err != nil {
		logger.Error("保存 Cookie 文件失败", "error", err.Error())
	}
}

// All 返回当前全部 Cookie 的副本。
func (j *Jar) All() []Cookie {
	j.mu.Lock()
	defer j.mu.Unlock()
	return append([]Cookie(nil), j.cookies...)
}

// Add 新增一条 Cookie（同域名+路径+名称则覆盖）。
func (j *Jar) Add(c Cookie) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if c.Path == "" {
		c.Path = "/"
	}
	j.upsertLocked(c)
	j.saveLocked()
}

// Update 修改指定 ID 的 Cookie。
func (j *Jar) Update(id string, fn func(*Cookie)) {
	j.mu.Lock()
	defer j.mu.Unlock()
	for i := range j.cookies {
		if j.cookies[i].ID == id {
			fn(&j.cookies[i])
			j.saveLocked()
			return
		}
	}
}

// Remove 删除指定 ID 的 Cookie。
func (j *Jar) Remove(id string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	out := j.cookies[:0]
	for _, c := range j.cookies {
		if c.ID != id {
			out = append(out, c)
		}
	}
	j.cookies = out
	j.saveLocked()
}

// Clear 清空全部 Cookie。
func (j *Jar) Clear() {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.cookies = nil
	j.saveLocked()
}

// ClearDomain 清空某个域名下的 Cookie。
func (j *Jar) ClearDomain(domain string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	d := NormalizeDomain(domain)
	out := j.cookies[:0]
	for _, c := range j.cookies {
		if NormalizeDomain(c.Domain) != d {
			out = append(out, c)
		}
	}
	j.cookies = out
	j.saveLocked()
}

var cookieAttrNames = map[string]bool{"path": true, "domain": true, "expires": true, "max-age": true, "secure": true,
	"httponly": true, "samesite": true, "priority": true, "partitioned": true}

// ImportHeader 解析形如 "a=1; b=2"（可带 "Cookie:" 前缀）的文本并导入，返回导入条数。
func (j *Jar) ImportHeader(raw, domain, path string) int {
	d := NormalizeDomain(domain)
	if d == "" {
		return 0
	}
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimLeft(cleaned, "*-• ")
	if len(cleaned) >= 7 && strings.EqualFold(cleaned[:7], "cookie:") {
		cleaned = cleaned[7:]
	}
	cleaned = strings.NewReplacer("\r\n", "; ", "\n", "; ").Replace(cleaned)
	if path == "" {
		path = "/"
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	count := 0
	for _, seg := range strings.Split(cleaned, ";") {
		name, value, ok := strings.Cut(strings.TrimSpace(seg), "=")
		name = strings.TrimSpace(name)
		if !ok || name == "" || cookieAttrNames[strings.ToLower(name)] {
			continue
		}
		j.upsertLocked(Cookie{Domain: d, Name: name, Value: strings.TrimSpace(value), Path: path, Enabled: true})
		count++
	}
	if count > 0 {
		j.saveLocked()
	}
	return count
}

func parseSetCookie(line, host string) (Cookie, bool) {
	parts := strings.Split(line, ";")
	name, value, ok := strings.Cut(strings.TrimSpace(parts[0]), "=")
	name = strings.TrimSpace(name)
	if !ok || name == "" {
		return Cookie{}, false
	}
	c := Cookie{Domain: host, Name: name, Value: strings.TrimSpace(value), Path: "/", Enabled: true}
	for _, attr := range parts[1:] {
		an, av, _ := strings.Cut(strings.TrimSpace(attr), "=")
		av = strings.TrimSpace(av)
		switch strings.ToLower(strings.TrimSpace(an)) {
		case "domain":
			if av != "" {
				c.Domain = strings.ToLower(strings.TrimPrefix(av, "."))
			}
		case "path":
			if av != "" {
				c.Path = av
			}
		case "expires":
			c.Expires = av
		case "max-age":
			if secs, err := strconv.Atoi(av); err == nil {
				c.Expires = time.Now().Add(time.Duration(secs) * time.Second).UTC().Format(time.RFC1123)
			}
		case "secure":
			c.Secure = true
		case "httponly":
			c.HTTPOnly = true
		}
	}
	return c, true
}

// Absorb 吸收响应头中的 Set-Cookie；已过期的 Cookie 会删除同名旧记录。
func (j *Jar) Absorb(requestURL string, headers map[string][]string) {
	u := parseRequestURL(requestURL)
	if u == nil {
		return
	}
	var lines []string
	for k, vs := range headers {
		if strings.EqualFold(k, "set-cookie") {
			lines = append(lines, vs...)
		}
	}
	if len(lines) == 0 {
		return
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	for _, line := range lines {
		c, ok := parseSetCookie(line, u.Hostname())
		if !ok {
			continue
		}
		if isExpired(c.Expires) {
			out := j.cookies[:0]
			for _, existing := range j.cookies {
				if !sameKey(existing, c) {
					out = append(out, existing)
				}
			}
			j.cookies = out
			continue
		}
		j.upsertLocked(c)
	}
	j.saveLocked()
	logger.Debug("已吸收响应 Cookie", "host", u.Hostname(), "count", len(lines))
}

// Header 返回应随请求携带的 Cookie 头值（域名按后缀且以点边界匹配，路径按前缀匹配）。
func (j *Jar) Header(requestURL string) string {
	u := parseRequestURL(requestURL)
	if u == nil {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	j.mu.Lock()
	defer j.mu.Unlock()
	var pairs []string
	for _, c := range j.cookies {
		if !c.Enabled || c.Name == "" || isExpired(c.Expires) {
			continue
		}
		if d := NormalizeDomain(c.Domain); d != "" && host != d && !strings.HasSuffix(host, "."+d) {
			continue
		}
		if c.Path != "" && !strings.HasPrefix(pathOrRoot(u), c.Path) {
			continue
		}
		if c.Secure && u.Scheme != "https" {
			continue
		}
		pairs = append(pairs, c.Name+"="+c.Value)
	}
	return strings.Join(pairs, "; ")
}

func pathOrRoot(u *url.URL) string {
	if u.Path == "" {
		return "/"
	}
	return u.Path
}

// MergeCookieHeaders 合并手写 Cookie 与 Jar 中的 Cookie，同名以手写为准。
func MergeCookieHeaders(manual, fromJar string) string {
	order := []string{}
	values := map[string]string{}
	add := func(src string, overwrite bool) {
		for _, pair := range strings.Split(src, ";") {
			name, value, ok := strings.Cut(strings.TrimSpace(pair), "=")
			name = strings.TrimSpace(name)
			if !ok || name == "" {
				continue
			}
			if _, exists := values[name]; !exists {
				order = append(order, name)
				values[name] = strings.TrimSpace(value)
			} else if overwrite {
				values[name] = strings.TrimSpace(value)
			}
		}
	}
	add(fromJar, false)
	add(manual, true)
	out := make([]string, 0, len(order))
	for _, n := range order {
		out = append(out, n+"="+values[n])
	}
	return strings.Join(out, "; ")
}
