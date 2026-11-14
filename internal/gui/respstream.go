package gui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	"minipost/internal/client"
	"minipost/internal/model"
)

// maxStreamRows 是 Stream 标签最多渲染的消息数（只保留最近的，避免界面卡顿）。
const maxStreamRows = 240

// streamViewState 是 Stream 标签的界面状态：搜索词、各条消息的展开与格式，以及详情编辑器。
type streamViewState struct {
	search   string
	expanded map[int]bool
	mode     map[int]string // 详情格式："json"（默认）或 "text"
	wrap     bool
	eds      map[int]*editor.State
	edKey    map[int]string // 编辑器当前载入内容的标识，变化时重新载入

	view   int              // 0 事件列表；1 合并文本（仅 token 流可用）
	scroll ui.ScrollState   // 事件列表的滚动位置：停在底部时自动跟随新消息
	merged mergedStreamText // 合并文本的缓存
}

// mergedStreamText 是 token 流合并结果的缓存：只在消息条数变化时重算。
type mergedStreamText struct {
	n    int // 计算时的 chunk 数
	text string
	ok   bool
	ed   *editor.State
	key  string
}

// sync 在消息条数变化时重新合并，返回是否是可合并的 token 流。
func (m *mergedStreamText) sync(chunks []model.StreamChunk) bool {
	if m.n != len(chunks) {
		m.n = len(chunks)
		m.text, m.ok = client.MergeStreamText(chunks)
	}
	return m.ok
}

// streamEntries 返回要展示的消息：去掉 response_start，只留最近 maxStreamRows 条，按时间先后排列（新消息在底部）。
func streamEntries(chunks []model.StreamChunk) []model.StreamChunk {
	kept := make([]model.StreamChunk, 0, len(chunks))
	for _, ch := range chunks {
		if ch.Kind != "response_start" {
			kept = append(kept, ch)
		}
	}
	if len(kept) > maxStreamRows {
		kept = kept[len(kept)-maxStreamRows:]
	}
	return kept
}

// filterStreamEntries 按搜索词（不区分大小写，匹配类型、数据与原文）过滤消息。
func filterStreamEntries(entries []model.StreamChunk, query string) []model.StreamChunk {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return entries
	}
	var out []model.StreamChunk
	for _, e := range entries {
		if strings.Contains(strings.ToLower(e.Kind+" "+e.Data+" "+e.Raw), query) {
			out = append(out, e)
		}
	}
	return out
}

// streamTimeLabel 把时间戳格式化为 HH:MM:SS.mmm，无法解析时返回占位。
func streamTimeLabel(ts string) string {
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return "--:--:--.---"
	}
	return t.Local().Format("15:04:05.000")
}

// streamOffsetLabel 是消息相对第一条消息的时间偏移：+0.123s、+12.40s、+1m 05s。
func streamOffsetLabel(ts string, base time.Time) string {
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil || base.IsZero() {
		return "+--"
	}
	d := t.Sub(base)
	if d < 0 {
		d = 0
	}
	switch {
	case d < 10*time.Second:
		return fmt.Sprintf("+%.3fs", d.Seconds())
	case d < time.Minute:
		return fmt.Sprintf("+%.2fs", d.Seconds())
	}
	s := int(d.Seconds())
	return fmt.Sprintf("+%dm %02ds", s/60, s%60)
}

// streamBase 是计时起点：第一条真实消息的时间。
func streamBase(chunks []model.StreamChunk) time.Time {
	for _, ch := range chunks {
		if ch.Kind == "response_start" {
			continue
		}
		if t, err := time.Parse(time.RFC3339Nano, ch.Timestamp); err == nil {
			return t
		}
	}
	return time.Time{}
}

// streamStats 是工具栏右侧的统计：消息条数与已接收字节数。
func streamStats(chunks []model.StreamChunk) (n int, bytes int64) {
	for _, ch := range chunks {
		if ch.Kind == "response_start" {
			continue
		}
		n++
		if ch.BytesTotal > bytes {
			bytes = ch.BytesTotal
		}
	}
	return n, bytes
}

func (a *app) streamSummary(e model.StreamChunk) string {
	switch e.Kind {
	case "connection_closed":
		return a.t("连接已关闭", "Connection closed")
	case "error":
		if e.Data != "" {
			return e.Data
		}
		return a.t("流式错误", "Stream error")
	case "comment":
		return a.t("心跳 · ", "Keep-alive · ") + e.Data
	}
	if e.Data != "" {
		return e.Data
	}
	return e.Raw
}

// streamDetailText 是展开后详情里的文本：JSON 模式尝试缩进，失败则原样。
func streamDetailText(e model.StreamChunk, mode string) string {
	text := e.Data
	if text == "" {
		text = e.Raw
	}
	if mode == "text" {
		return text
	}
	return client.FormatBody(client.ModeJSON, text)
}

// streamView 绘制 Stream 标签：工具栏（视图切换、搜索、统计）+ 事件列表（按时间先后，停在底部时自动跟随新消息），
// 点击消息展开详情；识别出大模型 token 流时可切到“合并文本”，直接读完整回答。
func (a *app) streamView(c *ui.Context, rs *respState) {
	p := a.p
	v := &rs.stream
	if v.expanded == nil {
		v.expanded, v.mode = map[int]bool{}, map[int]string{}
		v.eds, v.edKey = map[int]*editor.State{}, map[int]string{}
	}
	all := streamEntries(rs.chunks)
	if len(all) == 0 {
		a.respEmpty(c, a.t("暂无流式消息", "No stream messages"))
		return
	}
	canMerge := v.merged.sync(rs.chunks)
	if !canMerge {
		v.view = 0
	}
	entries := filterStreamEntries(all, v.search)
	base := streamBase(rs.chunks)
	count, bytes := streamStats(rs.chunks)
	ui.Column(c).Grow(1).Basis(0).Radius(10).Background(p.surface).Border(1, p.border).Children(func() {
		ui.Column(c).FillWidth().Shrink(0).Padding(8).BorderWidth(0, 0, 1, 0).BorderColor(p.border).Children(func() {
			if v.view == 0 && len(rs.chunks)-1 > maxStreamRows {
				txt(c, a.t("仅展示最近流消息，避免页面卡顿。", "Showing recent stream messages only to keep UI responsive."), fsTiny, p.muted, 0).Margin(0, 0, 4, 0)
			}
			ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Children(func() {
				if canMerge {
					a.pillTabs(c, "ssv", &v.view, []string{a.t("事件", "Events"), a.t("合并文本", "Merged text")}, nil)
				}
				if v.view == 0 {
					ui.Row(c).Grow(1).Basis(0).Gap(4).AlignItems(ui.Center).Children(func() {
						a.input(c, &v.search, a.t("搜索", "Search"), fieldOpts{h: 28, radius: 7, fs: fsBase, icon: "search", key: "stream-search"})
						if v.search != "" {
							if a.iconButton(c, "clear", a.t("清空", "Clear"), 12, p.muted) {
								v.search = ""
							}
						}
					})
				} else {
					ui.Spacer(c)
				}
				a.streamStatus(c, rs, count, bytes)
			})
		})
		if v.view == 1 {
			a.mergedView(c, v)
			return
		}
		// 停在底部（或还没滚动过）时自动跟随；用户向上滚动就不再打扰
		if rs.streaming && v.scroll.Y >= v.scroll.MaxY-4 {
			v.scroll.Y = math.MaxFloat32
		}
		ui.Scroll(c).Grow(1).Basis(0).TrackScroll(&v.scroll).Scrollbars(a.scrollbarsVisible()).Children(func() {
			ui.Column(c).FillWidth().Children(func() {
				for i, e := range entries {
					a.streamRow(c, rs, e, i, base)
				}
				if len(entries) == 0 {
					ui.Row(c).FillWidth().Padding(32, 12).Justify(ui.Center).Children(func() {
						txt(c, a.t("没有匹配的流式消息", "No matching stream messages"), fsBase, p.muted, 0)
					})
				}
			})
		})
	})
}

// streamStatus 是工具栏右侧：接收中（主题色呼吸点）或已结束，加上消息条数与大小。
func (a *app) streamStatus(c *ui.Context, rs *respState, count int, bytes int64) {
	p := a.p
	ui.Row(c).Gap(6).AlignItems(ui.Center).Shrink(0).Children(func() {
		if rs.streaming {
			c.AnimationFrame()
			ui.Box(c).Size(7, 7).Radius(3.5).Background(p.accent.Alpha(dotPulse(time.Duration(time.Now().UnixNano()), 0)))
			txt(c, a.t("接收中", "Receiving"), fsTiny, p.accent, 500)
			txt(c, "·", fsTiny, p.muted, 0)
		}
		txtMono(c, a.t(fmt.Sprintf("%d 条", count), fmt.Sprintf("%d msgs", count))+" · "+formatSizeShort(bytes), fsTiny, p.muted, 0)
	})
}

// mergedView 是“合并文本”视图：把所有增量拼成完整回答，只读、自动换行、可复制。
func (a *app) mergedView(c *ui.Context, v *streamViewState) {
	p := a.p
	m := &v.merged
	if m.ed == nil {
		m.ed = &editor.State{Buffer: editor.NewBuffer(""), ReadOnly: true, SoftWrap: true}
	}
	if m.key != m.text {
		m.key = m.text
		m.ed.Buffer.Reset(m.text)
	}
	ui.Row(c).FillWidth().Height(32).Shrink(0).PaddingX(8).Gap(8).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(p.border).Children(func() {
		txt(c, a.t("把所有增量片段按顺序拼成的完整文本", "All deltas joined in order"), fsSmall, p.muted, 0)
		ui.Spacer(c)
		txtMono(c, a.t(fmt.Sprintf("%d 字", len([]rune(m.text))), fmt.Sprintf("%d chars", len([]rune(m.text)))), fsTiny, p.muted, 0)
		if a.button(c, btnOpts{w: 24, h: 24, radius: 6, border: &p.btnBorder, tip: a.t("复制", "Copy"), key: "merged-copy"}, func(bool) {
			icon(c, "copy", 11, p.fg2)
		}) {
			text := m.text
			a.later(func() {
				a.writeClipboard(text)
				a.toastOK(a.t("已复制", "Copied"))
			})
		}
	})
	a.codeView(c, m.ed).Grow(1).Basis(0).FillWidth()
}

// streamRow 绘制一条消息：序号、事件名徽标、摘要、相对时间与（展开时的）详情区。
func (a *app) streamRow(c *ui.Context, rs *respState, e model.StreamChunk, index int, base time.Time) {
	p := a.p
	v := &rs.stream
	id := e.Sequence
	closable := e.Kind == "connection_closed"
	expanded := v.expanded[id]
	box := ui.Column(c.Key("sr" + itoa(id))).FillWidth()
	if index > 0 {
		box = box.BorderWidth(1, 0, 0, 0).BorderColor(p.border)
	}
	box.Children(func() {
		row := ui.ButtonBase(c.Key("srb"+itoa(id))).FillWidth().Padding(6, 8).Gap(8).AlignItems(ui.Center)
		if expanded || row.Hovered() {
			row = row.Background(p.surface2)
		}
		row.Children(func() {
			chip := p.sideHover
			if p.dark {
				chip = p.elevated()
			}
			ui.Row(c).Size(16, 16).Radius(4).Background(chip).AlignItems(ui.Center).Justify(ui.Center).Shrink(0).Children(func() {
				switch {
				case e.Kind == "error":
					icon(c, "info", 10, p.danger)
				case closable || e.Kind == "comment":
					icon(c, "info", 10, p.muted)
				default:
					icon(c, "download", 10, p.accent)
				}
			})
			if e.Event != "" {
				fg, bg := p.accent, p.accent.Alpha(0.12)
				if e.Event == "error" {
					fg, bg = p.danger, p.danger.Alpha(0.12)
				}
				ui.Row(c).Height(18).PaddingX(6).Radius(5).Background(bg).AlignItems(ui.Center).Shrink(0).Children(func() {
					txtMono(c, e.Event, fsTiny, fg, 500)
				})
			}
			summary := a.streamSummary(e)
			if summary == "" {
				summary = a.t("(空消息块)", "(empty chunk)")
			}
			sumCol := p.fg
			if e.Kind == "comment" {
				sumCol = p.muted
			}
			txtEMono(c, strings.ReplaceAll(summary, "\n", " ↵ "), fsSmall, sumCol, 0).Grow(1).Basis(0)
			ui.Row(c).Gap(4).AlignItems(ui.Center).Shrink(0).Children(func() {
				txtMono(c, streamOffsetLabel(e.Timestamp, base), fsSmall, p.muted, 0)
				if !closable {
					deg := float32(-90)
					if expanded {
						deg = 180
					}
					icon(c, "arrowDown", 10, p.muted).Rotate(deg)
				}
			})
		})
		if row.Clicked() && !closable {
			v.expanded[id] = !expanded
		}
		if expanded && !closable {
			a.streamDetail(c, rs, e)
		}
	})
}

// streamDetail 是展开后的详情：格式选择、换行、复制与 240px 高的只读编辑器。
func (a *app) streamDetail(c *ui.Context, rs *respState, e model.StreamChunk) {
	p := a.p
	v := &rs.stream
	id := e.Sequence
	mode := v.mode[id]
	if mode == "" {
		mode = "json"
	}
	text := streamDetailText(e, mode)
	ed := v.eds[id]
	if ed == nil {
		ed = &editor.State{Buffer: editor.NewBuffer(""), ReadOnly: true}
		v.eds[id] = ed
	}
	if key := mode + "|" + text; v.edKey[id] != key {
		v.edKey[id] = key
		ed.Buffer.Reset(text)
		file := "x.txt"
		if mode == "json" {
			file = "x.json"
		}
		ed.Buffer.SetLanguage(editor.LanguageFor(file))
	}
	ed.SoftWrap = v.wrap
	ui.Column(c).FillWidth().Background(p.surface).BorderWidth(1, 0, 0, 0).BorderColor(p.border).Children(func() {
		ui.Row(c).FillWidth().Height(32).Shrink(0).PaddingX(8).Gap(8).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(p.border).Children(func() {
			label := "JSON"
			if mode == "text" {
				label = a.t("文本", "Text")
			}
			ui.ButtonBase(c.Key("srm"+itoa(id))).Height(24).Radius(6).PaddingX(8).Gap(4).AlignItems(ui.Center).Border(1, p.btnBorder).Shrink(0).
				PopoverMenuButton(func(m *ui.Menu) {
					if m.Item("JSON").Checked(mode == "json").Chosen() {
						a.later(func() { v.mode[id] = "json" })
					}
					if m.Item(a.t("文本", "Text")).Checked(mode == "text").Chosen() {
						a.later(func() { v.mode[id] = "text" })
					}
				}).Children(func() {
				txt(c, label, fsSmall, p.fg, 0)
				icon(c, "arrowDown", 8, p.muted)
			})
			wrapBd := p.btnBorder
			if v.wrap {
				wrapBd = p.accent
			}
			if a.button(c, btnOpts{w: 24, h: 24, radius: 6, border: &wrapBd, active: v.wrap, tip: a.t("切换自动换行", "Toggle line wrap"), key: "srw" + itoa(id)}, func(bool) {
				icon(c, "arrowLeftRight", 11, p.fg2)
			}) {
				v.wrap = !v.wrap
			}
			if a.button(c, btnOpts{w: 24, h: 24, radius: 6, border: &p.btnBorder, tip: a.t("复制", "Copy"), key: "src" + itoa(id)}, func(bool) {
				icon(c, "copy", 11, p.fg2)
			}) {
				a.later(func() {
					a.writeClipboard(text)
					a.toastOK(a.t("已复制", "Copied"))
				})
			}
			ui.Spacer(c)
			meta := streamTimeLabel(e.Timestamp)
			if e.Event != "" {
				meta = "event: " + e.Event + "  " + meta
			}
			if e.ID != "" {
				meta = "id: " + e.ID + "  " + meta
			}
			txtMono(c, meta, fsTiny, p.muted, 0)
		})
		a.codeView(c, ed).Height(240).FillWidth()
	})
}
