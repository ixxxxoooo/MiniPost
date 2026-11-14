package client

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"minipost/internal/model"
	"minipost/internal/pkg/logger"
)

// SavedTab 会话中保存的一个标签页：已保存请求只记 ID，带未保存修改的记录草稿内容。
type SavedTab struct {
	RequestID string             `json:"requestId"`
	Unsaved   *model.RequestItem `json:"unsaved,omitempty"`
}

// ProjectSession 单个项目的标签页状态。
type ProjectSession struct {
	Tabs   []SavedTab `json:"tabs"`
	Active int        `json:"active"`
}

// Session 全部项目的标签页会话，用于重启后恢复标签。
type Session struct {
	Projects map[string]ProjectSession `json:"projects"`
}

// SessionStore 负责会话的读写（数据目录下的 session.json）。
type SessionStore struct {
	mu   sync.Mutex
	path string
}

// NewSessionStore 创建指向 baseDir/session.json 的存储。
func NewSessionStore(baseDir string) *SessionStore {
	return &SessionStore{path: filepath.Join(baseDir, "session.json")}
}

// Load 读取会话；缺失或损坏时返回空会话。
func (st *SessionStore) Load() Session {
	st.mu.Lock()
	defer st.mu.Unlock()
	s := Session{Projects: map[string]ProjectSession{}}
	data, err := os.ReadFile(st.path)
	if err != nil {
		return s
	}
	if err := json.Unmarshal(data, &s); err != nil || s.Projects == nil {
		logger.Warn("解析会话文件失败，已忽略", "error", errString(err))
		return Session{Projects: map[string]ProjectSession{}}
	}
	return s
}

// Save 原子写入会话。
func (st *SessionStore) Save(s Session) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	return writeJSONAtomic(st.path, s)
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
