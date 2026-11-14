package gui

import (
	"fmt"
	"math"
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

	bodyEd *editor.State
	view   bodyViewState // 响应体展示状态（格式、预览、过滤、下载卡片等）

	streamed  bool
	streaming bool
	start     *client.StreamStart
	chunks    []model.StreamChunk
	stream    streamViewState // Stream 标签的搜索、展开与详情编辑器状态
}

func newRespState() *respState {
	return &respState{
		bodyEd: &editor.State{Buffer: editor.NewBuffer(""), ReadOnly: true, SoftWrap: true},
	}
}

// reset 清空上一次响应，准备发送新的请求。
func (rs *respState) reset() {
	rs.resp, rs.err, rs.start, rs.chunks = nil, "", nil, nil
	rs.streamed, rs.streaming = false, false
	rs.view = bodyViewState{}
	rs.stream = streamViewState{}
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

// responsePane 绘制响应区：空态、发送中、失败与成功四种状态。
func (a *app) responsePane(c *ui.Context, t *tab) {
	rs := t.resp
	p := a.p
	ui.Column(c).Grow(1).Basis(0).Background(p.surface).Children(func() {
		a.sendShimmer(c, t)
		switch {
		case rs.err != "":
			a.respTitle(c)
			a.respError(c, t)
		case rs.resp == nil && !rs.streamed:
			a.respTitle(c)
			ui.Row(c).Grow(1).Basis(0).FillWidth().AlignItems(ui.Center).Justify(ui.Center).Children(func() {
				if t.sending {
					txt(c, a.t("发送请求...", "Sending request..."), fsLarge, p.fg2, 0)
					return
				}
				empty := strings.TrimSpace(t.draft.URL) == ""
				bg := p.surface2
				if p.dark {
					bg = ui.RGB(52, 52, 52)
				}
				col := p.fg2
				if empty {
					col = p.muted
				}
				if a.button(c, btnOpts{radius: 8, padX: 16, h: 36, gap: 8, bg: &bg, noHover: true, disabled: empty, key: "resp-empty"}, func(hover bool) {
					cc := col
					if hover {
						cc = p.fg
					}
					icon(c, "clock", 15, cc)
					txt(c, a.t("发送并获取成功响应", "Send + Get a successful response"), fsLarge, cc, 0)
					ui.Row(c).Gap(4).MarginStart(4).Children(func() {
						a.kbdSmall(c, "⌘")
						a.kbdSmall(c, "Enter")
					})
				}) {
					a.later(func() { a.sendTab(t) })
				}
			})
		default:
			a.responseBody(c, t)
		}
	})
}

// respTitle 是响应区没有内容时左上角的“响应”标题（32px 高）。
func (a *app) respTitle(c *ui.Context) {
	ui.Row(c).FillWidth().Height(32).Shrink(0).PaddingX(8).AlignItems(ui.Center).Children(func() {
		txt(c, a.t("响应", "Response"), fsLarge, a.p.fg, 600)
	})
}

// kbdSmall 是响应空态里的小快捷键（16px 高，圆角 4）。
func (a *app) kbdSmall(c *ui.Context, s string) {
	p := a.p
	ui.Row(c).Height(16).MinWidth(16).PaddingX(4).Radius(4).Background(p.surface).Border(1, p.btnBorder).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
		txtMono(c, s, fsTiny, p.muted, 0)
	})
}

// respError 绘制请求失败卡片。
func (a *app) respError(c *ui.Context, t *tab) {
	p := a.p
	rs := t.resp
	ui.Column(c).Grow(1).Basis(0).FillWidth().Padding(8).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
		ui.Column(c).FillWidth().MaxWidth(860).Padding(12, 16).Gap(10).Radius(12).Background(p.surface).Border(1, p.danger.Alpha(0.25)).Children(func() {
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Row(c).Size(24, 24).Radius(12).Background(p.danger.Alpha(0.12)).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
					icon(c, "info", 11, p.danger)
				})
				txt(c, a.t("无法发送请求", "Could not send request"), 14, p.fg, 600)
			})
			ui.Row(c).Gap(8).Children(func() {
				txtMono(c, t.draft.Method, fsBase, p.method(t.draft.Method), 600)
				url := client.EnsureProtocol(client.ResolveTemplate(t.draft.URL, a.vars()))
				if strings.TrimSpace(t.draft.URL) == "" {
					url = ""
				}
				txtMono(c, url, fsBase, p.fg, 0).Selectable()
			})
			parsed := client.ParseRequestError(strings.TrimPrefix(rs.err, "Error: "))
			info := client.DescribeRequestError(parsed)
			fg, bg := ui.Hex("#b44840"), ui.Hex("#fbeceb")
			if info.Tone == client.ToneWarning {
				fg, bg = ui.Hex("#a56412"), ui.Hex("#fbf2e6")
			}
			ui.Row(c).FillWidth().Padding(8, 10).Radius(9).Background(bg).Border(1, fg.Alpha(0.2)).Children(func() {
				ui.Text(c, a.t("错误: ", "Error: ")+info.Detail).FontSize(fsBase).TextColor(fg).Selectable()
			})
			ui.Text(c, a.t(info.DescZH+" "+info.SuggestZH, info.DescEN+" "+info.SuggestEN)).FontSize(fsBase).TextColor(p.fg2).Selectable().FillWidth()
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				if a.button(c, btnOpts{h: 24, radius: 7, padX: 8, gap: 6, key: "err-console"}, func(hover bool) {
					col := p.muted
					if hover {
						col = p.fg2
					}
					icon(c, "commandLine", 12, col)
					txt(c, a.t("在控制台查看", "View in console"), fsSmall, col, 0)
				}) {
					a.later(func() { a.setConsoleOpen(true) })
				}
				txt(c, "•", fsSmall, p.muted, 0)
				txt(c, a.t("错误码", "Code")+": "+parsed.Code, fsSmall, p.muted, 0)
			})
		})
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

// statusBadge 返回状态码徽标的文字色与底色，取值同旧版 statusBadgeClass（1xx 灰、2xx 绿、3xx 蓝、4xx 橙、其余红）。
func (a *app) statusBadge(code int) (fg, bg ui.Color) {
	if a.p.dark {
		switch {
		case code >= 100 && code < 200:
			return ui.Hex("#aeb4bd"), ui.Hex("#2d2f34")
		case code >= 200 && code < 300:
			return ui.Hex("#7ce0ad"), ui.Hex("#123a2c")
		case code >= 300 && code < 400:
			return ui.Hex("#72c7e4"), ui.Hex("#113342")
		case code >= 400 && code < 500:
			return ui.Hex("#f3bf67"), ui.Hex("#3f2f16")
		}
		return ui.Hex("#ff8a83"), ui.Hex("#402222")
	}
	switch {
	case code >= 100 && code < 200:
		return ui.Hex("#6f7782"), ui.Hex("#f1f2f4")
	case code >= 200 && code < 300:
		return ui.Hex("#2b8a57"), ui.Hex("#e6f6ec")
	case code >= 300 && code < 400:
		return ui.Hex("#2a7f95"), ui.Hex("#e6f3f8")
	case code >= 400 && code < 500:
		return ui.Hex("#a56412"), ui.Hex("#fbf2e6")
	}
	return ui.Hex("#b44840"), ui.Hex("#fbeceb")
}

// statusDot 返回状态点颜色：2xx 成功色、3xx info、4xx 警告色、其余危险色（同旧版 statusDotClass）。
func (a *app) statusDot(code int) ui.Color {
	switch {
	case code >= 200 && code < 300:
		return a.p.success
	case code >= 300 && code < 400:
		return a.p.info
	case code >= 400 && code < 500:
		return a.p.warning
	}
	return a.p.danger
}

// responseBody 绘制成功响应：标签行（含右侧状态/耗时/大小）与当前标签内容。
func (a *app) responseBody(c *ui.Context, t *tab) {
	rs := t.resp
	p := a.p
	labels := []string{a.t("响应体", "Body"), a.t("响应头", "Headers"), "Cookies"}
	counts := []int{0, a.respHeaderCount(rs), len(a.respCookies(rs))}
	for i := 1; i < 3; i++ {
		if counts[i] == 0 {
			counts[i] = countZero // 旧版响应头 / Cookies 标签始终显示数量
		}
	}
	if rs.streamed {
		labels = append(labels, "Stream")
		counts = append(counts, 0)
	}
	if rs.streaming && rs.resp == nil && len(labels) > 3 {
		rs.sub = 3
	}
	rs.sub = min(rs.sub, len(labels)-1)
	ui.Column(c).Grow(1).Basis(0).Children(func() {
		ui.Row(c).FillWidth().Height(32).Shrink(0).PaddingStart(16).PaddingEnd(8).Gap(4).AlignItems(ui.Center).Children(func() {
			a.pillTabs(c, "rs", &rs.sub, labels, counts)
			ui.Spacer(c)
			a.respMeta(c, rs)
		})
		switch {
		case rs.sub == 1:
			a.respHeadersView(c, rs)
		case rs.sub == 2:
			a.respCookiesView(c, rs)
		case rs.sub == 3 && rs.streamed:
			a.streamView(c, rs)
		default:
			a.bodyView(c, t, rs)
		}
	})
	_ = p
}

// respMeta 是标签行右侧的状态徽标、耗时、大小与网络信息；各项悬停时弹出详情卡片。
func (a *app) respMeta(c *ui.Context, rs *respState) {
	p := a.p
	code, text, _ := rs.statusInfo()
	if rs.resp == nil {
		if rs.streaming {
			ui.Spinner(c)
			txt(c, a.t("接收中…", "Receiving…"), fsSmall, p.muted, 0)
		}
		return
	}
	resp := rs.resp
	sep := func() { txt(c, "•", fsSmall, p.muted, 0).MarginX(0) }
	ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
		if len(resp.Warnings) > 0 {
			ui.Row(c).Height(20).PaddingX(6).Radius(7).Background(ui.RGB(251, 242, 230)).AlignItems(ui.Center).Children(func() {
				txtE(c, a.t("警告", "Warning")+": "+resp.Warnings[0], fsTiny, ui.RGB(165, 100, 18), 500).MaxWidth(260)
			})
			sep()
		}
		fg, bg := a.statusBadge(code)
		badge := ui.Row(c.Key("badge")).Height(20).PaddingX(6).Radius(7).Background(bg).AlignItems(ui.Center).Shrink(0)
		badge.Children(func() { txtMono(c, strings.TrimSpace(fmt.Sprintf("%d %s", code, text)), fsTiny, fg, 600) })
		a.hoverCard(c, "status", badge, 340, func() { a.statusCard(c, code, text) })
		sep()
		timeEl := txt(c, fmt.Sprintf("%.0fms", resp.Duration), fsSmall, p.muted, 0).Tooltip("")
		a.hoverCard(c, "time", timeEl, 460, func() { a.timingCard(c, resp) })
		sep()
		sizeEl := txt(c, formatSizeShort(resp.Size), fsSmall, p.muted, 0)
		a.hoverCard(c, "size", sizeEl, 280, func() { a.sizeCard(c, resp) })
		sep()
		net := ui.Row(c.Key("netbtn")).Size(20, 20).Radius(6).AlignItems(ui.Center).Justify(ui.Center).Shrink(0)
		net.Children(func() { icon(c, "globe", 13, p.muted) })
		a.hoverCard(c, "net", net, 460, func() { a.networkCard(c, resp) })
	})
}

// hoverCard 在锚点被悬停时，于其下方（右对齐）弹出详情卡片；鼠标移入卡片时保持显示。
func (a *app) hoverCard(c *ui.Context, key string, anchor ui.Element, w float32, build func()) {
	p := a.p
	open := anchor.Hovered() || a.side.cardHot == key
	hot := false
	ui.PopoverBase(c, anchor, &open, func(panel ui.Element) {
		panel = panel.AttachTo(anchor, ui.AnchorBottomRight, ui.AnchorTopRight).Margin(6, 0, 0, 0).Width(w).Radius(10).Background(p.surface).Border(1, p.border).Padding(10, 12)
		if panel.Hovered() {
			hot = true
		}
		ui.Column(c).FillWidth().Children(build)
	})
	if hot {
		a.side.cardHot = key
	} else if a.side.cardHot == key && !anchor.Hovered() {
		a.side.cardHot = ""
	}
}

// statusCard 是状态码卡片：圆点 + 状态 + 一句话说明。
func (a *app) statusCard(c *ui.Context, code int, text string) {
	p := a.p
	ui.Row(c).FillWidth().Gap(8).Children(func() {
		dot := a.statusDot(code)
		ui.Box(c).Size(8, 8).Radius(4).Background(dot).Margin(5, 0, 0, 0).Shrink(0)
		ui.Column(c).Grow(1).Basis(0).Gap(4).Children(func() {
			txt(c, strings.TrimSpace(fmt.Sprintf("%d %s", code, text)), fsLarge, p.fg, 600)
			ui.Text(c, a.statusHint(code)).FontSize(fsBase).TextColor(p.fg2)
		})
	})
}

// statusHint 返回状态码的中英文说明。
func (a *app) statusHint(code int) string {
	m := map[int][2]string{
		200: {"请求成功，服务器已返回预期结果。", "Request succeeded and the server returned the expected result."},
		201: {"资源创建成功。", "Resource created successfully."},
		204: {"请求成功，但响应体为空。", "Request succeeded, but the response body is empty."},
		301: {"资源已永久移动到新地址。", "Resource has moved permanently to a new location."},
		302: {"资源临时重定向到其他地址。", "Resource is temporarily redirected to another location."},
		304: {"资源未修改，可使用缓存。", "Resource not modified; cached content can be used."},
		400: {"请求参数或格式错误，服务器无法处理。", "Invalid request params or format; server cannot process it."},
		401: {"未通过身份认证，请检查凭据。", "Authentication failed. Check your credentials."},
		403: {"服务器理解请求，但拒绝访问。", "Server understood the request but refuses access."},
		404: {"请求资源不存在。", "Requested resource was not found."},
		405: {"请求方法不被该资源允许。", "Request method is not allowed for this resource."},
		408: {"请求超时，服务器等待客户端过久。", "Request timed out; server waited too long for the client."},
		409: {"请求冲突，通常与资源状态有关。", "Request conflict, usually related to resource state."},
		413: {"请求体过大，服务器拒绝处理。", "Request body is too large; server rejected it."},
		415: {"请求媒体类型不受支持。", "Unsupported request media type."},
		422: {"请求格式正确，但语义校验失败。", "Request format is valid but semantic validation failed."},
		429: {"请求过于频繁，触发限流。", "Too many requests; rate limit triggered."},
		500: {"服务器内部错误。", "Internal server error."},
		501: {"服务器不支持该请求能力。", "Server does not support this request capability."},
		502: {"网关收到上游无效响应。", "Gateway received an invalid upstream response."},
		503: {"服务暂时不可用。", "Service is temporarily unavailable."},
		504: {"网关等待上游响应超时。", "Gateway timed out waiting for upstream response."},
	}
	if v, ok := m[code]; ok {
		return a.t(v[0], v[1])
	}
	switch {
	case code >= 200 && code < 300:
		return a.t("请求已成功处理。", "Request was handled successfully.")
	case code >= 300 && code < 400:
		return a.t("请求被重定向，请关注 Location 或重定向策略。", "Request was redirected. Check Location or redirect policy.")
	case code >= 400 && code < 500:
		return a.t("客户端请求存在问题，请检查 URL、参数、认证和方法。", "Client request issue. Check URL, params, authentication, and method.")
	}
	return a.t("服务器处理请求时发生错误，请检查服务端日志或网关链路。", "Server error while processing request. Check server logs or gateway path.")
}

// timingCard 是响应时间卡片：每个阶段一行，带时间轴上的色块。
func (a *app) timingCard(c *ui.Context, resp *model.HttpResponse) {
	p := a.p
	tm := resp.Timings
	total := tm.Total
	if total <= 0 {
		total = resp.Duration
	}
	ttfb := tm.WaitingTTFB
	if ttfb == 0 {
		ttfb = resp.Duration
	}
	rows := []struct {
		label string
		v     float64
		col   ui.Color
	}{
		{a.t("准备", "Prepare"), tm.Prepare, ui.RGB(183, 189, 198)},
		{a.t("Socket 初始化", "Socket Initialization"), tm.SocketInitialization, ui.RGB(242, 190, 66)},
		{a.t("DNS 查询", "DNS Lookup"), tm.DNSLookup, ui.RGB(242, 190, 66)},
		{a.t("TCP 握手", "TCP Handshake"), tm.TCPHandshake, ui.RGB(76, 137, 227)},
		{a.t("SSL 握手", "SSL Handshake"), tm.SSLHandshake, ui.RGB(61, 120, 207)},
		{a.t("等待首字节 (TTFB)", "Waiting (TTFB)"), ttfb, ui.RGB(239, 143, 103)},
		{a.t("下载", "Download"), tm.Download, ui.RGB(92, 163, 121)},
		{a.t("处理", "Process"), tm.Process, ui.RGB(165, 173, 184)},
	}
	ui.Row(c).FillWidth().AlignItems(ui.Center).Padding(0, 0, 4, 0).Children(func() {
		icon(c, "clock", 14, p.muted)
		txt(c, a.t("响应时间", "Response Time"), fsLarge, p.fg, 600).MarginStart(8)
		ui.Spacer(c)
		txt(c, fmt.Sprintf("%.0fms", total), fsBase, p.fg, 600)
	})
	if total <= 0 {
		total = 1
	}
	const zone = 231.0
	cum := 0.0
	for _, r := range rows {
		off, w := cum/total*zone, math.Max(r.v/total*zone, 2.8)
		cum += r.v
		ui.Row(c).FillWidth().Height(22).Gap(6).AlignItems(ui.Center).Children(func() {
			txtE(c, r.label, fsTiny, p.fg2, 0).Width(120).Shrink(0)
			ui.Row(c).Width(zone+2).Height(20).Shrink(0).AlignItems(ui.Center).BorderWidth(0, 1, 0, 1).BorderColor(p.border).Children(func() {
				ui.Box(c).Width(float32(off)).Shrink(0)
				bar := ui.Box(c).Width(float32(w)).Height(12).Shrink(0)
				if r.col == ui.RGB(239, 143, 103) {
					bar.Background(ui.RGBA(253, 233, 229, 0.7)).Border(1, ui.RGB(230, 127, 113))
				} else {
					bar.Background(r.col)
				}
			})
			txt(c, fmt.Sprintf("%.2f ms", r.v), fsTiny, p.muted, 0).Width(68).Shrink(0).TextAlign(ui.End)
		})
	}
}

// sizeCard 是响应大小卡片：响应与请求各一组（头、体）。
func (a *app) sizeCard(c *ui.Context, resp *model.HttpResponse) {
	p := a.p
	sz := resp.SizeDetails
	group := func(iconName string, col ui.Color, title string, head, body int64, hl, bl string) {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			icon(c, iconName, 14, col)
			txt(c, title, fsLarge, p.fg, 600).MarginStart(8)
			ui.Spacer(c)
			txt(c, formatSize(head+body), fsLarge, p.fg, 600)
		})
		for _, r := range []struct {
			l string
			v int64
		}{{hl, head}, {bl, body}} {
			ui.Row(c).FillWidth().Height(22).AlignItems(ui.Center).Children(func() {
				txt(c, r.l, fsBase, p.fg2, 0)
				ui.Spacer(c)
				txt(c, formatSize(r.v), fsBase, p.fg2, 0)
			})
		}
	}
	body := sz.ResponseBody
	if body == 0 {
		body = resp.Size
	}
	group("download", ui.RGB(47, 111, 211), a.t("响应大小", "Response Size"), sz.ResponseHeaders, body, a.t("响应头", "Headers"), a.t("响应体", "Body"))
	hline(c, p.border).Margin(8, 0)
	group("upload", ui.RGB(168, 130, 31), a.t("请求大小", "Request Size"), sz.RequestHeaders, sz.RequestBody, a.t("请求头", "Headers"), a.t("请求体", "Body"))
}

// networkCard 是网络卡片：连接信息与 TLS 信息。
func (a *app) networkCard(c *ui.Context, resp *model.HttpResponse) {
	p := a.p
	n := resp.Network
	val := func(s string) string {
		if s == "" {
			return "-"
		}
		return s
	}
	var nd model.NetworkDetails
	if n != nil {
		nd = *n
	}
	ui.Row(c).FillWidth().AlignItems(ui.Center).Padding(0, 0, 6, 0).Children(func() {
		ui.Row(c).Size(20, 20).Radius(6).Background(p.surface2).AlignItems(ui.Center).Justify(ui.Center).Children(func() { icon(c, "globe", 12, p.fg2) })
		txt(c, a.t("网络", "Network"), fsBase, p.fg, 600).MarginStart(8)
	})
	block := func(rows [][2]string) {
		for _, r := range rows {
			ui.Row(c).FillWidth().MinHeight(24).Gap(12).Children(func() {
				txt(c, r[0], fsBase, p.fg2, 0).Width(126).Shrink(0)
				ui.Text(c, val(r[1])).FontSize(fsBase).Font(monoFont).TextColor(p.fg).Grow(1).Basis(0).Selectable()
			})
		}
	}
	block([][2]string{{a.t("HTTP 版本", "HTTP Version"), nd.HTTPVersion}, {a.t("本地地址", "Local Address"), nd.LocalAddress}, {a.t("远端地址", "Remote Address"), nd.RemoteAddress}})
	hline(c, p.border).Margin(6, 0)
	block([][2]string{{a.t("TLS 协议", "TLS Protocol"), nd.TLSProtocol}, {a.t("加密套件", "Cipher Name"), nd.CipherName}})
	hline(c, p.border).Margin(6, 0)
	block([][2]string{{a.t("证书 CN", "Certificate CN"), nd.CertificateCN}, {a.t("签发者 CN", "Issuer CN"), nd.IssuerCN}, {a.t("有效期至", "Valid Until"), nd.ValidUntil}})
}

// formatSizeShort 是标签行里的紧凑大小（"100 B"、"1.5 KB"）。
func formatSizeShort(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d B", n)
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
