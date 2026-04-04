import { cn } from "@/lib/utils"

interface ResponseHeadersProps {
  headers: Record<string, string[]>
}

export function ResponseHeaders({ headers }: ResponseHeadersProps) {
  const entries = Object.entries(headers)

  if (entries.length === 0) {
    return (
      <div className="flex items-center justify-center h-full text-2xs text-[var(--fg-muted)]">
        无响应头
      </div>
    )
  }

  return (
    <div className="overflow-auto h-full">
      <table className="w-full text-[length:var(--size-font-2xs)]">
        <thead>
          <tr className="border-b border-[var(--border-color)]">
            <th className="text-left px-3 py-1.5 text-2xs font-semibold text-[var(--fg-secondary)] uppercase bg-[var(--surface-secondary)] w-[200px]">
              Header
            </th>
            <th className="text-left px-3 py-1.5 text-2xs font-semibold text-[var(--fg-secondary)] uppercase bg-[var(--surface-secondary)]">
              Value
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
