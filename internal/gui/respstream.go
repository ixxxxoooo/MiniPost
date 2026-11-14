package gui

import (
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
}

// streamEntries 返回要展示的消息：去掉 response_start，只留最近 maxStreamRows 条，新消息在前。
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
	for i, j := 0, len(kept)-1; i < j; i, j = i+1, j-1 {
		kept[i], kept[j] = kept[j], kept[i]
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

func (a *app) streamSummary(e model.StreamChunk) string {
	switch e.Kind {
	case "connection_closed":
		return a.t("连接已关闭", "Connection closed")
	case "error":
		if e.Data != "" {
			return e.Data
		}
		return a.t("流式错误", "Stream error")
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

// streamView 绘制 Stream 标签：搜索栏 + 消息列表，点击消息展开详情（JSON/文本、换行、复制）。
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
	entries := filterStreamEntries(all, v.search)
	ui.Column(c).Grow(1).Basis(0).Radius(10).Background(p.surface).Border(1, p.border).Children(func() {
		ui.Column(c).FillWidth().Shrink(0).Padding(8).BorderWidth(0, 0, 1, 0).BorderColor(p.border).Children(func() {
			if len(rs.chunks)-1 > maxStreamRows {
				txt(c, a.t("仅展示最近流消息，避免页面卡顿。", "Showing recent stream messages only to keep UI responsive."), fsTiny, p.muted, 0).Margin(0, 0, 4, 0)
			}
			ui.Row(c).FillWidth().Gap(4).AlignItems(ui.Center).Children(func() {
				a.input(c, &v.search, a.t("搜索", "Search"), fieldOpts{h: 28, radius: 7, fs: fsBase, icon: "search", key: "stream-search"})
				if v.search != "" {
					if a.iconButton(c, "clear", a.t("清空", "Clear"), 12, p.muted) {
						v.search = ""
					}
				}
			})
		})
		ui.Scroll(c).Grow(1).Basis(0).Scrollbars(a.scrollbarsVisible()).Children(func() {
			ui.Column(c).FillWidth().Children(func() {
				for i, e := range entries {
					a.streamRow(c, rs, e, i)
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

// streamRow 绘制一条消息：摘要行与（展开时的）详情区。
func (a *app) streamRow(c *ui.Context, rs *respState, e model.StreamChunk, index int) {
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
				if closable {
					icon(c, "info", 10, p.muted)
				} else {
					icon(c, "download", 10, ui.RGB(47, 111, 211))
				}
			})
			summary := a.streamSummary(e)
			if summary == "" {
				summary = a.t("(空消息块)", "(empty chunk)")
			}
			txtEMono(c, summary, fsSmall, p.fg, 0).Grow(1).Basis(0)
			ui.Row(c).Gap(4).AlignItems(ui.Center).Shrink(0).Children(func() {
				txtMono(c, streamTimeLabel(e.Timestamp), fsSmall, p.muted, 0)
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
					a.toast(a.t("已复制", "Copied"))
				})
			}
		})
		a.codeView(c, ed).Height(240).FillWidth()
	})
}
