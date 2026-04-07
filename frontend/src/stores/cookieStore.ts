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
  importCookieHeader: (rawHeader: string, domain: string, path?: string) => number
  absorbResponseCookies: (requestUrl: string, headers: Record<string, string[]>) => void
  getCookieHeader: (url: string) => string
}

const COOKIE_STORAGE_KEY = "minipost:cookies"

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

function sanitizeCookie(input: unknown): CookieItem | null {
  if (!input || typeof input !== "object") return null
  const raw = input as Partial<CookieItem>

  const name = typeof raw.name === "string" ? raw.name : ""
  const value = typeof raw.value === "string" ? raw.value : ""
  const domain = typeof raw.domain === "string" ? raw.domain : ""
  const path = typeof raw.path === "string" && raw.path.trim() ? raw.path : "/"
  const expires = typeof raw.expires === "string" ? raw.expires : ""
  const id = typeof raw.id === "string" && raw.id.trim() ? raw.id : crypto.randomUUID()

  return {
    id,
    domain,
    name,
    value,
    path,
    expires,
    secure: Boolean(raw.secure),
    httpOnly: Boolean(raw.httpOnly),
    enabled: raw.enabled ?? true,
  }
}

function readPersistedCookies(): CookieItem[] {
  if (typeof window === "undefined") return []
  try {
    const raw = window.localStorage.getItem(COOKIE_STORAGE_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw) as unknown
    if (!Array.isArray(parsed)) return []
    const cookies = parsed
      .map((item) => sanitizeCookie(item))
      .filter((item): item is CookieItem => item !== null)
      .filter((item) => !isExpired(item))
    return cookies
  } catch {
    return []
  }
}

function persistCookies(cookies: CookieItem[]) {
  if (typeof window === "undefined") return
  try {
    window.localStorage.setItem(COOKIE_STORAGE_KEY, JSON.stringify(cookies))
  } catch {
    // ignore persistence errors
  }
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

function parseRequestUrl(rawUrl: string): URL | null {
  const value = rawUrl.trim()
  if (!value) return null
  try {
    return new URL(value)
  } catch {
    try {
      return new URL(`http://${value}`)
    } catch {
      return null
    }
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

function parseCookieHeader(rawHeader: string): Array<{ name: string; value: string }> {
  const cleaned = rawHeader
    .trim()
    .replace(/^[*\-•]\s*/, "")
    .replace(/^cookie:\s*/i, "")
    .replace(/\r?\n/g, "; ")
  if (!cleaned) return []

  const attributeNames = new Set(["path", "domain", "expires", "max-age", "secure", "httponly", "samesite", "priority", "partitioned"])
  const pairs: Array<{ name: string; value: string }> = []

  cleaned.split(";").forEach((segment) => {
    const token = segment.trim()
    if (!token) return
    const sep = token.indexOf("=")
    if (sep <= 0) return
    const name = token.slice(0, sep).trim()
    if (!name || attributeNames.has(name.toLowerCase())) return
    pairs.push({ name, value: token.slice(sep + 1).trim() })
  })

  return pairs
}

export const useCookieStore = create<CookieState>((set, get) => ({
  cookies: readPersistedCookies(),
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
    set((s) => {
      const next = [...s.cookies, item]
      persistCookies(next)
      return { cookies: next }
    })
  },

  updateCookie: (id, updates) => {
    set((s) => ({
      cookies: (() => {
        const next = s.cookies.map((c) => (c.id === id ? { ...c, ...updates } : c))
        persistCookies(next)
        return next
      })(),
    }))
  },

  removeCookie: (id) => {
    set((s) => {
      const next = s.cookies.filter((c) => c.id !== id)
      persistCookies(next)
      return { cookies: next }
    })
  },

  clearCookies: () => {
    persistCookies([])
    set({ cookies: [] })
  },
  clearDomainCookies: (domain) => set((s) => ({
    cookies: (() => {
      const next = s.cookies.filter((cookie) => normalizeDomain(cookie.domain) !== normalizeDomain(domain))
      persistCookies(next)
      return next
    })(),
  })),
  importCookieHeader: (rawHeader, domain, path = "/") => {
    const normalizedDomain = normalizeDomain(domain)
    if (!normalizedDomain) return 0

    const parsed = parseCookieHeader(rawHeader)
    if (parsed.length === 0) return 0

    let importedCount = 0
    set((s) => {
      let next = [...s.cookies]
      parsed.forEach((item) => {
        const candidate: Omit<CookieItem, "id"> = {
          domain: normalizedDomain,
          name: item.name,
          value: item.value,
          path: path || "/",
          expires: "",
          secure: false,
          httpOnly: false,
          enabled: true,
        }
        next = upsertCookie(next, candidate)
        importedCount += 1
      })
      persistCookies(next)
      return { cookies: next }
    })
    return importedCount
  },

  absorbResponseCookies: (requestUrl, headers) => {
    const parsedRequestURL = parseRequestUrl(requestUrl)
    if (!parsedRequestURL) return
    const requestHost = parsedRequestURL.hostname

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
      persistCookies(next)
      return { cookies: next }
    })
  },

  getCookieHeader: (url: string) => {
    const parsedURL = parseRequestUrl(url)
    if (!parsedURL) return ""

    const { cookies } = get()
    const matching = cookies.filter((c) => {
      if (!c.enabled || !c.name) return false
      if (isExpired(c)) return false
      if (c.domain && !parsedURL.hostname.endsWith(normalizeDomain(c.domain))) return false
      if (c.path && !parsedURL.pathname.startsWith(c.path)) return false
      if (c.secure && parsedURL.protocol !== "https:") return false
      return true
    })
    return matching.map((c) => `${c.name}=${c.value}`).join("; ")
  },
}))
