import { useState, useRef, useEffect } from "react"
import { HTTP_METHODS, METHOD_COLORS, type HttpMethod } from "@/lib/constants"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { AppIcon } from "@/components/ui/icon"
import { useTabStore, getProjectActiveTabFromState } from "@/stores/tabStore"
import { useUIStore } from "@/stores/uiStore"
import { cn } from "@/lib/utils"

interface UrlBarProps {
  onSend: (downloadAfter?: boolean) => void
  onCancel: () => void
  onSave: () => void
}

export function UrlBar({ onSend, onCancel, onSave }: UrlBarProps) {
  const activeTab = useTabStore(getProjectActiveTabFromState)
  const updateTabRequest = useTabStore((s) => s.updateTabRequest)
  const { isSending } = useUIStore()
  const [showSendMenu, setShowSendMenu] = useState(false)
  const sendMenuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!showSendMenu) return
    const handler = (e: MouseEvent) => {
      if (sendMenuRef.current && !sendMenuRef.current.contains(e.target as Node)) {
        setShowSendMenu(false)
      }
    }
    document.addEventListener("mousedown", handler)
    return () => document.removeEventListener("mousedown", handler)
  }, [showSendMenu])

  if (!activeTab) return null

  const { request } = activeTab

  return (
    <div
      className={cn(
        "flex items-center gap-[var(--size-gap)] px-[var(--size-padding)] flex-shrink-0",
        "bg-[var(--surface)]"
      )}
      style={{ height: "40px" }}
    >
      <div className="flex flex-1 items-center overflow-hidden rounded-[10px] border border-[var(--button-border)] bg-[var(--surface)] focus-within:border-[var(--accent)] focus-within:ring-1 focus-within:ring-[var(--accent)]/20">
        <Select value={request.method} onValueChange={(value) => updateTabRequest(activeTab.id, { method: value as HttpMethod })}>
          <SelectTrigger
            className={cn(
              "h-[30px] w-[96px] rounded-none border-0 border-r border-[var(--button-border)] bg-transparent px-3",
              "text-[11px] font-mono font-semibold shadow-none focus:ring-0",
              "justify-between",
              METHOD_COLORS[request.method as HttpMethod]
            )}
          >
            <SelectValue placeholder="方法" />
          </SelectTrigger>
          <SelectContent>
            {HTTP_METHODS.map((method) => (
              <SelectItem
                key={method}
                value={method}
                className={cn(
                  "rounded-[8px] py-1.5 px-2 text-[11px] font-mono font-semibold",
                  METHOD_COLORS[method]
                )}
              >
                {method}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <input
          value={request.url}
          onChange={(e) => updateTabRequest(activeTab.id, { url: e.target.value })}
          placeholder="输入请求 URL..."
          className={cn(
            "h-[30px] min-w-0 flex-1 bg-transparent px-3",
            "border-0 text-[var(--fg)] font-mono text-[length:var(--size-font-xs)]",
            "placeholder:text-[var(--fg-muted)] focus:outline-none focus:ring-0"
          )}
          onKeyDown={(e) => {
            if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) onSend()
          }}
        />
      </div>

      <button
        className={cn(
          "h-[var(--size-btn)] px-3 flex items-center justify-center gap-1 rounded-[var(--radius-btn)]",
          "text-[length:var(--size-font-2xs)] font-medium transition-colors",
          "hover:bg-[var(--sidebar-hover)]",
          activeTab.dirty ? "text-[var(--fg-secondary)]" : "text-[var(--fg-muted)]"
        )}
        onClick={onSave}
        disabled={!activeTab.dirty}
        title="保存 (⌘S)"
      >
        <AppIcon name="save" size={12} strokeWidth={1.9} />
        Save
      </button>

      {/* 发送/取消按钮组 */}
      <div className="relative flex items-center flex-shrink-0 w-[112px]" ref={sendMenuRef}>
        {isSending ? (
          <button
            onClick={onCancel}
            className={cn(
              "no-press-feedback h-[30px] w-full flex items-center justify-center gap-1.5 rounded-[8px]",
              "bg-[#f87171] text-white text-[13px] font-medium",
              "hover:bg-[#ef6666] active:bg-[#e65c5c] transition-[background-color,color]"
            )}
          >
            Cancel
          </button>
        ) : (
          <>
            <button
              onClick={() => onSend()}
              disabled={!request.url.trim()}
              className={cn(
                "no-press-feedback h-[30px] flex-1 flex items-center justify-center gap-1.5 rounded-l-[8px]",
                "bg-[#3b82f6] text-white text-[13px] font-medium",
                "hover:bg-[#3477e6] active:bg-[#2f6ed6] transition-[background-color,color]",
                "disabled:opacity-50 disabled:pointer-events-none"
              )}
            >
              Send
            </button>
            <button
              onClick={() => setShowSendMenu(!showSendMenu)}
              disabled={!request.url.trim()}
              className={cn(
                "no-press-feedback relative h-[30px] w-[28px] flex items-center justify-center rounded-r-[8px]",
                "before:pointer-events-none before:absolute before:left-0 before:top-[6px] before:h-[18px] before:w-px before:bg-white/25",
                "bg-[#3b82f6] text-white",
                "hover:bg-[#3477e6] active:bg-[#2f6ed6] transition-[background-color,color]",
                "disabled:opacity-50 disabled:pointer-events-none"
              )}
            >
              <AppIcon name="arrowDown" size={10} />
            </button>
          </>
        )}

        {showSendMenu && (
          <div
            className={cn(
              "absolute right-0 top-full mt-1 z-50 py-1 rounded-[var(--radius-menu)] shadow-lg border",
              "w-max bg-[var(--surface-elevated)] border-[var(--border-color)]"
            )}
          >
            <button
              className="no-press-feedback w-full whitespace-nowrap px-3 py-1.5 text-[11px] text-left hover:bg-[var(--sidebar-hover)] text-[var(--fg)] flex items-center gap-2 transition-[background-color,color] rounded-[7px] mx-1"
              onClick={() => { setShowSendMenu(false); onSend(true) }}
            >
              <AppIcon name="download" size={12} />
              Send and Download
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
