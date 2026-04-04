package model

// HistoryEntry 请求历史记录条目
type HistoryEntry struct {
	ID         string  `json:"id"`
	RequestID  string  `json:"requestId,omitempty"`
	Name       string  `json:"name"`
	Method     string  `json:"method"`
	URL        string  `json:"url"`
	StatusCode int     `json:"statusCode"`
	Duration   float64 `json:"duration"`
	Size       int64   `json:"size"`
	Timestamp  string  `json:"timestamp"`
}
