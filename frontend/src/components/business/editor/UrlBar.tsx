import { Send, Loader2, Save } from "lucide-react"
import { HTTP_METHODS, METHOD_COLORS, type HttpMethod } from "@/lib/constants"
import { useTabStore } from "@/stores/tabStore"
import { useUIStore } from "@/stores/uiStore"
import { cn } from "@/lib/utils"

interface UrlBarProps {
  onSend: () => void
  onSave: () => void
}

export function UrlBar({ onSend, onSave }: UrlBarProps) {
  const activeTab = useTabStore((s) => s.getActiveTab())
  const updateTabRequest = useTabStore((s) => s.updateTabRequest)
  const { isSending } = useUIStore()

  if (!activeTab) return null

  const { request } = activeTab

  return (
    <div
      className={cn(
        "flex items-center gap-[var(--size-gap)] px-[var(--size-padding)] border-b flex-shrink-0",
        "border-[var(--border-color)] bg-[var(--surface)]"
      )}
      style={{ height: "40px" }}
    >
      {activeTab.dirty && (
        <div className="w-1.5 h-1.5 rounded-full bg-[var(--warning)] flex-shrink-0" title="未保存" />
      )}

      {/* 方法选择 */}
      <select
        value={request.method}
        onChange={(e) => updateTabRequest(activeTab.id, { method: e.target.value as HttpMethod })}
        className={cn(
          "h-[var(--size-btn-sm)] px-2 rounded-[var(--radius-btn)] font-mono font-bold text-[length:var(--size-font-xs)]",
          "bg-transparent border border-[var(--border-color)] outline-none cursor-pointer",
          METHOD_COLORS[request.method]
        )}
      >
        {HTTP_METHODS.map((m) => (
          <option key={m} value={m}>{m}</option>
        ))}
      </select>

      {/* URL 输入 */}
      <input
        value={request.url}
        onChange={(e) => updateTabRequest(activeTab.id, { url: e.target.value })}
        placeholder="输入请求 URL..."
        className={cn(
          "flex-1 h-[var(--size-btn)] px-3 rounded-[var(--radius-input)]",
          "border border-[var(--border-color)] bg-[var(--surface)] text-[var(--fg)]",
          "font-mono text-[length:var(--size-font-xs)] placeholder:text-[var(--fg-muted)]",
          "focus:outline-none focus:border-[var(--accent)] focus:ring-1 focus:ring-[var(--accent)]/30"
        )}
        onKeyDown={(e) => {
          if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) onSend()
        }}
      />

      {/* 保存按钮 */}
      <button
        className={cn(
          "h-[var(--size-btn)] w-[var(--size-btn)] flex items-center justify-center rounded-[var(--radius-btn)]",
          "hover:bg-[var(--sidebar-hover)] transition-colors",
          activeTab.dirty ? "text-[var(--warning)]" : "text-[var(--fg-muted)]"
        )}
        onClick={onSave}
        disabled={!activeTab.dirty}
        title="保存 (⌘S)"
      >
        <Save className="h-[var(--size-btn-icon-sm)] w-[var(--size-btn-icon-sm)]" />
      </button>

      {/* 发送按钮 */}
      <button
        onClick={onSend}
        disabled={isSending || !request.url.trim()}
        className={cn(
          "h-[var(--size-btn)] px-3 flex items-center gap-1.5 rounded-[var(--radius-btn)]",
          "bg-[var(--accent)] text-[var(--accent-fg)] text-[length:var(--size-font-xs)] font-medium",
          "hover:bg-[var(--accent-hover)] transition-colors",
          "disabled:opacity-50 disabled:pointer-events-none flex-shrink-0"
        )}
      >
        {isSending ? (
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
        ) : (
          <Send className="h-3.5 w-3.5" />
        )}
        发送
      </button>
    </div>
  )
}
