package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"minipost/internal/model"
	"minipost/internal/repository"
)

// ---- Postman Collection v2.1 结构 ----

type postmanCollection struct {
	Info     postmanInfo       `json:"info"`
	Item     []postmanItem     `json:"item"`
	Variable []postmanVariable `json:"variable,omitempty"`
}

type postmanInfo struct {
	Name   string `json:"name"`
	Schema string `json:"schema,omitempty"`
}

type postmanItem struct {
	Name        string        `json:"name"`
	Item        []postmanItem `json:"item,omitempty"`
	Request     *postmanReq   `json:"request,omitempty"`
	Description string        `json:"description,omitempty"`
}

type postmanEnvironment struct {
	Name          string            `json:"name"`
	Values        []postmanVariable `json:"values"`
	VariableScope string            `json:"_postman_variable_scope,omitempty"`
}

type postmanReq struct {
	Method string       `json:"method"`
	Header []postmanKV  `json:"header"`
	Body   *postmanBody `json:"body,omitempty"`
	URL    postmanURL   `json:"url"`
}

type postmanURL struct {
	Raw string `json:"raw"`
}

func (u *postmanURL) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		*u = postmanURL{}
		return nil
	}

	// Postman 有两种 url 结构：字符串或对象
	if trimmed[0] == '"' {
		var raw string
		if err := json.Unmarshal(trimmed, &raw); err != nil {
			return err
		}
		u.Raw = raw
		return nil
	}

	var parsed struct {
		Raw      string      `json:"raw"`
		Protocol string      `json:"protocol"`
		Host     []string    `json:"host"`
		Path     []string    `json:"path"`
		Query    []postmanKV `json:"query"`
	}
	if err := json.Unmarshal(trimmed, &parsed); err != nil {
		return err
	}

	if parsed.Raw != "" {
		u.Raw = parsed.Raw
		return nil
	}

	host := strings.Join(parsed.Host, ".")
	path := strings.Join(parsed.Path, "/")
	var b strings.Builder
	if parsed.Protocol != "" {
		b.WriteString(parsed.Protocol)
		b.WriteString("://")
	}
	b.WriteString(host)
	if path != "" {
		if !strings.HasPrefix(path, "/") {
			b.WriteString("/")
		}
		b.WriteString(path)
	}
	if len(parsed.Query) > 0 {
		values := url.Values{}
		for _, q := range parsed.Query {
			if strings.TrimSpace(q.Key) == "" {
				continue
			}
			values.Add(q.Key, q.Value)
		}
		encoded := values.Encode()
		if encoded != "" {
			b.WriteString("?")
			b.WriteString(encoded)
		}
	}

	u.Raw = b.String()
	return nil
}

type postmanKV struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
}

type postmanVariable struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Type     string `json:"type,omitempty"`
	Disabled bool   `json:"disabled,omitempty"`
}

type postmanBody struct {
	Mode       string            `json:"mode"`
	Raw        string            `json:"raw,omitempty"`
	URLEncoded []postmanKV       `json:"urlencoded,omitempty"`
	FormData   []postmanFormData `json:"formdata,omitempty"`
	Options    postmanOpt        `json:"options,omitempty"`
}

type postmanFormData struct {
	Key         string          `json:"key"`
	Value       string          `json:"value,omitempty"`
	Description string          `json:"description,omitempty"`
	Type        string          `json:"type,omitempty"` // text | file
	Src         json.RawMessage `json:"src,omitempty"`  // string or string[]
}

type postmanOpt struct {
	Raw struct {
		Language string `json:"language"`
	} `json:"raw"`
}

// ---- OpenAPI/Swagger 结构 ----

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

type RequestService struct {
	store *repository.FileStore
}

func NewRequestService(store *repository.FileStore) *RequestService {
	return &RequestService{store: store}
}

func stripJSONComments(raw []byte) []byte {
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

func unmarshalJSONPossiblyCommented(raw []byte, target any) error {
	if err := json.Unmarshal(raw, target); err == nil {
		return nil
	}
	cleaned := stripJSONComments(raw)
	return json.Unmarshal(cleaned, target)
}

func normalizeImportedVariables(variables []model.Variable) []model.Variable {
	if len(variables) == 0 {
		return nil
	}

	seen := make(map[string]int, len(variables))
	result := make([]model.Variable, 0, len(variables))
	for _, variable := range variables {
		key := strings.TrimSpace(variable.Key)
		if key == "" {
			continue
		}
		variable.Key = key
		if variable.ID == "" {
			variable.ID = uuid.New().String()
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = len(result)
		result = append(result, variable)
	}
	return result
}

func (s *RequestService) uniqueEnvironmentName(projectID, preferred string) (string, error) {
	base := strings.TrimSpace(preferred)
	if base == "" {
		base = "Imported Environment"
	}

	envs, err := s.store.ListEnvironments(projectID)
	if err != nil {
		return "", err
	}
	existing := make(map[string]struct{}, len(envs))
	for _, env := range envs {
		existing[env.Name] = struct{}{}
	}
	if _, ok := existing[base]; !ok {
		return base, nil
	}
	for index := 2; ; index++ {
		candidate := fmt.Sprintf("%s (%d)", base, index)
		if _, ok := existing[candidate]; !ok {
			return candidate, nil
		}
	}
}

func (s *RequestService) saveImportedEnvironment(projectID, preferredName string, variables []model.Variable) error {
	variables = normalizeImportedVariables(variables)
	if len(variables) == 0 {
		return nil
	}

	name, err := s.uniqueEnvironmentName(projectID, preferredName)
	if err != nil {
		return err
	}

	return s.store.SaveEnvironment(&model.Environment{
		ID:        uuid.New().String(),
		Name:      name,
		ProjectID: projectID,
		Variables: variables,
	})
}

func convertPostmanVariables(variables []postmanVariable) []model.Variable {
	result := make([]model.Variable, 0, len(variables))
	for _, variable := range variables {
		key := strings.TrimSpace(variable.Key)
		if key == "" {
			continue
		}
		result = append(result, model.Variable{
			ID:       uuid.New().String(),
			Key:      key,
			Value:    variable.Value,
			Enabled:  !variable.Disabled,
			IsSecret: strings.EqualFold(variable.Type, "secret"),
		})
	}
	return result
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

func (s *RequestService) importOpenAPIEnvironments(projectID string, doc openAPIDoc, sourceURL string) (string, error) {
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
			if err := s.saveImportedEnvironment(projectID, envName, variables); err != nil {
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
		}); err != nil {
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
	}); err != nil {
		return "", err
	}
	return "{{baseUrl}}", nil
}

func (s *RequestService) GetCollectionData(projectID string) (*model.CollectionData, error) {
	return s.store.GetCollectionData(projectID)
}

func (s *RequestService) ListRequests(projectID string) ([]model.RequestItem, error) {
	return s.store.ListRequests(projectID)
}

func (s *RequestService) CreateRequest(projectID, folderID, name string) (*model.RequestItem, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	req := &model.RequestItem{
		ID:        uuid.New().String(),
		Name:      name,
		Method:    "GET",
		URL:       "",
		Params:    []model.KeyValue{},
		Headers:   []model.KeyValue{},
		Body:      model.RequestBody{Type: "none"},
		Auth:      model.AuthConfig{Type: "none"},
		FolderID:  folderID,
		SortOrder: 0,
		ProjectID: projectID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.store.SaveRequest(req); err != nil {
		return nil, err
	}
	return req, nil
}

func (s *RequestService) SaveRequest(request *model.RequestItem) error {
	request.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return s.store.SaveRequest(request)
}

func (s *RequestService) DeleteRequest(projectID, requestID string) error {
	return s.store.DeleteRequest(projectID, requestID)
}

func (s *RequestService) ListFolders(projectID string) ([]model.Folder, error) {
	return s.store.ListFolders(projectID)
}

func (s *RequestService) CreateFolder(projectID, parentID, name string) (*model.Folder, error) {
	folder := &model.Folder{
		ID:        uuid.New().String(),
		Name:      name,
		ProjectID: projectID,
		ParentID:  parentID,
		SortOrder: 0,
	}
	if err := s.store.SaveFolder(folder); err != nil {
		return nil, err
	}
	return folder, nil
}

func (s *RequestService) RenameFolder(projectID, folderID, name string) error {
	folders, err := s.store.ListFolders(projectID)
	if err != nil {
		return err
	}
	for _, f := range folders {
		if f.ID == folderID {
			f.Name = name
			return s.store.SaveFolder(&f)
		}
	}
	return nil
}

func (s *RequestService) MoveCollectionNode(projectID, nodeID string, nodeType model.CollectionNodeType, targetParentID string, targetIndex int) error {
	return s.store.MoveCollectionNode(projectID, nodeID, nodeType, targetParentID, targetIndex)
}

func (s *RequestService) MoveFolder(projectID, folderID, targetParentID string, targetIndex int) error {
	return s.store.MoveFolder(projectID, folderID, targetParentID, targetIndex)
}

func (s *RequestService) MoveRequest(projectID, requestID, targetFolderID string, targetIndex int) error {
	return s.store.MoveRequest(projectID, requestID, targetFolderID, targetIndex)
}

func (s *RequestService) DeleteFolder(projectID, folderID string) error {
	return s.store.DeleteFolder(projectID, folderID)
}

func (s *RequestService) RenameRequest(projectID, requestID, name string) error {
	requests, err := s.store.ListRequests(projectID)
	if err != nil {
		return err
	}
	for _, r := range requests {
		if r.ID == requestID {
			r.Name = name
			r.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			return s.store.SaveRequest(&r)
		}
	}
	return nil
}

func (s *RequestService) DuplicateRequest(projectID, requestID string) (*model.RequestItem, error) {
	requests, err := s.store.ListRequests(projectID)
	if err != nil {
		return nil, err
	}
	for _, r := range requests {
		if r.ID == requestID {
			now := time.Now().UTC().Format(time.RFC3339)
			dup := r
			dup.ID = uuid.New().String()
			dup.Name = r.Name + " (副本)"
			dup.CreatedAt = now
			dup.UpdatedAt = now
			if err := s.store.SaveRequest(&dup); err != nil {
				return nil, err
			}
			return &dup, nil
		}
	}
	return nil, fmt.Errorf("请求 %s 不存在", requestID)
}

func (s *RequestService) DuplicateFolder(projectID, folderID string) (*model.Folder, error) {
	folders, err := s.store.ListFolders(projectID)
	if err != nil {
		return nil, err
	}
	for _, f := range folders {
		if f.ID == folderID {
			dup := &model.Folder{
				ID:        uuid.New().String(),
				Name:      f.Name + " (副本)",
				ProjectID: projectID,
				ParentID:  f.ParentID,
				SortOrder: 0,
			}
			if err := s.store.SaveFolder(dup); err != nil {
				return nil, err
			}
			// 复制文件夹内的请求
			requests, _ := s.store.ListRequests(projectID)
			for _, r := range requests {
				if r.FolderID == folderID {
					now := time.Now().UTC().Format(time.RFC3339)
					dupReq := r
					dupReq.ID = uuid.New().String()
					dupReq.FolderID = dup.ID
					dupReq.CreatedAt = now
					dupReq.UpdatedAt = now
					_ = s.store.SaveRequest(&dupReq)
				}
			}
			return dup, nil
		}
	}
	return nil, fmt.Errorf("文件夹 %s 不存在", folderID)
}

// ExportProjectJSON 导出项目为 Postman Collection v2.1 JSON
func (s *RequestService) ExportProjectJSON(projectID string) ([]byte, error) {
	data, err := s.store.GetCollectionData(projectID)
	if err != nil {
		return nil, err
	}
	if data == nil {
		data = &model.CollectionData{}
	}

	projectName := "MiniPost Export"
	if project, err := s.store.GetProject(projectID); err == nil && strings.TrimSpace(project.Name) != "" {
		projectName = project.Name
	}

	collection := postmanCollection{
		Info: postmanInfo{
			Name:   projectName,
			Schema: "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
		},
		Item: s.buildPostmanItems(data),
	}

	return json.MarshalIndent(collection, "", "  ")
}

func (s *RequestService) buildPostmanItems(data *model.CollectionData) []postmanItem {
	folderByID := make(map[string]model.Folder, len(data.Folders))
	for _, folder := range data.Folders {
		folderByID[folder.ID] = folder
	}
	requestByID := make(map[string]model.RequestItem, len(data.Requests))
	for _, request := range data.Requests {
		requestByID[request.ID] = request
	}

	children := make(map[string][]model.CollectionNode)
	for _, node := range data.TreeNodes {
		parentID := strings.TrimSpace(node.ParentFolderID)
		children[parentID] = append(children[parentID], node)
	}
	for parentID := range children {
		sort.SliceStable(children[parentID], func(i, j int) bool {
			if children[parentID][i].SortOrder == children[parentID][j].SortOrder {
				if children[parentID][i].NodeType == children[parentID][j].NodeType {
					return children[parentID][i].NodeID < children[parentID][j].NodeID
				}
				return children[parentID][i].NodeType < children[parentID][j].NodeType
			}
			return children[parentID][i].SortOrder < children[parentID][j].SortOrder
		})
	}

	var walk func(parentID string) []postmanItem
	walk = func(parentID string) []postmanItem {
		nodes := children[parentID]
		result := make([]postmanItem, 0, len(nodes))
		for _, node := range nodes {
			if node.NodeType == model.CollectionNodeTypeFolder {
				folder, ok := folderByID[node.NodeID]
				if !ok {
					continue
				}
				result = append(result, postmanItem{
					Name: folder.Name,
					Item: walk(folder.ID),
				})
				continue
			}
			if node.NodeType == model.CollectionNodeTypeRequest {
				request, ok := requestByID[node.NodeID]
				if !ok {
					continue
				}
				result = append(result, postmanItem{
					Name:    request.Name,
					Request: convertRequestToPostman(request),
				})
			}
		}
		return result
	}

	return walk("")
}

func convertRequestToPostman(request model.RequestItem) *postmanReq {
	postmanRequest := &postmanReq{
		Method: strings.ToUpper(strings.TrimSpace(request.Method)),
		URL:    postmanURL{Raw: buildRequestRawURL(request)},
	}
	if postmanRequest.Method == "" {
		postmanRequest.Method = "GET"
	}

	if len(request.Headers) > 0 {
		headers := make([]postmanKV, 0, len(request.Headers))
		for _, header := range request.Headers {
			if strings.TrimSpace(header.Key) == "" {
				continue
			}
			headers = append(headers, postmanKV{
				Key:         header.Key,
				Value:       header.Value,
				Description: header.Description,
			})
		}
		postmanRequest.Header = headers
	}

	switch request.Body.Type {
	case "json":
		body := &postmanBody{
			Mode: "raw",
			Raw:  request.Body.JSON,
		}
		body.Options.Raw.Language = "json"
		postmanRequest.Body = body
	case "raw":
		postmanRequest.Body = &postmanBody{
			Mode: "raw",
			Raw:  request.Body.Raw,
		}
	case "form-urlencoded":
		formData := make([]postmanKV, 0, len(request.Body.FormUrlEncoded))
		for _, field := range request.Body.FormUrlEncoded {
			if strings.TrimSpace(field.Key) == "" {
				continue
			}
			formData = append(formData, postmanKV{
				Key:         field.Key,
				Value:       field.Value,
				Description: field.Description,
			})
		}
		if len(formData) > 0 {
			postmanRequest.Body = &postmanBody{
				Mode:       "urlencoded",
				URLEncoded: formData,
			}
		}
	case "form-data":
		formData := make([]postmanFormData, 0, len(request.Body.FormData))
		for _, field := range request.Body.FormData {
			if strings.TrimSpace(field.Key) == "" {
				continue
			}
			itemType := strings.TrimSpace(field.Type)
			if itemType == "" {
				itemType = "text"
			}
			item := postmanFormData{
				Key:         field.Key,
				Type:        itemType,
				Value:       field.Value,
				Description: field.Description,
			}
			if itemType == "file" {
				src := strings.TrimSpace(field.FilePath)
				if src == "" {
					src = strings.TrimSpace(field.Value)
				}
				if src != "" {
					if encoded, err := json.Marshal(src); err == nil {
						item.Src = encoded
					}
				}
			}
			formData = append(formData, item)
		}
		if len(formData) > 0 {
			postmanRequest.Body = &postmanBody{
				Mode:     "formdata",
				FormData: formData,
			}
		}
	}

	return postmanRequest
}

func buildRequestRawURL(request model.RequestItem) string {
	raw := strings.TrimSpace(request.URL)
	if len(request.Params) == 0 {
		return raw
	}

	values := url.Values{}
	for _, param := range request.Params {
		key := strings.TrimSpace(param.Key)
		if key == "" {
			continue
		}
		values.Add(key, param.Value)
	}
	encoded := values.Encode()
	if encoded == "" {
		return raw
	}
	if strings.Contains(raw, "?") {
		if strings.HasSuffix(raw, "?") || strings.HasSuffix(raw, "&") {
			return raw + encoded
		}
		return raw + "&" + encoded
	}
	return raw + "?" + encoded
}

// ImportPostmanCollection 导入 Postman Collection v2.1 格式
func (s *RequestService) ImportPostmanCollection(projectID string, raw []byte) error {
	var collection postmanCollection
	if err := unmarshalJSONPossiblyCommented(raw, &collection); err != nil {
		return fmt.Errorf("解析 Postman JSON 失败: %w", err)
	}
	if err := s.saveImportedEnvironment(projectID, strings.TrimSpace(collection.Info.Name), convertPostmanVariables(collection.Variable)); err != nil {
		return err
	}
	return s.importPostmanItems(projectID, "", collection.Item)
}

func (s *RequestService) ImportPostmanEnvironment(projectID string, raw []byte) error {
	var env postmanEnvironment
	if err := unmarshalJSONPossiblyCommented(raw, &env); err != nil {
		return fmt.Errorf("解析 Postman Environment JSON 失败: %w", err)
	}
	name := strings.TrimSpace(env.Name)
	if name == "" {
		name = "Postman Environment"
	}
	return s.saveImportedEnvironment(projectID, name, convertPostmanVariables(env.Values))
}

func parsePostmanFormDataFilePath(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var single string
	if err := json.Unmarshal(raw, &single); err == nil {
		return strings.TrimSpace(single)
	}
	var multiple []string
	if err := json.Unmarshal(raw, &multiple); err == nil {
		for _, path := range multiple {
			trimmed := strings.TrimSpace(path)
			if trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func (s *RequestService) importPostmanItems(projectID, parentFolderID string, items []postmanItem) error {
	for _, item := range items {
		if len(item.Item) > 0 {
			// 这是一个文件夹
			folder, err := s.CreateFolder(projectID, parentFolderID, item.Name)
			if err != nil {
				return err
			}
			if err := s.importPostmanItems(projectID, folder.ID, item.Item); err != nil {
				return err
			}
		} else if item.Request != nil {
			// 这是一个请求
			req, err := s.CreateRequest(projectID, parentFolderID, item.Name)
			if err != nil {
				return err
			}
			req.Method = item.Request.Method
			if item.Request.URL.Raw != "" {
				req.URL = item.Request.URL.Raw
			}
			for _, h := range item.Request.Header {
				req.Headers = append(req.Headers, model.KeyValue{
					Key:         h.Key,
					Value:       h.Value,
					Description: h.Description,
				})
			}
			if item.Request.Body != nil {
				switch item.Request.Body.Mode {
				case "raw":
					req.Body = model.RequestBody{Type: "raw", Raw: item.Request.Body.Raw}
					if strings.EqualFold(item.Request.Body.Options.Raw.Language, "json") {
						req.Body.Type = "json"
						req.Body.JSON = item.Request.Body.Raw
						req.Body.Raw = ""
					}
				case "urlencoded":
					var formData []model.KeyValue
					for _, kv := range item.Request.Body.URLEncoded {
						formData = append(formData, model.KeyValue{
							Key:         kv.Key,
							Value:       kv.Value,
							Description: kv.Description,
						})
					}
					req.Body = model.RequestBody{Type: "form-urlencoded", FormUrlEncoded: formData}
				case "formdata":
					formData := make([]model.FormData, 0, len(item.Request.Body.FormData))
					for _, field := range item.Request.Body.FormData {
						itemType := strings.TrimSpace(field.Type)
						if itemType == "" {
							itemType = "text"
						}
						filePath := ""
						fileName := ""
						if itemType == "file" {
							filePath = parsePostmanFormDataFilePath(field.Src)
							if filePath != "" {
								parts := strings.Split(strings.ReplaceAll(filePath, "\\", "/"), "/")
								fileName = parts[len(parts)-1]
							}
						}
						formData = append(formData, model.FormData{
							Key:         field.Key,
							Value:       field.Value,
							Description: field.Description,
							Type:        itemType,
							FilePath:    filePath,
							FileName:    fileName,
						})
					}
					req.Body = model.RequestBody{Type: "form-data", FormData: formData}
				}
			}
			if err := s.SaveRequest(req); err != nil {
				return err
			}
		}
	}
	return nil
}

// ImportSwagger 导入 OpenAPI/Swagger 2.0 或 3.x 格式
func (s *RequestService) ImportSwagger(projectID string, raw []byte) error {
	return s.ImportSwaggerWithSource(projectID, raw, "")
}

func (s *RequestService) ImportSwaggerWithSource(projectID string, raw []byte, sourceURL string) error {
	var doc openAPIDoc
	if err := unmarshalJSONPossiblyCommented(raw, &doc); err != nil {
		return fmt.Errorf("解析 OpenAPI JSON 失败: %w", err)
	}

	baseURLTemplate, err := s.importOpenAPIEnvironments(projectID, doc, sourceURL)
	if err != nil {
		return err
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
					folder, err := s.CreateFolder(projectID, "", tag)
					if err != nil {
						return err
					}
					tagFolders[tag] = folder.ID
					folderID = folder.ID
				}
			}

			name := op.Summary
			if name == "" {
				name = op.OperationID
			}
			if name == "" {
				name = methodUpper + " " + pathStr
			}

			req, err := s.CreateRequest(projectID, folderID, name)
			if err != nil {
				return err
			}
			req.Method = methodUpper
			req.URL = joinURLTemplate(baseURLTemplate, replacePathPlaceholders(pathStr))
			applyOpenAPIParameters(req, append(entry.parameters, op.Parameters...), doc)
			applyOpenAPIRequestBody(req, op.RequestBody, doc)
			if err := s.SaveRequest(req); err != nil {
				return err
			}
		}
	}
	return nil
}
