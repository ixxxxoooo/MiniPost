package gui

import (
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"

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
			a.respStateBar(c, func() { a.respFailedMeta(c) })
			a.respError(c, t)
		case rs.resp == nil && !rs.streamed && t.sending:
			a.respStateBar(c, func() { a.sendingMeta(c, t) })
			a.sendingSkeleton(c, t)
		case rs.resp == nil && !rs.streamed:
			a.respTitle(c)
			ui.Row(c).Grow(1).Basis(0).FillWidth().AlignItems(ui.Center).Justify(ui.Center).Children(func() {
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

// respStateBar 是失败/发送中状态下的标签行：与成功态同样的三个标签（不可切换），右侧放状态内容。
func (a *app) respStateBar(c *ui.Context, right func()) {
	p := a.p
	sel := 0
	labels := []string{a.t("响应体", "Body"), a.t("响应头", "Headers"), "Cookies"}
	ui.Row(c).FillWidth().Height(32).Shrink(0).PaddingX(8).Gap(2).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(p.border).Children(func() {
		a.underlineTabs(c, "rst", &sel, labels, nil, nil, 32)
		ui.Spacer(c)
		right()
	})
}

// respFailedMeta 是失败状态标签行右侧的“请求失败”红色胶囊。
func (a *app) respFailedMeta(c *ui.Context) {
	fg, bg := a.statusBadge(0)
	ui.Row(c.Key("failbadge")).Height(20).PaddingX(8).Gap(5).Radius(10).Background(bg).AlignItems(ui.Center).Shrink(0).Children(func() {
		ui.Box(c).Size(6, 6).Radius(3).Background(fg).Shrink(0)
		txt(c, a.t("请求失败", "Request failed"), fsTiny, fg, 600)
	})
}

// sendingMeta 是发送中标签行右侧：旋转图标 + 已用秒数 + 取消按钮。
func (a *app) sendingMeta(c *ui.Context, t *tab) {
	p := a.p
	elapsed := time.Since(t.sendAt)
	if t.sendAt.IsZero() {
		elapsed = 0
	}
	c.AnimationFrame()
	ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
		ui.Spinner(c)
		txtMono(c, a.t("发送中… ", "Sending… ")+sendingElapsed(elapsed), fsSmall, p.muted, 0)
		bg := p.surface
		if a.button(c, btnOpts{h: 22, radius: 6, padX: 10, bg: &bg, border: &p.btnBorder, key: "resp-cancel"}, func(bool) {
			txt(c, a.t("取消", "Cancel"), fsSmall, p.fg, 500)
		}) {
			a.be.CancelHTTPRequest(t.sendID)
		}
	})
}

// sendingSkeleton 是发送中的骨架屏：几条灰色占位条（轻微脉动）+ 等待提示。
func (a *app) sendingSkeleton(c *ui.Context, t *tab) {
	p := a.p
	elapsed := time.Since(t.sendAt)
	if t.sendAt.IsZero() {
		elapsed = 0
	}
	c.AnimationFrame()
	pulse := 0.55 + 0.45*dotPulse(elapsed, 0)
	host := ""
	if u, err := url.Parse(client.EnsureProtocol(client.ResolveTemplate(t.draft.URL, a.vars()))); err == nil {
		host = u.Host
	}
	ui.Column(c.Key("sending")).Grow(1).Basis(0).FillWidth().Padding(20, 16).Gap(10).Children(func() {
		for _, w := range []float32{0.34, 0.62, 0.46, 0.8, 0.28} {
			ui.Box(c).Width(520 * w / 0.8).Height(10).Radius(5).Background(p.border.Alpha(pulse)).Shrink(1)
		}
		ui.Row(c).Gap(6).Padding(8, 0, 0, 0).AlignItems(ui.Center).Children(func() {
			icon(c, "globe", 11, p.muted)
			label := a.t("正在等待响应首字节…", "Waiting for the first response byte…")
			if host != "" {
				label = a.t("正在等待 "+host+" 响应首字节…", "Waiting for the first byte from "+host+"…")
			}
			txt(c, label, fsSmall, p.muted, 0)
		})
	})
}

// respError 绘制请求失败卡片：原因说明、错误详情、逐步诊断清单与“重试 / 在控制台查看”。
func (a *app) respError(c *ui.Context, t *tab) {
	p := a.p
	rs := t.resp
	parsed := client.ParseRequestError(strings.TrimPrefix(rs.err, "Error: "))
	info := client.DescribeRequestError(parsed)
	url := ""
	if strings.TrimSpace(t.draft.URL) != "" {
		url = client.EnsureProtocol(client.ResolveTemplate(t.draft.URL, a.vars()))
	}
	steps := client.DiagnoseRequestError(parsed, url)
	ui.Scroll(c).Grow(1).Basis(0).Scrollbars(a.scrollbarsVisible()).Children(func() {
		ui.Column(c).FillWidth().Padding(20, 24).Gap(14).Children(func() {
			ui.Row(c).Gap(12).AlignItems(ui.Start).Children(func() {
				ui.Row(c).Size(30, 30).Radius(8).Background(p.danger.Alpha(0.12)).AlignItems(ui.Center).Justify(ui.Center).Shrink(0).Children(func() {
					icon(c, "toastWarn", 15, p.danger)
				})
				ui.Column(c).Gap(2).Grow(1).Basis(0).Children(func() {
					txt(c, a.t("无法发送请求", "Could not send request"), 14, p.fg, 600)
					ui.Text(c, a.t(info.DescZH+" "+info.SuggestZH, info.DescEN+" "+info.SuggestEN)).FontSize(fsBase).TextColor(p.fg2).Selectable().FillWidth()
				})
			})
			ui.Column(c).FillWidth().MaxWidth(720).Padding(8, 12).Gap(2).Radius(8).Background(p.surface2).Children(func() {
				ui.Text(c, a.t("错误: ", "Error: ")+info.Detail).FontSize(fsSmall).Font(monoFont).TextColor(p.fg2).Selectable()
				ui.Text(c, a.t("错误码: ", "Code: ")+parsed.Code).FontSize(fsSmall).Font(monoFont).TextColor(p.muted).Selectable()
			})
			ui.Column(c).Gap(6).Children(func() {
				for _, st := range steps {
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						if st.OK {
							icon(c, "tick", 12, p.success)
						} else {
							icon(c, "clear", 12, p.danger)
						}
						txt(c, a.t(st.ZH, st.EN), fsBase, p.fg, 0)
					})
				}
			})
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				retry := ui.ButtonBase(c.Key("err-retry")).Height(26).Radius(6).PaddingX(12).Gap(6).AlignItems(ui.Center).Background(p.accent)
				if retry.Hovered() {
					retry = retry.Background(p.accentHover)
				}
				retry.Children(func() {
					icon(c, "send", 11, p.accentFg)
					txt(c, a.t("重试", "Retry"), fsBase, p.accentFg, 600)
				})
				if retry.Clicked() {
					a.later(func() { a.sendTab(t) })
				}
				bg := p.surface
				if a.button(c, btnOpts{h: 26, radius: 6, padX: 10, gap: 6, bg: &bg, border: &p.btnBorder, key: "err-console"}, func(hover bool) {
					col := p.fg2
					if hover {
						col = p.fg
					}
					icon(c, "commandLine", 12, col)
					txt(c, a.t("在控制台查看", "View in console"), fsBase, col, 0)
				}) {
					a.later(func() { a.setConsoleOpen(true) })
				}
			})
		})
	})
}

// respStatusLabel 是状态胶囊里的文字，如 "200 OK"；状态说明已含状态码时不重复拼接。
func respStatusLabel(code int, text string) string {
	text = strings.TrimSpace(text)
	prefix := fmt.Sprint(code)
	if strings.HasPrefix(text, prefix) {
		return text
	}
	return strings.TrimSpace(prefix + " " + text)
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
		ui.Row(c).FillWidth().Height(32).Shrink(0).PaddingStart(12).PaddingEnd(10).Gap(2).AlignItems(ui.Center).Gradient(ui.RGBA(0, 0, 0, 0), p.surface2.Alpha(0.35), 180).BorderWidth(0, 0, 1, 0).BorderColor(p.border).Children(func() {
			a.underlineTabs(c, "rs", &rs.sub, labels, counts, nil, 32)
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
	ui.Row(c).Gap(6).AlignItems(ui.Center).Shrink(0).Children(func() {
		if len(resp.Warnings) > 0 {
			ui.Row(c).Height(20).PaddingX(8).Radius(10).Background(p.warning.Alpha(0.14)).AlignItems(ui.Center).Children(func() {
				txtE(c, a.t("警告", "Warning")+": "+resp.Warnings[0], fsSmall, p.warning, 600).MaxWidth(260)
			})
		}
		// 设计稿 C：状态胶囊（圆点 + 状态，1px 同色细环）+ 带图标的灰色度量（耗时、大小、TLS）
		fg, bg := a.statusBadge(code)
		badge := ui.Row(c.Key("badge")).Height(20).PaddingX(8).Gap(5).Radius(10).Background(bg).Border(1, fg.Alpha(0.13)).AlignItems(ui.Center).Shrink(0)
		badge.Children(func() {
			ui.Box(c).Size(6, 6).Radius(3).Background(fg).Shrink(0)
			txt(c, respStatusLabel(code, text), fsSmall, fg, 600)
		})
		a.hoverCard(c, "status", badge, 340, func() { a.statusCard(c, code, text) })
		mi := func(key, ic, label string, w float32, card func()) {
			el := ui.Row(c.Key(key)).Height(22).PaddingX(6).Gap(4).Radius(6).AlignItems(ui.Center).Shrink(0)
			if el.Hovered() {
				el = el.Background(p.hover)
			}
			el.Children(func() {
				icon(c, ic, 12, p.muted)
				if label != "" {
					txt(c, label, fsSmall+0.5, p.fg2, 0)
				}
			})
			a.hoverCard(c, strings.TrimSuffix(key, "anchor"), el, w, card)
		}
		mi("timeanchor", "clock", spaceUnit(formatDuration(resp.Duration)), 460, func() { a.timingCard(c, resp) })
		mi("sizeanchor", "database", formatSizeShort(resp.Size), 280, func() { a.sizeCard(c, resp) })
		mi("netbtn", "lock", "", 460, func() { a.networkCard(c, resp) })
	})
}

// spaceUnit 在数值与单位之间补一个空格（"128ms" → "128 ms"），对应设计稿的度量写法。
func spaceUnit(s string) string {
	for i, r := range s {
		if r != '.' && (r < '0' || r > '9') {
			if i == 0 || r == ' ' || s[i-1] == ' ' {
				return s
			}
			return s[:i] + " " + s[i:]
		}
	}
	return s
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
	if tm == (model.TimingBreakdown{}) {
		ttfb = resp.Duration // 没有分阶段数据（旧记录）时，整段耗时都算作等待
	}
	rows := []struct {
		label  string
		v      float64
		col    ui.Color
		dashed bool // 等待首字节用浅红底加虚线框，突出最耗时的等待阶段
	}{
		{a.t("准备", "Prepare"), tm.Prepare, ui.RGB(183, 189, 198), false},
		{a.t("Socket 初始化", "Socket Initialization"), tm.SocketInitialization, ui.RGB(242, 190, 66), false},
		{a.t("DNS 查询", "DNS Lookup"), tm.DNSLookup, ui.RGB(242, 190, 66), false},
		{a.t("TCP 握手", "TCP Handshake"), tm.TCPHandshake, ui.RGB(76, 137, 227), false},
		{a.t("SSL 握手", "SSL Handshake"), tm.SSLHandshake, ui.RGB(61, 120, 207), false},
		{a.t("等待首字节 (TTFB)", "Waiting (TTFB)"), ttfb, ui.RGB(239, 143, 103), true},
		{a.t("下载", "Download"), tm.Download, ui.RGB(92, 163, 121), false},
		{a.t("处理", "Process"), tm.Process, ui.RGB(165, 173, 184), false},
	}
	ui.Row(c).FillWidth().AlignItems(ui.Center).Padding(0, 0, 4, 0).Children(func() {
		icon(c, "clock", 14, p.muted)
		txt(c, a.t("响应时间", "Response Time"), fsLarge, p.fg, 600).MarginStart(8)
		ui.Spacer(c)
		txt(c, formatDuration(total), fsBase, p.fg, 600)
	})
	if total <= 0 {
		total = 1
	}
	const zone = 231.0
	vals := make([]float64, len(rows))
	for i, r := range rows {
		vals[i] = r.v
	}
	bars := timelineLayout(vals, total, zone)
	for i, r := range rows {
		off, w := bars[i].off, bars[i].w
		ui.Row(c).FillWidth().Height(22).Gap(6).AlignItems(ui.Center).Children(func() {
			txtE(c, r.label, fsTiny, p.fg2, 0).Width(120).Shrink(0)
			ui.Row(c).Width(zone+2).Height(20).Shrink(0).AlignItems(ui.Center).BorderWidth(0, 1, 0, 1).BorderColor(p.border).Children(func() {
				ui.Box(c).Width(off).Shrink(0)
				bar := ui.Box(c).Width(w).Height(12).Shrink(0)
				if r.dashed {
					bar.Background(ui.RGBA(253, 233, 229, 0.7)).Border(1, ui.RGB(230, 127, 113)).BorderStyle(ui.BorderDashed)
				} else {
					bar.Background(r.col)
				}
			})
			txt(c, formatPhaseMs(r.v), fsTiny, p.muted, 0).Width(68).Shrink(0).TextAlign(ui.End)
		})
	}
}

// barSpan 是某个阶段在时间轴上的起点偏移与宽度（像素）。
type barSpan struct{ off, w float32 }

// timelineMinBar 是色块的最小宽度：再短的阶段也要看得见。
const timelineMinBar = 2.5

// timelineLayout 把各阶段按顺序排到宽 zone 的时间轴上，像瀑布一样一级一级错开：
// 后一段从前一段“画出来的右端”开始，各段合起来正好铺满时间轴。
//   - 宽度按耗时占比，最短 timelineMinBar，保证极短的阶段（准备、Socket、DNS、TCP）也各占一格，
//     不会全叠在同一个位置；
//   - 最小宽度撑出的多余长度，由超过最小宽度的阶段按各自“多出来的部分”等比让出，
//     所以最长的阶段（通常是等待首字节）会略微缩短，整条仍刚好到右端。
func timelineLayout(vals []float64, total, zone float64) []barSpan {
	n := len(vals)
	ws := make([]float64, n)
	sum, flex := 0.0, 0.0
	for i, v := range vals {
		ws[i] = math.Max(v/total*zone, timelineMinBar)
		sum += ws[i]
		flex += ws[i] - timelineMinBar
	}
	if over := sum - zone; over > 0 && flex > 0 {
		k := math.Max(1-over/flex, 0)
		for i := range ws {
			ws[i] = timelineMinBar + (ws[i]-timelineMinBar)*k
		}
	}
	out := make([]barSpan, n)
	x := 0.0
	for i, w := range ws {
		out[i] = barSpan{off: float32(x), w: float32(w)}
		x += w
	}
	return out
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
	head, body := sz.ResponseHeaders, sz.ResponseBody
	if sz == (model.SizeBreakdown{}) {
		head, body = 0, resp.Size // 没有分项数据（旧记录）时，只能把总大小算作响应体
	}
	group("download", ui.RGB(47, 111, 211), a.t("响应大小", "Response Size"), head, body, a.t("响应头", "Headers"), a.t("响应体", "Body"))
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
