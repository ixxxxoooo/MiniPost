import { useRef, useEffect, useCallback } from "react"
import { EditorView, keymap, placeholder as cmPlaceholder, lineNumbers, highlightActiveLine, highlightActiveLineGutter } from "@codemirror/view"
import { EditorState } from "@codemirror/state"
import { json } from "@codemirror/lang-json"
import { javascript } from "@codemirror/lang-javascript"
import { xml } from "@codemirror/lang-xml"
import { defaultKeymap, indentWithTab } from "@codemirror/commands"
import { syntaxHighlighting, defaultHighlightStyle, bracketMatching, foldGutter, foldKeymap } from "@codemirror/language"
import { searchKeymap, highlightSelectionMatches, openSearchPanel } from "@codemirror/search"
import { oneDark } from "@codemirror/theme-one-dark"
import { cn } from "@/lib/utils"

export type EditorLanguage = "json" | "javascript" | "xml" | "text"

interface CodeEditorProps {
  value: string
  onChange?: (value: string) => void
  language?: EditorLanguage
  placeholder?: string
  isDark?: boolean
  className?: string
  readOnly?: boolean
  /** 编辑器自动撑满父容器 */
  fillParent?: boolean
}

export function stripJsonComments(text: string): string {
  let result = ""
  let i = 0
  let inString = false
  let escape = false

  while (i < text.length) {
    const ch = text[i]
    const next = text[i + 1]

    if (inString) {
      result += ch
      if (escape) {
        escape = false
      } else if (ch === "\\") {
        escape = true
      } else if (ch === '"') {
        inString = false
      }
      i++
      continue
    }

    if (ch === '"') {
      inString = true
      result += ch
      i++
      continue
    }

    if (ch === "/" && next === "/") {
      while (i < text.length && text[i] !== "\n") i++
      continue
    }

    if (ch === "/" && next === "*") {
      i += 2
      while (i < text.length && !(text[i] === "*" && text[i + 1] === "/")) i++
      i += 2
      continue
    }

    result += ch
    i++
  }

  return result
}

export function formatJsonWithComments(text: string): string {
  const stripped = stripJsonComments(text)
  try {
    const parsed = JSON.parse(stripped)
    return JSON.stringify(parsed, null, 2)
  } catch {
    return text
  }
}

const editorThemeLight = EditorView.theme({
  "&": {
    backgroundColor: "var(--surface, #ffffff)",
    color: "var(--fg, #1d1d1f)",
    fontSize: "12px",
  },
  ".cm-content": {
    fontFamily: "ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Consolas, monospace",
    padding: "8px 0",
    caretColor: "#007aff",
  },
  ".cm-gutters": {
    backgroundColor: "var(--surface-secondary, #f9f9f9)",
    borderRight: "1px solid var(--border-color, #ededed)",
    color: "#aeaeb2",
    fontSize: "11px",
    minWidth: "36px",
  },
  ".cm-activeLineGutter": {
    backgroundColor: "rgba(0, 0, 0, 0.04)",
    color: "#6e6e73",
  },
  ".cm-activeLine": {
    backgroundColor: "rgba(0, 122, 255, 0.04)",
  },
  "&.cm-focused .cm-cursor": {
    borderLeftColor: "#007aff",
  },
  "&.cm-focused .cm-selectionBackground, .cm-selectionBackground": {
    backgroundColor: "rgba(0, 122, 255, 0.15) !important",
  },
  ".cm-selectionMatch": {
    backgroundColor: "rgba(0, 122, 255, 0.1)",
  },
  ".cm-searchMatch": {
    backgroundColor: "rgba(255, 200, 0, 0.3)",
    borderRadius: "2px",
  },
  ".cm-searchMatch.cm-searchMatch-selected": {
    backgroundColor: "rgba(255, 150, 0, 0.4)",
  },
  ".cm-foldGutter span": {
    color: "#aeaeb2",
    fontSize: "10px",
  },
  ".cm-matchingBracket": {
    backgroundColor: "rgba(0, 122, 255, 0.12)",
    outline: "1px solid rgba(0, 122, 255, 0.3)",
  },
  ".cm-panels": {
    backgroundColor: "var(--surface-secondary, #f9f9f9)",
    borderBottom: "1px solid var(--border-color, #ededed)",
  },
  ".cm-panel.cm-search": {
    padding: "4px 8px",
  },
  ".cm-panel.cm-search input": {
    border: "1px solid var(--border-color, #ccc)",
    borderRadius: "4px",
    padding: "2px 6px",
    fontSize: "12px",
  },
  ".cm-panel.cm-search button": {
    border: "1px solid var(--border-color, #ccc)",
    borderRadius: "4px",
    padding: "2px 8px",
    fontSize: "11px",
    cursor: "pointer",
  },
})

const editorThemeDark = EditorView.theme({
  "&": {
    backgroundColor: "var(--surface, #1e1e1e)",
    color: "var(--fg, #d4d4d4)",
    fontSize: "12px",
  },
  ".cm-content": {
    fontFamily: "ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Consolas, monospace",
    padding: "8px 0",
    caretColor: "#0a84ff",
  },
  ".cm-gutters": {
    backgroundColor: "var(--surface-secondary, #252526)",
    borderRight: "1px solid var(--border-color, #333)",
    color: "#858585",
    fontSize: "11px",
    minWidth: "36px",
  },
  ".cm-activeLineGutter": {
    backgroundColor: "rgba(255, 255, 255, 0.04)",
    color: "#c6c6c6",
  },
  ".cm-activeLine": {
    backgroundColor: "rgba(255, 255, 255, 0.04)",
  },
  "&.cm-focused .cm-cursor": {
    borderLeftColor: "#0a84ff",
  },
  "&.cm-focused .cm-selectionBackground, .cm-selectionBackground": {
    backgroundColor: "rgba(10, 132, 255, 0.25) !important",
  },
  ".cm-searchMatch": {
    backgroundColor: "rgba(255, 200, 0, 0.2)",
    borderRadius: "2px",
  },
  ".cm-searchMatch.cm-searchMatch-selected": {
    backgroundColor: "rgba(255, 150, 0, 0.35)",
  },
  ".cm-matchingBracket": {
    backgroundColor: "rgba(10, 132, 255, 0.15)",
    outline: "1px solid rgba(10, 132, 255, 0.3)",
  },
  ".cm-panels": {
    backgroundColor: "var(--surface-secondary, #252526)",
    borderBottom: "1px solid var(--border-color, #333)",
  },
  ".cm-panel.cm-search": {
    padding: "4px 8px",
  },
  ".cm-panel.cm-search input": {
    border: "1px solid var(--border-color, #555)",
    borderRadius: "4px",
    padding: "2px 6px",
    fontSize: "12px",
    backgroundColor: "var(--surface, #1e1e1e)",
    color: "var(--fg, #d4d4d4)",
  },
  ".cm-panel.cm-search button": {
    border: "1px solid var(--border-color, #555)",
    borderRadius: "4px",
    padding: "2px 8px",
    fontSize: "11px",
    cursor: "pointer",
    color: "var(--fg, #d4d4d4)",
  },
})

function getLanguageExtension(language: EditorLanguage) {
  switch (language) {
    case "json": return json()
    case "javascript": return javascript()
    case "xml": return xml()
    default: return []
  }
}

export function CodeEditor({
  value,
  onChange,
  language = "json",
  placeholder = "",
  isDark = false,
  className,
  readOnly = false,
  fillParent = false,
}: CodeEditorProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const viewRef = useRef<EditorView | null>(null)
  const onChangeRef = useRef(onChange)
  onChangeRef.current = onChange

  const createExtensions = useCallback(() => {
    const extensions = [
      lineNumbers(),
      highlightActiveLine(),
      highlightActiveLineGutter(),
      bracketMatching(),
      foldGutter(),
      highlightSelectionMatches(),
      syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
      keymap.of([...defaultKeymap, ...foldKeymap, ...searchKeymap, indentWithTab]),
      EditorView.updateListener.of((update) => {
        if (update.docChanged && onChangeRef.current) {
          onChangeRef.current(update.state.doc.toString())
        }
      }),
      EditorView.lineWrapping,
      isDark ? editorThemeDark : editorThemeLight,
    ]

    if (isDark) {
      extensions.push(oneDark)
    }

    if (readOnly) {
      extensions.push(EditorState.readOnly.of(true))
      extensions.push(EditorView.editable.of(false))
    }

    if (placeholder) {
      extensions.push(cmPlaceholder(placeholder))
    }

    const langExt = getLanguageExtension(language)
    if (Array.isArray(langExt)) {
      extensions.push(...langExt)
    } else {
      extensions.push(langExt)
    }

    if (fillParent) {
      extensions.push(EditorView.theme({
        "&": { height: "100%" },
        ".cm-scroller": { overflow: "auto" },
      }))
    }

    return extensions
  }, [language, placeholder, isDark, readOnly, fillParent])

  useEffect(() => {
    if (!containerRef.current) return

    const state = EditorState.create({
      doc: value,
      extensions: createExtensions(),
    })

    const view = new EditorView({
      state,
      parent: containerRef.current,
    })

    viewRef.current = view

    return () => {
      view.destroy()
      viewRef.current = null
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [createExtensions])

  useEffect(() => {
    const view = viewRef.current
    if (!view) return
    const currentDoc = view.state.doc.toString()
    if (currentDoc !== value) {
      view.dispatch({
        changes: { from: 0, to: currentDoc.length, insert: value },
      })
    }
  }, [value])

  return (
    <div
      ref={containerRef}
      className={cn(
        "overflow-hidden",
        "[&_.cm-editor]:!outline-none",
        fillParent && "h-full [&_.cm-editor]:h-full",
        !fillParent && "border border-[var(--border-color)] rounded-[var(--radius-input)]",
        className
      )}
    />
  )
}

export { openSearchPanel }
