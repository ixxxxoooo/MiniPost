package gui

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	"minipost/internal/client"
	"minipost/internal/model"
)

// respState 保存一个标签页的响应展示状态。
type respState struct {
	resp *model.HttpResponse
	err  string
	sub  int // 响应区子标签下标

	bodyMode int // 0 Pretty，1 Raw，2 Preview（图片）
	bodyEd   *editor.State

	loadedResp *model.HttpResponse // 已载入编辑器的响应与模式，用于判断是否需要重新载入
	loadedMode int
	image      ui.ImageSource

	streamed  bool
	streaming bool
	start     *client.StreamStart
	chunks    []model.StreamChunk
	chunkList ui.ListState
	chunkRow  int
	hdrList   ui.ListState
}

func newRespState() *respState {
	return &respState{
		bodyEd:   &editor.State{Buffer: editor.NewBuffer(""), ReadOnly: true, SoftWrap: true},
		chunkRow: -1,
	}
}

// reset 清空上一次响应，准备发送新的请求。
func (rs *respState) reset() {
	rs.resp, rs.err, rs.start, rs.chunks = nil, "", nil, nil
	rs.streamed, rs.streaming, rs.image = false, false, nil
	rs.loadedResp = nil
	rs.chunkRow = -1
}

// onChunk 处理一个流式数据块。
func (rs *respState) onChunk(ch model.StreamChunk) {
	rs.streamed = true
	if ch.Kind == "response_start" {
		if s, ok := client.ParseStreamStart(ch.Data); ok {
			rs.start = &s
		}
		return
	}
	rs.chunks = append(rs.chunks, ch)
}

// isJSONBody 判断响应体是否应作为 JSON 展示。
func isJSONBody(resp *model.HttpResponse, detect string) bool {
	ct := strings.ToLower(resp.ContentType)
	if strings.Contains(ct, "json") {
		return true
	}
	if detect == "json" {
		return json.Valid([]byte(resp.Body))
	}
	t := strings.TrimSpace(resp.Body)
	return (strings.HasPrefix(t, "{") || strings.HasPrefix(t, "[")) && json.Valid([]byte(t))
}

// prettyBody 返回响应体的展示文本：JSON 缩进，其余原样。
func prettyBody(resp *model.HttpResponse, detect string) string {
	if isJSONBody(resp, detect) {
		var out bytes.Buffer
		if err := json.Indent(&out, []byte(resp.Body), "", "  "); err == nil {
			return out.String()
		}
	}
	return resp.Body
}

// languageFile 按内容类型选择编辑器的高亮语言。
func languageFile(resp *model.HttpResponse, json bool) string {
	ct := strings.ToLower(resp.ContentType)
	switch {
	case json:
		return "x.json"
	case strings.Contains(ct, "html"):
		return "x.html"
	case strings.Contains(ct, "xml"):
		return "x.xml"
	case strings.Contains(ct, "javascript"):
		return "x.js"
	case strings.Contains(ct, "css"):
		return "x.css"
	}
	return "x.txt"
}

// loadBody 在响应或展示模式变化时，把文本载入只读编辑器。
func (rs *respState) loadBody(detect string) {
	resp := rs.resp
	if resp == nil || (rs.loadedResp == resp && rs.loadedMode == rs.bodyMode) {
		return
	}
	rs.loadedResp, rs.loadedMode = resp, rs.bodyMode
	if resp.BodyIsBinary {
		rs.bodyEd.Buffer.Reset("")
		rs.image = nil
		if rs.bodyMode == 2 || strings.HasPrefix(strings.ToLower(resp.ContentType), "image/") {
			if data, err := base64.StdEncoding.DecodeString(resp.BodyBase64); err == nil {
				if img, err := ui.DecodeImage(data); err == nil {
					rs.image = img
				}
			}
		}
		return
	}
	text := resp.Body
	isJSON := isJSONBody(resp, detect)
	if rs.bodyMode == 0 {
		text = prettyBody(resp, detect)
	}
	rs.bodyEd.Buffer.Reset(text)
	if lang := editor.LanguageFor(languageFile(resp, isJSON && rs.bodyMode == 0)); lang != nil {
		rs.bodyEd.Buffer.SetLanguage(lang)
	}
}

// responsePane 绘制响应区。
func (a *app) responsePane(c *ui.Context, t *tab) {
	rs := t.resp
	th := c.Theme()
	ui.Column(c).Grow(1).Children(func() {
		switch {
		case rs.err != "":
			ui.Column(c).Padding(12).Gap(6).Children(func() {
				ui.Text(c, a.t("请求失败", "Request failed")).Bold().TextColor(th.Danger)
				ui.Text(c, rs.err).Selectable()
			})
		case rs.resp == nil && !rs.streamed:
			ui.Column(c).Grow(1).Gap(8).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
				if t.sending {
					ui.Spinner(c)
					ui.Text(c, a.t("发送中…", "Sending…")).TextColor(th.TextMuted)
				} else {
					ui.Text(c, a.t("点击“发送”获取响应", "Press Send to get a response")).TextColor(th.TextMuted)
				}
			})
		default:
			a.responseBody(c, t)
		}
	})
}

// statusInfo 返回状态行需要的字段，兼容流式进行中（只有响应头）与完成后两种状态。
func (rs *respState) statusInfo() (code int, text, proto string) {
	switch {
	case rs.resp != nil:
		return rs.resp.StatusCode, rs.resp.StatusText, rs.resp.Protocol
	case rs.start != nil:
		return rs.start.StatusCode, rs.start.StatusText, rs.start.Protocol
	}
	return 0, "", ""
}

func (a *app) responseBody(c *ui.Context, t *tab) {
	rs := t.resp
	th := c.Theme()
	code, text, proto := rs.statusInfo()
	ui.Row(c).Gap(12).Padding(8, 10).AlignItems(ui.Center).Children(func() {
		ui.Text(c, fmt.Sprintf("%d %s", code, text)).Bold().TextColor(statusColor(code, th))
		if rs.resp != nil {
			ui.Text(c, fmt.Sprintf("%.0f ms", rs.resp.Duration)).TextColor(th.TextMuted)
			ui.Text(c, formatSize(rs.resp.Size)).TextColor(th.TextMuted)
		} else if rs.streaming {
			ui.Spinner(c)
			ui.Text(c, a.t("接收中…", "Receiving…")).TextColor(th.TextMuted)
		}
		if proto != "" {
			ui.Text(c, proto).TextColor(th.TextMuted)
		}
		ui.Spacer(c)
		if rs.resp != nil {
			if ui.Button(c, a.t("复制", "Copy")).Clicked() {
				a.copyResponse(rs.resp)
			}
			if ui.Button(c, a.t("保存到文件…", "Save…")).Clicked() {
				a.saveResponse(rs.resp)
			}
		}
	})
	if rs.resp != nil && len(rs.resp.Warnings) > 0 {
		ui.Text(c, strings.Join(rs.resp.Warnings, "；")).TextColor(th.Warning).Padding(0, 10)
	}

	labels := []string{"Body", withCount("Headers", a.respHeaderCount(rs)), "Cookies", "Timing", "Network"}
	if rs.streamed {
		labels = append(labels, "Stream")
	}
	if rs.streaming && rs.resp == nil && rs.sub != 5 {
		rs.sub = 5
	}
	rs.sub = min(rs.sub, len(labels)-1)
	ui.Tabs(c, &rs.sub, labels...).Padding(0, 8)
	ui.Divider(c)
	switch rs.sub {
	case 0:
		a.bodyView(c, rs)
	case 1:
		a.respHeadersView(c, rs)
	case 2:
		a.respCookiesView(c, rs)
	case 3:
		a.respTimingView(c, rs)
	case 4:
		a.respNetworkView(c, rs)
	case 5:
		a.streamView(c, rs)
	}
}

func (a *app) respHeadersMap(rs *respState) map[string][]string {
	switch {
	case rs.resp != nil:
		return rs.resp.Headers
	case rs.start != nil:
		return rs.start.Headers
	}
	return nil
}

func (a *app) respHeaderCount(rs *respState) int { return len(a.respHeadersMap(rs)) }

// formatSize 把字节数格式化为易读的大小。
func formatSize(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.2f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.2f KB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d B", n)
}

func (a *app) bodyView(c *ui.Context, rs *respState) {
	if rs.resp == nil {
		ui.Text(c, a.t("响应尚未结束，请查看 Stream 标签", "Response is still streaming — see the Stream tab")).Padding(12).TextColor(c.Theme().TextMuted)
		return
	}
	resp := rs.resp
	if resp.BodyIsBinary {
		ui.Column(c).Grow(1).Gap(10).Padding(12).Children(func() {
			ui.Text(c, a.t("二进制内容", "Binary content")+" · "+formatSize(resp.Size)).TextColor(c.Theme().TextMuted)
			rs.loadBody(a.set.ResponseFormatDetection)
			if rs.image != nil {
				ui.Scroll(c).Grow(1).Children(func() { ui.Image(c, rs.image).Fit(ui.Contain) })
			}
		})
		return
	}
	ui.Column(c).Grow(1).Children(func() {
		ui.Row(c).Padding(6, 10).Children(func() {
			ui.Segmented(c, &rs.bodyMode, "Pretty", "Raw")
		})
		rs.loadBody(a.set.ResponseFormatDetection)
		editor.View(c, rs.bodyEd).Grow(1).FillWidth()
	})
}
