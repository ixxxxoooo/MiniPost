package gui

import (
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// respCookie 是响应里解析出的一个 Set-Cookie；attrs 保留原始属性串，其余字段是从中解析出的常用属性。
type respCookie struct {
	name, value, attrs string
	path, expires      string
	secure, httpOnly   bool
}

// respCookies 解析响应头里的 Set-Cookie。
func (a *app) respCookies(rs *respState) []respCookie {
	var out []respCookie
	for _, line := range a.respHeadersMap(rs)["Set-Cookie"] {
		first, attrs, _ := strings.Cut(line, ";")
		name, value, _ := strings.Cut(first, "=")
		ck := respCookie{name: strings.TrimSpace(name), value: strings.TrimSpace(value), attrs: strings.TrimSpace(attrs), path: "/", expires: a.t("会话", "Session")}
		for _, part := range strings.Split(attrs, ";") {
			k, v, _ := strings.Cut(strings.TrimSpace(part), "=")
			switch strings.ToLower(k) {
			case "path":
				ck.path = v
			case "secure":
				ck.secure = true
			case "httponly":
				ck.httpOnly = true
			case "expires":
				ck.expires = cookieExpiry(v)
			case "max-age":
				ck.expires = v + " s"
			}
		}
		out = append(out, ck)
	}
	return out
}

// cookieExpiry 把 Expires 属性值（RFC 1123）整理成「年-月-日」；解析不了就原样返回。
func cookieExpiry(v string) string {
	for _, layout := range []string{time.RFC1123, time.RFC1123Z, "Mon, 02-Jan-2006 15:04:05 MST"} {
		if t, err := time.Parse(layout, v); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return v
}

// respCol 描述只读表格的一列：width 为 0 表示按权重伸缩（grow），否则是固定宽度。
type respCol struct {
	head  string
	width float32 // >0 固定宽度；0 伸缩
	grow  float32 // 伸缩权重
	kind  int     // respText 等宽文字 / respMuted 次要文字 / respName 着色的名称
}

const (
	respText  = iota // 等宽、主色
	respName         // 等宽、信息色（名称列）
	respMuted        // 普通字体、弱色
)

// respTable 绘制响应头 / Cookies 的只读表格：与请求侧键值表同一套规范（表头 28 高、列间发丝线、
// 首列 34 宽放序号），名称列着色，悬停整行高亮；copyCol>=0 时行尾显示复制该列值的按钮。
// tags 非 nil 时，最后一列用它绘制（Cookies 的 S / H 标记）。
func (a *app) respTable(c *ui.Context, cols []respCol, rows [][]string, copyCol int, tags func(row int)) {
	p := a.p
	cellW := func(rc respCol) float32 {
		if rc.width > 0 {
			return -rc.width
		}
		return max(rc.grow, 1)
	}
	hasAct := copyCol >= 0
	ui.Scroll(c).Grow(1).Basis(0).Scrollbars(a.scrollbarsVisible()).Children(func() {
		ui.Column(c).FillWidth().Children(func() {
			a.kvHeaderRow(c, func() {
				a.kvHeaderCell(c, colCheck, "", nil)
				for i, rc := range cols {
					if i == len(cols)-1 && !hasAct {
						a.kvHeaderCellEnd(c, cellW(rc), rc.head)
					} else {
						a.kvHeaderCell(c, cellW(rc), rc.head, nil)
					}
				}
				if hasAct {
					a.kvHeaderCellEnd(c, colAct, "")
				}
			})
			for ri, r := range rows {
				ri, r := ri, r
				row := ui.Row(c.Key(ri)).FillWidth()
				hover := row.Hovered()
				if hover {
					row = row.Background(p.surface2)
				}
				row.Children(func() {
					a.tableCell(c, colCheck, kvRowH, func() {
						ui.Row(c).FillWidth().Justify(ui.Center).Children(func() { txtMono(c, itoa(ri+1), fsTiny, p.muted, 0) })
					})
					for i, rc := range cols {
						body := func() {
							if i == len(cols)-1 && tags != nil {
								ui.Row(c).FillWidth().PaddingX(8).Gap(4).Children(func() { tags(ri) })
								return
							}
							v := ""
							if i < len(r) {
								v = r[i]
							}
							switch rc.kind {
							case respMuted:
								txtE(c, v, fsBase, p.muted, 0).PaddingX(8).Grow(1)
							case respName:
								txtEMono(c, v, fsBase, p.info, 0).Selectable().PaddingX(8).Grow(1).Basis(0)
							default:
								txtEMono(c, v, fsBase, p.fg, 0).Selectable().PaddingX(8).Grow(1).Basis(0)
							}
						}
						if i == len(cols)-1 && !hasAct {
							a.tableCellEnd(c, cellW(rc), kvRowH, body)
						} else {
							a.tableCell(c, cellW(rc), kvRowH, body)
						}
					}
					if hasAct {
						a.tableCellEnd(c, colAct, kvRowH, func() {
							ui.Row(c).FillWidth().Justify(ui.Center).Children(func() {
								if hover && copyCol < len(r) && a.iconButton(c.Key("cp"), "copy", a.t("复制值", "Copy value"), 12, p.info) {
									a.copyText(r[copyCol], a.t("已复制", "Copied"))
								}
							})
						})
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
	a.respTable(c, []respCol{{head: a.t("名称", "Name"), grow: 1, kind: respName}, {head: a.t("值", "Value"), grow: 2.2, kind: respText}}, rows, 1, nil)
}

func (a *app) respCookiesView(c *ui.Context, rs *respState) {
	cookies := a.respCookies(rs)
	if len(cookies) == 0 {
		a.respEmpty(c, a.t("无 Cookies", "No cookies"))
		return
	}
	rows := make([][]string, 0, len(cookies))
	for _, ck := range cookies {
		rows = append(rows, []string{ck.name, ck.value, ck.path, ck.expires, ""})
	}
	cols := []respCol{
		{head: a.t("名称", "Name"), grow: 1, kind: respName},
		{head: a.t("值", "Value"), grow: 1.6, kind: respText},
		{head: a.t("路径", "Path"), width: 80, kind: respText},
		{head: a.t("过期", "Expires"), width: 110, kind: respMuted},
		{head: a.t("属性", "Flags"), width: 72},
	}
	p := a.p
	flag := func(s string, tip string) {
		ui.Row(c).Height(16).PaddingX(5).Radius(4).Background(p.surface2).Border(0.5, p.border).AlignItems(ui.Center).Children(func() {
			txtMono(c, s, 9.5, p.fg2, 600)
		}).Tooltip(tip)
	}
	a.respTable(c, cols, rows, -1, func(i int) {
		if cookies[i].secure {
			flag("S", "Secure")
		}
		if cookies[i].httpOnly {
			flag("H", "HttpOnly")
		}
	})
}

// respEmpty 是响应子标签没有内容时居中的灰色提示。
func (a *app) respEmpty(c *ui.Context, s string) {
	ui.Row(c).Grow(1).Basis(0).FillWidth().AlignItems(ui.Center).Justify(ui.Center).Children(func() {
		txt(c, s, 10.4, a.p.muted, 0)
	})
}
