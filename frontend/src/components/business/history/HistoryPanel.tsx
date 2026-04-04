import { useEffect, useState, useCallback } from "react"
import { Clock, Trash2 } from "lucide-react"
import { cn } from "@/lib/utils"
import { METHOD_COLORS, type HttpMethod } from "@/lib/constants"
import { historyService } from "@/services/historyService"
import { useProjectStore } from "@/stores/projectStore"
import { useTabStore } from "@/stores/tabStore"
import { createDefaultRequest } from "@/types/request"
import type { model } from "../../../../wailsjs/go/models"

function formatTime(timestamp: string): string {
  const d = new Date(timestamp)
  const now = new Date()
  const diff = now.getTime() - d.getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return "刚刚"
  if (mins < 60) return `${mins}分钟前`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours}小时前`
  return d.toLocaleDateString("zh-CN", { month: "short", day: "numeric" })
}

function getStatusColor(code: number): string {
  if (code >= 200 && code < 300) return "text-[var(--success)]"
  if (code >= 400) return "text-[var(--danger)]"
  return "text-[var(--warning)]"
}

export function HistoryPanel() {
  const { currentProjectId } = useProjectStore()
  const { openRequestTab } = useTabStore()
  const [entries, setEntries] = useState<model.HistoryEntry[]>([])
  const [loading, setLoading] = useState(false)

  const loadHistory = useCallback(async () => {
    if (!currentProjectId) return
    setLoading(true)
    try {
      const result = await historyService.getHistory(currentProjectId)
      setEntries(result ?? [])
    } finally {
      setLoading(false)
    }
  }, [currentProjectId])

  useEffect(() => {
    loadHistory()
  }, [loadHistory])

  const handleClear = async () => {
    if (!currentProjectId) return
    if (!confirm("确认清空所有历史记录？")) return
    await historyService.clearHistory(currentProjectId)
    setEntries([])
  }

  const handleRestore = (entry: model.HistoryEntry) => {
    if (!currentProjectId) return
    const request = createDefaultRequest({
      method: entry.method as HttpMethod,
      url: entry.url,
      name: entry.name || entry.url,
    })
    openRequestTab(currentProjectId, request)
  }

  if (!currentProjectId) {
    return (
      <div className="flex items-center justify-center h-full text-[length:var(--size-font-xs)] text-[var(--fg-muted)]">
        请先选择项目
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-2 py-1">
        <div className="flex items-center gap-1 text-2xs text-[var(--fg-muted)] font-medium">
          <Clock className="h-3 w-3" />
          <span>历史</span>
          {entries.length > 0 && <span>({entries.length})</span>}
        </div>
        {entries.length > 0 && (
          <button
            className="flex items-center gap-0.5 h-5 px-1.5 rounded-[var(--radius-sm)] text-2xs text-[var(--fg-muted)] hover:text-[var(--danger)] hover:bg-[var(--sidebar-hover)] transition-colors"
            onClick={handleClear}
          >
            <Trash2 className="h-2.5 w-2.5" /> 清空
          </button>
        )}
      </div>

      <div className="flex-1 overflow-y-auto">
        {loading ? (
          <div className="text-center py-6 text-2xs text-[var(--fg-muted)]">加载中...</div>
        ) : entries.length === 0 ? (
          <div className="text-center py-6 text-2xs text-[var(--fg-muted)]">暂无历史记录</div>
        ) : (
          entries.map((entry) => (
            <div
              key={entry.id}
              className="flex items-center h-[24px] px-2 mx-1 rounded-[var(--radius-btn)] cursor-pointer hover:bg-[var(--sidebar-hover)] group"
              onClick={() => handleRestore(entry)}
            >
              <span className={cn(
                "text-[9px] font-mono font-bold w-[32px] text-right flex-shrink-0 uppercase mr-1.5",
                METHOD_COLORS[(entry.method as HttpMethod)] || "text-[var(--fg-muted)]"
              )}>
                {entry.method?.substring(0, 3) || "GET"}
              </span>
              <span className="text-[length:var(--size-font-2xs)] truncate flex-1 text-[var(--fg)]">
                {entry.url || entry.name}
              </span>
              <span className={cn("text-2xs font-mono flex-shrink-0 ml-1", getStatusColor(entry.statusCode))}>
                {entry.statusCode}
              </span>
              <span className="text-2xs text-[var(--fg-muted)] ml-1.5 flex-shrink-0">
                {formatTime(entry.timestamp)}
              </span>
            </div>
          ))
        )}
      </div>
    </div>
  )
}
