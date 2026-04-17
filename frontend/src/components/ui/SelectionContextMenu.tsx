import { useEffect, useMemo, useRef, useState } from "react"
import { createPortal } from "react-dom"
import { AppIcon } from "@/components/ui/icon"
import { useI18n } from "@/hooks/useI18n"
import { getSelectionFromRegisteredEditors } from "@/lib/editorSelectionBridge"
import { cn } from "@/lib/utils"
import { writeClipboardText } from "@/lib/clipboard"

interface MenuState {
  x: number
  y: number
  text: string
}

const MENU_WIDTH = 164
const MENU_HEIGHT = 38
const VIEWPORT_PADDING = 8

function isTextControl(element: Element | null): element is HTMLInputElement | HTMLTextAreaElement {
  return element instanceof HTMLInputElement || element instanceof HTMLTextAreaElement
}

function getClosestElement(target: EventTarget | null): Element | null {
  if (target instanceof Element) return target
  if (target instanceof Node) return target.parentElement
  return null
}

function getSelectedTextFromTarget(target: EventTarget | null, requireTargetWithinSelection: boolean): string {
  const fromRegisteredEditor = getSelectionFromRegisteredEditors(target)
  if (fromRegisteredEditor) return fromRegisteredEditor

  const element = getClosestElement(target)

  const textControl = element ? element.closest("input, textarea") : null
  if (isTextControl(textControl)) {
    const start = textControl.selectionStart ?? 0
    const end = textControl.selectionEnd ?? 0
    return end > start ? textControl.value.slice(start, end) : ""
  }

  const selection = window.getSelection()
  if (!selection || selection.isCollapsed) return ""
  if (requireTargetWithinSelection && target instanceof Node && !selection.containsNode(target, true)) return ""
  return selection.toString()
}

function getSelectedTextFallback(): string {
  const selection = window.getSelection()
  if (!selection || selection.isCollapsed) return ""
  return selection.toString()
}

function clampMenuPosition(x: number, y: number) {
  const maxX = Math.max(VIEWPORT_PADDING, window.innerWidth - MENU_WIDTH - VIEWPORT_PADDING)
  const maxY = Math.max(VIEWPORT_PADDING, window.innerHeight - MENU_HEIGHT - VIEWPORT_PADDING)
  return {
    x: Math.min(Math.max(VIEWPORT_PADDING, x), maxX),
    y: Math.min(Math.max(VIEWPORT_PADDING, y), maxY),
  }
}

export function SelectionContextMenu() {
  const { t } = useI18n()
  const [menu, setMenu] = useState<MenuState | null>(null)
  const lastSelectedTextRef = useRef<{ text: string; timestamp: number }>({ text: "", timestamp: 0 })
  const hotkeyLabel = useMemo(() => {
    if (typeof navigator === "undefined") return "Ctrl+C"
    return navigator.platform.toLowerCase().includes("mac") ? "⌘C" : "Ctrl+C"
  }, [])

  useEffect(() => {
    const closeMenu = () => setMenu(null)
    const rememberSelectionFromTarget = (target: EventTarget | null) => {
      const text = getSelectedTextFromTarget(target, false) || getSelectedTextFallback()
      if (!text.trim()) return
      lastSelectedTextRef.current = { text, timestamp: Date.now() }
    }
    const handleRememberSelection = (event: Event) => rememberSelectionFromTarget(event.target)

    const handleContextMenu = (event: MouseEvent) => {
      if (getClosestElement(event.target)?.closest("[data-selection-context-menu]")) {
        return
      }

      const recent = Date.now() - lastSelectedTextRef.current.timestamp < 1800
        ? lastSelectedTextRef.current.text
        : ""
      const text = getSelectedTextFromTarget(event.target, true) || getSelectedTextFallback() || recent
      if (!text.trim()) {
        closeMenu()
        return
      }

      event.preventDefault()
      const position = clampMenuPosition(event.clientX + 2, event.clientY + 2)
      setMenu({ x: position.x, y: position.y, text })
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      const isCopy = (event.metaKey || event.ctrlKey) && !event.altKey && !event.shiftKey && event.key.toLowerCase() === "c"
      if (!isCopy) return

      const text = getSelectedTextFromTarget(event.target, false)
        || getSelectedTextFallback()
        || lastSelectedTextRef.current.text
      if (!text.trim()) return

      event.preventDefault()
      void writeClipboardText(text)
      closeMenu()
    }

    const handleSelectionChange = () => {
      const selection = window.getSelection()
      if (!selection || selection.isCollapsed) {
        closeMenu()
        return
      }
      rememberSelectionFromTarget(document.activeElement)
    }

    window.addEventListener("contextmenu", handleContextMenu, true)
    window.addEventListener("mouseup", handleRememberSelection, true)
    window.addEventListener("keyup", handleRememberSelection, true)
    window.addEventListener("keydown", handleKeyDown, true)
    window.addEventListener("resize", closeMenu)
    window.addEventListener("blur", closeMenu)
    document.addEventListener("scroll", closeMenu, true)
    document.addEventListener("selectionchange", handleSelectionChange)

    return () => {
      window.removeEventListener("contextmenu", handleContextMenu, true)
      window.removeEventListener("mouseup", handleRememberSelection, true)
      window.removeEventListener("keyup", handleRememberSelection, true)
      window.removeEventListener("keydown", handleKeyDown, true)
      window.removeEventListener("resize", closeMenu)
      window.removeEventListener("blur", closeMenu)
      document.removeEventListener("scroll", closeMenu, true)
      document.removeEventListener("selectionchange", handleSelectionChange)
    }
  }, [])

  if (!menu) return null

  return createPortal(
    <>
      <div className="fixed inset-0 z-[420]" onMouseDown={() => setMenu(null)} />
      <div
        data-selection-context-menu="true"
        className={cn(
          "fixed z-[421] w-[164px] rounded-[10px] border border-[var(--border-color)] bg-[var(--surface-elevated)] p-1 shadow-[var(--shadow-lg)] animate-fade-in"
        )}
        style={{ left: menu.x, top: menu.y }}
        onMouseDown={(event) => event.stopPropagation()}
      >
        <button
          type="button"
          className="flex w-full items-center gap-2 rounded-[7px] px-2.5 py-1.5 text-left text-[length:var(--size-font-2xs)] text-[var(--fg)] transition-colors hover:bg-[var(--sidebar-hover)]"
          onClick={() => {
            void writeClipboardText(menu.text)
            setMenu(null)
          }}
        >
          <AppIcon name="copy" size={12} />
          <span className="flex-1">{t("复制", "Copy")}</span>
          <span className="text-[10px] text-[var(--fg-muted)]">{hotkeyLabel}</span>
        </button>
      </div>
    </>,
    document.body
  )
}
