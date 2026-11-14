package gui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"

	"minipost/internal/model"
)

// kvText 绘制“名称：值”的只读两列表。
func (a *app) kvText(c *ui.Context, rows [][2]string) {
	th := c.Theme()
	ui.Scroll(c).Grow(1).Children(func() {
		ui.Column(c).Gap(4).Padding(10).Children(func() {
			for i, r := range rows {
				ui.Row(c.Key(i)).Gap(12).Children(func() {
					ui.Text(c, r[0]).Bold().Width(180).Shrink(0).TextColor(th.TextMuted).Selectable()
					ui.Text(c, r[1]).Grow(1).Basis(0).Selectable()
				})
			}
		})
	})
}

func (a *app) respHeadersView(c *ui.Context, rs *respState) {
	headers := a.respHeadersMap(rs)
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var rows [][2]string
	for _, k := range keys {
		for _, v := range headers[k] {
			rows = append(rows, [2]string{k, v})
		}
	}
	if len(rows) == 0 {
		ui.Text(c, a.t("没有响应头", "No headers")).Padding(12).TextColor(c.Theme().TextMuted)
		return
	}
	a.kvText(c, rows)
}

func (a *app) respCookiesView(c *ui.Context, rs *respState) {
	var rows [][2]string
	for _, line := range a.respHeadersMap(rs)["Set-Cookie"] {
		first, attrs, _ := strings.Cut(line, ";")
		name, value, _ := strings.Cut(first, "=")
		rows = append(rows, [2]string{strings.TrimSpace(name), strings.TrimSpace(value) + "  " + strings.TrimSpace(attrs)})
	}
	if len(rows) == 0 {
		ui.Text(c, a.t("响应没有设置 Cookie", "The response set no cookies")).Padding(12).TextColor(c.Theme().TextMuted)
		return
	}
	a.kvText(c, rows)
}

func ms(v float64) string { return fmt.Sprintf("%.1f ms", v) }

func (a *app) respTimingView(c *ui.Context, rs *respState) {
	if rs.resp == nil {
		ui.Text(c, a.t("响应结束后显示", "Shown when the response completes")).Padding(12).TextColor(c.Theme().TextMuted)
		return
	}
	tm, sz := rs.resp.Timings, rs.resp.SizeDetails
	a.kvText(c, [][2]string{
		{"Prepare", ms(tm.Prepare)},
		{"Socket Initialization", ms(tm.SocketInitialization)},
		{"DNS Lookup", ms(tm.DNSLookup)},
		{"TCP Handshake", ms(tm.TCPHandshake)},
		{"SSL Handshake", ms(tm.SSLHandshake)},
		{"Waiting (TTFB)", ms(tm.WaitingTTFB)},
		{"Download", ms(tm.Download)},
		{"Process", ms(tm.Process)},
		{"Total", ms(tm.Total)},
		{a.t("响应头大小", "Response headers"), formatSize(sz.ResponseHeaders)},
		{a.t("响应体大小", "Response body"), formatSize(sz.ResponseBody)},
		{a.t("请求头大小", "Request headers"), formatSize(sz.RequestHeaders)},
		{a.t("请求体大小", "Request body"), formatSize(sz.RequestBody)},
	})
}

func (a *app) respNetworkView(c *ui.Context, rs *respState) {
	var n *model.NetworkDetails
	switch {
	case rs.resp != nil:
		n = rs.resp.Network
	case rs.start != nil:
		n = rs.start.Network
	}
	if n == nil {
		ui.Text(c, a.t("没有网络信息", "No network details")).Padding(12).TextColor(c.Theme().TextMuted)
		return
	}
	var rows [][2]string
	add := func(k, v string) {
		if v != "" {
			rows = append(rows, [2]string{k, v})
		}
	}
	add("HTTP Version", n.HTTPVersion)
	add("Local Address", n.LocalAddress)
	add("Remote Address", n.RemoteAddress)
	add("TLS Protocol", n.TLSProtocol)
	add("Cipher", n.CipherName)
	add("Certificate CN", n.CertificateCN)
	add("Issuer CN", n.IssuerCN)
	add("Valid Until", n.ValidUntil)
	a.kvText(c, rows)
}

// streamView 列出流式（SSE）收到的数据块。
func (a *app) streamView(c *ui.Context, rs *respState) {
	if len(rs.chunks) == 0 {
		ui.Text(c, a.t("等待数据…", "Waiting for data…")).Padding(12).TextColor(c.Theme().TextMuted)
		return
	}
	th := c.Theme()
	rs.chunkList.FollowEnd = true
	ui.List(c, &rs.chunkList, len(rs.chunks), func(i int) {
		ch := rs.chunks[i]
		ui.Row(c).Gap(10).Children(func() {
			ui.Textf(c, "#%d", ch.Sequence).TextColor(th.TextMuted).Width(48).Shrink(0)
			ui.Text(c, ch.Kind).TextColor(th.TextMuted).Width(110).Shrink(0)
			ui.Text(c, ch.Data).Grow(1).Basis(0).Selectable()
		})
	}).Grow(1)
}
