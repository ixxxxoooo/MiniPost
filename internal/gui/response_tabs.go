package gui

import (
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"
)

// respCookie 是响应里解析出的一个 Set-Cookie。
type respCookie struct{ name, value, attrs string }

// respCookies 解析响应头里的 Set-Cookie。
func (a *app) respCookies(rs *respState) []respCookie {
	var out []respCookie
	for _, line := range a.respHeadersMap(rs)["Set-Cookie"] {
		first, attrs, _ := strings.Cut(line, ";")
		name, value, _ := strings.Cut(first, "=")
		out = append(out, respCookie{strings.TrimSpace(name), strings.TrimSpace(value), strings.TrimSpace(attrs)})
	}
	return out
}

// respTable 绘制响应头/Cookies 的只读表格：灰底大写表头，行间细线，等宽字体。
func (a *app) respTable(c *ui.Context, heads []string, widths []float32, rows [][]string) {
	p := a.p
	ui.Scroll(c).Grow(1).Basis(0).Children(func() {
		ui.Column(c).FillWidth().Padding(8).Children(func() {
			ui.Row(c).FillWidth().Background(p.surface2).BorderWidth(0, 0, 1, 0).BorderColor(p.btnBorder).Children(func() {
				for i, h := range heads {
					cell := txt(c, strings.ToUpper(h), 10.4, p.fg2, 600).PaddingX(12).PaddingY(6)
					if widths[i] > 0 {
						cell.Width(widths[i]).Shrink(0)
					} else {
						cell.Grow(1).Basis(0)
					}
				}
			})
			for ri, r := range rows {
				row := ui.Row(c.Key(ri)).FillWidth().BorderWidth(0, 0, 1, 0).BorderColor(p.border)
				if row.Hovered() {
					row = row.Background(p.sideHover)
				}
				row.Children(func() {
					for i, v := range r {
						weight, col := 0, p.fg2
						if i == 0 {
							weight, col = 500, p.fg
						}
						cell := ui.Text(c, v).FontSize(fsBase).Font(monoFont).TextColor(col).Selectable().PaddingX(12).PaddingY(4)
						if weight > 0 {
							cell = cell.FontWeight(weight)
						}
						if widths[i] > 0 {
							cell.Width(widths[i]).Shrink(0)
						} else {
							cell.Grow(1).Basis(0)
						}
					}
				})
			}
		})
	})
}

func (a *app) respHeadersView(c *ui.Context, rs *respState) {
	headers := a.respHeadersMap(rs)
	if len(headers) == 0 {
		a.respEmpty(c, a.t("无响应头", "No response headers"))
		return
	}
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	rows := make([][]string, 0, len(keys))
	for _, k := range keys {
		rows = append(rows, []string{k, strings.Join(headers[k], ", ")})
	}
	a.respTable(c, []string{a.t("请求头", "Header"), a.t("值", "Value")}, []float32{200, 0}, rows)
}

func (a *app) respCookiesView(c *ui.Context, rs *respState) {
	cookies := a.respCookies(rs)
	if len(cookies) == 0 {
		a.respEmpty(c, a.t("无 Cookies", "No cookies"))
		return
	}
	rows := make([][]string, 0, len(cookies))
	for _, ck := range cookies {
		attrs := ck.attrs
		if attrs == "" {
			attrs = "-"
		}
		rows = append(rows, []string{ck.name, ck.value, attrs})
	}
	a.respTable(c, []string{"Name", "Value", "Attributes"}, []float32{220, 260, 0}, rows)
}

// respEmpty 是响应子标签没有内容时居中的灰色提示。
func (a *app) respEmpty(c *ui.Context, s string) {
	ui.Row(c).Grow(1).Basis(0).FillWidth().AlignItems(ui.Center).Justify(ui.Center).Children(func() {
		txt(c, s, 10.4, a.p.muted, 0)
	})
}

// streamView 列出流式（SSE）收到的数据块。
func (a *app) streamView(c *ui.Context, rs *respState) {
	if len(rs.chunks) == 0 {
		a.respEmpty(c, a.t("等待数据…", "Waiting for data…"))
		return
	}
	p := a.p
	rs.chunkList.FollowEnd = true
	ui.List(c, &rs.chunkList, len(rs.chunks), func(i int) {
		ch := rs.chunks[i]
		ui.Row(c).Gap(10).PaddingX(12).Children(func() {
			txtMono(c, "#"+itoa(int(ch.Sequence)), fsSmall, p.muted, 0).Width(48).Shrink(0)
			txtMono(c, ch.Kind, fsSmall, p.muted, 0).Width(110).Shrink(0)
			ui.Text(c, ch.Data).FontSize(fsSmall).Font(monoFont).TextColor(p.fg).Grow(1).Basis(0).Selectable()
		})
	}).Grow(1).Basis(0)
}
