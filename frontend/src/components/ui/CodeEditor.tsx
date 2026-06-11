import { useCallback, useEffect, useMemo, useRef } from "react"
import MonacoEditor, { loader, useMonaco } from "@monaco-editor/react"
import * as Monaco from "monaco-editor/esm/vs/editor/editor.api.js"
import type { editor as MonacoEditorType } from "monaco-editor"
import "monaco-editor/esm/vs/language/json/monaco.contribution"
import editorWorker from "monaco-editor/esm/vs/editor/editor.worker?worker"
import jsonWorker from "monaco-editor/esm/vs/language/json/json.worker?worker"
import { getEditorValueSyncDecision, shouldApplyPendingEditorValue } from "@/lib/codeEditorValueSync"
import { registerMonacoSelectionProvider } from "@/lib/editorSelectionBridge"
import { cn } from "@/lib/utils"

// 在 Vite/Wails 环境下显式配置 Monaco loader 与 worker，避免运行时 Promise 拒绝和编辑器空白。
loader.config({ monaco: Monaco })
void loader.init().catch(() => undefined)

if (typeof self !== "undefined") {
  self.MonacoEnvironment = {
    getWorker(_: unknown, label: string) {
      if (label === "json") return new jsonWorker()
      return new editorWorker()
    },
  }
}

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
  /** 响应区可用的 Postman 风格语法色 */
  syntaxStyle?: "default" | "postman"
  /** 每次值变更时触发打开搜索面板 */
  searchSignal?: number
  /** 是否启用自动换行 */
  lineWrap?: boolean
  /** 每次值变更时触发全部折叠 */
  foldAllSignal?: number
  /** 每次值变更时触发全部展开 */
  unfoldAllSignal?: number
  /** 每次值变更时触发文档格式化 */
  formatSignal?: number
  /** 是否启用 ⌘/Ctrl+Enter 发送快捷键 */
  enableSendShortcut?: boolean
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

function toMonacoLanguage(language: EditorLanguage): string {
  switch (language) {
    case "json":
      return "json"
    case "javascript":
      return "javascript"
    case "xml":
      return "xml"
    default:
      return "plaintext"
  }
}

function applyExternalEditorValue(editor: MonacoEditorType.IStandaloneCodeEditor, nextValue: string) {
  const position = editor.getPosition()
  const selections = editor.getSelections()
  const scrollTop = editor.getScrollTop()
  const scrollLeft = editor.getScrollLeft()

  editor.setValue(nextValue)
  if (selections) editor.setSelections(selections)
  if (position) editor.setPosition(position)
  editor.setScrollTop(scrollTop)
  editor.setScrollLeft(scrollLeft)
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
  syntaxStyle = "default",
  searchSignal,
  lineWrap = true,
  foldAllSignal,
  unfoldAllSignal,
  formatSignal,
  enableSendShortcut = false,
}: CodeEditorProps) {
  const monaco = useMonaco()
  const containerRef = useRef<HTMLDivElement | null>(null)
  const editorRef = useRef<MonacoEditorType.IStandaloneCodeEditor | null>(null)
  const unregisterSelectionProviderRef = useRef<(() => void) | null>(null)
  const resizeObserverRef = useRef<ResizeObserver | null>(null)
  const tooltipObserverRef = useRef<MutationObserver | null>(null)
  const compositionDisposablesRef = useRef<Array<{ dispose: () => void }>>([])
  const isComposingRef = useRef(false)
  const pendingExternalValueRef = useRef<string | null>(null)
  const valueRef = useRef(value)

  useEffect(() => {
    valueRef.current = value
    const editor = editorRef.current
    if (!editor) return

    const currentValue = editor.getValue()
    const decision = getEditorValueSyncDecision(currentValue, value, isComposingRef.current)

    if (decision.type === "none") {
      pendingExternalValueRef.current = null
      return
    }

    if (decision.type === "defer") {
      pendingExternalValueRef.current = decision.pendingValue
      return
    }

    pendingExternalValueRef.current = null
    applyExternalEditorValue(editor, value)
  }, [value])

  const layoutEditor = useCallback((editor = editorRef.current) => {
    const container = containerRef.current
    if (!editor || !container) return

    const rect = container.getBoundingClientRect()
    if (rect.width <= 0 || rect.height <= 0) return

    editor.layout({
      width: Math.floor(rect.width),
      height: Math.floor(rect.height),
    })
  }, [])

  const watchEditorContainerSize = useCallback((editor: MonacoEditorType.IStandaloneCodeEditor) => {
    resizeObserverRef.current?.disconnect()
    resizeObserverRef.current = null

    const container = containerRef.current
    if (container && typeof ResizeObserver !== "undefined") {
      const observer = new ResizeObserver(() => layoutEditor(editor))
      observer.observe(container)
      resizeObserverRef.current = observer
    }

    requestAnimationFrame(() => layoutEditor(editor))
  }, [layoutEditor])

  const stabilizeFindWidgetTooltips = useCallback((editor: MonacoEditorType.IStandaloneCodeEditor) => {
    const container = editor.getContainerDomNode()
    if (!container) return

    const syncTooltipAttrs = () => {
      const tooltipTargets = container.querySelectorAll<HTMLElement>(".find-widget [title]")
      tooltipTargets.forEach((node) => {
        const title = node.getAttribute("title")
        if (!title) return
        node.setAttribute("data-minipost-tooltip", title)
        node.removeAttribute("title")
      })
    }

    syncTooltipAttrs()
    tooltipObserverRef.current?.disconnect()
    const observer = new MutationObserver(() => {
      syncTooltipAttrs()
    })
    observer.observe(container, {
      childList: true,
      subtree: true,
      attributes: true,
      attributeFilter: ["title"],
    })
    tooltipObserverRef.current = observer
  }, [])

  useEffect(() => {
    if (!monaco) return

    const jsonDefaults = (monaco.languages as unknown as {
      json?: { jsonDefaults?: { setDiagnosticsOptions: (options: { validate: boolean; allowComments: boolean }) => void } }
    }).json?.jsonDefaults
    jsonDefaults?.setDiagnosticsOptions({
      validate: true,
      allowComments: true,
    })

    monaco.editor.defineTheme("minipost-light-default", {
      base: "vs",
      inherit: true,
      colors: {
        "editor.background": "#ffffff",
        "editor.foreground": "#1d1d1f",
        "editorLineNumber.foreground": "#a0a0a8",
        "editorLineNumber.activeForeground": "#6e6e73",
        "editor.lineHighlightBackground": "#f6f9ff",
        "editor.selectionBackground": "#d8e7ff",
      },
      rules: [],
    })

    monaco.editor.defineTheme("minipost-dark-default", {
      base: "vs-dark",
      inherit: true,
      colors: {
        "editor.background": "#212121",
        "editor.foreground": "#d4d4d4",
        "editorGutter.background": "#212121",
        "editorWidget.background": "#212121",
        "editorSuggestWidget.background": "#212121",
        "peekViewEditor.background": "#212121",
        "peekViewResult.background": "#212121",
        "panel.background": "#212121",
        "editorLineNumber.foreground": "#7d828a",
        "editorLineNumber.activeForeground": "#c6c6c6",
        "editor.lineHighlightBackground": "#212121",
        "editor.selectionBackground": "#2d4f82",
      },
      rules: [],
    })

    monaco.editor.defineTheme("minipost-light-postman", {
      base: "vs",
      inherit: true,
      colors: {
        "editor.background": "#ffffff",
        "editor.foreground": "#2f2f33",
        "editorLineNumber.foreground": "#a0a0a8",
        "editorLineNumber.activeForeground": "#6e6e73",
        "editor.lineHighlightBackground": "#f3f8ff",
        "editor.selectionBackground": "#d8e7ff",
      },
      rules: [
        { token: "string.key.json", foreground: "C02F1D" },
        { token: "string.value.json", foreground: "0A4FA8" },
        { token: "number.json", foreground: "0F7B45" },
        { token: "keyword.json", foreground: "8F2D56" },
        { token: "string", foreground: "0A4FA8" },
        { token: "number", foreground: "0F7B45" },
        { token: "keyword", foreground: "8B5CF6" },
      ],
    })

    monaco.editor.defineTheme("minipost-dark-postman", {
      base: "vs-dark",
      inherit: true,
      colors: {
        "editor.background": "#212121",
        "editor.foreground": "#d3d7de",
        "editorGutter.background": "#212121",
        "editorWidget.background": "#212121",
        "editorSuggestWidget.background": "#212121",
        "peekViewEditor.background": "#212121",
        "peekViewResult.background": "#212121",
        "panel.background": "#212121",
        "editorLineNumber.foreground": "#7d828a",
        "editorLineNumber.activeForeground": "#c6c6c6",
        "editor.lineHighlightBackground": "#212121",
        "editor.selectionBackground": "#2d4f82",
      },
      rules: [
        { token: "string.key.json", foreground: "F28B82" },
        { token: "string.value.json", foreground: "80B7FF" },
        { token: "number.json", foreground: "8DDF99" },
        { token: "keyword.json", foreground: "F59CB5" },
        { token: "string", foreground: "80B7FF" },
        { token: "number", foreground: "8DDF99" },
        { token: "keyword", foreground: "C6A6FF" },
      ],
    })
  }, [monaco])

  const theme = useMemo(() => {
    if (syntaxStyle === "postman") {
      return isDark ? "minipost-dark-postman" : "minipost-light-postman"
    }
    return isDark ? "minipost-dark-default" : "minipost-light-default"
  }, [isDark, syntaxStyle])

  const options = useMemo<MonacoEditorType.IStandaloneEditorConstructionOptions>(() => ({
    readOnly,
    minimap: { enabled: false },
    scrollBeyondLastLine: false,
    smoothScrolling: false,
    automaticLayout: false,
    contextmenu: true,
    renderValidationDecorations: "off",
    fontSize: 12,
    lineHeight: 20,
    fontFamily: "ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Consolas, monospace",
    wordWrap: lineWrap ? "on" : "off",
    lineNumbers: "on",
    glyphMargin: false,
    folding: true,
    cursorBlinking: "smooth",
    cursorSmoothCaretAnimation: "off",
    padding: { top: 8, bottom: 8 },
    suggestOnTriggerCharacters: !readOnly,
    quickSuggestions: !readOnly,
    occurrencesHighlight: "singleFile",
    find: {
      addExtraSpaceOnTop: false,
      autoFindInSelection: "never",
      seedSearchStringFromSelection: "always",
    },
    ...(placeholder ? { placeholder } : {}),
  }), [lineWrap, placeholder, readOnly])

  useEffect(() => {
    if (searchSignal === undefined) return
    const editor = editorRef.current
    if (!editor) return
    editor.focus()
    void editor.getAction("actions.find")?.run()
  }, [searchSignal])

  useEffect(() => {
    if (foldAllSignal === undefined) return
    const editor = editorRef.current
    if (!editor) return
    editor.focus()
    const action = editor.getAction("editor.foldAll")
    if (action) {
      void action.run()
      return
    }
    editor.trigger("minipost", "editor.foldAll", null)
  }, [foldAllSignal])

  useEffect(() => {
    if (unfoldAllSignal === undefined) return
    const editor = editorRef.current
    if (!editor) return
    editor.focus()
    const action = editor.getAction("editor.unfoldAll")
    if (action) {
      void action.run()
      return
    }
    editor.trigger("minipost", "editor.unfoldAll", null)
  }, [unfoldAllSignal])

  useEffect(() => {
    if (formatSignal === undefined) return
    const editor = editorRef.current
    if (!editor) return
    editor.focus()
    const action = editor.getAction("editor.action.formatDocument")
    if (action) {
      void action.run()
      return
    }
    editor.trigger("minipost", "editor.action.formatDocument", null)
  }, [formatSignal])

  useEffect(() => () => {
    unregisterSelectionProviderRef.current?.()
    unregisterSelectionProviderRef.current = null
    compositionDisposablesRef.current.forEach((disposable) => disposable.dispose())
    compositionDisposablesRef.current = []
    resizeObserverRef.current?.disconnect()
    resizeObserverRef.current = null
    tooltipObserverRef.current?.disconnect()
    tooltipObserverRef.current = null
  }, [])

  return (
    <div
      ref={containerRef}
      className={cn(
        "relative overflow-hidden min-h-0",
        fillParent ? "h-full" : "min-h-[220px] border border-[var(--border-color)] rounded-[var(--radius-input)]",
        className
      )}
    >
      <MonacoEditor
        height="100%"
        language={toMonacoLanguage(language)}
        defaultValue={value}
        theme={theme}
        options={options}
        onMount={(editor, monacoInstance) => {
          editorRef.current = editor
          unregisterSelectionProviderRef.current?.()
          unregisterSelectionProviderRef.current = registerMonacoSelectionProvider(editor)
          compositionDisposablesRef.current.forEach((disposable) => disposable.dispose())
          compositionDisposablesRef.current = [
            editor.onDidCompositionStart(() => {
              isComposingRef.current = true
              pendingExternalValueRef.current = null
            }),
            editor.onDidCompositionEnd(() => {
              isComposingRef.current = false
              const pendingValue = pendingExternalValueRef.current
              pendingExternalValueRef.current = null
              if (shouldApplyPendingEditorValue(editor.getValue(), pendingValue)) {
                applyExternalEditorValue(editor, pendingValue)
              }
            }),
          ]
          watchEditorContainerSize(editor)
          stabilizeFindWidgetTooltips(editor)
          if (!enableSendShortcut || readOnly) return
          editor.addCommand(monacoInstance.KeyMod.CtrlCmd | monacoInstance.KeyCode.Enter, () => {
            window.dispatchEvent(new CustomEvent("minipost:send"))
          })
        }}
        onChange={(nextValue) => {
          const normalizedValue = nextValue ?? ""
          valueRef.current = normalizedValue
          if (isComposingRef.current) {
            pendingExternalValueRef.current = null
          }
          onChange?.(normalizedValue)
        }}
      />
    </div>
  )
}
