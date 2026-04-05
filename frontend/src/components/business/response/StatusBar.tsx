import { AppIcon } from "@/components/ui/icon"
import { cn } from "@/lib/utils"
import type { HttpResponse } from "@/types/response"

interface StatusBarProps {
  response: HttpResponse
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`
}

function getStatusColor(code: number): string {
  if (code >= 200 && code < 300) return "text-[var(--success)]"
  if (code >= 300 && code < 400) return "text-[var(--info)]"
  if (code >= 400 && code < 500) return "text-[var(--warning)]"
  return "text-[var(--danger)]"
}

export function StatusBar({ response }: StatusBarProps) {
  return (
    <div
      className={cn(
        "flex items-center h-[24px] px-[var(--size-padding)] gap-3 border-b flex-shrink-0",
        "bg-[var(--surface-secondary)] border-[var(--border-color)] select-none"
      )}
    >
      {/* 状态码 */}
      <div className="flex items-center gap-1">
        <div className={cn(
          "w-1.5 h-1.5 rounded-full flex-shrink-0",
          response.statusCode >= 200 && response.statusCode < 300 ? "bg-[var(--success)]" :
          response.statusCode >= 400 ? "bg-[var(--danger)]" : "bg-[var(--warning)]"
        )} />
        <span className={cn("text-2xs font-mono font-bold", getStatusColor(response.statusCode))}>
          {response.statusCode}
        </span>
        <span className="text-2xs text-[var(--fg-secondary)]">{response.statusText}</span>
      </div>

      {/* 耗时 */}
      <div className="flex items-center gap-0.5 text-[var(--fg-muted)]">
        <AppIcon name="clock" size={10} />
        <span className="text-2xs font-mono">
          {response.duration < 1000
            ? `${Math.round(response.duration)}ms`
            : `${(response.duration / 1000).toFixed(2)}s`}
        </span>
      </div>

      {/* 大小 */}
      <div className="flex items-center gap-0.5 text-[var(--fg-muted)]">
        <AppIcon name="cube" size={10} />
        <span className="text-2xs font-mono">{formatSize(response.size)}</span>
      </div>

      <div className="flex-1" />

      {/* 内容类型 */}
      <span className="text-2xs text-[var(--fg-muted)] font-mono truncate max-w-[200px]">
        {response.contentType}
      </span>
    </div>
  )
}
