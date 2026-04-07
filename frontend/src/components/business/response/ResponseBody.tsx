import { useState, useMemo, useEffect, useLayoutEffect, useRef, useCallback } from "react"
import { createPortal } from "react-dom"
import { cn } from "@/lib/utils"
import { CodeEditor, type EditorLanguage } from "@/components/ui/CodeEditor"
import { AppIcon } from "@/components/ui/icon"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { useUIStore } from "@/stores/uiStore"
import { useI18n } from "@/hooks/useI18n"
import {
  buildSaveResponsePayload,
  decodeResponseBodyToText,
  shouldPreferDownload,
  suggestResponseFilename,
} from "@/lib/responseDownload"

interface ResponseBodyProps {
  body: string
  bodyBase64?: string
  bodyIsBinary?: boolean
  headers: Record<string, string[]>
  contentType: string
  requestUrl?: string
  isDark: boolean
}

type DisplayMode = "json" | "xml" | "html" | "javascript" | "raw" | "hex" | "base64"

interface DisplayOption {
  value: DisplayMode
  label: string
  group: "structured" | "encoded"
  glyph: string
}

const DISPLAY_OPTION_META: Array<Omit<DisplayOption, "label">> = [
  { value: "json", group: "structured", glyph: "{}" },
  { value: "xml", group: "structured", glyph: "</>" },
  { value: "html", group: "structured", glyph: "</>" },
  { value: "javascript", group: "structured", glyph: "JS" },
  { value: "raw", group: "encoded", glyph: "T" },
  { value: "hex", group: "encoded", glyph: "0x" },
  { value: "base64", group: "encoded", glyph: "64" },
]

function getDisplayOptionLabel(value: DisplayMode, isZh: boolean): string {
  if (value === "raw") return isZh ? "原始" : "Raw"
  if (value === "hex") return isZh ? "十六进制" : "Hex"
  return value === "javascript" ? "JavaScript" : value.toUpperCase()
}

function getDisplayOptions(isZh: boolean): DisplayOption[] {
  return DISPLAY_OPTION_META.map((option) => ({
    ...option,
    label: getDisplayOptionLabel(option.value, isZh),
  }))
}

function looksLikeJson(text: string): boolean {
  const trimmed = text.trimStart()
  if (!trimmed.startsWith("{") && !trimmed.startsWith("[")) return false
  try {
    JSON.parse(text)
    return true
  } catch {
    return false
  }
}

function looksLikeHtml(text: string): boolean {
  const trimmed = text.trimStart().toLowerCase()
  return trimmed.startsWith("<!doctype html") || trimmed.startsWith("<html")
}

function looksLikeXml(text: string): boolean {
  const trimmed = text.trimStart()
  return trimmed.startsWith("<?xml") || (trimmed.startsWith("<") && trimmed.includes(">"))
}

function looksLikeJs(text: string): boolean {
  const trimmed = text.trimStart()
  return /^(var|let|const|function|class|import|export|return)\s/.test(trimmed)
    || /^\(function/.test(trimmed)
    || /^!function/.test(trimmed)
}

function detectDefaultMode(contentType: string, body: string, responseFormatDetection: "auto" | "json"): DisplayMode {
  if (responseFormatDetection === "json") {
    return "json"
  }
  const normalized = contentType.toLowerCase()
  if (normalized.includes("json")) return "json"
  if (normalized.includes("html")) return "html"
  if (normalized.includes("xml") || normalized.includes("svg")) return "xml"
  if (normalized.includes("javascript") || normalized.includes("ecmascript")) return "javascript"

  if (looksLikeJson(body)) return "json"
  if (looksLikeHtml(body)) return "html"
  if (looksLikeXml(body)) return "xml"
  if (looksLikeJs(body)) return "javascript"
  return "raw"
}

function toEditorLanguage(mode: DisplayMode): EditorLanguage {
  if (mode === "json") return "json"
  if (mode === "xml" || mode === "html") return "xml"
  if (mode === "javascript") return "javascript"
  return "text"
}

function toHex(text: string): string {
  const bytes = new TextEncoder().encode(text)
  return Array.from(bytes).map((b) => b.toString(16).padStart(2, "0")).join(" ")
}

function toBase64(text: string): string {
  const bytes = new TextEncoder().encode(text)
  let binary = ""
  bytes.forEach((b) => {
    binary += String.fromCharCode(b)
  })
  return btoa(binary)
}

function tryFormatJson(text: string): string {
  try {
    return JSON.stringify(JSON.parse(text), null, 2)
  } catch {
    // 支持 NDJSON / JSON Lines：每行一个独立 JSON 对象
    const lines = text.split(/\r?\n/).map((line) => line.trim()).filter(Boolean)
    if (lines.length === 0) return text

    const prettyLines: string[] = []
    for (const line of lines) {
      try {
        prettyLines.push(JSON.stringify(JSON.parse(line), null, 2))
      } catch {
        return text
      }
    }

    return prettyLines.join("\n")
  }
}

function tryFormatXml(text: string): string {
  let formatted = ""
  let indent = 0
  const lines = text.replace(/>\s*</g, ">\n<").split("\n")
  for (const line of lines) {
    const trimmed = line.trim()
    if (!trimmed) continue
    if (trimmed.startsWith("</")) indent = Math.max(0, indent - 1)
    formatted += "  ".repeat(indent) + trimmed + "\n"
    if (
      trimmed.startsWith("<")
      && !trimmed.startsWith("</")
      && !trimmed.startsWith("<?")
      && !trimmed.endsWith("/>")
      && !trimmed.includes("</")
    ) {
      indent++
    }
  }
  return formatted.trimEnd()
}

function tryFormatJs(text: string): string {
  let result = ""
  let indent = 0
  let inString: string | null = null
  let i = 0
  while (i < text.length) {
    const ch = text[i]
    if (inString) {
      result += ch
      if (ch === "\\" && i + 1 < text.length) {
        result += text[i + 1]
        i += 2
        continue
      }
      if (ch === inString) inString = null
      i++
      continue
    }
    if (ch === "\"" || ch === "'" || ch === "`") {
      inString = ch
      result += ch
      i++
      continue
    }
    if (ch === "{" || ch === "[" || ch === "(") {
      indent++
      result += `${ch}\n${"  ".repeat(indent)}`
      i++
      continue
    }
    if (ch === "}" || ch === "]" || ch === ")") {
      indent = Math.max(0, indent - 1)
      result += `\n${"  ".repeat(indent)}${ch}`
      i++
      continue
    }
    if (ch === ";" || ch === ",") {
      result += `${ch}\n${"  ".repeat(indent)}`
      i++
      while (i < text.length && (text[i] === " " || text[i] === "\t")) i++
      continue
    }
    result += ch
    i++
  }
  return result
}

function escapeHtml(text: string): string {
  return text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
}

function buildPreviewDocument(mode: DisplayMode, rawBody: string): string {
  if (mode === "html") {
    return rawBody
  }
  const text = mode === "xml"
    ? tryFormatXml(rawBody)
    : mode === "json"
      ? tryFormatJson(rawBody)
      : mode === "javascript"
        ? tryFormatJs(rawBody)
        : rawBody
  return `<!doctype html><html><body style="margin:16px;font-family:-apple-system,BlinkMacSystemFont,'SF Pro Text','Segoe UI',sans-serif;"><pre style="white-space:pre-wrap;word-break:break-word;">${escapeHtml(text)}</pre></body></html>`
}

function canPreview(mode: DisplayMode, contentType: string, body: string): boolean {
  if (mode === "html") return true
  if (mode === "xml") return true
  if (mode === "json") return true
  if (mode === "javascript") return true
  const normalized = contentType.toLowerCase()
  return normalized.includes("html") || normalized.includes("xml") || looksLikeHtml(body) || looksLikeXml(body)
}

function parseJsonPathTokens(expression: string): Array<string | number> | null {
  const input = expression.trim()
  if (!input) return []
  if (input === "$") return []
  if (!input.startsWith("$")) return null
  const tokens: Array<string | number> = []
  let i = 1

  while (i < input.length) {
    const ch = input[i]
    if (ch === ".") {
      i += 1
      const start = i
      while (i < input.length && /[A-Za-z0-9_$-]/.test(input[i])) i += 1
      if (start === i) return null
      tokens.push(input.slice(start, i))
      continue
    }
    if (ch === "[") {
      i += 1
      if (i >= input.length) return null
      if (input[i] === "'" || input[i] === "\"") {
        const quote = input[i]
        i += 1
        const start = i
        while (i < input.length && input[i] !== quote) i += 1
        if (i >= input.length) return null
        const key = input.slice(start, i)
        i += 1
        if (input[i] !== "]") return null
        i += 1
        tokens.push(key)
        continue
      }
      const start = i
      while (i < input.length && /[0-9]/.test(input[i])) i += 1
      if (start === i) return null
      if (input[i] !== "]") return null
      const index = Number.parseInt(input.slice(start, i), 10)
      i += 1
      tokens.push(index)
      continue
    }
    return null
  }
  return tokens
}

function tryApplyJsonFilter(
  rawBody: string,
  expression: string,
  t: (zh: string, en: string) => string
): { text: string; error: string | null } {
  try {
    const parsed = JSON.parse(rawBody)
    const tokens = parseJsonPathTokens(expression)
    if (!tokens) {
      return { text: rawBody, error: t("JSONPath 表达式无效（示例：$.headers.host）", "Invalid JSONPath expression (example: $.headers.host)") }
    }
    let current: unknown = parsed
    for (const token of tokens) {
      if (typeof token === "number") {
        if (!Array.isArray(current) || token < 0 || token >= current.length) {
          return { text: rawBody, error: t("过滤结果为空", "No results after filtering") }
        }
        current = current[token]
      } else {
        if (typeof current !== "object" || current === null || !(token in (current as Record<string, unknown>))) {
          return { text: rawBody, error: t("过滤结果为空", "No results after filtering") }
        }
        current = (current as Record<string, unknown>)[token]
      }
    }
    if (typeof current === "string") return { text: current, error: null }
    if (typeof current === "number" || typeof current === "boolean" || current === null) {
      return { text: String(current), error: null }
    }
    return { text: JSON.stringify(current, null, 2), error: null }
  } catch {
    return { text: rawBody, error: t("JSON 解析失败，无法按路径过滤", "JSON parse failed, cannot filter by path") }
  }
}

function tryApplyTextFilter(
  text: string,
  expression: string,
  t: (zh: string, en: string) => string
): { text: string; error: string | null } {
  const query = expression.trim().toLowerCase()
  if (!query) return { text, error: null }
  const lines = text.split("\n").filter((line) => line.toLowerCase().includes(query))
  if (lines.length === 0) return { text, error: t("过滤结果为空", "No results after filtering") }
  return { text: lines.join("\n"), error: null }
}

function isJsonObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value)
}

type JsonNodeKind = "primitive" | "object" | "array"
type JsonPathToken = string | number

type JsonNodeRow = {
  key: string
  token: JsonPathToken
  value: unknown
  kind: JsonNodeKind
}

function jsonRowKind(value: unknown): JsonNodeKind {
  if (Array.isArray(value)) return "array"
  if (isJsonObject(value)) return "object"
  return "primitive"
}

function jsonBadge(value: unknown): string {
  if (Array.isArray(value)) return `[${value.length}]`
  if (isJsonObject(value)) return Object.keys(value).length === 0 ? "{}" : `{${Object.keys(value).length}}`
  return ""
}

function resolveJsonPath(root: unknown, tokens: JsonPathToken[]): unknown {
  let current: unknown = root
  for (const token of tokens) {
    if (typeof token === "number") {
      if (!Array.isArray(current) || token < 0 || token >= current.length) return undefined
      current = current[token]
      continue
    }
    if (!isJsonObject(current) || !(token in current)) return undefined
    current = current[token]
  }
  return current
}

function buildJsonNodeRows(node: unknown): JsonNodeRow[] {
  if (Array.isArray(node)) {
    return node.map((value, index) => ({
      key: `[${index}]`,
      token: index,
      value,
      kind: jsonRowKind(value),
    }))
  }
  if (isJsonObject(node)) {
    return Object.entries(node).map(([key, value]) => ({
      key,
      token: key,
      value,
      kind: jsonRowKind(value),
    }))
  }
  return []
}

function buildPathSegments(tokens: JsonPathToken[], rootLabel: string): Array<{ label: string; tokens: JsonPathToken[] }> {
  const segments: Array<{ label: string; tokens: JsonPathToken[] }> = [{ label: rootLabel, tokens: [] }]
  const currentTokens: JsonPathToken[] = []
  tokens.forEach((token) => {
    currentTokens.push(token)
    segments.push({
      label: typeof token === "number" ? `[${token}]` : token,
      tokens: [...currentTokens],
    })
  })
  return segments
}

function formatPreviewPrimitive(value: unknown): string {
  if (value === null) return "null"
  if (typeof value === "string") return value
  if (typeof value === "number" || typeof value === "boolean") return String(value)
  return ""
}

function JsonStructuredPreview({
  value,
  rootLabel,
  valueLabel,
}: {
  value: unknown
  rootLabel: string
  valueLabel: string
}) {
  const [tokens, setTokens] = useState<JsonPathToken[]>([])
  const currentNode = useMemo(() => resolveJsonPath(value, tokens), [tokens, value])
  const rows = useMemo(() => buildJsonNodeRows(currentNode), [currentNode])
  const segments = useMemo(() => buildPathSegments(tokens, rootLabel), [rootLabel, tokens])

  useEffect(() => {
    setTokens([])
  }, [value])

  return (
    <div className="h-full overflow-auto border border-[var(--border-color)] bg-[var(--surface)]">
      <div className="flex h-7 items-center gap-1 border-b border-[var(--border-subtle)] px-2 text-[12px]">
        {segments.map((segment, index) => {
          const isLast = index === segments.length - 1
          return (
            <div key={`${segment.label}-${index}`} className="flex items-center gap-1">
              {isLast ? (
                <span className="font-semibold text-[var(--fg)]">{segment.label}</span>
              ) : (
                <button
                  type="button"
                  className="text-[var(--fg-secondary)] hover:text-[var(--accent)] transition-colors"
                  onClick={() => setTokens(segment.tokens)}
                >
                  {segment.label}
                </button>
              )}
              {!isLast && <AppIcon name="arrowRight" size={9} className="text-[var(--fg-muted)]" />}
            </div>
          )
        })}
      </div>

      <table className="w-full border-collapse text-[12px]">
        <tbody>
          {rows.length > 0 ? rows.map((row) => {
            const complex = row.kind !== "primitive"
            return (
              <tr key={`${row.key}-${String(row.token)}`} className="border-b border-[var(--border-subtle)] last:border-b-0">
                <td className="w-[32%] border-r border-[var(--border-subtle)] px-3 py-1.5 align-top">
                  {complex ? (
                    <button
                      type="button"
                      className="flex items-center gap-1 text-left text-[var(--fg)] hover:text-[var(--accent)] transition-colors"
                      onClick={() => setTokens((prev) => [...prev, row.token])}
                    >
                      <AppIcon name="arrowRight" size={10} className="text-[var(--fg-muted)]" />
                      <span className="font-semibold">{row.key}</span>
                    </button>
                  ) : (
                    <span className="font-semibold text-[var(--fg)]">{row.key}</span>
                  )}
                </td>
                <td className="px-3 py-1.5 align-top">
                  {complex ? (
                    <button
                      type="button"
                      className="font-mono text-[var(--fg-secondary)] hover:text-[var(--accent)] transition-colors"
                      onClick={() => setTokens((prev) => [...prev, row.token])}
                    >
                      {jsonBadge(row.value)}
                    </button>
                  ) : (
                    <span className={cn(
                      "break-all",
                      typeof row.value === "string" ? "text-[var(--fg-secondary)]" : "font-mono text-[var(--fg)]"
                    )}>
                      {formatPreviewPrimitive(row.value)}
                    </span>
                  )}
                </td>
              </tr>
            )
          }) : (
            <tr>
              <td className="w-[32%] border-r border-[var(--border-subtle)] px-3 py-1.5 align-top font-semibold text-[var(--fg)]">{valueLabel}</td>
              <td className="px-3 py-1.5 align-top">
                <span className={cn(
                  "break-all",
                  typeof currentNode === "string" ? "text-[var(--fg-secondary)]" : "font-mono text-[var(--fg)]"
                )}>
                  {formatPreviewPrimitive(currentNode)}
                </span>
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  )
}

function getDisplayLabel(mode: DisplayMode, isZh: boolean): string {
  return getDisplayOptions(isZh).find((option) => option.value === mode)?.label ?? (isZh ? "原始" : "Raw")
}

function getDisplayGlyph(mode: DisplayMode): string {
  return DISPLAY_OPTION_META.find((option) => option.value === mode)?.glyph ?? "T"
}

function ModeIcon({ glyph, checked = false }: { glyph: string; checked?: boolean }) {
  return (
    <span
      className={cn(
        "inline-flex min-w-[14px] items-center justify-center",
        "text-[10px] font-mono leading-none",
        checked ? "text-[var(--fg-secondary)]" : "text-[var(--fg-muted)]"
      )}
    >
      {glyph}
    </span>
  )
}

function FormatDropdown({
  value,
  onChange,
  previewActive,
  onSwitchToCodeView,
  isZh,
}: {
  value: DisplayMode
  onChange: (mode: DisplayMode) => void
  previewActive: boolean
  onSwitchToCodeView: () => void
  isZh: boolean
}) {
  const [open, setOpen] = useState(false)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const longestLabelLength = useMemo(
    () => getDisplayOptions(isZh).reduce((max, option) => Math.max(max, option.label.length), 0),
    [isZh]
  )
  const [menuRect, setMenuRect] = useState<{ top: number; left: number; width: number }>({
    top: 0,
    left: 0,
    width: 180,
  })

  const updateMenuRect = () => {
    if (!triggerRef.current) return
    const rect = triggerRef.current.getBoundingClientRect()
    const menuWidthByLabel = 30 + 60 + longestLabelLength * 8
    setMenuRect({
      top: rect.bottom + 6,
      left: rect.left,
      width: Math.max(rect.width + 30, menuWidthByLabel),
    })
  }

  useLayoutEffect(() => {
    if (!open) return
    updateMenuRect()
    const handleResize = () => updateMenuRect()
    const handleScroll = () => updateMenuRect()
    window.addEventListener("resize", handleResize)
    window.addEventListener("scroll", handleScroll, true)
    return () => {
      window.removeEventListener("resize", handleResize)
      window.removeEventListener("scroll", handleScroll, true)
    }
  }, [open])

  useEffect(() => {
    if (!open) return
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault()
        setOpen(false)
      }
    }
    window.addEventListener("keydown", onKeyDown)
    return () => window.removeEventListener("keydown", onKeyDown)
  }, [open])

  const options = useMemo(() => getDisplayOptions(isZh), [isZh])
  const structured = options.filter((option) => option.group === "structured")
  const encoded = options.filter((option) => option.group === "encoded")

  return (
    <div className="relative">
      <button
        ref={triggerRef}
        type="button"
        className={cn(
          "h-6 px-1.5 rounded-[8px] border border-transparent bg-transparent",
          "text-[10px] text-[var(--fg)] flex items-center gap-1 transition-colors",
          "hover:bg-[var(--button-bg)]",
          !previewActive && "bg-[var(--selected-bg)]"
        )}
        onClick={() => {
          if (previewActive) {
            onSwitchToCodeView()
            setOpen(false)
            return
          }
          setOpen((prev) => !prev)
        }}
      >
        <ModeIcon glyph={getDisplayGlyph(value)} checked />
        <span>{getDisplayLabel(value, isZh)}</span>
        <AppIcon name="arrowDown" size={8} className="text-[var(--fg-muted)]" />
      </button>

      {open && createPortal(
        <>
          <div className="fixed inset-0 z-[260]" onClick={() => setOpen(false)} />
          <div
            className={cn(
              "fixed z-[261] rounded-[10px] border shadow-[var(--shadow-lg)] overflow-hidden",
              "border-[var(--border-color)] bg-[var(--surface-elevated)]"
            )}
            style={{ top: menuRect.top, left: menuRect.left, width: `${menuRect.width}px` }}
          >
            <div className="p-1.5">
              {structured.map((option) => {
                const checked = option.value === value
                return (
                  <button
                    key={option.value}
                    type="button"
                    className={cn(
                      "w-full h-7 px-2 rounded-[7px] text-left text-[11px] transition-colors flex items-center gap-2",
                      checked ? "bg-[var(--selected-bg)] text-[var(--fg)]" : "text-[var(--fg)] hover:bg-[var(--sidebar-hover)]"
                    )}
                    onClick={() => { onChange(option.value); setOpen(false) }}
                  >
                    <span className="w-3 text-[12px] text-[var(--fg-muted)]">{checked ? "✓" : ""}</span>
                    <ModeIcon glyph={option.glyph} checked={checked} />
                    <span>{option.label}</span>
                  </button>
                )
              })}
            </div>
            <div className="h-px bg-[var(--border-subtle)] mx-2" />
            <div className="p-1.5">
              {encoded.map((option) => {
                const checked = option.value === value
                return (
                  <button
                    key={option.value}
                    type="button"
                    className={cn(
                      "w-full h-7 px-2 rounded-[7px] text-left text-[11px] transition-colors flex items-center gap-2",
                      checked ? "bg-[var(--selected-bg)] text-[var(--fg)]" : "text-[var(--fg)] hover:bg-[var(--sidebar-hover)]"
                    )}
                    onClick={() => { onChange(option.value); setOpen(false) }}
                  >
                    <span className="w-3 text-[12px] text-[var(--fg-muted)]">{checked ? "✓" : ""}</span>
                    <ModeIcon glyph={option.glyph} checked={checked} />
                    <span>{option.label}</span>
                  </button>
                )
              })}
            </div>
          </div>
        </>,
        document.body
      )}
    </div>
  )
}

function FilterGlyph() {
  return (
    <span className="inline-flex h-3.5 w-3.5 items-center justify-center">
      <svg width="12" height="12" viewBox="0 0 12 12" fill="none" aria-hidden="true">
        <path d="M2 2.5H10" stroke="currentColor" strokeWidth="1.25" strokeLinecap="round" />
        <path d="M2 6H8" stroke="currentColor" strokeWidth="1.25" strokeLinecap="round" />
        <path d="M2 9.5H6.5" stroke="currentColor" strokeWidth="1.25" strokeLinecap="round" />
      </svg>
    </span>
  )
}

function WrapGlyph() {
  return (
    <span className="inline-flex h-3.5 w-3.5 items-center justify-center">
      <svg width="12" height="12" viewBox="0 0 12 12" fill="none" aria-hidden="true">
        <path d="M2 3H9" stroke="currentColor" strokeWidth="1.25" strokeLinecap="round" />
        <path d="M2 6H8.7C9.7 6 10.5 6.8 10.5 7.8C10.5 8.8 9.7 9.6 8.7 9.6H5.9" stroke="currentColor" strokeWidth="1.25" strokeLinecap="round" />
        <path d="M7.4 11L5.8 9.6L7.4 8.2" stroke="currentColor" strokeWidth="1.25" strokeLinecap="round" strokeLinejoin="round" />
      </svg>
    </span>
  )
}

export function ResponseBody({ body, bodyBase64, bodyIsBinary, headers, contentType, requestUrl, isDark }: ResponseBodyProps) {
  const { t, isZh } = useI18n()
  const rootRef = useRef<HTMLDivElement>(null)
  const responseFormatDetection = useUIStore((s) => s.responseFormatDetection)
  const rawBodyText = useMemo(
    () => decodeResponseBodyToText({ body, bodyBase64, bodyIsBinary }),
    [body, bodyBase64, bodyIsBinary]
  )
  const preferDownload = useMemo(
    () => shouldPreferDownload({ headers, contentType, bodyIsBinary }),
    [headers, contentType, bodyIsBinary]
  )
  const suggestedFilename = useMemo(
    () => suggestResponseFilename({ headers, contentType, requestUrl }),
    [headers, contentType, requestUrl]
  )
  const defaultMode = useMemo(
    () => detectDefaultMode(contentType, rawBodyText, responseFormatDetection),
    [contentType, rawBodyText, responseFormatDetection]
  )
  const [mode, setMode] = useState<DisplayMode>(defaultMode)
  const [downloadMode, setDownloadMode] = useState(preferDownload)
  const [preview, setPreview] = useState(false)
  const [copied, setCopied] = useState(false)
  const [filterOpen, setFilterOpen] = useState(false)
  const [filterExpr, setFilterExpr] = useState("")
  const [searchSignal, setSearchSignal] = useState<number | undefined>(undefined)
  const [lineWrap, setLineWrap] = useState(true)
  const [foldAllSignal, setFoldAllSignal] = useState<number | undefined>(undefined)
  const [unfoldAllSignal, setUnfoldAllSignal] = useState<number | undefined>(undefined)

  useEffect(() => {
    setMode(defaultMode)
    setPreview(false)
    setFilterOpen(false)
    setFilterExpr("")
    setDownloadMode(preferDownload)
  }, [defaultMode, rawBodyText, contentType, preferDownload])

  useEffect(() => {
    if (!downloadMode) return
    setPreview(false)
    setFilterOpen(false)
  }, [downloadMode])

  const editorLanguage = useMemo(() => toEditorLanguage(mode), [mode])
  const previewAvailable = useMemo(() => canPreview(mode, contentType, rawBodyText), [mode, contentType, rawBodyText])

  const formattedBody = useMemo(() => {
    if (!rawBodyText) return ""
    if (mode === "json") return tryFormatJson(rawBodyText)
    if (mode === "xml" || mode === "html") return tryFormatXml(rawBodyText)
    if (mode === "javascript") return tryFormatJs(rawBodyText)
    if (mode === "hex") return toHex(rawBodyText)
    if (mode === "base64") return toBase64(rawBodyText)
    return rawBodyText
  }, [rawBodyText, mode])

  const filteredBodyState = useMemo(() => {
    if (!filterOpen || !filterExpr.trim()) return { text: formattedBody, error: null as string | null }
    if (mode === "json") return tryApplyJsonFilter(rawBodyText, filterExpr, t)
    return tryApplyTextFilter(formattedBody, filterExpr, t)
  }, [rawBodyText, filterExpr, filterOpen, formattedBody, mode, t])

  const displayBody = filteredBodyState.text

  const previewDoc = useMemo(() => buildPreviewDocument(mode, rawBodyText), [mode, rawBodyText])
  const jsonPreviewValue = useMemo(() => {
    if (mode !== "json") return null
    try {
      return JSON.parse(rawBodyText)
    } catch {
      return null
    }
  }, [rawBodyText, mode])

  const handleCopy = async () => {
    await navigator.clipboard.writeText(displayBody)
    setCopied(true)
    window.setTimeout(() => setCopied(false), 1200)
  }

  const handleDownload = useCallback(async () => {
    const { SaveResponseToFile } = await import("../../../../wailsjs/go/main/App")
    const payload = buildSaveResponsePayload({ body, bodyBase64, bodyIsBinary })
    await SaveResponseToFile(suggestedFilename, payload)
  }, [body, bodyBase64, bodyIsBinary, suggestedFilename])

  const handleSearch = useCallback(() => {
    if (preview) setPreview(false)
    window.setTimeout(() => setSearchSignal(Date.now()), 0)
  }, [preview])

  const handleFoldAll = useCallback(() => {
    if (preview) setPreview(false)
    window.setTimeout(() => setFoldAllSignal(Date.now()), 0)
  }, [preview])

  const handleUnfoldAll = useCallback(() => {
    if (preview) setPreview(false)
    window.setTimeout(() => setUnfoldAllSignal(Date.now()), 0)
  }, [preview])

  const showJsonFoldActions = mode === "json" && !preview

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (!(event.metaKey || event.ctrlKey) || event.altKey) return
      const key = event.key.toLowerCase()
      const active = document.activeElement as HTMLElement | null
      const inResponseBody = !!(rootRef.current && active && rootRef.current.contains(active))
      const isInputLike = !!active && (
        active.tagName === "INPUT"
        || active.tagName === "TEXTAREA"
        || active.isContentEditable
      )
      const inEditor = !!active?.closest(".cm-editor, .monaco-editor")

      if (key === "f") {
        if (inResponseBody) {
          event.preventDefault()
          event.stopPropagation()
          handleSearch()
        }
        return
      }

      if (key === "a") {
        if (!isInputLike && !inEditor) {
          event.preventDefault()
          event.stopPropagation()
        }
      }
    }

    window.addEventListener("keydown", onKeyDown, true)
    return () => window.removeEventListener("keydown", onKeyDown, true)
  }, [handleSearch])

  return (
    <div
      ref={rootRef}
      tabIndex={-1}
      className="h-full flex flex-col outline-none"
      onMouseDownCapture={() => rootRef.current?.focus({ preventScroll: true })}
    >
      <div className="flex items-center justify-between h-[34px] px-3 flex-shrink-0">
        <div className="flex items-center gap-2 min-w-0">
          {downloadMode ? (
            <div className="inline-flex min-w-0 items-center gap-1 rounded-[8px] bg-[var(--selected-bg)] px-2 py-1 text-[10px] text-[var(--fg-secondary)]">
              <AppIcon name="download" size={10} />
              <span className="truncate">{suggestedFilename}</span>
            </div>
          ) : (
            <>
              <FormatDropdown
                value={mode}
                onChange={setMode}
                previewActive={preview}
                onSwitchToCodeView={() => setPreview(false)}
                isZh={isZh}
              />
              <Tooltip>
                <TooltipTrigger asChild>
                  <button
                    type="button"
                    className={cn(
                      "h-6 px-1.5 rounded-[8px] border border-transparent bg-transparent text-[10px] flex items-center gap-1 transition-colors",
                      preview && previewAvailable
                        ? "bg-[var(--selected-bg)] text-[var(--accent)]"
                        : "text-[var(--fg-secondary)] hover:text-[var(--fg)] hover:bg-[var(--button-bg)]",
                      !previewAvailable && "opacity-40 pointer-events-none"
                    )}
                    onClick={() => setPreview(true)}
                  >
                    <AppIcon name="arrowRight" size={10} />
                    {t("预览", "Preview")}
                  </button>
                </TooltipTrigger>
                <TooltipContent>{t("预览", "Preview")}</TooltipContent>
              </Tooltip>
            </>
          )}
        </div>

        <div className="flex items-center gap-1.5">
          {downloadMode ? (
            <>
              <button
                type="button"
                className="h-6 rounded-[8px] px-2 text-[10px] bg-[var(--accent)] text-[var(--accent-fg)] hover:bg-[var(--accent-hover)] transition-colors inline-flex items-center gap-1"
                onClick={() => void handleDownload()}
              >
                <AppIcon name="download" size={11} />
                {t("下载文件", "Download file")}
              </button>
              <button
                type="button"
                className="h-6 rounded-[8px] px-2 text-[10px] text-[var(--fg-secondary)] hover:text-[var(--fg)] hover:bg-[var(--button-bg)] transition-colors"
                onClick={() => setDownloadMode(false)}
              >
                {t("文本查看", "View as text")}
              </button>
            </>
          ) : (
            <>
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                type="button"
                className={cn(
                  "h-6 w-6 rounded-[8px] border border-transparent bg-transparent transition-colors flex items-center justify-center",
                  lineWrap
                    ? "bg-[var(--selected-bg)] text-[var(--accent)]"
                    : "text-[var(--fg-secondary)] hover:text-[var(--fg)] hover:bg-[var(--button-bg)]"
                )}
                onClick={() => setLineWrap((prev) => !prev)}
              >
                <WrapGlyph />
              </button>
            </TooltipTrigger>
            <TooltipContent>{t("自动换行", "Line wrap")}</TooltipContent>
          </Tooltip>
          <span className="mx-0.5 h-3.5 w-px bg-[var(--border-subtle)]" aria-hidden="true" />
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                type="button"
                className={cn(
                  "h-6 w-6 rounded-[8px] border border-transparent bg-transparent transition-colors flex items-center justify-center",
                  filterOpen
                    ? "bg-[var(--selected-bg)] text-[var(--accent)]"
                    : "text-[var(--fg-secondary)] hover:text-[var(--fg)] hover:bg-[var(--button-bg)]"
                )}
                onClick={() => setFilterOpen((prev) => !prev)}
              >
                <FilterGlyph />
              </button>
            </TooltipTrigger>
            <TooltipContent>{t("过滤", "Filter")}</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                type="button"
                className="h-6 w-6 rounded-[8px] border border-transparent bg-transparent text-[var(--fg-secondary)] hover:text-[var(--fg)] hover:bg-[var(--button-bg)] transition-colors flex items-center justify-center"
                onClick={handleSearch}
              >
                <AppIcon name="search" size={12} />
              </button>
            </TooltipTrigger>
            <TooltipContent>{t("查找 (⌘F)", "Search (⌘F)")}</TooltipContent>
          </Tooltip>
          {showJsonFoldActions && (
            <>
              <span className="mx-0.5 h-3.5 w-px bg-[var(--border-subtle)]" aria-hidden="true" />
              <Tooltip>
                <TooltipTrigger asChild>
                  <button
                    type="button"
                    className="h-6 rounded-[8px] border border-transparent px-2 text-[10px] text-[var(--fg-secondary)] hover:text-[var(--fg)] hover:bg-[var(--button-bg)] transition-colors"
                    onClick={handleFoldAll}
                  >
                    {t("全部折叠", "Fold all")}
                  </button>
                </TooltipTrigger>
                <TooltipContent>{t("一键折叠所有 JSON 节点", "Collapse all JSON nodes")}</TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger asChild>
                  <button
                    type="button"
                    className="h-6 rounded-[8px] border border-transparent px-2 text-[10px] text-[var(--fg-secondary)] hover:text-[var(--fg)] hover:bg-[var(--button-bg)] transition-colors"
                    onClick={handleUnfoldAll}
                  >
                    {t("全部展开", "Unfold all")}
                  </button>
                </TooltipTrigger>
                <TooltipContent>{t("一键展开所有 JSON 节点", "Expand all JSON nodes")}</TooltipContent>
              </Tooltip>
            </>
          )}
          <span className="mx-0.5 h-3.5 w-px bg-[var(--border-subtle)]" aria-hidden="true" />
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                type="button"
                className="no-press-feedback h-6 px-1.5 rounded-[8px] border border-transparent bg-transparent text-[var(--fg-secondary)] hover:text-[var(--fg)] hover:bg-[var(--button-bg)] transition-colors flex items-center justify-center"
                onClick={handleCopy}
              >
                <AppIcon name="copy" size={12} className={cn(copied && "text-[var(--accent)]")} />
              </button>
            </TooltipTrigger>
            <TooltipContent>{copied ? t("已复制", "Copied") : t("复制响应体", "Copy response body")}</TooltipContent>
          </Tooltip>
            {preferDownload && (
              <>
                <span className="mx-0.5 h-3.5 w-px bg-[var(--border-subtle)]" aria-hidden="true" />
                <Tooltip>
                  <TooltipTrigger asChild>
                    <button
                      type="button"
                      className="h-6 px-1.5 rounded-[8px] border border-transparent bg-transparent text-[var(--fg-secondary)] hover:text-[var(--fg)] hover:bg-[var(--button-bg)] transition-colors flex items-center justify-center"
                      onClick={() => setDownloadMode(true)}
                    >
                      <AppIcon name="download" size={12} />
                    </button>
                  </TooltipTrigger>
                  <TooltipContent>{t("下载文件", "Download file")}</TooltipContent>
                </Tooltip>
              </>
            )}
            </>
          )}
        </div>
      </div>

      {filterOpen && !downloadMode && (
        <div className="px-3 pb-2">
          <div className="flex items-center gap-1.5">
            <input
              value={filterExpr}
              onChange={(e) => setFilterExpr(e.target.value)}
              placeholder={mode === "json"
                ? t("JSONPath 过滤（如 $.headers.host）", "JSONPath filter (e.g. $.headers.host)")
                : t("输入关键字过滤行", "Filter lines by keyword")}
              className="h-7 flex-1 rounded-[8px] border border-[var(--button-border)] bg-[var(--surface)] px-2.5 text-[11px] text-[var(--fg)] outline-none placeholder:text-[var(--fg-muted)] focus:border-[var(--accent)]"
            />
            {filterExpr && (
              <button
                type="button"
                className="h-7 rounded-[8px] border border-[var(--button-border)] px-2 text-[11px] text-[var(--fg-secondary)] hover:bg-[var(--button-bg)]"
                onClick={() => setFilterExpr("")}
              >
                {t("清空", "Clear")}
              </button>
            )}
          </div>
          {filteredBodyState.error && filterExpr.trim() && (
            <div className="mt-1 text-[10px] text-[var(--warning)]">{filteredBodyState.error}</div>
          )}
        </div>
      )}

      <div className="flex-1 min-h-0">
        {downloadMode ? (
          <div className="flex h-full items-center justify-center px-4">
            <div className="w-full max-w-[520px] rounded-[12px] border border-[var(--border-color)] bg-[var(--surface-elevated)] p-4">
              <div className="mb-3 flex items-center gap-2">
                <div className="inline-flex h-8 w-8 items-center justify-center rounded-[8px] bg-[var(--selected-bg)] text-[var(--accent)]">
                  <AppIcon name="download" size={14} />
                </div>
                <div className="min-w-0">
                  <div className="truncate text-[13px] font-semibold text-[var(--fg)]">{suggestedFilename}</div>
                  <div className="truncate text-[11px] text-[var(--fg-muted)]">{contentType || t("二进制响应", "Binary response")}</div>
                </div>
              </div>
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  className="h-8 rounded-[8px] bg-[var(--accent)] px-3 text-[12px] text-[var(--accent-fg)] hover:bg-[var(--accent-hover)] transition-colors inline-flex items-center gap-1.5"
                  onClick={() => void handleDownload()}
                >
                  <AppIcon name="download" size={12} />
                  {t("下载文件", "Download file")}
                </button>
                <button
                  type="button"
                  className="h-8 rounded-[8px] px-3 text-[12px] text-[var(--fg-secondary)] hover:text-[var(--fg)] hover:bg-[var(--button-bg)] transition-colors"
                  onClick={() => setDownloadMode(false)}
                >
                  {t("文本查看", "View as text")}
                </button>
              </div>
            </div>
          </div>
        ) : preview && previewAvailable ? (
          mode === "json" && jsonPreviewValue !== null ? (
            <JsonStructuredPreview value={jsonPreviewValue} rootLabel={t("根节点", "Root")} valueLabel={t("值", "value")} />
          ) : (
            <iframe
              title={t("响应预览", "Response Preview")}
              sandbox=""
              srcDoc={previewDoc}
              className="h-full w-full bg-[var(--surface)]"
              style={{ border: "none" }}
            />
          )
        ) : (
          <CodeEditor
            value={displayBody || t("(空响应)", "(empty response)")}
            language={editorLanguage}
            isDark={isDark}
            readOnly
            fillParent
            syntaxStyle="postman"
            searchSignal={searchSignal}
            lineWrap={lineWrap}
            foldAllSignal={foldAllSignal}
            unfoldAllSignal={unfoldAllSignal}
          />
        )}
      </div>
    </div>
  )
}
