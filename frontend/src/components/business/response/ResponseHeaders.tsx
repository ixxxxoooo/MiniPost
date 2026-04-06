import { cn } from "@/lib/utils"
import { useI18n } from "@/hooks/useI18n"

interface ResponseHeadersProps {
  headers: Record<string, string[]>
}

export function ResponseHeaders({ headers }: ResponseHeadersProps) {
  const { t } = useI18n()
  const entries = Object.entries(headers)

  if (entries.length === 0) {
    return (
      <div className="flex items-center justify-center h-full text-2xs text-[var(--fg-muted)]">
        {t("无响应头", "No response headers")}
      </div>
    )
  }

  return (
    <div className="overflow-auto h-full">
      <table className="w-full text-[length:var(--size-font-2xs)]">
        <thead>
          <tr className="border-b border-[var(--border-color)]">
            <th className="text-left px-3 py-1.5 text-2xs font-semibold text-[var(--fg-secondary)] uppercase bg-[var(--surface-secondary)] w-[200px]">
              {t("请求头", "Header")}
            </th>
            <th className="text-left px-3 py-1.5 text-2xs font-semibold text-[var(--fg-secondary)] uppercase bg-[var(--surface-secondary)]">
              {t("值", "Value")}
            </th>
          </tr>
        </thead>
        <tbody>
          {entries.map(([key, values]) => (
            <tr key={key} className="border-b border-[var(--border-subtle)] hover:bg-[var(--sidebar-hover)]">
              <td className="px-3 py-1 font-mono font-medium text-[var(--fg)] align-top">
                {key}
              </td>
              <td className="px-3 py-1 font-mono text-[var(--fg-secondary)] break-all">
                {values.join(", ")}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
