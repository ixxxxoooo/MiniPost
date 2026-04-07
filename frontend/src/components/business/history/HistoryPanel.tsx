import { useEffect, useState, useCallback, useMemo } from "react"
import { AppIcon } from "@/components/ui/icon"
import { useI18n } from "@/hooks/useI18n"
import { cn } from "@/lib/utils"
import { METHOD_COLORS, type HttpMethod } from "@/lib/constants"
import { historyService } from "@/services/historyService"
import { useProjectStore } from "@/stores/projectStore"
import { useTabStore } from "@/stores/tabStore"
import { useUIStore } from "@/stores/uiStore"
import { createDefaultRequest } from "@/types/request"
import type { model } from "../../../../wailsjs/go/models"

function formatTimeOnly(timestamp: string, locale: string): string {
  const d = new Date(timestamp)
  return d.toLocaleTimeString(locale, { hour12: false, hour: "2-digit", minute: "2-digit" })
}

function getDateKey(timestamp: string, locale: string, t: (zh: string, en: string) => string): string {
  const d = new Date(timestamp)
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const yesterday = new Date(today.getTime() - 86400000)
  const target = new Date(d.getFullYear(), d.getMonth(), d.getDate())

  if (target.getTime() === today.getTime()) return t("今天", "Today")
  if (target.getTime() === yesterday.getTime()) return t("昨天", "Yesterday")
  return d.toLocaleDateString(locale, { month: "long", day: "numeric", weekday: "short" })
}

function getStatusColor(code: number): string {
  if (code >= 200 && code < 300) return "text-[var(--success)]"
  if (code >= 400) return "text-[var(--danger)]"
  return "text-[var(--warning)]"
}

interface DayGroup {
  label: string
  entries: model.HistoryEntry[]
}

export function HistoryPanel() {
  const { t, locale } = useI18n()
  const { currentProjectId } = useProjectStore()
  const { openRequestTab } = useTabStore()
  const setEditingEnvironmentId = useUIStore((s) => s.setEditingEnvironmentId)
  const [entries, setEntries] = useState<model.HistoryEntry[]>([])
  const [loading, setLoading] = useState(false)
  const [collapsedDays, setCollapsedDays] = useState<Set<string>>(new Set())

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

  useEffect(() => { loadHistory() }, [loadHistory])

  const dayGroups = useMemo<DayGroup[]>(() => {
    const groups = new Map<string, model.HistoryEntry[]>()
    for (const entry of entries) {
      const key = getDateKey(entry.timestamp, locale, t)
      if (!groups.has(key)) groups.set(key, [])
      groups.get(key)!.push(entry)
    }
    return Array.from(groups.entries()).map(([label, items]) => ({ label, entries: items }))
  }, [entries, locale, t])

  const toggleDay = (label: string) => {
    setCollapsedDays((prev) => {
      const next = new Set(prev)
      if (next.has(label)) next.delete(label)
      else next.add(label)
      return next
    })
  }

  const handleClear = async () => {
    if (!currentProjectId) return
    if (!confirm(t("确认清空所有历史记录？", "Clear all history records?"))) return
    await historyService.clearHistory(currentProjectId)
    setEntries([])
  }

  const handleRefresh = async () => {
    await loadHistory()
  }

  const handleRestore = (entry: model.HistoryEntry) => {
    if (!currentProjectId) return
    setEditingEnvironmentId(null)
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
        {t("请先选择项目", "Please select a project first")}
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-2 py-1">
        <div className="flex items-center gap-1 text-2xs text-[var(--fg-muted)] font-medium">
          <AppIcon name="clock" size={12} />
          <span>{t("历史", "History")}</span>
          {entries.length > 0 && <span>({entries.length})</span>}
        </div>
        <div className="flex items-center gap-1">
          <button
            className="flex items-center gap-0.5 h-5 px-1.5 rounded-[var(--radius-sm)] text-2xs text-[var(--fg-muted)] hover:text-[var(--fg)] hover:bg-[var(--sidebar-hover)] transition-colors disabled:opacity-50"
            onClick={() => void handleRefresh()}
            disabled={loading}
            title={t("刷新历史", "Refresh history")}
          >
            <AppIcon name="arrowUpDown" size={10} className={loading ? "animate-spin" : undefined} /> {t("刷新", "Refresh")}
          </button>
          {entries.length > 0 && (
            <button
              className="flex items-center gap-0.5 h-5 px-1.5 rounded-[var(--radius-sm)] text-2xs text-[var(--fg-muted)] hover:text-[var(--danger)] hover:bg-[var(--sidebar-hover)] transition-colors"
              onClick={handleClear}
            >
              <AppIcon name="delete" size={10} /> {t("清空", "Clear")}
            </button>
          )}
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        {entries.length === 0 ? (
          loading ? (
            <div className="text-center py-6 text-2xs text-[var(--fg-muted)]">{t("加载中...", "Loading...")}</div>
          ) : (
            <div className="text-center py-6 text-2xs text-[var(--fg-muted)]">{t("暂无历史记录", "No history yet")}</div>
          )
        ) : (
          dayGroups.map((group) => {
            const isCollapsed = collapsedDays.has(group.label)
            return (
              <div key={group.label}>
                <div
                  className="flex items-center gap-1 px-2 py-1 cursor-pointer select-none hover:bg-[var(--sidebar-hover)] transition-colors"
                  onClick={() => toggleDay(group.label)}
                >
                  <AppIcon name={isCollapsed ? "arrowRight" : "arrowDown"} size={8} className="text-[var(--fg-muted)]" />
                  <span className="text-[10px] font-medium text-[var(--fg-secondary)]">{group.label}</span>
                  <span className="text-[10px] text-[var(--fg-muted)]">({group.entries.length})</span>
                </div>
                {!isCollapsed && group.entries.map((entry) => (
                  <div
                    key={entry.id}
                    className="flex items-center h-[24px] px-2 pl-5 mx-1 rounded-[var(--radius-btn)] cursor-pointer hover:bg-[var(--sidebar-hover)] group"
                    onClick={() => handleRestore(entry)}
                  >
                    <span className={cn(
                      "text-[9px] font-mono font-bold w-[40px] text-right flex-shrink-0 uppercase mr-1.5",
                      METHOD_COLORS[(entry.method as HttpMethod)] || "text-[var(--fg-muted)]"
                    )}>
                      {entry.method || "GET"}
                    </span>
                    <span className="text-[11px] truncate flex-1 text-[var(--fg)]">
                      {entry.url || entry.name}
                    </span>
                    <span className={cn("text-[10px] font-mono flex-shrink-0 ml-1", getStatusColor(entry.statusCode))}>
                      {entry.statusCode}
                    </span>
                    <span className="text-[10px] text-[var(--fg-muted)] ml-1.5 flex-shrink-0">
                      {formatTimeOnly(entry.timestamp, locale)}
                    </span>
                  </div>
                ))}
              </div>
            )
          })
        )}
      </div>
    </div>
  )
}
