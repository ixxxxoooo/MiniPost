// OpenAPI / Swagger 的结构与导入（含环境变量推断、Schema 展开）。

package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/google/uuid"

	"minipost/internal/model"
)

type openAPIDoc struct {
	Info        openAPIInfo                `json:"info,omitempty"`
	Swagger     string                     `json:"swagger,omitempty"`
	OpenAPI     string                     `json:"openapi,omitempty"`
	Host        string                     `json:"host,omitempty"`
	BasePath    string                     `json:"basePath,omitempty"`
	Schemes     []string                   `json:"schemes,omitempty"`
	Servers     []openAPIServer            `json:"servers,omitempty"`
	Paths       map[string]openAPIPathItem `json:"paths"`
	Definitions map[string]openAPISchema   `json:"definitions,omitempty"`
	Components  openAPIComponents          `json:"components,omitempty"`
}

type openAPIInfo struct {
	Title string `json:"title,omitempty"`
}

type openAPIServer struct {
	URL         string                           `json:"url"`
	Description string                           `json:"description,omitempty"`
	Variables   map[string]openAPIServerVariable `json:"variables,omitempty"`
}

type openAPIServerVariable struct {
	Default     string   `json:"default,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Description string   `json:"description,omitempty"`
}

type openAPIPathItem struct {
	Get        *openAPIOp         `json:"get,omitempty"`
	Post       *openAPIOp         `json:"post,omitempty"`
	Put        *openAPIOp         `json:"put,omitempty"`
	Patch      *openAPIOp         `json:"patch,omitempty"`
	Delete     *openAPIOp         `json:"delete,omitempty"`
	Head       *openAPIOp         `json:"head,omitempty"`
	Options    *openAPIOp         `json:"options,omitempty"`
	Trace      *openAPIOp         `json:"trace,omitempty"`
	Parameters []openAPIParameter `json:"parameters,omitempty"`
}

type openAPIOp struct {
	Tags        []string            `json:"tags,omitempty"`
	Summary     string              `json:"summary,omitempty"`
	OperationID string              `json:"operationId,omitempty"`
	Consumes    []string            `json:"consumes,omitempty"`
	Produces    []string            `json:"produces,omitempty"`
	Parameters  []openAPIParameter  `json:"parameters,omitempty"`
	RequestBody *openAPIRequestBody `json:"requestBody,omitempty"`
}

type openAPIParameter struct {
	Name        string         `json:"name,omitempty"`
	In          string         `json:"in,omitempty"`
	Description string         `json:"description,omitempty"`
	Required    bool           `json:"required,omitempty"`
	Type        string         `json:"type,omitempty"`
	Format      string         `json:"format,omitempty"`
	Schema      *openAPISchema `json:"schema,omitempty"`
}

type openAPIRequestBody struct {
	Description string                      `json:"description,omitempty"`
	Content     map[string]openAPIMediaType `json:"content,omitempty"`
	Required    bool                        `json:"required,omitempty"`
}

type openAPIMediaType struct {
	Schema  *openAPISchema `json:"schema,omitempty"`
	Example any            `json:"example,omitempty"`
}

type openAPIComponents struct {
	Schemas map[string]openAPISchema `json:"schemas,omitempty"`
}

type openAPISchema struct {
	Ref                  string                      `json:"$ref,omitempty"`
	Type                 string                      `json:"type,omitempty"`
	Format               string                      `json:"format,omitempty"`
	Description          string                      `json:"description,omitempty"`
	Properties           map[string]openAPISchema    `json:"properties,omitempty"`
	Items                *openAPISchema              `json:"items,omitempty"`
	AdditionalProperties openAPIAdditionalProperties `json:"additionalProperties,omitempty"`
	Enum                 []any                       `json:"enum,omitempty"`
	Example              any                         `json:"example,omitempty"`
}

type openAPIAdditionalProperties struct {
	Allowed bool
	Schema  *openAPISchema
	Set     bool
}

func (a *openAPIAdditionalProperties) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		*a = openAPIAdditionalProperties{}
		return nil
	}

	if bytes.Equal(trimmed, []byte("true")) {
		a.Allowed = true
		a.Schema = nil
		a.Set = true
		return nil
	}
	if bytes.Equal(trimmed, []byte("false")) {
		a.Allowed = false
		a.Schema = nil
		a.Set = true
		return nil
	}

	var schema openAPISchema
	if err := json.Unmarshal(trimmed, &schema); err != nil {
		return err
	}
	a.Allowed = true
	a.Schema = &schema
	a.Set = true
	return nil
}

func selectOpenAPIVariableValue(variable openAPIServerVariable) string {
	if strings.TrimSpace(variable.Default) != "" {
		return variable.Default
	}
	if len(variable.Enum) > 0 {
		return variable.Enum[0]
	}
	return ""
}

func convertOpenAPIServerURLTemplate(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	var builder strings.Builder
	for i := 0; i < len(trimmed); i++ {
		if trimmed[i] != '{' {
			builder.WriteByte(trimmed[i])
			continue
		}
		end := strings.IndexByte(trimmed[i+1:], '}')
		if end < 0 {
			builder.WriteByte(trimmed[i])
			continue
		}
		key := strings.TrimSpace(trimmed[i+1 : i+1+end])
		if key == "" {
			builder.WriteString("{}")
		} else {
			builder.WriteString("{{")
			builder.WriteString(key)
			builder.WriteString("}}")
		}
		i += end + 1
	}
	return builder.String()
}

func joinURLTemplate(baseURL, path string) string {
	trimmedBase := strings.TrimSpace(baseURL)
	trimmedPath := strings.TrimSpace(path)
	switch {
	case trimmedBase == "":
		return trimmedPath
	case trimmedPath == "":
		return trimmedBase
	case strings.HasSuffix(trimmedBase, "/") && strings.HasPrefix(trimmedPath, "/"):
		return trimmedBase + strings.TrimPrefix(trimmedPath, "/")
	case !strings.HasSuffix(trimmedBase, "/") && !strings.HasPrefix(trimmedPath, "/"):
		return trimmedBase + "/" + trimmedPath
	default:
		return trimmedBase + trimmedPath
	}
}

func inferBaseURLFromImportSource(sourceURL string) string {
	trimmed := strings.TrimSpace(sourceURL)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}

	path := parsed.Path
	suffixes := []string{
		"/swagger/doc.json",
		"/swagger/openapi.json",
		"/swagger/swagger.json",
		"/v3/api-docs",
		"/v2/api-docs",
		"/api-docs",
		"/openapi.json",
		"/swagger.json",
		"/doc.json",
	}
	for _, suffix := range suffixes {
		if strings.HasSuffix(path, suffix) {
			path = strings.TrimSuffix(path, suffix)
			if path == "" {
				path = "/"
			}
			parsed.Path = path
			parsed.RawQuery = ""
			parsed.Fragment = ""
			return strings.TrimRight(parsed.String(), "/")
		}
	}

	lastSlash := strings.LastIndex(path, "/")
	if lastSlash >= 0 {
		path = path[:lastSlash]
	}
	if path == "" {
		path = "/"
	}
	parsed.Path = path
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/")
}

func replacePathPlaceholders(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	var builder strings.Builder
	for i := 0; i < len(trimmed); i++ {
		if trimmed[i] != '{' {
			builder.WriteByte(trimmed[i])
			continue
		}
		end := strings.IndexByte(trimmed[i+1:], '}')
		if end < 0 {
			builder.WriteByte(trimmed[i])
			continue
		}
		key := strings.TrimSpace(trimmed[i+1 : i+1+end])
		if key == "" {
			builder.WriteString("{}")
		} else {
			builder.WriteString("{{")
			builder.WriteString(key)
			builder.WriteString("}}")
		}
		i += end + 1
	}
	return builder.String()
}

func isJSONMediaType(mediaType string) bool {
	normalized := strings.ToLower(strings.TrimSpace(mediaType))
	return strings.Contains(normalized, "json") || strings.HasSuffix(normalized, "+json")
}

func looksLikeFileParameter(param openAPIParameter) bool {
	return strings.EqualFold(strings.TrimSpace(param.Type), "file") || strings.EqualFold(strings.TrimSpace(param.Format), "binary")
}

func defaultValueForPrimitiveType(schemaType string) any {
	switch strings.ToLower(strings.TrimSpace(schemaType)) {
	case "integer", "number":
		return 0
	case "boolean":
		return false
	case "array":
		return []any{}
	case "object":
		return map[string]any{}
	default:
		return ""
	}
}

func resolveOpenAPISchema(schema *openAPISchema, doc openAPIDoc, seen map[string]bool) any {
	if schema == nil {
		return map[string]any{}
	}
	if schema.Example != nil {
		return schema.Example
	}
	if ref := strings.TrimSpace(schema.Ref); ref != "" {
		if seen[ref] {
			return map[string]any{}
		}
		seen[ref] = true
		defer delete(seen, ref)

		name := strings.TrimPrefix(ref, "#/definitions/")
		if definition, ok := doc.Definitions[name]; ok {
			return resolveOpenAPISchema(&definition, doc, seen)
		}
		name = strings.TrimPrefix(ref, "#/components/schemas/")
		if definition, ok := doc.Components.Schemas[name]; ok {
			return resolveOpenAPISchema(&definition, doc, seen)
		}
		return map[string]any{}
	}

	switch strings.ToLower(strings.TrimSpace(schema.Type)) {
	case "object", "":
		if len(schema.Properties) == 0 {
			if schema.AdditionalProperties.Set {
				if schema.AdditionalProperties.Schema != nil {
					return map[string]any{
						"key": resolveOpenAPISchema(schema.AdditionalProperties.Schema, doc, seen),
					}
				}
				if schema.AdditionalProperties.Allowed {
					return map[string]any{
						"key": "",
					}
				}
				return map[string]any{}
			}
			if schema.AdditionalProperties.Schema != nil {
				return map[string]any{
					"key": resolveOpenAPISchema(schema.AdditionalProperties.Schema, doc, seen),
				}
			}
			return map[string]any{}
		}
		keys := make([]string, 0, len(schema.Properties))
		for key := range schema.Properties {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		result := make(map[string]any, len(keys))
		for _, key := range keys {
			property := schema.Properties[key]
			result[key] = resolveOpenAPISchema(&property, doc, seen)
		}
		return result
	case "array":
		if schema.Items == nil {
			return []any{}
		}
		return []any{resolveOpenAPISchema(schema.Items, doc, seen)}
	default:
		if len(schema.Enum) > 0 {
			return schema.Enum[0]
		}
		return defaultValueForPrimitiveType(schema.Type)
	}
}

func resolveOpenAPISchemaRef(schema *openAPISchema, doc openAPIDoc, seen map[string]bool) *openAPISchema {
	if schema == nil {
		return nil
	}
	if ref := strings.TrimSpace(schema.Ref); ref != "" {
		if seen[ref] {
			return schema
		}
		seen[ref] = true
		defer delete(seen, ref)

		name := strings.TrimPrefix(ref, "#/definitions/")
		if definition, ok := doc.Definitions[name]; ok {
			return resolveOpenAPISchemaRef(&definition, doc, seen)
		}
		name = strings.TrimPrefix(ref, "#/components/schemas/")
		if definition, ok := doc.Components.Schemas[name]; ok {
			return resolveOpenAPISchemaRef(&definition, doc, seen)
		}
	}
	return schema
}

func collectSchemaDescriptions(prefix string, schema *openAPISchema, doc openAPIDoc, lines *[]string, seen map[string]bool) {
	resolved := resolveOpenAPISchemaRef(schema, doc, seen)
	if resolved == nil {
		return
	}
	if desc := strings.TrimSpace(resolved.Description); desc != "" && prefix != "" {
		*lines = append(*lines, fmt.Sprintf("// %s: %s", prefix, desc))
	}
	if len(resolved.Properties) == 0 {
		return
	}
	keys := make([]string, 0, len(resolved.Properties))
	for key := range resolved.Properties {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		prop := resolved.Properties[key]
		nextPrefix := key
		if prefix != "" {
			nextPrefix = prefix + "." + key
		}
		collectSchemaDescriptions(nextPrefix, &prop, doc, lines, seen)
	}
}

func buildOpenAPIBody(schema *openAPISchema, doc openAPIDoc) string {
	value := resolveOpenAPISchema(schema, doc, map[string]bool{})
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "{}"
	}
	lines := make([]string, 0)
	if resolved := resolveOpenAPISchemaRef(schema, doc, map[string]bool{}); resolved != nil {
		if desc := strings.TrimSpace(resolved.Description); desc != "" {
			lines = append(lines, fmt.Sprintf("// body: %s", desc))
		}
	}
	collectSchemaDescriptions("", schema, doc, &lines, map[string]bool{})
	if len(lines) == 0 {
		return string(body)
	}
	return strings.Join(append(lines, string(body)), "\n")
}

func applyOpenAPIParameters(req *model.RequestItem, parameters []openAPIParameter, doc openAPIDoc) {
	for _, parameter := range parameters {
		name := strings.TrimSpace(parameter.Name)
		if name == "" {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(parameter.In)) {
		case "query":
			req.Params = append(req.Params, model.KeyValue{Key: name, Value: "", Description: strings.TrimSpace(parameter.Description)})
		case "header":
			req.Headers = append(req.Headers, model.KeyValue{Key: name, Value: "", Description: strings.TrimSpace(parameter.Description)})
		case "path":
			if !strings.Contains(req.URL, "{{"+name+"}}") {
				req.URL = strings.ReplaceAll(req.URL, "{"+name+"}", "{{"+name+"}}")
			}
		case "formdata":
			if req.Body.Type != "form-data" {
				req.Body = model.RequestBody{Type: "form-data", FormData: []model.FormData{}}
			}
			entryType := "text"
			if looksLikeFileParameter(parameter) {
				entryType = "file"
			}
			req.Body.FormData = append(req.Body.FormData, model.FormData{
				Key:         name,
				Type:        entryType,
				Value:       "",
				Description: strings.TrimSpace(parameter.Description),
			})
		case "body":
			if parameter.Schema != nil {
				bodyJSON := buildOpenAPIBody(parameter.Schema, doc)
				if desc := strings.TrimSpace(parameter.Description); desc != "" {
					bodyJSON = "// body: " + desc + "\n" + bodyJSON
				}
				req.Body = model.RequestBody{Type: "json", JSON: bodyJSON}
				req.Headers = append(req.Headers, model.KeyValue{Key: "Content-Type", Value: "application/json"})
			}
		}
	}
}

func applyOpenAPIRequestBody(req *model.RequestItem, body *openAPIRequestBody, doc openAPIDoc) {
	if body == nil || len(body.Content) == 0 {
		return
	}

	var mediaTypes []string
	for mediaType := range body.Content {
		mediaTypes = append(mediaTypes, mediaType)
	}
	sort.Strings(mediaTypes)
	for _, mediaType := range mediaTypes {
		content := body.Content[mediaType]
		if isJSONMediaType(mediaType) {
			bodyJSON := buildOpenAPIBody(content.Schema, doc)
			if desc := strings.TrimSpace(body.Description); desc != "" {
				bodyJSON = "// body: " + desc + "\n" + bodyJSON
			}
			req.Body = model.RequestBody{Type: "json", JSON: bodyJSON}
			req.Headers = append(req.Headers, model.KeyValue{Key: "Content-Type", Value: mediaType})
			return
		}
		if strings.Contains(strings.ToLower(mediaType), "x-www-form-urlencoded") {
			req.Body = model.RequestBody{Type: "form-urlencoded", FormUrlEncoded: []model.KeyValue{}}
			req.Headers = append(req.Headers, model.KeyValue{Key: "Content-Type", Value: mediaType})
			return
		}
		if strings.Contains(strings.ToLower(mediaType), "form-data") {
			req.Body = model.RequestBody{Type: "form-data", FormData: []model.FormData{}}
			req.Headers = append(req.Headers, model.KeyValue{Key: "Content-Type", Value: mediaType})
			return
		}
	}
}

func collectOpenAPIOperations(pathItem openAPIPathItem) []struct {
	method     string
	operation  *openAPIOp
	parameters []openAPIParameter
} {
	return []struct {
		method     string
		operation  *openAPIOp
		parameters []openAPIParameter
	}{
		{method: "GET", operation: pathItem.Get, parameters: append([]openAPIParameter{}, pathItem.Parameters...)},
		{method: "POST", operation: pathItem.Post, parameters: append([]openAPIParameter{}, pathItem.Parameters...)},
		{method: "PUT", operation: pathItem.Put, parameters: append([]openAPIParameter{}, pathItem.Parameters...)},
		{method: "PATCH", operation: pathItem.Patch, parameters: append([]openAPIParameter{}, pathItem.Parameters...)},
		{method: "DELETE", operation: pathItem.Delete, parameters: append([]openAPIParameter{}, pathItem.Parameters...)},
		{method: "HEAD", operation: pathItem.Head, parameters: append([]openAPIParameter{}, pathItem.Parameters...)},
		{method: "OPTIONS", operation: pathItem.Options, parameters: append([]openAPIParameter{}, pathItem.Parameters...)},
		{method: "TRACE", operation: pathItem.Trace, parameters: append([]openAPIParameter{}, pathItem.Parameters...)},
	}
}

func (s *RequestService) importOpenAPIEnvironments(projectID string, doc openAPIDoc, sourceURL string, reuseExisting bool) (string, error) {
	baseName := strings.TrimSpace(doc.Info.Title)
	if baseName == "" {
		baseName = "OpenAPI"
	}

	if len(doc.Servers) > 0 {
		for index, server := range doc.Servers {
			baseURL := convertOpenAPIServerURLTemplate(server.URL)
			if baseURL == "" {
				continue
			}

			variableNames := make([]string, 0, len(server.Variables))
			for name := range server.Variables {
				variableNames = append(variableNames, name)
			}
			sort.Strings(variableNames)

			variables := make([]model.Variable, 0, len(variableNames)+1)
			variables = append(variables, model.Variable{
				ID:       uuid.New().String(),
				Key:      "baseUrl",
				Value:    baseURL,
				Enabled:  true,
				IsSecret: false,
			})
			for _, name := range variableNames {
				variables = append(variables, model.Variable{
					ID:       uuid.New().String(),
					Key:      name,
					Value:    selectOpenAPIVariableValue(server.Variables[name]),
					Enabled:  true,
					IsSecret: false,
				})
			}

			envName := baseName
			if len(doc.Servers) > 1 {
				envName = fmt.Sprintf("%s Server %d", baseName, index+1)
			}
			if strings.TrimSpace(server.Description) != "" && len(doc.Servers) > 1 {
				envName = fmt.Sprintf("%s - %s", baseName, strings.TrimSpace(server.Description))
			}
			if err := s.saveImportedEnvironment(projectID, envName, variables, reuseExisting); err != nil {
				return "", err
			}
		}
		return "{{baseUrl}}", nil
	}

	if strings.TrimSpace(doc.Host) == "" {
		fallbackBaseURL := inferBaseURLFromImportSource(sourceURL)
		if fallbackBaseURL == "" {
			return "", nil
		}
		if err := s.saveImportedEnvironment(projectID, baseName, []model.Variable{
			{
				ID:       uuid.New().String(),
				Key:      "baseUrl",
				Value:    fallbackBaseURL,
				Enabled:  true,
				IsSecret: false,
			},
		}, reuseExisting); err != nil {
			return "", err
		}
		return "{{baseUrl}}", nil
	}

	scheme := "https"
	if len(doc.Schemes) > 0 && strings.TrimSpace(doc.Schemes[0]) != "" {
		scheme = strings.TrimSpace(doc.Schemes[0])
	}
	baseURL := scheme + "://" + strings.TrimSpace(doc.Host)
	if basePath := strings.TrimSpace(doc.BasePath); basePath != "" && basePath != "/" {
		baseURL = joinURLTemplate(baseURL, basePath)
	}
	if err := s.saveImportedEnvironment(projectID, baseName, []model.Variable{
		{
			ID:       uuid.New().String(),
			Key:      "baseUrl",
			Value:    baseURL,
			Enabled:  true,
			IsSecret: false,
		},
	}, reuseExisting); err != nil {
		return "", err
	}
	return "{{baseUrl}}", nil
}

// ImportSwagger 导入 OpenAPI/Swagger 2.0 或 3.x 格式
func (s *RequestService) ImportSwagger(projectID string, raw []byte) error {
	return s.ImportSwaggerWithSource(projectID, raw, "")
}

// ImportSwaggerWithSource 导入 OpenAPI/Swagger（历史行为：冲突时直接新建副本）
func (s *RequestService) ImportSwaggerWithSource(projectID string, raw []byte, sourceURL string) error {
	_, err := s.ImportSwaggerWithSourceStrategy(projectID, raw, sourceURL, ImportStrategyCopy)
	return err
}

// ImportSwaggerWithSourceStrategy 按指定冲突策略导入 OpenAPI/Swagger 2.0 或 3.x 格式
func (s *RequestService) ImportSwaggerWithSourceStrategy(projectID string, raw []byte, sourceURL, strategy string) (*ImportSummary, error) {
	var doc openAPIDoc
	if err := unmarshalJSONPossiblyCommented(raw, &doc); err != nil {
		return nil, fmt.Errorf("解析 OpenAPI JSON 失败: %w", err)
	}

	applier, err := s.newImportApplier(projectID, strategy)
	if err != nil {
		return nil, err
	}

	baseURLTemplate, err := s.importOpenAPIEnvironments(projectID, doc, sourceURL, applier.strategy != ImportStrategyCopy)
	if err != nil {
		return nil, err
	}

	// 按 tag 分组创建文件夹
	tagFolders := make(map[string]string) // tag -> folderID

	for pathStr, pathItem := range doc.Paths {
		for _, entry := range collectOpenAPIOperations(pathItem) {
			if entry.operation == nil {
				continue
			}
			op := entry.operation
			methodUpper := entry.method
			folderID := ""
			if len(op.Tags) > 0 {
				tag := op.Tags[0]
				if fid, ok := tagFolders[tag]; ok {
					folderID = fid
				} else {
					fid, err := applier.ensureFolder("", tag)
					if err != nil {
						return nil, err
					}
					tagFolders[tag] = fid
					folderID = fid
				}
			}

			name := op.Summary
			if name == "" {
				name = op.OperationID
			}
			if name == "" {
				name = methodUpper + " " + pathStr
			}

			req := newImportedRequestTemplate(name)
			req.Method = methodUpper
			req.URL = joinURLTemplate(baseURLTemplate, replacePathPlaceholders(pathStr))
			applyOpenAPIParameters(req, append(entry.parameters, op.Parameters...), doc)
			applyOpenAPIRequestBody(req, op.RequestBody, doc)
			if err := applier.applyRequest(folderID, req); err != nil {
				return nil, err
			}
		}
	}
	return &applier.summary, nil
}
