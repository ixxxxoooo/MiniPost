import { useMemo, useState } from "react"
import { cn } from "@/lib/utils"
import { AppIcon } from "@/components/ui/icon"
import type { HttpStreamEntry } from "@/types/response"
import { CodeEditor, type EditorLanguage } from "@/components/ui/CodeEditor"
import { useI18n } from "@/hooks/useI18n"

type EntryViewMode = "text" | "json"

interface ResponseStreamProps {
  entries: HttpStreamEntry[]
  isDark: boolean
}

const MAX_RENDERED_STREAM_ROWS = 240

function pad2(value: number): string {
  return String(value).padStart(2, "0")
}

function pad3(value: number): string {
  return String(value).padStart(3, "0")
}

function formatTimeLabel(timestamp: string): string {
  const date = new Date(timestamp)
  if (Number.isNaN(date.getTime())) return "--:--:--.---"
  return `${pad2(date.getHours())}:${pad2(date.getMinutes())}:${pad2(date.getSeconds())}.${pad3(date.getMilliseconds())}`
}

function tryPrettyJson(text: string): string {
  try {
    return JSON.stringify(JSON.parse(text), null, 2)
  } catch {
    return text
  }
}

function normalizeEntrySummary(entry: HttpStreamEntry, t: (zh: string, en: string) => string): string {
  if (entry.kind === "connection_closed") return t("连接已关闭", "Connection closed")
  if (entry.kind === "error") return entry.data || t("流式错误", "Stream error")
  return entry.data || entry.raw || ""
}

function modeToLanguage(mode: EntryViewMode): EditorLanguage {
  return mode === "json" ? "json" : "text"
}

export function ResponseStream({ entries, isDark }: ResponseStreamProps) {
  const { t } = useI18n()
  const [expandedById, setExpandedById] = useState<Record<string, boolean>>({})
  const [modeById, setModeById] = useState<Record<string, EntryViewMode>>({})
  const [lineWrap, setLineWrap] = useState(false)
  const [searchQuery, setSearchQuery] = useState("")

  const orderedEntries = useMemo(
    () => {
      const filtered = entries.filter((entry) => entry.kind !== "response_start")
      const sliced = filtered.length > MAX_RENDERED_STREAM_ROWS
        ? filtered.slice(filtered.length - MAX_RENDERED_STREAM_ROWS)
        : filtered
      return [...sliced].reverse()
    },
    [entries]
  )

  const filteredEntries = useMemo(() => {
    const query = searchQuery.trim().toLowerCase()
    if (!query) return orderedEntries
    return orderedEntries.filter((entry) => {
      const text = `${entry.kind} ${entry.data || ""} ${entry.raw || ""}`.toLowerCase()
      return text.includes(query)
    })
  }, [orderedEntries, searchQuery])

  if (orderedEntries.length === 0) {
    return (
      <div className="flex h-full items-center justify-center text-2xs text-[var(--fg-muted)]">
        {t("暂无流式消息", "No stream messages")}
      </div>
    )
  }

  return (
    <div className="h-full overflow-auto rounded-[10px] border border-[var(--border-color)] bg-[var(--surface)]">
      <div className="sticky top-0 z-10 border-b border-[var(--border-subtle)] bg-[var(--surface)] px-2 py-2">
        {entries.length > MAX_RENDERED_STREAM_ROWS && (
          <div className="mb-1 text-[10px] text-[var(--fg-muted)]">
            {t("仅展示最近流消息，避免页面卡顿。", "Showing recent stream messages only to keep UI responsive.")}
          </div>
        )}
        <div className="relative">
          <AppIcon name="search" size={12} className="pointer-events-none absolute left-2 top-1/2 -translate-y-1/2 text-[var(--fg-muted)]" />
          <input
            value={searchQuery}
            onChange={(event) => setSearchQuery(event.target.value)}
            placeholder={t("搜索", "Search")}
            className="h-7 w-full rounded-[7px] border border-[var(--border-color)] bg-[var(--surface)] pl-7 pr-7 text-[12px] text-[var(--fg)] outline-none placeholder:text-[var(--fg-muted)]"
          />
          {searchQuery && (
            <button
              type="button"
              onClick={() => setSearchQuery("")}
              className="no-press-feedback absolute right-1.5 top-1/2 -translate-y-1/2 text-[var(--fg-muted)] hover:text-[var(--fg)]"
            >
              <AppIcon name="clear" size={12} />
            </button>
          )}
        </div>
      </div>

      {filteredEntries.map((entry, index) => {
        const expanded = Boolean(expandedById[entry.id])
        const isClosableInfo = entry.kind === "connection_closed"
        const rowText = normalizeEntrySummary(entry, t)
        const mode = modeById[entry.id] ?? "json"
        const detailText = mode === "json" ? tryPrettyJson(entry.data || entry.raw || "") : (entry.data || entry.raw || "")

        return (
          <div key={entry.id} className={cn(index > 0 && "border-t border-[var(--border-subtle)]")}>
            <button
              type="button"
              className={cn(
                "no-press-feedback flex w-full items-center gap-2 px-2 py-1.5 text-left text-[11px] leading-5 transition-colors",
                expanded ? "bg-[var(--surface-secondary)]" : "hover:bg-[var(--surface-secondary)]"
              )}
              onClick={() => {
                if (isClosableInfo) return
                setExpandedById((prev) => ({ ...prev, [entry.id]: !prev[entry.id] }))
              }}
            >
              <span
                className={cn(
                  "inline-flex h-4 w-4 items-center justify-center rounded-[4px]",
                  isDark ? "bg-[var(--surface-elevated)]" : "bg-[var(--sidebar-hover)]"
                )}
              >
                {isClosableInfo ? (
                  <AppIcon name="info" size={10} className="text-[var(--fg-muted)]" />
                ) : (
                  <AppIcon name="download" size={10} className="text-[#2f6fd3]" />
                )}
              </span>

              <span className="min-w-0 flex-1 truncate font-mono text-[var(--fg)]">
                {rowText || t("(空消息块)", "(empty chunk)")}
              </span>

              <span className="flex items-center gap-1">
                <span className="font-mono text-[var(--fg-muted)] [font-variant-numeric:tabular-nums]">
                  {formatTimeLabel(entry.timestamp)}
                </span>
                {!isClosableInfo && (
                  <AppIcon
                    name="arrowDown"
                    size={10}
                    className={cn("text-[var(--fg-muted)] transition-transform", expanded ? "rotate-180" : "-rotate-90")}
                  />
                )}
              </span>
            </button>

            {expanded && !isClosableInfo && (
              <div className="border-t border-[var(--border-subtle)] bg-[var(--surface)]">
                <div className="flex h-8 items-center gap-2 border-b border-[var(--border-subtle)] px-2">
                  <select
                    value={mode}
                    className="h-6 rounded-[6px] border border-[var(--border-color)] bg-[var(--surface)] px-2 text-[11px] text-[var(--fg)] outline-none"
                    onChange={(event) => {
                      const nextMode = event.target.value === "text" ? "text" : "json"
                      setModeById((prev) => ({ ...prev, [entry.id]: nextMode }))
                    }}
                  >
                    <option value="json">JSON</option>
                    <option value="text">{t("文本", "Text")}</option>
                  </select>

                  <button
                    type="button"
                    className={cn(
                      "no-press-feedback inline-flex h-6 w-6 items-center justify-center rounded-[6px] border text-[11px] transition-colors",
                      lineWrap
                        ? "border-[var(--accent)] bg-[var(--selected-bg)] text-[var(--fg)]"
                        : "border-[var(--border-color)] text-[var(--fg-secondary)] hover:text-[var(--fg)]"
                    )}
                    title={t("切换自动换行", "Toggle line wrap")}
                    onClick={() => setLineWrap((prev) => !prev)}
                  >
                    <AppIcon name="arrowLeftRight" size={11} />
                  </button>

                  <button
                    type="button"
                    className="no-press-feedback inline-flex h-6 w-6 items-center justify-center rounded-[6px] border border-[var(--border-color)] text-[var(--fg-secondary)] transition-colors hover:text-[var(--fg)]"
                    title={t("复制", "Copy")}
                    onClick={() => void navigator.clipboard.writeText(detailText)}
                  >
                    <AppIcon name="copy" size={11} />
                  </button>
                </div>

                <div className="h-[240px]">
                  <CodeEditor
                    value={detailText}
                    language={modeToLanguage(mode)}
                    isDark={isDark}
                    readOnly
                    fillParent
                    syntaxStyle="postman"
                    lineWrap={lineWrap}
                    className="h-full"
                  />
                </div>
              </div>
            )}
          </div>
        )
      })}

      {filteredEntries.length === 0 && (
        <div className="px-3 py-8 text-center text-[12px] text-[var(--fg-muted)]">
          {t("没有匹配的流式消息", "No matching stream messages")}
        </div>
      )}
    </div>
  )
}
