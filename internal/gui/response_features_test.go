package gui

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"minipost/internal/client"
	"minipost/internal/model"
)

// sendWith 让后端返回 resp，打开第一条请求并发送，返回当前标签的响应状态。
func sendWith(t *testing.T, resp *model.HttpResponse) (*app, *ui.Tester, *fakeBackend, *respState) {
	t.Helper()
	a, tt, be := zhApp(t)
	be.resp = resp
	a.openRequest(a.coll.Requests[0].ID)
	tt.Frame()
	if err := tt.Click("Send"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	rs := a.activeTab().resp
	if rs.resp == nil {
		t.Fatalf("应收到响应，实际 %q", tt.Texts())
	}
	return a, tt, be, rs
}

func jsonResp() *model.HttpResponse {
	return &model.HttpResponse{
		StatusCode: 200, StatusText: "OK", ContentType: "application/json",
		Headers: map[string][]string{"Content-Type": {"application/json"}},
		Body:    `{"data":{"total":2,"users":[{"id":1}]},"msg":"ok"}`,
	}
}

func TestResponseBodyDefaultsToPrettyJSON(t *testing.T) {
	_, tt, _, rs := sendWith(t, jsonResp())
	if rs.view.effMode != client.ModeJSON {
		t.Fatalf("应识别为 JSON，实际 %q", rs.view.effMode)
	}
	if !strings.Contains(rs.bodyEd.Buffer.String(), "\n  \"data\": {") {
		t.Fatalf("应缩进展示，实际 %q", rs.bodyEd.Buffer.String())
	}
	if !tt.HasText("JSON") {
		t.Fatalf("格式下拉应显示 JSON，实际 %q", tt.Texts())
	}
}

func TestResponseFormatSwitch(t *testing.T) {
	_, tt, _, rs := sendWith(t, &model.HttpResponse{StatusCode: 200, ContentType: "text/plain", Body: "Hi"})
	if rs.view.effMode != client.ModeRaw {
		t.Fatalf("纯文本应为 Raw，实际 %q", rs.view.effMode)
	}
	rs.view.mode = client.ModeHex
	tt.Frame()
	if rs.view.display != "48 69" || rs.bodyEd.Buffer.String() != "48 69" {
		t.Fatalf("Hex 视图不符: %q", rs.view.display)
	}
	rs.view.mode = client.ModeBase64
	tt.Frame()
	if rs.view.display != "SGk=" {
		t.Fatalf("Base64 视图不符: %q", rs.view.display)
	}
}

func TestResponseFilterAndCopy(t *testing.T) {
	a, tt, _, rs := sendWith(t, jsonResp())
	if err := tt.Click("过滤"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !rs.view.filterOpen {
		t.Fatalf("应展开过滤栏，实际 %q", tt.Texts())
	}
	rs.view.filterExpr = "$.data.total"
	tt.Frame()
	snapshot(t, tt, "rb_filter")
	if rs.view.display != "2" || rs.bodyEd.Buffer.String() != "2" {
		t.Fatalf("过滤结果应为 2，实际 %q", rs.view.display)
	}
	// 复制的是当前展示（过滤后）的内容，而不是原始响应体
	if err := tt.Click("复制响应体"); err != nil {
		t.Fatal(err)
	}
	if a.clipboard != "2" {
		t.Fatalf("剪贴板应为过滤结果，实际 %q", a.clipboard)
	}
	// 无效表达式给出提示并保留未过滤内容
	rs.view.filterExpr = "data"
	tt.Frame()
	if rs.view.filterErr == nil || !tt.HasText("JSONPath 表达式无效（示例：$.headers.host）") {
		t.Fatalf("应提示表达式无效，实际 %q", tt.Texts())
	}
	// 关闭过滤栏会清空表达式并恢复完整内容
	if err := tt.Click("过滤"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if rs.view.filterExpr != "" || !strings.Contains(rs.bodyEd.Buffer.String(), "\"users\"") {
		t.Fatalf("关闭过滤应恢复全文，实际 %q", rs.bodyEd.Buffer.String())
	}
}

func TestResponseKeywordFilterForText(t *testing.T) {
	_, tt, _, rs := sendWith(t, &model.HttpResponse{StatusCode: 200, ContentType: "text/plain", Body: "alpha\nBETA\nalphabet"})
	rs.view.filterOpen = true
	rs.view.filterExpr = "alpha"
	tt.Frame()
	if rs.view.display != "alpha\nalphabet" {
		t.Fatalf("按关键字过滤行，实际 %q", rs.view.display)
	}
}

func TestResponseBinaryShowsDownloadCard(t *testing.T) {
	data := []byte{0x50, 0x4b, 0x03, 0x04, 0xff, 0x00}
	b64 := base64.StdEncoding.EncodeToString(data)
	a, tt, be, rs := sendWith(t, &model.HttpResponse{
		StatusCode: 200, ContentType: "application/zip", BodyIsBinary: true, BodyBase64: b64, Size: int64(len(data)),
		Headers: map[string][]string{"Content-Disposition": {`attachment; filename="backup.zip"`}},
	})
	if !rs.view.download || rs.view.filename != "backup.zip" {
		t.Fatalf("应进入下载卡片并使用头里的文件名，实际 download=%v name=%q", rs.view.download, rs.view.filename)
	}
	if !tt.HasText("backup.zip") || !tt.HasText("application/zip") {
		t.Fatalf("卡片应显示文件名与类型，实际 %q", tt.Texts())
	}
	snapshot(t, tt, "rb_card")
	if err := tt.Click("下载文件"); err != nil {
		t.Fatal(err)
	}
	if len(be.saved) != 1 || be.saved[0].name != "backup.zip" || be.saved[0].content != model.BinarySavePrefix+b64 {
		t.Fatalf("应按二进制保存，实际 %+v", be.saved)
	}
	if err := tt.Click("文本查看"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if rs.view.download {
		t.Fatalf("文本查看应关闭下载卡片")
	}
	_ = a
}

func TestResponseTextDownloadButtonUsesURLName(t *testing.T) {
	// 文本响应带 attachment：工具栏出现下载按钮，文件名按头/类型推断
	_, tt, be, _ := sendWith(t, &model.HttpResponse{
		StatusCode: 200, ContentType: "text/csv", Body: "a,b\n1,2",
		Headers: map[string][]string{"Content-Disposition": {"attachment"}},
	})
	// attachment 无文件名时回落到 URL 末段（alpha）并按类型补扩展名
	if err := tt.Click("下载文件"); err != nil {
		t.Fatal(err)
	}
	if len(be.saved) != 1 || be.saved[0].name != "alpha.csv" || be.saved[0].content != "a,b\n1,2" {
		t.Fatalf("保存内容或文件名不符: %+v", be.saved)
	}
}

func TestResponseNoDownloadButtonForPlainJSON(t *testing.T) {
	_, tt, _, _ := sendWith(t, jsonResp())
	if tt.HasText("下载文件") {
		t.Fatalf("普通 JSON 不应出现下载卡片")
	}
	if err := tt.Click("下载文件"); err == nil {
		t.Fatalf("普通 JSON 不应有下载按钮")
	}
}

func TestResponseJSONPreviewDrillDown(t *testing.T) {
	_, tt, _, rs := sendWith(t, jsonResp())
	if err := tt.Click("预览"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !rs.view.preview || !tt.HasText("根节点") || !tt.HasText("data") || !tt.HasText("{2}") {
		t.Fatalf("应显示根节点的键与角标，实际 %q", tt.Texts())
	}
	snapshot(t, tt, "rb_preview")
	if err := tt.Click("data"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if len(rs.view.path) != 1 || !tt.HasText("total") || !tt.HasText("[1]") {
		t.Fatalf("应钻取到 data，实际 path=%v %q", rs.view.path, tt.Texts())
	}
	if err := tt.Click("根节点"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if len(rs.view.path) != 0 {
		t.Fatalf("点击面包屑应回到根")
	}
	// 点击格式下拉回到代码视图
	if err := tt.Click("JSON"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if rs.view.preview {
		t.Fatalf("预览中点击格式应回到代码视图")
	}
}

func TestResponseStateResetsOnNewResponse(t *testing.T) {
	a, tt, be, rs := sendWith(t, jsonResp())
	rs.view.filterOpen, rs.view.filterExpr, rs.view.mode = true, "$.msg", client.ModeRaw
	tt.Frame()
	be.resp = &model.HttpResponse{StatusCode: 200, ContentType: "application/json", Body: `{"x":1}`}
	if err := tt.Click("Send"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if v := a.activeTab().resp.view; v.filterOpen || v.filterExpr != "" || v.mode != "" {
		t.Fatalf("新响应应重置展示状态，实际 %+v", v)
	}
}

// streamApp 构造一个已收到流式消息、停在 Stream 标签的应用。
func streamApp(t *testing.T) (*app, *ui.Tester, *respState) {
	t.Helper()
	a, tt, _, rs := sendWith(t, &model.HttpResponse{StatusCode: 200, ContentType: "text/event-stream", Body: ""})
	rs.onChunk(model.StreamChunk{Kind: "response_start", Data: `{"statusCode":200,"statusText":"OK","contentType":"text/event-stream"}`, Sequence: 0})
	rs.onChunk(model.StreamChunk{Kind: "message", Data: `{"n":1}`, Timestamp: "2026-11-14T10:00:00.123Z", Sequence: 1})
	rs.onChunk(model.StreamChunk{Kind: "message", Data: `hello world`, Timestamp: "2026-11-14T10:00:01.000Z", Sequence: 2})
	rs.onChunk(model.StreamChunk{Kind: "connection_closed", Timestamp: "2026-11-14T10:00:02.000Z", Sequence: 3})
	rs.sub = 3
	tt.Frame()
	return a, tt, rs
}

func TestStreamEntriesOrderAndFilter(t *testing.T) {
	chunks := []model.StreamChunk{
		{Kind: "response_start", Sequence: 0},
		{Kind: "message", Data: "Alpha", Sequence: 1},
		{Kind: "message", Data: "beta", Sequence: 2},
	}
	got := streamEntries(chunks)
	if len(got) != 2 || got[0].Sequence != 2 {
		t.Fatalf("应去掉 response_start 并让新消息在前，实际 %+v", got)
	}
	if f := filterStreamEntries(got, " ALPHA "); len(f) != 1 || f[0].Sequence != 1 {
		t.Fatalf("搜索应不区分大小写，实际 %+v", f)
	}
	many := make([]model.StreamChunk, maxStreamRows+50)
	for i := range many {
		many[i] = model.StreamChunk{Kind: "message", Sequence: i + 1}
	}
	if got := streamEntries(many); len(got) != maxStreamRows || got[0].Sequence != maxStreamRows+50 {
		t.Fatalf("应只保留最近 %d 条，实际 %d", maxStreamRows, len(got))
	}
	if streamTimeLabel("bad") != "--:--:--.---" {
		t.Fatalf("无法解析的时间应显示占位")
	}
}

func TestStreamViewExpandAndCopy(t *testing.T) {
	a, tt, rs := streamApp(t)
	if !tt.HasText("连接已关闭") || !tt.HasText(`{"n":1}`) || !tt.HasText("hello world") {
		t.Fatalf("应列出消息，实际 %q", tt.Texts())
	}
	if err := tt.Click("hello world"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !rs.stream.expanded[2] {
		t.Fatalf("点击应展开消息 2")
	}
	// 展开后默认 JSON 格式；非 JSON 内容保持原样
	if got := rs.stream.eds[2].Buffer.String(); got != "hello world" {
		t.Fatalf("详情内容 = %q", got)
	}
	if err := tt.Click("复制"); err != nil {
		t.Fatal(err)
	}
	if a.clipboard != "hello world" {
		t.Fatalf("应复制详情文本，实际 %q", a.clipboard)
	}
	// 关闭连接的消息不可展开
	if err := tt.Click("连接已关闭"); err != nil {
		t.Fatal(err)
	}
	if rs.stream.expanded[3] {
		t.Fatalf("connection_closed 不应展开")
	}
	// 收起消息 2，让消息 1 回到可视区域；JSON 消息展开后缩进
	if err := tt.Click("hello world"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if rs.stream.expanded[2] {
		t.Fatalf("再次点击应收起")
	}
	if err := tt.Click(`{"n":1}`); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if got := rs.stream.eds[1].Buffer.String(); got != "{\n  \"n\": 1\n}" {
		t.Fatalf("JSON 详情应缩进，实际 %q", got)
	}
	snapshot(t, tt, "rb_stream")
}

func TestStreamViewSearchFilters(t *testing.T) {
	_, tt, rs := streamApp(t)
	rs.stream.search = "hello"
	tt.Frame()
	if tt.HasText(`{"n":1}`) || !tt.HasText("hello world") {
		t.Fatalf("搜索后只应剩匹配消息，实际 %q", tt.Texts())
	}
	rs.stream.search = "zzz"
	tt.Frame()
	if !tt.HasText("没有匹配的流式消息") {
		t.Fatalf("无匹配应提示，实际 %q", tt.Texts())
	}
}

func TestRequestErrorCardShowsDiagnosis(t *testing.T) {
	a, tt, be := zhApp(t)
	be.sendErr = errors.New("[CONNECTION_REFUSED] 连接被拒绝: dial tcp 127.0.0.1:1: connection refused")
	a.openRequest(a.coll.Requests[0].ID)
	tt.Frame()
	if err := tt.Click("Send"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	for _, want := range []string{"无法发送请求", "连接被目标地址拒绝，服务可能未启动或端口不可达。 请检查目标服务状态、端口、代理或防火墙策略。", "在控制台查看", "错误码: CONNECTION_REFUSED"} {
		if !tt.HasText(want) {
			t.Fatalf("错误卡片应包含 %q，实际 %q", want, tt.Texts())
		}
	}
	snapshot(t, tt, "rb_error")
	if err := tt.Click("在控制台查看"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.set.ConsoleOpen {
		t.Fatalf("应打开控制台")
	}
}

func TestInstallSamplesSwitchesToFirstNewProject(t *testing.T) {
	a, _, be := zhApp(t)
	be.sampleNames = []string{"示例 · A", "示例 · B"}
	before := len(a.projects)
	a.installSamples()
	if len(a.projects) != before+2 {
		t.Fatalf("应重新加载项目列表，实际 %d", len(a.projects))
	}
	if pr := a.project(); pr == nil || pr.Name != "示例 · A" {
		t.Fatalf("应切换到第一个新项目，实际 %+v", pr)
	}
	// 没有新建项目时只提示，不切换
	cur := a.projectID
	be.sampleNames = nil
	a.toastMsgs = nil
	a.installSamples()
	if a.projectID != cur || len(a.toastMsgs) == 0 || !strings.Contains(a.toastMsgs[0], "已存在") {
		t.Fatalf("已存在时应只提示，实际 %q 项目 %s", a.toastMsgs, a.projectID)
	}
}

func TestTruncateForDisplay(t *testing.T) {
	if s, cut := truncateForDisplay("abc", 10); s != "abc" || cut {
		t.Fatalf("未超限不应截断: %q %v", s, cut)
	}
	if s, cut := truncateForDisplay("abcdef", 3); s != "abc" || !cut {
		t.Fatalf("应截到 3 字节: %q %v", s, cut)
	}
	// "你好" 每个字 3 字节；限制 4 字节时不能把第二个字劈成两半
	if s, cut := truncateForDisplay("你好", 4); s != "你" || !cut {
		t.Fatalf("应落在 UTF-8 边界上: %q %v", s, cut)
	}
}

// 小响应：不出现截断提示；文本响应也能“保存到文件”（保存的是完整原始响应体）。
func TestSaveTextResponseToFile(t *testing.T) {
	a, tt, be, rs := sendWith(t, jsonResp())
	if rs.view.truncated || tt.HasText("保存完整响应") {
		t.Fatal("小响应不应有截断提示")
	}
	a.saveResponse(rs.resp, rs.view.filename)
	if len(be.saved) != 1 || be.saved[0].content != jsonResp().Body {
		t.Fatalf("应保存完整响应体: %+v", be.saved)
	}
}

// 大响应：视图只载入开头并出现提示条；点“保存完整响应”保存的仍是完整内容，复制也是完整内容。
func TestLargeResponseTruncatedWithBanner(t *testing.T) {
	big := strings.Repeat("0123456789abcdef\n", bodyDisplayLimit/16+4096)
	a, tt, be, rs := sendWith(t, &model.HttpResponse{StatusCode: 200, ContentType: "text/plain", Body: big, Size: int64(len(big))})
	if !rs.view.truncated {
		t.Fatal("超过上限的响应应被标记为截断")
	}
	if n := rs.bodyEd.Buffer.Len(); n > bodyDisplayLimit {
		t.Fatalf("编辑器只应载入开头 %d 字节，实际 %d", bodyDisplayLimit, n)
	}
	if !tt.HasText("保存完整响应") {
		t.Fatalf("应出现截断提示条: %q", tt.Texts())
	}
	if err := tt.Click("保存完整响应"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if len(be.saved) != 1 || len(be.saved[0].content) != len(big) {
		t.Fatalf("应保存完整响应（%d 字节）: %d 条", len(big), len(be.saved))
	}
	a.copyBody(rs)
	if len(a.clipboard) != len(rs.view.display) || len(a.clipboard) <= bodyDisplayLimit {
		t.Fatalf("复制应得到完整内容: %d", len(a.clipboard))
	}
}
