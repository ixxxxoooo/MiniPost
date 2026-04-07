import { useI18n } from "@/hooks/useI18n"

interface ParsedCookie {
  name: string
  value: string
  attributes: string
}

interface ResponseCookiesProps {
  cookies: ParsedCookie[]
}

export function ResponseCookies({ cookies }: ResponseCookiesProps) {
  const { t } = useI18n()
  if (cookies.length === 0) {
    return (
      <div className="flex h-full items-center justify-center text-2xs text-[var(--fg-muted)]">
        {t("无 Cookies", "No cookies")}
      </div>
    )
  }

  return (
    <div className="h-full overflow-auto">
      <table className="w-full text-[length:var(--size-font-2xs)]">
        <thead>
          <tr className="border-b border-[var(--border-color)]">
            <th className="select-text w-[220px] bg-[var(--surface-secondary)] px-3 py-1.5 text-left text-2xs font-semibold uppercase text-[var(--fg-secondary)]">
              Name
            </th>
            <th className="select-text w-[260px] bg-[var(--surface-secondary)] px-3 py-1.5 text-left text-2xs font-semibold uppercase text-[var(--fg-secondary)]">
              Value
            </th>
            <th className="select-text bg-[var(--surface-secondary)] px-3 py-1.5 text-left text-2xs font-semibold uppercase text-[var(--fg-secondary)]">
              Attributes
            </th>
          </tr>
        </thead>
        <tbody>
          {cookies.map((cookie, idx) => (
            <tr key={`${cookie.name}-${idx}`} className="border-b border-[var(--border-subtle)] hover:bg-[var(--sidebar-hover)]">
              <td className="select-text px-3 py-1 font-mono font-medium text-[var(--fg)]">{cookie.name}</td>
              <td className="select-text px-3 py-1 break-all font-mono text-[var(--fg-secondary)]">{cookie.value}</td>
              <td className="select-text px-3 py-1 break-all text-[var(--fg-secondary)]">{cookie.attributes || "-"}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
