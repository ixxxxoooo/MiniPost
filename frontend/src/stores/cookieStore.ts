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
  getCookieHeader: (url: string) => string
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

  getCookieHeader: (url: string) => {
    try {
      const u = new URL(url)
      const { cookies } = get()
      const matching = cookies.filter((c) => {
        if (!c.enabled || !c.name) return false
        if (c.domain && !u.hostname.endsWith(c.domain.replace(/^\./, ""))) return false
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
