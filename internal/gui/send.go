package gui

import (
	"strings"

	"github.com/egoist/mygo"

	"github.com/google/uuid"

	"minipost/internal/client"
	"minipost/internal/model"
	"minipost/internal/pkg/logger"
)

// sendTab 发送标签页中的请求：在后台执行，结果通过 win.Update 回到主线程。
func (a *app) sendTab(t *tab) {
	if t.sending {
		return
	}
	t.flush()
	d := t.draft
	if strings.TrimSpace(d.URL) == "" {
		a.toast(a.t("请先输入请求地址", "Enter a request URL first"))
		return
	}
	reqID := uuid.NewString()
	in, info := client.BuildSendInput(d, reqID, client.SendEnv{Vars: a.vars(), Settings: a.set, Jar: a.jar})
	projectID := a.projectID
	stream := client.WantsStreaming(d)

	t.sending, t.sendID = true, reqID
	t.resp.reset()
	t.resp.streaming = stream
	a.logRequest(d.Method, info.ResolvedURL)
	logger.Info("发送请求", "method", d.Method, "target", logger.RedactRequestTarget(info.ResolvedURL), "streaming", stream)

	method := d.Method
	a.async(func() func() {
		var (
			resp *model.HttpResponse
			err  error
		)
		// 变量已在本地解析，这里不再传环境 ID，避免后端二次替换。
		if stream {
			resp, err = a.be.SendRequestWithEnvStream(in, projectID, "", func(ch model.StreamChunk) {
				a.post(func() { t.resp.onChunk(ch) })
			})
		} else {
			resp, err = a.be.SendRequestWithEnv(in, projectID, "")
		}
		return func() { a.finishSend(t, method, info.ResolvedURL, resp, err) }
	})
}

// finishSend 在主线程处理发送结果。
func (a *app) finishSend(t *tab, method, url string, resp *model.HttpResponse, err error) {
	t.sending = false
	rs := t.resp
	rs.streaming = false
	if err != nil {
		rs.err = err.Error()
		a.logResponse(method, url, 0, 0, err.Error())
		logger.Warn("请求失败", "error", err.Error())
		return
	}
	rs.resp = resp
	rs.err = ""
	if !a.set.DisableCookies {
		a.jar.Absorb(url, resp.Headers)
	}
	if !rs.streamed {
		rs.sub = 0
	}
	a.logResponse(method, url, resp.StatusCode, resp.Duration, "")
	a.reloadHistory()
}

// copyResponse 复制响应体文本到剪贴板。
func (a *app) copyResponse(resp *model.HttpResponse) {
	text := resp.Body
	if resp.BodyIsBinary {
		a.toast(a.t("二进制内容无法复制，请保存到文件", "Binary content can't be copied; save it to a file"))
		return
	}
	a.writeClipboard(text)
	a.toast(a.t("已复制响应体", "Response body copied"))
}

// saveResponse 保存响应体到文件（二进制内容以 base64 前缀传递给后端）。
func (a *app) saveResponse(resp *model.HttpResponse) {
	content, name := resp.Body, "response.txt"
	switch {
	case resp.BodyIsBinary:
		content, name = model.BinarySavePrefix+resp.BodyBase64, "response.bin"
	case strings.Contains(strings.ToLower(resp.ContentType), "json"):
		name = "response.json"
	case strings.Contains(strings.ToLower(resp.ContentType), "html"):
		name = "response.html"
	}
	a.async(func() func() {
		err := a.be.SaveResponseToFile(a.win, name, content)
		return func() { a.fail(a.t("保存响应", "Save response"), err) }
	})
}

// copyCurl 把当前请求（含环境变量与 Cookie）复制为 cURL 命令。
func (a *app) copyCurl(t *tab) {
	t.flush()
	cmd, _ := client.RuntimeCurl(t.draft, a.vars(), a.set, a.jar)
	a.writeClipboard(cmd)
	a.toast(a.t("已复制 cURL 命令", "cURL command copied"))
}

// writeClipboard 写入系统剪贴板。
func (a *app) writeClipboard(text string) {
	a.clipboard = text
	if a.native {
		mygo.Clipboard.WriteText(text)
	}
}
