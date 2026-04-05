import { useState, useMemo, useEffect, useLayoutEffect, useRef } from "react"
import { createPortal } from "react-dom"
import { cn } from "@/lib/utils"
import { CodeEditor, type EditorLanguage } from "@/components/ui/CodeEditor"
import { AppIcon } from "@/components/ui/icon"

interface ResponseBodyProps {
  body: string
  contentType: string
  isDark: boolean
}

type DisplayMode = "json" | "xml" | "html" | "javascript" | "raw" | "hex" | "base64"

interface DisplayOption {
  value: DisplayMode
  label: string
  group: "structured" | "encoded"
  glyph: string
}

const DISPLAY_OPTIONS: DisplayOption[] = [
  { value: "json", label: "JSON", group: "structured", glyph: "{}" },
  { value: "xml", label: "XML", group: "structured", glyph: "</>" },
  { value: "html", label: "HTML", group: "structured", glyph: "</>" },
  { value: "javascript", label: "JavaScript", group: "structured", glyph: "JS" },
  { value: "raw", label: "Raw", group: "encoded", glyph: "T" },
  { value: "hex", label: "Hex", group: "encoded", glyph: "0x" },
  { value: "base64", label: "Base64", group: "encoded", glyph: "64" },
]

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

function detectDefaultMode(contentType: string, body: string): DisplayMode {
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
    return text
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

function getDisplayLabel(mode: DisplayMode): string {
  return DISPLAY_OPTIONS.find((option) => option.value === mode)?.label ?? "Raw"
}

function getDisplayGlyph(mode: DisplayMode): string {
  return DISPLAY_OPTIONS.find((option) => option.value === mode)?.glyph ?? "T"
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
}: {
  value: DisplayMode
  onChange: (mode: DisplayMode) => void
  previewActive: boolean
  onSwitchToCodeView: () => void
}) {
  const [open, setOpen] = useState(false)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const longestLabelLength = useMemo(
    () => DISPLAY_OPTIONS.reduce((max, option) => Math.max(max, option.label.length), 0),
    []
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

  const structured = DISPLAY_OPTIONS.filter((option) => option.group === "structured")
  const encoded = DISPLAY_OPTIONS.filter((option) => option.group === "encoded")

  return (
    <div className="relative">
      <button
        ref={triggerRef}
        type="button"
        className={cn(
          "h-6 px-1.5 rounded-[8px] border border-transparent bg-transparent",
          "text-[10px] text-[var(--fg)] flex items-center gap-1 transition-colors",
          "hover:bg-[var(--button-bg)]",
          !previewActive && "bg-[rgb(237,237,237)]"
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
        <span>{getDisplayLabel(value)}</span>
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
                      checked ? "bg-[rgb(237,237,237)] text-[var(--fg)]" : "text-[var(--fg)] hover:bg-[var(--sidebar-hover)]"
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
                      checked ? "bg-[rgb(237,237,237)] text-[var(--fg)]" : "text-[var(--fg)] hover:bg-[var(--sidebar-hover)]"
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

export function ResponseBody({ body, contentType, isDark }: ResponseBodyProps) {
  const defaultMode = useMemo(() => detectDefaultMode(contentType, body), [contentType, body])
  const [mode, setMode] = useState<DisplayMode>(defaultMode)
  const [preview, setPreview] = useState(false)
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    setMode(defaultMode)
    setPreview(false)
  }, [defaultMode, body, contentType])

  const editorLanguage = useMemo(() => toEditorLanguage(mode), [mode])
  const previewAvailable = useMemo(() => canPreview(mode, contentType, body), [mode, contentType, body])

  const displayBody = useMemo(() => {
    if (!body) return ""
    if (mode === "json") return tryFormatJson(body)
    if (mode === "xml" || mode === "html") return tryFormatXml(body)
    if (mode === "javascript") return tryFormatJs(body)
    if (mode === "hex") return toHex(body)
    if (mode === "base64") return toBase64(body)
    return body
  }, [body, mode])

  const previewDoc = useMemo(() => buildPreviewDocument(mode, body), [mode, body])

  const handleCopy = async () => {
    await navigator.clipboard.writeText(displayBody)
    setCopied(true)
    window.setTimeout(() => setCopied(false), 1200)
  }

  return (
    <div className="h-full flex flex-col">
      <div className="flex items-center justify-between h-[34px] px-3 flex-shrink-0">
        <div className="flex items-center gap-2">
          <FormatDropdown
            value={mode}
            onChange={setMode}
            previewActive={preview}
            onSwitchToCodeView={() => setPreview(false)}
          />
          <button
            type="button"
            className={cn(
              "h-6 px-1.5 rounded-[8px] border border-transparent bg-transparent text-[10px] flex items-center gap-1 transition-colors",
              preview && previewAvailable
                ? "bg-[rgb(237,237,237)] text-[var(--accent)]"
                : "text-[var(--fg-secondary)] hover:text-[var(--fg)] hover:bg-[var(--button-bg)]",
              !previewAvailable && "opacity-40 pointer-events-none"
            )}
            onClick={() => setPreview(true)}
            title="Preview"
          >
            <AppIcon name="arrowRight" size={10} />
            Preview
          </button>
        </div>

        <div className="flex items-center gap-1.5">
          <button
            type="button"
            className="h-6 px-1.5 rounded-[8px] border border-transparent bg-transparent text-[var(--fg-secondary)] hover:text-[var(--fg)] hover:bg-[var(--button-bg)] transition-colors flex items-center justify-center"
            onClick={handleCopy}
            title="复制响应"
          >
            <AppIcon name="copy" size={12} className={cn(copied && "text-[var(--success)]")} />
          </button>
        </div>
      </div>

      <div className="flex-1 min-h-0">
        {preview && previewAvailable ? (
          <iframe
            title="Response Preview"
            sandbox=""
            srcDoc={previewDoc}
            className="h-full w-full bg-[var(--surface)]"
            style={{ border: "none" }}
          />
        ) : (
          <CodeEditor
            value={displayBody || "(空响应)"}
            language={editorLanguage}
            isDark={isDark}
            readOnly
            fillParent
            className="[&_.cm-gutters]:bg-transparent [&_.cm-gutters]:border-r-0 [&_.cm-activeLineGutter]:bg-transparent"
          />
        )}
      </div>
    </div>
  )
}
