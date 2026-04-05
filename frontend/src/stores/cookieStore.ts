import { create } from "zustand"

export interface CookieItem {
  id: string
  domain: string
  name: string
  value: string
  path: string
  expires: string
  secure: boolean
  httpOnly: boolean
  enabled: boolean
}

interface CookieState {
  cookies: CookieItem[]
  cookiePanelOpen: boolean

  toggleCookiePanel: () => void
  setCookiePanelOpen: (v: boolean) => void
  addCookie: (cookie: Partial<CookieItem>) => void
  updateCookie: (id: string, updates: Partial<CookieItem>) => void
  removeCookie: (id: string) => void
  clearCookies: () => void
  clearDomainCookies: (domain: string) => void
  absorbResponseCookies: (requestUrl: string, headers: Record<string, string[]>) => void
  getCookieHeader: (url: string) => string
}

function isExpired(cookie: Pick<CookieItem, "expires">): boolean {
  if (!cookie.expires) return false
  const ts = Date.parse(cookie.expires)
  if (Number.isNaN(ts)) return false
  return ts <= Date.now()
}

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

function parseSetCookie(
  line: string,
  requestHost: string
): Omit<CookieItem, "id" | "enabled"> | null {
  const parts = line.split(";").map((part) => part.trim()).filter(Boolean)
  if (parts.length === 0) return null
  const first = parts[0]
  const sep = first.indexOf("=")
  if (sep <= 0) return null

  const name = first.slice(0, sep).trim()
  const value = first.slice(sep + 1).trim()
  if (!name) return null

  let domain = requestHost
  let path = "/"
  let expires = ""
  let secure = false
  let httpOnly = false

  for (let i = 1; i < parts.length; i += 1) {
    const attr = parts[i]
    const attrSep = attr.indexOf("=")
    const attrName = (attrSep >= 0 ? attr.slice(0, attrSep) : attr).trim().toLowerCase()
    const attrValue = (attrSep >= 0 ? attr.slice(attrSep + 1) : "").trim()

    if (attrName === "domain" && attrValue) {
      domain = attrValue.replace(/^\./, "").toLowerCase()
      continue
    }
    if (attrName === "path" && attrValue) {
      path = attrValue
      continue
    }
    if (attrName === "expires" && attrValue) {
      expires = attrValue
      continue
    }
    if (attrName === "max-age" && attrValue) {
      const seconds = Number.parseInt(attrValue, 10)
      if (!Number.isNaN(seconds)) {
        expires = new Date(Date.now() + seconds * 1000).toUTCString()
      }
      continue
    }
    if (attrName === "secure") {
      secure = true
      continue
    }
    if (attrName === "httponly") {
      httpOnly = true
    }
  }

  return {
    domain,
    name,
    value,
    path,
    expires,
    secure,
    httpOnly,
  }
}

function upsertCookie(list: CookieItem[], nextCookie: Omit<CookieItem, "id">): CookieItem[] {
  const idx = list.findIndex((cookie) =>
    normalizeDomain(cookie.domain) === normalizeDomain(nextCookie.domain)
    && cookie.path === nextCookie.path
    && cookie.name === nextCookie.name
  )
  if (idx === -1) {
    return [...list, { ...nextCookie, id: crypto.randomUUID() }]
  }
  const existing = list[idx]
  const updated = [...list]
  updated[idx] = {
    ...existing,
    ...nextCookie,
    id: existing.id,
  }
  return updated
}

export const useCookieStore = create<CookieState>((set, get) => ({
  cookies: [],
  cookiePanelOpen: false,

  toggleCookiePanel: () => set((s) => ({ cookiePanelOpen: !s.cookiePanelOpen })),
  setCookiePanelOpen: (cookiePanelOpen) => set({ cookiePanelOpen }),

  addCookie: (cookie) => {
    const item: CookieItem = {
      id: crypto.randomUUID(),
      domain: cookie.domain ?? "",
      name: cookie.name ?? "",
      value: cookie.value ?? "",
      path: cookie.path ?? "/",
      expires: cookie.expires ?? "",
      secure: cookie.secure ?? false,
      httpOnly: cookie.httpOnly ?? false,
      enabled: cookie.enabled ?? true,
    }
    set((s) => ({ cookies: [...s.cookies, item] }))
  },

  updateCookie: (id, updates) => {
    set((s) => ({
      cookies: s.cookies.map((c) => (c.id === id ? { ...c, ...updates } : c)),
    }))
  },

  removeCookie: (id) => {
    set((s) => ({ cookies: s.cookies.filter((c) => c.id !== id) }))
  },

  clearCookies: () => set({ cookies: [] }),
  clearDomainCookies: (domain) => set((s) => ({
    cookies: s.cookies.filter((cookie) => normalizeDomain(cookie.domain) !== normalizeDomain(domain)),
  })),

  absorbResponseCookies: (requestUrl, headers) => {
    let requestHost = ""
    try {
      requestHost = new URL(requestUrl).hostname
    } catch {
      return
    }

    const setCookieValues: string[] = []
    Object.entries(headers).forEach(([key, values]) => {
      if (key.toLowerCase() === "set-cookie") {
        setCookieValues.push(...values)
      }
    })
    if (setCookieValues.length === 0) return

    set((s) => {
      let next = [...s.cookies]
      for (const line of setCookieValues) {
        const parsed = parseSetCookie(line, requestHost)
        if (!parsed) continue
        const candidate: Omit<CookieItem, "id"> = {
          ...parsed,
          enabled: true,
        }
        if (isExpired(candidate)) {
          next = next.filter((cookie) => !(
            normalizeDomain(cookie.domain) === normalizeDomain(candidate.domain)
            && cookie.path === candidate.path
            && cookie.name === candidate.name
          ))
          continue
        }
        next = upsertCookie(next, candidate)
      }
      return { cookies: next }
    })
  },

  getCookieHeader: (url: string) => {
    try {
      const u = new URL(url)
      const { cookies } = get()
      const matching = cookies.filter((c) => {
        if (!c.enabled || !c.name) return false
        if (isExpired(c)) return false
        if (c.domain && !u.hostname.endsWith(normalizeDomain(c.domain))) return false
        if (c.path && !u.pathname.startsWith(c.path)) return false
        if (c.secure && u.protocol !== "https:") return false
        return true
      })
      return matching.map((c) => `${c.name}=${c.value}`).join("; ")
    } catch {
      return ""
    }
  },
}))
