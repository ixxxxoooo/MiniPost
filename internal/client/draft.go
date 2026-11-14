package client

import (
	"encoding/json"

	"minipost/internal/model"
)

// 请求体与鉴权类型常量，取值与持久化格式保持一致。
const (
	BodyNone       = "none"
	BodyFormData   = "form-data"
	BodyRaw        = "raw"
	BodyJSON       = "json"
	BodyURLEncoded = "form-urlencoded"

	AuthNone   = "none"
	AuthBasic  = "basic"
	AuthBearer = "bearer"
	AuthAPIKey = "api-key"
)

// Methods 支持的 HTTP 方法。
var Methods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

// BodyDraft 请求体草稿。
type BodyDraft struct {
	Type       string
	Raw        string
	JSON       string
	URLEncoded []*Row
	FormData   []*FormRow
}

// AuthDraft 鉴权草稿。
type AuthDraft struct {
	Type           string
	Username       string
	Password       string
	Token          string
	APIKeyName     string
	APIKeyValue    string
	APIKeyLocation string // header | query
}

// Draft 可编辑的请求草稿：界面直接绑定其字段，保存时转换为 model.RequestItem。
type Draft struct {
	ID        string
	Name      string
	Method    string
	URL       string
	Params    []*Row
	Headers   []*Row
	Body      BodyDraft
	Auth      AuthDraft
	FolderID  string
	ProjectID string
	SortOrder int
	CreatedAt string
	UpdatedAt string
}

func rowsFromKV(items []model.KeyValue) []*Row {
	rows := make([]*Row, 0, len(items)+1)
	for _, kv := range items {
		r := NewRow()
		r.Key, r.Value, r.Description, r.Enabled = kv.Key, kv.Value, kv.Description, !kv.Disabled
		rows = append(rows, r)
	}
	return EnsureTrailingRow(rows)
}

func rowsToKV(rows []*Row) []model.KeyValue {
	out := make([]model.KeyValue, 0, len(rows))
	for _, r := range TrimTrailingPlaceholders(rows) {
		out = append(out, model.KeyValue{Key: r.Key, Value: r.Value, Description: r.Description, Disabled: !r.Enabled})
	}
	return out
}

// DraftFromItem 由已保存的请求构造草稿，并为各表格补齐末尾空行。
func DraftFromItem(it model.RequestItem) *Draft {
	d := &Draft{
		ID: it.ID, Name: it.Name, Method: it.Method, URL: it.URL,
		Params: rowsFromKV(it.Params), Headers: rowsFromKV(it.Headers),
		FolderID: it.FolderID, ProjectID: it.ProjectID, SortOrder: it.SortOrder,
		CreatedAt: it.CreatedAt, UpdatedAt: it.UpdatedAt,
	}
	if d.Method == "" {
		d.Method = "GET"
	}
	d.Body = BodyDraft{
		Type: orDefault(it.Body.Type, BodyNone), Raw: it.Body.Raw, JSON: it.Body.JSON,
		URLEncoded: rowsFromKV(it.Body.FormUrlEncoded),
	}
	for _, f := range it.Body.FormData {
		fr := NewFormRow()
		fr.Key, fr.Value, fr.Description, fr.Enabled = f.Key, f.Value, f.Description, !f.Disabled
		fr.Type, fr.FilePath, fr.FileName = orDefault(f.Type, "text"), f.FilePath, f.FileName
		d.Body.FormData = append(d.Body.FormData, fr)
	}
	d.Body.FormData = EnsureTrailingFormRow(d.Body.FormData)
	d.Auth = AuthDraft{
		Type: orDefault(it.Auth.Type, AuthNone), Username: it.Auth.Basic.Username, Password: it.Auth.Basic.Password,
		Token: it.Auth.Bearer.Token, APIKeyName: it.Auth.APIKey.Key, APIKeyValue: it.Auth.APIKey.Value,
		APIKeyLocation: orDefault(it.Auth.APIKey.AddTo, "header"),
	}
	return d
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// ToItem 转为持久化模型（丢弃空占位行）。
func (d *Draft) ToItem() model.RequestItem {
	it := model.RequestItem{
		ID: d.ID, Name: d.Name, Method: d.Method, URL: d.URL,
		Params: rowsToKV(d.Params), Headers: rowsToKV(d.Headers),
		FolderID: d.FolderID, ProjectID: d.ProjectID, SortOrder: d.SortOrder,
		CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
	}
	it.Body = model.RequestBody{
		Type: d.Body.Type, Raw: d.Body.Raw, JSON: d.Body.JSON, FormUrlEncoded: rowsToKV(d.Body.URLEncoded),
		FormData: []model.FormData{},
	}
	for _, f := range d.Body.FormData {
		if f.IsPlaceholder() {
			continue
		}
		it.Body.FormData = append(it.Body.FormData, model.FormData{
			Key: f.Key, Value: f.Value, Description: f.Description, Disabled: !f.Enabled,
			Type: f.Type, FilePath: f.FilePath, FileName: f.FileName,
		})
	}
	it.Auth = model.AuthConfig{
		Type:   d.Auth.Type,
		Basic:  model.BasicAuth{Username: d.Auth.Username, Password: d.Auth.Password},
		Bearer: model.BearerAuth{Token: d.Auth.Token},
		APIKey: model.APIKeyAuth{Key: d.Auth.APIKeyName, Value: d.Auth.APIKeyValue, AddTo: d.Auth.APIKeyLocation},
	}
	return it
}

// Fingerprint 返回草稿内容的指纹（忽略更新时间），用于判断是否有未保存修改。
func (d *Draft) Fingerprint() string {
	it := d.ToItem()
	it.UpdatedAt = ""
	data, _ := json.Marshal(it)
	return string(data)
}

// Clone 深拷贝草稿。
func (d *Draft) Clone() *Draft { return DraftFromItem(d.ToItem()) }

// NewDraft 创建一个全新的空请求草稿。
func NewDraft(projectID, folderID, name string) *Draft {
	return DraftFromItem(model.RequestItem{Name: name, Method: "GET", ProjectID: projectID, FolderID: folderID})
}

// DraftFromSendInput 把解析出的 cURL 请求转换为草稿（用于"导入 cURL"）。
func DraftFromSendInput(in model.SendRequestInput, name, projectID, folderID string) *Draft {
	it := model.RequestItem{
		Name: orDefault(name, "Imported cURL"), Method: orDefault(in.Method, "GET"), URL: in.URL,
		Params: in.Params, Headers: in.Headers, Body: in.Body, Auth: in.Auth,
		ProjectID: projectID, FolderID: folderID,
	}
	switch it.Body.Type {
	case BodyNone, BodyRaw, BodyJSON, BodyURLEncoded, BodyFormData:
	default:
		it.Body.Type = BodyNone
	}
	switch it.Auth.Type {
	case AuthNone, AuthBasic, AuthBearer, AuthAPIKey:
	default:
		it.Auth.Type = AuthNone
	}
	return DraftFromItem(it)
}

// TemplateTexts 列出发送时会做 {{变量}} 替换的所有文本：地址、启用的参数与请求头、
// 当前请求体类型对应的内容，以及当前鉴权方式用到的字段。
func (d *Draft) TemplateTexts() []string {
	out := []string{d.URL}
	addRows := func(rows []*Row) {
		for _, r := range rows {
			if r.Enabled {
				out = append(out, r.Key, r.Value)
			}
		}
	}
	addRows(d.Params)
	addRows(d.Headers)
	switch d.Body.Type {
	case BodyJSON:
		out = append(out, d.Body.JSON)
	case BodyRaw:
		out = append(out, d.Body.Raw)
	case BodyURLEncoded:
		addRows(d.Body.URLEncoded)
	case BodyFormData:
		for _, r := range d.Body.FormData {
			if r.Enabled {
				out = append(out, r.Key)
				if r.Type != "file" {
					out = append(out, r.Value)
				}
			}
		}
	}
	switch d.Auth.Type {
	case AuthBasic:
		out = append(out, d.Auth.Username, d.Auth.Password)
	case AuthBearer:
		out = append(out, d.Auth.Token)
	case AuthAPIKey:
		out = append(out, d.Auth.APIKeyName, d.Auth.APIKeyValue)
	}
	return out
}
