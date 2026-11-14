// Package client 承载原先由前端（TypeScript）完成的客户端逻辑：
// 请求草稿、URL 与 Params 同步、变量解析、Cookie Jar、自动 Header、cURL 导出与设置持久化。
package client

import "github.com/google/uuid"

// Row 键值行，对应编辑器里的一行 Params / Headers / urlencoded 参数。
type Row struct {
	ID          string
	Enabled     bool
	Key         string
	Value       string
	Description string
}

// NewRow 创建一个启用状态的空行。
func NewRow() *Row { return &Row{ID: uuid.NewString(), Enabled: true} }

// IsPlaceholder 判断是否为末尾的空占位行。
func (r *Row) IsPlaceholder() bool {
	return r.Key == "" && r.Value == "" && r.Description == ""
}

// FormRow form-data 行，类型为 text 或 file。
type FormRow struct {
	Row
	Type     string
	FilePath string
	FileName string
}

// NewFormRow 创建一个文本类型的空 form-data 行。
func NewFormRow() *FormRow {
	return &FormRow{Row: *NewRow(), Type: "text"}
}

// IsPlaceholder 判断 form-data 行是否为空占位行。
func (r *FormRow) IsPlaceholder() bool {
	return r.Key == "" && r.Value == "" && r.FilePath == "" && r.FileName == ""
}

// EnsureTrailingRow 保证列表末尾有且仅有一个空占位行（用于"输入即新增"的表格交互）。
func EnsureTrailingRow(rows []*Row) []*Row {
	if len(rows) == 0 || !rows[len(rows)-1].IsPlaceholder() {
		return append(rows, NewRow())
	}
	return rows
}

// EnsureTrailingFormRow 同 EnsureTrailingRow，作用于 form-data 行。
func EnsureTrailingFormRow(rows []*FormRow) []*FormRow {
	if len(rows) == 0 || !rows[len(rows)-1].IsPlaceholder() {
		return append(rows, NewFormRow())
	}
	return rows
}

// TrimTrailingPlaceholders 移除所有空占位行（保存、比较前使用）。
func TrimTrailingPlaceholders(rows []*Row) []*Row {
	out := make([]*Row, 0, len(rows))
	for _, r := range rows {
		if !r.IsPlaceholder() {
			out = append(out, r)
		}
	}
	return out
}
