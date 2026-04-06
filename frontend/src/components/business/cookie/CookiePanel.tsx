import { useEffect, useMemo, useState } from "react"
import { createPortal } from "react-dom"
import { AppIcon } from "@/components/ui/icon"
import { useCookieStore, type CookieItem } from "@/stores/cookieStore"
import { useTabStore, getProjectActiveTabFromState } from "@/stores/tabStore"
import { cn } from "@/lib/utils"
import { useI18n } from "@/hooks/useI18n"

function normalizeDomain(domain: string): string {
  let value = domain.trim().toLowerCase()
  if (!value) return ""
  if (value.includes("://")) {
    try {
      value = new URL(value).hostname.toLowerCase()
    } catch {
      // ignore parse error, continue fallback cleanup
    }
  }
  value = value.replace(/^\./, "")
  value = value.split("/")[0]
  value = value.split(":")[0]
  return value
}

function inferHostFromUrl(url: string): string {
  try {
    return new URL(url).hostname
  } catch {
    return ""
  }
}

function DomainListItem({
  domain,
  selected,
  count,
  onClick,
}: {
  domain: string
  selected: boolean
  count: number
  onClick: () => void
}) {
  const { t } = useI18n()
  return (
    <button
      type="button"
      className={cn(
        "w-full rounded-[7px] px-2.5 py-1.5 text-left text-[11px] transition-colors",
        "flex items-center gap-2",
        selected ? "bg-[var(--selected-bg)] text-[var(--fg)]" : "text-[var(--fg)] hover:bg-[var(--sidebar-hover)]"
      )}
      onClick={onClick}
      title={domain}
    >
      <AppIcon name="globe" size={11} className={selected ? "text-[var(--accent)]" : "text-[var(--fg-muted)]"} />
      <span className="truncate flex-1">{domain || t("无域名", "No Domain")}</span>
      <span className="text-[10px] text-[var(--fg-muted)]">{count}</span>
    </button>
  )
}

function CookieRow({
  cookie,
  onUpdate,
  onRemove,
}: {
  cookie: CookieItem
  onUpdate: (updates: Partial<CookieItem>) => void
  onRemove: () => void
}) {
  const { t } = useI18n()
  return (
    <tr className="group border-b border-[var(--border-subtle)] last:border-b-0">
      <td className="px-2 py-1 text-center">
        <input
          type="checkbox"
          checked={cookie.enabled}
          onChange={(e) => onUpdate({ enabled: e.target.checked })}
          className="h-3 w-3 rounded accent-[var(--accent)]"
        />
      </td>
      <td className="px-2 py-1">
        <input
          value={cookie.name}
          onChange={(e) => onUpdate({ name: e.target.value })}
          placeholder={t("名称", "Name")}
          className="h-6 w-full rounded-[5px] border border-[var(--button-border)] bg-[var(--surface)] px-2 text-[11px] text-[var(--fg)] outline-none focus:border-[var(--accent)]"
        />
      </td>
      <td className="px-2 py-1">
        <input
          value={cookie.value}
          onChange={(e) => onUpdate({ value: e.target.value })}
          placeholder={t("值", "Value")}
          className="h-6 w-full rounded-[5px] border border-[var(--button-border)] bg-[var(--surface)] px-2 text-[11px] text-[var(--fg)] outline-none focus:border-[var(--accent)]"
        />
      </td>
      <td className="px-2 py-1">
        <input
          value={cookie.path}
          onChange={(e) => onUpdate({ path: e.target.value })}
          placeholder="/"
          className="h-6 w-full rounded-[5px] border border-[var(--button-border)] bg-[var(--surface)] px-2 text-[11px] text-[var(--fg)] outline-none focus:border-[var(--accent)]"
        />
      </td>
      <td className="px-2 py-1">
        <input
          value={cookie.expires}
          onChange={(e) => onUpdate({ expires: e.target.value })}
          placeholder={t("会话", "Session")}
          className="h-6 w-full rounded-[5px] border border-[var(--button-border)] bg-[var(--surface)] px-2 text-[11px] text-[var(--fg)] outline-none focus:border-[var(--accent)]"
        />
      </td>
      <td className="px-2 py-1 text-center">
        <input
          type="checkbox"
          checked={cookie.secure}
          onChange={(e) => onUpdate({ secure: e.target.checked })}
          className="h-3 w-3 rounded accent-[var(--accent)]"
          title="Secure"
        />
      </td>
      <td className="px-2 py-1 text-center">
        <input
          type="checkbox"
          checked={cookie.httpOnly}
          onChange={(e) => onUpdate({ httpOnly: e.target.checked })}
          className="h-3 w-3 rounded accent-[var(--accent)]"
          title="HttpOnly"
        />
      </td>
      <td className="px-1 py-1 text-center">
        <button
          type="button"
          className="inline-flex h-5 w-5 items-center justify-center rounded-[4px] text-[var(--fg-muted)] opacity-0 transition-all hover:bg-[var(--sidebar-hover)] hover:text-[var(--danger)] group-hover:opacity-100"
          onClick={onRemove}
          title={t("删除", "Delete")}
        >
          <AppIcon name="delete" size={10} />
        </button>
      </td>
    </tr>
  )
}

export function CookiePanel() {
  const { t } = useI18n()
  const activeTab = useTabStore(getProjectActiveTabFromState)
  const {
    cookies,
    cookiePanelOpen,
    setCookiePanelOpen,
    addCookie,
    updateCookie,
    removeCookie,
    clearCookies,
    clearDomainCookies,
  } = useCookieStore()
  const [selectedDomain, setSelectedDomain] = useState("")
  const [search, setSearch] = useState("")
  const [domainInput, setDomainInput] = useState("")

  const activeHost = useMemo(() => inferHostFromUrl(activeTab?.request.url ?? ""), [activeTab?.request.url])
  const domainGroups = useMemo(() => {
    const map = new Map<string, CookieItem[]>()
    cookies.forEach((cookie) => {
      const key = normalizeDomain(cookie.domain || activeHost || "")
      const list = map.get(key) ?? []
      list.push(cookie)
      map.set(key, list)
    })
    if (activeHost && !map.has(activeHost)) {
      map.set(activeHost, [])
    }
    return Array.from(map.entries()).sort((a, b) => a[0].localeCompare(b[0]))
  }, [activeHost, cookies])

  useEffect(() => {
    if (!cookiePanelOpen) return
    if (selectedDomain && domainGroups.some(([domain]) => domain === selectedDomain)) return
    if (activeHost && domainGroups.some(([domain]) => domain === activeHost)) {
      setSelectedDomain(activeHost)
      return
    }
    setSelectedDomain(domainGroups[0]?.[0] ?? activeHost ?? "")
  }, [activeHost, cookiePanelOpen, domainGroups, selectedDomain])

  useEffect(() => {
    if (!cookiePanelOpen) return
    setDomainInput(activeHost || "")
  }, [activeHost, cookiePanelOpen])

  useEffect(() => {
    if (!cookiePanelOpen) return
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault()
        setCookiePanelOpen(false)
      }
    }
    window.addEventListener("keydown", onKeyDown)
    return () => window.removeEventListener("keydown", onKeyDown)
  }, [cookiePanelOpen, setCookiePanelOpen])

  if (!cookiePanelOpen) return null

  const currentDomain = selectedDomain || activeHost || domainGroups[0]?.[0] || ""
  const currentCookies = cookies.filter((cookie) => normalizeDomain(cookie.domain || activeHost || "") === currentDomain)
  const q = search.trim().toLowerCase()
  const filteredCookies = q
    ? currentCookies.filter((cookie) => {
      return (
        cookie.name.toLowerCase().includes(q)
        || cookie.value.toLowerCase().includes(q)
        || cookie.path.toLowerCase().includes(q)
      )
    })
    : currentCookies

  const handleAddCookie = () => {
    const domain = currentDomain || activeHost
    addCookie({
      domain,
      path: "/",
      name: "",
      value: "",
      expires: "",
      enabled: true,
    })
  }

  const handleAddDomain = () => {
    const domain = normalizeDomain(domainInput)
    if (!domain) return
    setSelectedDomain(domain)
    setDomainInput("")
    const exists = cookies.some((cookie) => normalizeDomain(cookie.domain || "") === domain)
    if (!exists) {
      addCookie({
        domain,
        path: "/",
        name: "",
        value: "",
        expires: "",
        enabled: true,
      })
    }
  }

  return createPortal(
    <div className="fixed inset-0 z-[320] flex items-center justify-center" onClick={() => setCookiePanelOpen(false)}>
      <div className="absolute inset-0 bg-black/35 backdrop-blur-[1px]" />
      <div
        className={cn(
          "relative z-[321] flex h-[min(760px,88vh)] w-[min(1280px,92vw)] overflow-hidden rounded-[14px] border",
          "border-[var(--border-color)] bg-[var(--surface)] shadow-[var(--shadow-lg)]"
        )}
        onClick={(e) => e.stopPropagation()}
      >
        <aside className="flex w-[300px] flex-col border-r border-[var(--border-color)] bg-[var(--surface-secondary)]">
          <div className="flex h-[44px] items-center gap-2 border-b border-[var(--border-color)] px-3">
            <AppIcon name="cookie" size={13} className="text-[var(--fg-secondary)]" />
            <span className="flex-1 text-[12px] font-semibold text-[var(--fg)]">{t("Cookie 管理器", "Cookie Jar")}</span>
          </div>
          <div className="space-y-2 p-2.5">
            <div className="flex items-center gap-1.5">
              <input
                value={domainInput}
                onChange={(e) => setDomainInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    e.preventDefault()
                    handleAddDomain()
                  }
                }}
                placeholder={t("输入域名", "Type a domain name")}
                className="h-7 flex-1 rounded-[7px] border border-[var(--button-border)] bg-[var(--surface)] px-2 text-[11px] text-[var(--fg)] outline-none placeholder:text-[var(--fg-muted)] focus:border-[var(--accent)]"
              />
              <button
                type="button"
                className="flex h-7 items-center gap-1 rounded-[7px] border border-[var(--button-border)] bg-[var(--surface)] px-2.5 text-[11px] text-[var(--fg-secondary)] transition-colors hover:bg-[var(--button-bg)] hover:text-[var(--fg)]"
                onClick={handleAddDomain}
                title={t("新增域名", "Add domain")}
              >
                {t("添加", "Add")}
              </button>
            </div>
            <div className="relative">
              <AppIcon name="search" size={11} className="pointer-events-none absolute left-2 top-1/2 -translate-y-1/2 text-[var(--fg-muted)]" />
              <input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder={t("搜索 Cookie", "Search cookies")}
                className="h-7 w-full rounded-[7px] border border-[var(--button-border)] bg-[var(--surface)] pl-7 pr-2 text-[11px] text-[var(--fg)] outline-none placeholder:text-[var(--fg-muted)] focus:border-[var(--accent)]"
              />
            </div>
          </div>
          <div className="min-h-0 flex-1 space-y-0.5 overflow-y-auto px-2 pb-2">
            {domainGroups.length === 0 ? (
              <div className="px-2 py-6 text-center text-[11px] text-[var(--fg-muted)]">{t("暂无域名", "No domains")}</div>
            ) : (
              domainGroups.map(([domain, items]) => (
                <DomainListItem
                  key={domain || "__empty__"}
                  domain={domain}
                  count={items.length}
                  selected={domain === currentDomain}
                  onClick={() => setSelectedDomain(domain)}
                />
              ))
            )}
          </div>
        </aside>

        <section className="flex min-w-0 flex-1 flex-col">
          <div className="flex h-[44px] items-center justify-between border-b border-[var(--border-color)] px-3.5">
            <div className="flex min-w-0 items-center gap-2">
              <AppIcon name="globe" size={12} className="text-[var(--fg-secondary)]" />
              <span className="truncate text-[12px] font-semibold text-[var(--fg)]">{currentDomain || t("无域名", "No Domain")}</span>
              <span className="text-[11px] text-[var(--fg-muted)]">({filteredCookies.length})</span>
            </div>
            <div className="flex items-center gap-1">
              <button
                type="button"
                className="flex h-6 items-center gap-1 rounded-[6px] border border-[var(--button-border)] bg-[var(--surface)] px-2 text-[11px] text-[var(--fg)] transition-colors hover:bg-[var(--button-bg)]"
                onClick={handleAddCookie}
              >
                <AppIcon name="add" size={11} />
                {t("添加 Cookie", "Add Cookie")}
              </button>
              <button
                type="button"
                className="flex h-6 items-center gap-1 rounded-[6px] border border-[var(--button-border)] bg-[var(--surface)] px-2 text-[11px] text-[var(--fg-secondary)] transition-colors hover:bg-[var(--button-bg)]"
                onClick={() => currentDomain && clearDomainCookies(currentDomain)}
                disabled={!currentDomain}
              >
                {t("清空当前域名", "Clear Domain")}
              </button>
              <button
                type="button"
                className="flex h-6 items-center gap-1 rounded-[6px] border border-[var(--button-border)] bg-[var(--surface)] px-2 text-[11px] text-[var(--fg-secondary)] transition-colors hover:bg-[var(--button-bg)]"
                onClick={clearCookies}
              >
                {t("清空全部", "Clear All")}
              </button>
              <button
                type="button"
                className="ml-1 flex h-6 w-6 items-center justify-center rounded-[6px] text-[var(--fg-muted)] transition-colors hover:bg-[var(--button-bg)] hover:text-[var(--fg)]"
                onClick={() => setCookiePanelOpen(false)}
                title={t("关闭", "Close")}
              >
                <AppIcon name="clear" size={11} />
              </button>
            </div>
          </div>

          <div className="min-h-0 flex-1 overflow-auto">
            {filteredCookies.length === 0 ? (
              <div className="flex h-full items-center justify-center text-[11px] text-[var(--fg-muted)]">
                {t("当前域名暂无 Cookies，点击右上角 添加 Cookie", "No cookies for this domain. Click Add Cookie in the top-right.")}
              </div>
            ) : (
              <table className="w-full border-collapse text-[11px]">
                <thead>
                  <tr className="border-b border-[var(--border-color)] bg-[var(--surface-secondary)]/60">
                    <th className="w-8 px-2 py-1.5 text-center font-semibold text-[var(--fg-secondary)]">{t("启用", "On")}</th>
                    <th className="min-w-[140px] px-2 py-1.5 text-left font-semibold text-[var(--fg-secondary)]">{t("名称", "Name")}</th>
                    <th className="min-w-[190px] px-2 py-1.5 text-left font-semibold text-[var(--fg-secondary)]">{t("值", "Value")}</th>
                    <th className="min-w-[90px] px-2 py-1.5 text-left font-semibold text-[var(--fg-secondary)]">{t("路径", "Path")}</th>
                    <th className="min-w-[150px] px-2 py-1.5 text-left font-semibold text-[var(--fg-secondary)]">{t("过期时间", "Expires")}</th>
                    <th className="w-[60px] px-2 py-1.5 text-center font-semibold text-[var(--fg-secondary)]">Secure</th>
                    <th className="w-[70px] px-2 py-1.5 text-center font-semibold text-[var(--fg-secondary)]">HttpOnly</th>
                    <th className="w-8 px-1 py-1.5" />
                  </tr>
                </thead>
                <tbody>
                  {filteredCookies.map((cookie) => (
                    <CookieRow
                      key={cookie.id}
                      cookie={cookie}
                      onUpdate={(updates) => updateCookie(cookie.id, updates)}
                      onRemove={() => removeCookie(cookie.id)}
                    />
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </section>
      </div>
    </div>,
    document.body
  )
}
