package client

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"minipost/internal/pkg/logger"
)

// Settings 用户设置，持久化在数据目录的 settings.json 中。
type Settings struct {
	Locale            string  `json:"locale"`      // zh-CN | en-US，空表示跟随系统
	Theme             string  `json:"theme"`       // light | dark | system
	ColorScheme       string  `json:"colorScheme"` // mist（雾蓝）| ink（墨白）| jade（松石）
	FontSize          int     `json:"fontSize"`
	SidebarWidth      float32 `json:"sidebarWidth"`
	SidebarCollapsed  bool    `json:"sidebarCollapsed"`
	ScrollbarAutoHide bool    `json:"scrollbarAutoHide"`
	LayoutDirection   string  `json:"layoutDirection"` // vertical | horizontal
	ConsoleOpen       bool    `json:"consoleOpen"`
	ConsoleHeight     float32 `json:"consoleHeight"`

	FollowRedirects         bool   `json:"followRedirects"`
	HTTPVersion             string `json:"httpVersion"` // auto | http1 | http2
	RequestTimeoutMs        int    `json:"requestTimeoutMs"`
	MaxResponseSizeMB       int    `json:"maxResponseSizeMB"`
	SSLVerify               bool   `json:"sslVerify"`
	SSLTLSKeyLog            bool   `json:"sslTlsKeyLog"`
	DisableCookies          bool   `json:"disableCookies"`
	SendNoCacheHeader       bool   `json:"sendNoCacheHeader"`
	SendPostmanTokenHeader  bool   `json:"sendPostmanTokenHeader"`
	ResponseFormatDetection string `json:"responseFormatDetection"` // auto | json

	AlwaysDiscardUnsaved bool `json:"alwaysDiscardUnsaved"`
	AlwaysSaveUnsaved    bool `json:"alwaysSaveUnsaved"`

	AutoBackupEnabled         bool `json:"autoBackupEnabled"`
	AutoBackupIntervalMinutes int  `json:"autoBackupIntervalMinutes"`

	NoAutoUpdate bool `json:"noAutoUpdate,omitempty"` // 发现新版本后不自动下载（默认自动下载）

	LastProjectID string            `json:"lastProjectId"`
	EnvByProject  map[string]string `json:"envByProject"`
}

// DefaultSettings 返回与旧版一致的默认设置。
// 控制台面板高度的取值范围与默认值（px），可在界面里拖动面板上边缘调整。
const (
	ConsoleMinHeight     = 80
	ConsoleMaxHeight     = 1200
	ConsoleDefaultHeight = 220
)

func DefaultSettings() Settings {
	return Settings{
		Theme: "system", ColorScheme: "mist", FontSize: 12, SidebarWidth: 240, LayoutDirection: "vertical", ConsoleHeight: ConsoleDefaultHeight,
		FollowRedirects: true, HTTPVersion: "auto", MaxResponseSizeMB: 50, SSLVerify: true, ScrollbarAutoHide: true,
		ResponseFormatDetection: "auto", AutoBackupIntervalMinutes: 30, EnvByProject: map[string]string{},
	}
}

func clampInt(v, lo, hi int) int { return max(lo, min(hi, v)) }

// Normalize 修正非法取值，保证各字段落在合法范围内。
func (s *Settings) Normalize() {
	switch s.Theme {
	case "light", "dark", "system":
	default:
		s.Theme = "system"
	}
	switch s.ColorScheme {
	case "mist", "ink", "jade":
	default:
		s.ColorScheme = "mist"
	}
	switch s.HTTPVersion {
	case "auto", "http1", "http2":
	default:
		s.HTTPVersion = "auto"
	}
	if s.ResponseFormatDetection != "json" {
		s.ResponseFormatDetection = "auto"
	}
	if s.LayoutDirection != "horizontal" {
		s.LayoutDirection = "vertical"
	}
	s.FontSize = clampInt(s.FontSize, 10, 16)
	s.RequestTimeoutMs = clampInt(s.RequestTimeoutMs, 0, 3_600_000)
	s.MaxResponseSizeMB = clampInt(s.MaxResponseSizeMB, 0, 2048)
	s.AutoBackupIntervalMinutes = clampInt(s.AutoBackupIntervalMinutes, 5, 1440)
	s.SidebarWidth = max(180, min(500, s.SidebarWidth))
	s.ConsoleHeight = max(ConsoleMinHeight, min(ConsoleMaxHeight, s.ConsoleHeight))
	if s.AlwaysSaveUnsaved {
		s.AlwaysDiscardUnsaved = false
	}
	if s.EnvByProject == nil {
		s.EnvByProject = map[string]string{}
	}
}

// SettingsStore 负责设置的读写。
type SettingsStore struct {
	mu   sync.Mutex
	path string
}

// NewSettingsStore 创建指向 baseDir/settings.json 的存储。
func NewSettingsStore(baseDir string) *SettingsStore {
	return &SettingsStore{path: filepath.Join(baseDir, "settings.json")}
}

// Load 读取设置；文件缺失或损坏时返回默认值。
func (st *SettingsStore) Load() Settings {
	st.mu.Lock()
	defer st.mu.Unlock()
	s := DefaultSettings()
	data, err := os.ReadFile(st.path)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Warn("读取设置失败，使用默认值", "error", err.Error())
		}
		return s
	}
	if err := json.Unmarshal(data, &s); err != nil {
		logger.Warn("解析设置失败，使用默认值", "error", err.Error())
		return DefaultSettings()
	}
	s.Normalize()
	return s
}

// Save 原子写入设置。
func (st *SettingsStore) Save(s Settings) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	s.Normalize()
	return writeJSONAtomic(st.path, s)
}

func writeJSONAtomic(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
