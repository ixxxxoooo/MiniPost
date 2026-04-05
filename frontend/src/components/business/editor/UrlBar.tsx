import { Send, Loader2, Save } from "lucide-react"
import { HTTP_METHODS, METHOD_COLORS, type HttpMethod } from "@/lib/constants"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
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
        "flex items-center gap-[var(--size-gap)] px-[var(--size-padding)] flex-shrink-0",
        "bg-[var(--surface)]"
      )}
      style={{ height: "40px" }}
    >
      {activeTab.dirty && (
        <div className="w-1.5 h-1.5 rounded-full bg-[var(--warning)] flex-shrink-0" title="未保存" />
      )}

      <div className="flex flex-1 items-center overflow-hidden rounded-[10px] border border-[var(--button-border)] bg-[var(--surface)] focus-within:border-[var(--accent)] focus-within:ring-1 focus-within:ring-[var(--accent)]/20">
        {/* 方法选择 */}
        <Select value={request.method} onValueChange={(value) => updateTabRequest(activeTab.id, { method: value as HttpMethod })}>
          <SelectTrigger
            className={cn(
              "h-[30px] w-[96px] rounded-none border-0 border-r border-[var(--button-border)] bg-transparent px-3",
              "text-[11px] font-mono font-semibold shadow-none focus:ring-0",
              "justify-between",
              METHOD_COLORS[request.method]
            )}
          >
            <SelectValue placeholder="方法" />
          </SelectTrigger>
          <SelectContent className="rounded-[10px] border-[var(--button-border)] bg-[var(--surface-elevated)] p-1 shadow-[var(--shadow-lg)]">
            {HTTP_METHODS.map((method) => (
              <SelectItem
                key={method}
                value={method}
                className={cn(
                  "rounded-[8px] py-1.5 pl-7 pr-2 text-[11px] font-mono font-semibold",
                  METHOD_COLORS[method]
                )}
              >
                {method}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        {/* URL 输入 */}
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
