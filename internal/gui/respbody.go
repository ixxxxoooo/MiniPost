package gui

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	"minipost/internal/client"
	"minipost/internal/model"
	"minipost/internal/pkg/logger"
)

// bodyDisplayLimit 是响应体在代码视图里最多展示的字节数；更大的响应只展示开头，
// 完整内容可复制或保存到文件，避免超大文本拖垮编辑器。
const bodyDisplayLimit = 1 << 20

// truncateForDisplay 把 s 截到不超过 limit 字节（落在合法的 UTF-8 边界上），并报告是否发生了截断。
func truncateForDisplay(s string, limit int) (string, bool) {
	if len(s) <= limit {
		return s, false
	}
	cut := limit
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut], true
}

// copiedLinger 是复制成功后图标保持强调色的时长（同旧版 1.2s）。
const copiedLinger = 1200 * time.Millisecond

// bodyViewState 是响应体区域的展示状态：格式、预览、过滤、下载卡片以及由它们派生出的缓存。
// 收到新响应时整体重置（对应旧版 ResponseBody 的 effect）。
type bodyViewState struct {
	synced *model.HttpResponse // 当前状态对应的响应，变化时重置

	mode       string // 用户选择的格式，空表示自动识别
	preview    bool
	filterOpen bool
	filterExpr string
	download   bool // 显示「下载文件」卡片而非文本
	copiedAt   time.Time
	truncated  bool  // 展示内容被截断（超过 bodyDisplayLimit）
	path       []any // JSON 预览的钻取路径

	// 以下为派生缓存
	raw       string
	effMode   string
	display   string
	filterErr error
	filename  string
	isImage   bool
	image     ui.ImageSource
	loadKey   string
	root      *client.JSONNode
}

// responseRawText 取响应体文本；二进制内容按 UTF-8 宽松解码（用于「文本查看」）。
func responseRawText(resp *model.HttpResponse) string {
	if resp.BodyIsBinary && resp.BodyBase64 != "" {
		if data, err := base64.StdEncoding.DecodeString(resp.BodyBase64); err == nil {
			return strings.ToValidUTF8(string(data), "\uFFFD")
		}
	}
	return resp.Body
}

// responseFilename 是保存响应时的建议文件名（Content-Disposition > URL 末段 > response）。
func (a *app) responseFilename(resp *model.HttpResponse, rawURL string) string {
	u := client.EnsureProtocol(client.ResolveTemplate(rawURL, a.vars()))
	return client.SuggestResponseFilename(resp.Headers, resp.ContentType, u, "response")
}

// syncBody 在响应、格式或过滤条件变化时重新计算展示文本并载入只读编辑器。
func (a *app) syncBody(rs *respState, t *tab) {
	v, resp := &rs.view, rs.resp
	if v.synced != resp {
		*v = bodyViewState{synced: resp}
		v.raw = responseRawText(resp)
		if resp.BodyIsBinary && strings.HasPrefix(strings.ToLower(resp.ContentType), "image/") {
			if data, err := base64.StdEncoding.DecodeString(resp.BodyBase64); err == nil {
				if img, err := ui.DecodeImage(data); err == nil {
					v.isImage, v.image = true, img
				}
			}
		}
		v.download = !v.isImage && client.PreferDownload(resp.Headers, resp.ContentType, resp.BodyIsBinary)
		v.filename = a.responseFilename(resp, t.draft.URL)
	}
	mode := v.mode
	if mode == "" {
		mode = client.DetectBodyMode(resp.ContentType, v.raw, a.set.ResponseFormatDetection)
	}
	filterOn := v.filterOpen && strings.TrimSpace(v.filterExpr) != ""
	key := mode + "|" + v.filterExpr
	if filterOn {
		key += "|f"
	}
	if key == v.loadKey {
		return
	}
	v.loadKey, v.effMode = key, mode
	formatted := client.FormatBody(mode, v.raw)
	v.display, v.filterErr = formatted, nil
	if filterOn {
		v.display, v.filterErr = client.FilterBody(mode, v.raw, formatted, v.filterExpr)
	}
	v.root = nil
	if mode == client.ModeJSON {
		if n, err := client.ParseJSONNode(v.raw); err == nil {
			v.root = n
		}
	}
	text := v.display
	if text == "" {
		text = a.t("(空响应)", "(empty response)")
	}
	text, v.truncated = truncateForDisplay(text, bodyDisplayLimit)
	if v.truncated {
		logger.Info("响应体过大，视图只展示开头", "total", len(v.display), "shown", len(text))
	}
	rs.bodyEd.Buffer.Reset(text)
	rs.bodyEd.Buffer.SetLanguage(editor.LanguageFor(client.BodyLanguageFile(mode)))
}

// bodyView 绘制响应体：工具栏、过滤栏与内容区（代码视图 / 预览 / 下载卡片 / 图片）。
func (a *app) bodyView(c *ui.Context, t *tab, rs *respState) {
	p := a.p
	if rs.resp == nil {
		ui.Row(c).FillWidth().Padding(24, 0).Justify(ui.Center).Children(func() {
			txt(c, a.t("响应尚未结束，请查看 Stream 标签", "Response is still streaming — see the Stream tab"), 10.4, p.muted, 0)
		})
		return
	}
	a.syncBody(rs, t)
	v := &rs.view
	canPreview := client.CanPreview(v.effMode, rs.resp.ContentType, v.raw)
	ui.Column(c).Grow(1).Basis(0).Padding(8).Children(func() {
		a.bodyToolbar(c, rs, canPreview)
		if v.filterOpen && !v.download && !v.isImage {
			a.filterBar(c, rs)
		}
		if v.truncated && !v.download && !v.isImage {
			a.truncatedBanner(c, rs)
		}
		switch {
		case v.isImage:
			ui.Column(c).Grow(1).Basis(0).Gap(10).Padding(8, 12).Children(func() {
				txt(c, a.t("二进制内容", "Binary content")+" · "+formatSize(rs.resp.Size), fsBase, p.muted, 0)
				ui.Scroll(c).Grow(1).Scrollbars(a.scrollbarsVisible()).Children(func() { ui.Image(c, v.image).Fit(ui.Contain) })
			})
		case v.download:
			a.downloadCard(c, rs)
		case v.preview && canPreview:
			a.bodyPreview(c, rs)
		default:
			a.codeView(c, rs.bodyEd).Grow(1).Basis(0).FillWidth()
		}
	})
}

// modeGlyph 是格式下拉里每种格式前的小图标文字。
func modeGlyph(mode string) string {
	switch mode {
	case client.ModeJSON:
		return "{}"
	case client.ModeXML, client.ModeHTML:
		return "</>"
	case client.ModeJavaScript:
		return "JS"
	case client.ModeHex:
		return "0x"
	case client.ModeBase64:
		return "64"
	}
	return "T"
}

// modeLabel 是格式名称：Raw/Hex 有中文名，其余沿用大写缩写。
func (a *app) modeLabel(mode string) string {
	switch mode {
	case client.ModeRaw:
		return a.t("原始", "Raw")
	case client.ModeHex:
		return a.t("十六进制", "Hex")
	case client.ModeJavaScript:
		return "JavaScript"
	}
	return strings.ToUpper(mode)
}

// bodyToolbar 是响应体上方 24px 的工具栏：左侧格式与预览，右侧换行、过滤、搜索、折叠、复制、下载。
func (a *app) bodyToolbar(c *ui.Context, rs *respState, canPreview bool) {
	p := a.p
	resp, v := rs.resp, &rs.view
	preferDownload := client.PreferDownload(resp.Headers, resp.ContentType, resp.BodyIsBinary)
	sep := func() { ui.Box(c).Width(1).Height(14).Background(p.border).Shrink(0) }
	ui.Row(c).FillWidth().Shrink(0).Padding(6, 12).Gap(6).AlignItems(ui.Center).Children(func() {
		switch {
		case v.isImage:
			ui.Row(c).Height(24).Radius(8).PaddingX(6).Gap(4).AlignItems(ui.Center).Background(p.selected).Shrink(0).Children(func() {
				txt(c, "Image", fsTiny, p.fg, 0)
			})
		case v.download:
			ui.Row(c).Height(24).Radius(8).PaddingX(8).Gap(4).AlignItems(ui.Center).Background(p.selected).Shrink(1).Children(func() {
				icon(c, "download", 10, p.fg2)
				txtE(c, v.filename, fsTiny, p.fg2, 0)
			})
		default:
			a.formatDropdown(c, rs)
			prevOn := v.preview && canPreview
			prevCol, eyeCol := p.fg2, p.success
			if prevOn {
				prevCol, eyeCol = p.accent, p.accent
			}
			if a.button(c, btnOpts{h: 26, radius: 8, padX: 8, gap: 5, active: prevOn, disabled: !canPreview, tip: a.t("预览", "Preview"), key: "preview"}, func(bool) {
				icon(c, "eye", 13, eyeCol)
				txt(c, a.t("预览", "Preview"), fsBase, prevCol, 0)
			}) {
				v.preview = true
			}
		}
		ui.Spacer(c)
		tb := func(name, tip string, active bool, key string) bool {
			col := p.fg2
			if active {
				col = p.accent
			}
			return a.button(c, btnOpts{w: 26, h: 26, radius: 8, active: active, tip: tip, key: key}, func(bool) { icon(c, name, 14, col) })
		}
		switch {
		case v.isImage:
			// 图片只提供下载
		case v.download:
			if a.button(c, btnOpts{h: 24, radius: 8, padX: 8, gap: 4, bg: &p.accent, hoverBg: &p.accentHover, key: "dl-file"}, func(bool) {
				icon(c, "download", 11, p.accentFg)
				txt(c, a.t("下载文件", "Download file"), fsTiny, p.accentFg, 0)
			}) {
				a.later(func() { a.saveResponse(resp, v.filename) })
			}
			if a.button(c, btnOpts{h: 24, radius: 8, padX: 8, key: "dl-text"}, func(bool) {
				txt(c, a.t("文本查看", "View as text"), fsTiny, p.fg2, 0)
			}) {
				v.download = false
			}
			return
		default:
			if tb("wrapText", a.t("自动换行", "Line wrap"), rs.bodyEd.SoftWrap, "tb-wrap") {
				rs.bodyEd.SoftWrap = !rs.bodyEd.SoftWrap
			}
			if tb("formatList", a.t("过滤", "Filter"), v.filterOpen, "tb-filter") {
				v.filterOpen = !v.filterOpen
				if !v.filterOpen {
					v.filterExpr = ""
				}
			}
			if tb("search", a.t("查找 (⌘F)", "Search (⌘F)"), false, "tb-find") && !v.preview {
				rs.bodyEd.Find("")
			}
			if v.effMode == client.ModeJSON && !v.preview {
				if tb("arrowRight", a.t("一键折叠所有 JSON 节点", "Collapse all JSON nodes"), false, "tb-fold") {
					n := rs.bodyEd.Buffer.LineCount()
					for i := 0; i < min(n, 2000); i++ {
						rs.bodyEd.Fold(i)
					}
				}
				if tb("arrowDown", a.t("一键展开所有 JSON 节点", "Expand all JSON nodes"), false, "tb-unfold") {
					for _, l := range rs.bodyEd.Folded() {
						rs.bodyEd.Unfold(l)
					}
				}
			}
			sep()
			copied := time.Since(v.copiedAt) < copiedLinger
			tip := a.t("复制响应体", "Copy response body")
			if copied {
				tip = a.t("已复制", "Copied")
			}
			if tb2 := a.button(c, btnOpts{w: 26, h: 26, radius: 8, tip: tip, key: "tb-copy"}, func(bool) {
				col := p.fg2
				if copied {
					col = p.accent
				}
				icon(c, "copy", 14, col)
			}); tb2 {
				a.later(func() { a.copyBody(rs) })
			}
		}
		// 任何文本响应都可以保存为文件（二进制响应在上面的下载卡片里处理）
		tip := a.t("保存到文件", "Save to file")
		if preferDownload {
			tip = a.t("下载文件", "Download file")
		}
		if tb("download", tip, false, "tb-download") {
			a.later(func() { a.saveResponse(resp, v.filename) })
		}
	})
}

// truncatedBanner 提示响应体过大、视图只展示了开头，并给出保存完整响应的入口。
func (a *app) truncatedBanner(c *ui.Context, rs *respState) {
	p := a.p
	resp, v := rs.resp, &rs.view
	ui.Row(c.Key("trunc-banner")).FillWidth().Shrink(0).Margin(0, 12, 4, 12).Padding(5, 10).Radius(8).Gap(8).AlignItems(ui.Center).
		Background(p.warning.Alpha(0.12)).Children(func() {
		icon(c, "info", 12, p.warning)
		txtE(c, a.t("响应较大（", "Large response (")+formatSize(resp.Size)+a.t("），仅显示前 ", "), showing only the first ")+formatSize(int64(bodyDisplayLimit))+
			a.t("。复制或保存可获取完整内容。", ". Copy or save to get everything."), fsTiny, p.fg2, 0).Grow(1)
		if a.button(c, btnOpts{h: 22, radius: 6, padX: 8, key: "trunc-save"}, func(bool) {
			txt(c, a.t("保存完整响应", "Save full response"), fsTiny, p.accent, 0)
		}) {
			a.later(func() { a.saveResponse(resp, v.filename) })
		}
	})
}

// formatDropdown 是左侧的格式下拉：预览中点击它会回到代码视图。
func (a *app) formatDropdown(c *ui.Context, rs *respState) {
	p := a.p
	v := &rs.view
	body := func() {
		txtMono(c, modeGlyph(v.effMode), fsSmall, p.accent, 600)
		txt(c, a.modeLabel(v.effMode), fsBase, p.fg, 500)
		icon(c, "arrowDown", 9, p.muted)
	}
	b := ui.ButtonBase(c.Key("fmt")).Height(26).Radius(8).PaddingX(8).Gap(5).AlignItems(ui.Center).Shrink(0)
	switch {
	case !v.preview:
		b = b.Background(p.accent.Alpha(0.13))
	case b.Hovered():
		b = b.Background(p.hover)
	}
	if v.preview {
		b.Children(body)
		if b.Clicked() {
			v.preview = false
		}
		return
	}
	b.PopoverMenuButton(func(m *ui.Menu) {
		for i, mode := range client.BodyModes {
			if i == 4 {
				m.Separator()
			}
			if m.Item(modeGlyph(mode) + "  " + a.modeLabel(mode)).Checked(mode == v.effMode).Chosen() {
				a.later(func() { v.mode = mode })
			}
		}
	}).Children(body)
}

// copyBody 把当前展示的内容（含格式化与过滤结果）复制到剪贴板。
func (a *app) copyBody(rs *respState) {
	a.writeClipboard(rs.view.display)
	rs.view.copiedAt = time.Now()
	if a.native && a.win != nil {
		time.AfterFunc(copiedLinger+50*time.Millisecond, a.win.Invalidate)
	}
	a.toastOK(a.t("已复制响应体", "Response body copied"))
	logger.Info("已复制响应体", "bytes", len(rs.view.display))
}

// filterErrText 把过滤错误转成本地化提示。
func (a *app) filterErrText(err error) string {
	switch {
	case errors.Is(err, client.ErrFilterPath):
		return a.t("JSONPath 表达式无效（示例：$.headers.host）", "Invalid JSONPath expression (example: $.headers.host)")
	case errors.Is(err, client.ErrFilterEmpty):
		return a.t("过滤结果为空", "No results after filtering")
	case errors.Is(err, client.ErrFilterParse):
		return a.t("JSON 解析失败，无法按路径过滤", "JSON parse failed, cannot filter by path")
	}
	return err.Error()
}

// filterBar 是工具栏下方的过滤输入行：JSON 按 JSONPath，其余按关键字过滤行。
func (a *app) filterBar(c *ui.Context, rs *respState) {
	p := a.p
	v := &rs.view
	ph := a.t("输入关键字过滤行", "Filter lines by keyword")
	if v.effMode == client.ModeJSON {
		ph = a.t("JSONPath 过滤（如 $.headers.host）", "JSONPath filter (e.g. $.headers.host)")
	}
	ui.Column(c).FillWidth().Shrink(0).Padding(0, 12, 8, 12).Children(func() {
		ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Children(func() {
			a.input(c, &v.filterExpr, ph, fieldOpts{h: 28, radius: 8, fs: fsSmall, padX: 10, bd: &p.btnBorder, key: "resp-filter", focus: true})
			if v.filterExpr != "" {
				if a.button(c, btnOpts{h: 28, radius: 8, padX: 8, border: &p.btnBorder, key: "resp-filter-clear"}, func(bool) {
					txt(c, a.t("清空", "Clear"), fsSmall, p.fg2, 0)
				}) {
					v.filterExpr = ""
				}
			}
		})
		if v.filterErr != nil && strings.TrimSpace(v.filterExpr) != "" {
			txt(c, a.filterErrText(v.filterErr), fsTiny, p.warning, 0).Margin(4, 0, 0, 0)
		}
	})
}

// downloadCard 是二进制 / attachment 响应的居中下载卡片。
func (a *app) downloadCard(c *ui.Context, rs *respState) {
	p := a.p
	v, resp := &rs.view, rs.resp
	ui.Row(c).Grow(1).Basis(0).FillWidth().AlignItems(ui.Center).Justify(ui.Center).Padding(0, 16).Children(func() {
		ui.Column(c).FillWidth().MaxWidth(520).Padding(16).Radius(12).Background(p.elevated()).Border(1, p.border).Children(func() {
			ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Padding(0, 0, 12, 0).Children(func() {
				ui.Row(c).Size(32, 32).Radius(8).Background(p.selected).AlignItems(ui.Center).Justify(ui.Center).Shrink(0).Children(func() {
					icon(c, "download", 14, p.accent)
				})
				ui.Column(c).Grow(1).Basis(0).Children(func() {
					txtE(c, v.filename, fsLarge, p.fg, 600)
					ct := resp.ContentType
					if ct == "" {
						ct = a.t("二进制响应", "Binary response")
					}
					txtE(c, ct, fsSmall, p.muted, 0)
				})
			})
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				if a.button(c, btnOpts{h: 32, radius: 8, padX: 12, gap: 6, bg: &p.accent, hoverBg: &p.accentHover, key: "card-dl"}, func(bool) {
					icon(c, "download", 12, p.accentFg)
					txt(c, a.t("下载文件", "Download file"), fsBase, p.accentFg, 0)
				}) {
					a.later(func() { a.saveResponse(resp, v.filename) })
				}
				if a.button(c, btnOpts{h: 32, radius: 8, padX: 12, key: "card-text"}, func(bool) {
					txt(c, a.t("文本查看", "View as text"), fsBase, p.fg2, 0)
				}) {
					v.download = false
				}
			})
		})
	})
}

// bodyPreview 绘制预览：JSON 为可逐层钻取的结构化表格，其余格式显示整理后的文本
// （原生界面无法渲染 HTML，因此 HTML 预览展示排版后的源码）。
func (a *app) bodyPreview(c *ui.Context, rs *respState) {
	p := a.p
	v := &rs.view
	if v.effMode == client.ModeJSON && v.root != nil {
		a.jsonPreview(c, v)
		return
	}
	text := client.FormatBody(v.effMode, v.raw)
	ui.Scroll(c).Grow(1).Basis(0).Scrollbars(a.scrollbarsVisible()).Children(func() {
		ui.Text(c, text).FontSize(fsBase).TextColor(p.fg).Selectable().Padding(16).FillWidth()
	})
}

// jsonPreview 是结构化 JSON 预览：顶部面包屑 + 键/值表格，点击对象或数组行向下钻取。
func (a *app) jsonPreview(c *ui.Context, v *bodyViewState) {
	p := a.p
	cur, ok := v.root.Resolve(v.path)
	if !ok {
		v.path, cur = nil, v.root
	}
	label := func(tok any) string {
		if n, isInt := tok.(int); isInt {
			return "[" + itoa(n) + "]"
		}
		return tok.(string)
	}
	ui.Column(c).Grow(1).Basis(0).Border(1, p.border).Background(p.surface).Children(func() {
		// 面包屑
		ui.Row(c).FillWidth().Height(28).Shrink(0).PaddingX(8).Gap(4).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(p.border).Children(func() {
			segs := append([]any{nil}, v.path...)
			for i := range segs {
				name := a.t("根节点", "Root")
				if i > 0 {
					name = label(segs[i])
				}
				if i == len(segs)-1 {
					txt(c, name, fsBase, p.fg, 600)
					continue
				}
				depth := i
				if a.button(c, btnOpts{padX: 0, h: 20, key: "pv-seg" + itoa(i)}, func(hover bool) {
					col := p.fg2
					if hover {
						col = p.accent
					}
					txt(c, name, fsBase, col, 0)
				}) {
					a.later(func() { v.path = append([]any(nil), v.path[:depth]...) })
				}
				icon(c, "arrowRight", 9, p.muted)
			}
		})
		ui.Scroll(c).Grow(1).Basis(0).Scrollbars(a.scrollbarsVisible()).Children(func() {
			ui.Column(c).FillWidth().Children(func() {
				row := func(key string, tok any, n *client.JSONNode, idx int) {
					complex := n.Kind == client.JSONArray || n.Kind == client.JSONObject
					r := ui.Row(c.Key("pv"+itoa(idx))).FillWidth().BorderWidth(0, 0, 1, 0).BorderColor(p.border)
					r.Children(func() {
						kc := ui.Row(c).Grow(32).Basis(0).Padding(6, 12).BorderWidth(0, 1, 0, 0).BorderColor(p.border)
						kc.Children(func() {
							if complex {
								b := ui.ButtonBase(c.Key("pvk" + itoa(idx))).Gap(4).AlignItems(ui.Center)
								col := p.fg
								if b.Hovered() {
									col = p.accent
								}
								b.Children(func() {
									icon(c, "arrowRight", 10, p.muted)
									txt(c, key, fsBase, col, 600)
								})
								if b.Clicked() {
									a.later(func() { v.path = append(append([]any(nil), v.path...), tok) })
								}
								return
							}
							txt(c, key, fsBase, p.fg, 600)
						})
						vc := ui.Row(c).Grow(68).Basis(0).Padding(6, 12)
						vc.Children(func() {
							if complex {
								b := ui.ButtonBase(c.Key("pvv" + itoa(idx)))
								col := p.fg2
								if b.Hovered() {
									col = p.accent
								}
								b.Children(func() { txtMono(c, n.Badge(), fsBase, col, 0) })
								if b.Clicked() {
									a.later(func() { v.path = append(append([]any(nil), v.path...), tok) })
								}
								return
							}
							previewScalar(c, p, n)
						})
					})
				}
				switch cur.Kind {
				case client.JSONArray:
					for i, k := range cur.Kids {
						row("["+itoa(i)+"]", i, k, i)
					}
				case client.JSONObject:
					for i, k := range cur.Kids {
						row(cur.Keys[i], cur.Keys[i], k, i)
					}
				}
				if (cur.Kind == client.JSONArray || cur.Kind == client.JSONObject) && len(cur.Kids) > 0 {
					return
				}
				// 空容器或标量：只有一行「值」
				ui.Row(c).FillWidth().Children(func() {
					ui.Row(c).Grow(32).Basis(0).Padding(6, 12).BorderWidth(0, 1, 0, 0).BorderColor(p.border).Children(func() {
						txt(c, a.t("值", "value"), fsBase, p.fg, 600)
					})
					ui.Row(c).Grow(68).Basis(0).Padding(6, 12).Children(func() {
						if cur.Kind == client.JSONArray || cur.Kind == client.JSONObject {
							txtMono(c, cur.Badge(), fsBase, p.fg2, 0)
							return
						}
						previewScalar(c, p, cur)
					})
				})
			})
		})
	})
}

// previewScalar 绘制标量值：字符串用次要色，数字/布尔/null 用等宽主色；长内容换行。
func previewScalar(c *ui.Context, p *palette, n *client.JSONNode) {
	e := ui.Text(c, n.Scalar()).FontSize(fsBase).Selectable().Grow(1).Basis(0)
	if n.Kind == client.JSONString {
		e.TextColor(p.fg2)
		return
	}
	e.Font(monoFont).TextColor(p.fg)
}

// saveResponse 保存响应体到文件（二进制内容以 base64 前缀传递给后端）。
func (a *app) saveResponse(resp *model.HttpResponse, filename string) {
	content := resp.Body
	if resp.BodyIsBinary {
		content = model.BinarySavePrefix + resp.BodyBase64
	}
	if filename == "" {
		filename = "response"
	}
	if !utf8.ValidString(filename) {
		filename = "response"
	}
	logger.Info("保存响应体", "filename", filename, "binary", resp.BodyIsBinary, "bytes", resp.Size)
	a.async(func() func() {
		err := a.be.SaveResponseToFile(a.win, filename, content)
		return func() { a.fail(a.t("保存响应", "Save response"), err) }
	})
}
