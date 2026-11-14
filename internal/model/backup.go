package model

import "time"

// BackupFile 描述备份目录里的一个备份文件，用于设置·数据页的“最近备份”列表。
type BackupFile struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
}
