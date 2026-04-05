import { create } from "zustand"

export type Theme = "light" | "dark" | "system"
export type LayoutDirection = "vertical" | "horizontal"
export type HttpVersion = "auto" | "http1" | "http2"
export type ResponseFormatDetection = "auto" | "json"

export interface ConsoleEntry {
  id: string
  timestamp: string
  method: string
  url: string
  status?: number
  duration?: number
  size?: number
  error?: string
  requestHeaders?: Record<string, string>
  responseHeaders?: Record<string, string[]>
  responseBody?: string
}

interface UIState {
  theme: Theme
  resolved: "light" | "dark"
  sidebarWidth: number
  sidebarCollapsed: boolean
  layoutDirection: LayoutDirection
  isSending: boolean
  consoleOpen: boolean
  consoleHeight: number
  consoleLogs: ConsoleEntry[]
  settingsOpen: boolean
  fontSize: number
  scrollbarAutoHide: boolean
  followRedirects: boolean
  httpVersion: HttpVersion
  requestTimeoutMs: number
  maxResponseSizeMB: number
  sslCertificateVerification: boolean
  sslTlsKeyLog: boolean
  disableCookies: boolean
  responseFormatDetection: ResponseFormatDetection
  editingEnvironmentId: string | null
  openEnvironmentTabIds: string[]

  setTheme: (theme: Theme) => void
  setSidebarWidth: (width: number) => void
  toggleSidebar: () => void
  setLayoutDirection: (d: LayoutDirection) => void
  setIsSending: (v: boolean) => void
  toggleConsole: () => void
  setConsoleHeight: (h: number) => void
  setSettingsOpen: (v: boolean) => void
  setFontSize: (size: number) => void
  setScrollbarAutoHide: (v: boolean) => void
  setFollowRedirects: (v: boolean) => void
  setHttpVersion: (v: HttpVersion) => void
  setRequestTimeoutMs: (v: number) => void
  setMaxResponseSizeMB: (v: number) => void
  setSSLCertificateVerification: (v: boolean) => void
  setSSLTlsKeyLog: (v: boolean) => void
  setDisableCookies: (v: boolean) => void
  setResponseFormatDetection: (v: ResponseFormatDetection) => void
  setEditingEnvironmentId: (id: string | null) => void
  closeEnvironmentTab: (id: string) => void
  closeActiveEnvironmentTab: () => void
  clearEnvironmentTabs: () => void
  addConsoleRequest: (entry: Pick<ConsoleEntry, "method" | "url" | "requestHeaders">) => string
  updateConsoleResponse: (id: string, data: Pick<ConsoleEntry, "status" | "duration" | "size" | "responseHeaders" | "responseBody">) => void
  updateConsoleError: (id: string, error: string) => void
  clearConsoleLogs: () => void
}

function getSystemTheme(): "light" | "dark" {
  if (typeof window === "undefined") return "light"
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light"
}

function applyTheme(resolved: "light" | "dark") {
  const root = document.documentElement
  if (resolved === "dark") {
    root.classList.add("dark")
  } else {
    root.classList.remove("dark")
  }
}

const SCROLLBAR_AUTO_HIDE_STORAGE_KEY = "minipost:scrollbar-auto-hide"
const REQUEST_SETTINGS_STORAGE_KEY = "minipost:request-settings"

type PersistedRequestSettings = {
  followRedirects: boolean
  httpVersion: HttpVersion
  requestTimeoutMs: number
  maxResponseSizeMB: number
  sslCertificateVerification: boolean
  sslTlsKeyLog: boolean
  disableCookies: boolean
  responseFormatDetection: ResponseFormatDetection
}

function defaultRequestSettings(): PersistedRequestSettings {
  return {
    followRedirects: true,
    httpVersion: "auto",
    requestTimeoutMs: 0,
    maxResponseSizeMB: 50,
    sslCertificateVerification: true,
    sslTlsKeyLog: false,
    disableCookies: false,
    responseFormatDetection: "auto",
  }
}

function clampSettingNumber(value: number, min: number, max: number): number {
  if (!Number.isFinite(value)) return min
  return Math.max(min, Math.min(max, value))
}

function readRequestSettings(): PersistedRequestSettings {
  const defaults = defaultRequestSettings()
  if (typeof window === "undefined") return defaults
  try {
    const raw = window.localStorage.getItem(REQUEST_SETTINGS_STORAGE_KEY)
    if (!raw) return defaults
    const parsed = JSON.parse(raw) as Partial<PersistedRequestSettings>
    const httpVersion = parsed.httpVersion === "http1" || parsed.httpVersion === "http2" || parsed.httpVersion === "auto"
      ? parsed.httpVersion
      : defaults.httpVersion
    const responseFormatDetection = parsed.responseFormatDetection === "json" || parsed.responseFormatDetection === "auto"
      ? parsed.responseFormatDetection
      : defaults.responseFormatDetection

    return {
      followRedirects: parsed.followRedirects ?? defaults.followRedirects,
      httpVersion,
      requestTimeoutMs: clampSettingNumber(parsed.requestTimeoutMs ?? defaults.requestTimeoutMs, 0, 3_600_000),
      maxResponseSizeMB: clampSettingNumber(parsed.maxResponseSizeMB ?? defaults.maxResponseSizeMB, 0, 2048),
      sslCertificateVerification: parsed.sslCertificateVerification ?? defaults.sslCertificateVerification,
      sslTlsKeyLog: parsed.sslTlsKeyLog ?? defaults.sslTlsKeyLog,
      disableCookies: parsed.disableCookies ?? defaults.disableCookies,
      responseFormatDetection,
    }
  } catch {
    return defaults
  }
}

function persistRequestSettings(settings: PersistedRequestSettings) {
  if (typeof window === "undefined") return
  try {
    window.localStorage.setItem(REQUEST_SETTINGS_STORAGE_KEY, JSON.stringify(settings))
  } catch {
    // ignore persistence errors
  }
}

function readScrollbarAutoHide(): boolean {
  if (typeof window === "undefined") return true
  try {
    const raw = window.localStorage.getItem(SCROLLBAR_AUTO_HIDE_STORAGE_KEY)
    if (raw === null) return true
    return raw !== "0"
  } catch {
    return true
  }
}

function persistScrollbarAutoHide(value: boolean) {
  if (typeof window === "undefined") return
  try {
    window.localStorage.setItem(SCROLLBAR_AUTO_HIDE_STORAGE_KEY, value ? "1" : "0")
  } catch {
    // ignore persistence errors
  }
}

function applyScrollbarMode(autoHide: boolean) {
  if (typeof document === "undefined") return
  document.documentElement.classList.toggle("scrollbar-auto-hide", autoHide)
}

export const useUIStore = create<UIState>((set) => ({
  ...readRequestSettings(),
  theme: "system",
  resolved: getSystemTheme(),
  sidebarWidth: 240,
  sidebarCollapsed: false,
  layoutDirection: "vertical",
  isSending: false,
  consoleOpen: false,
  consoleHeight: 220,
  consoleLogs: [],
  settingsOpen: false,
  fontSize: 12,
  scrollbarAutoHide: readScrollbarAutoHide(),
  editingEnvironmentId: null,
  openEnvironmentTabIds: [],

  setTheme: (theme) => {
    const resolved = theme === "system" ? getSystemTheme() : theme
    applyTheme(resolved)
    set({ theme, resolved })
  },
  setSidebarWidth: (sidebarWidth) => set({ sidebarWidth }),
  toggleSidebar: () => set((s) => ({ sidebarCollapsed: !s.sidebarCollapsed })),
  setLayoutDirection: (layoutDirection) => set({ layoutDirection }),
  setIsSending: (isSending) => set({ isSending }),
  toggleConsole: () => set((s) => ({ consoleOpen: !s.consoleOpen })),
  setConsoleHeight: (consoleHeight) => set({ consoleHeight: Math.max(80, Math.min(600, consoleHeight)) }),
  setSettingsOpen: (settingsOpen) => set({ settingsOpen }),
  setFontSize: (fontSize) => set({ fontSize }),
  setScrollbarAutoHide: (scrollbarAutoHide) => {
    applyScrollbarMode(scrollbarAutoHide)
    persistScrollbarAutoHide(scrollbarAutoHide)
    set({ scrollbarAutoHide })
  },
  setFollowRedirects: (followRedirects) => set((state) => {
    const next = { ...state, followRedirects }
    persistRequestSettings({
      followRedirects: next.followRedirects,
      httpVersion: next.httpVersion,
      requestTimeoutMs: next.requestTimeoutMs,
      maxResponseSizeMB: next.maxResponseSizeMB,
      sslCertificateVerification: next.sslCertificateVerification,
      sslTlsKeyLog: next.sslTlsKeyLog,
      disableCookies: next.disableCookies,
      responseFormatDetection: next.responseFormatDetection,
    })
    return { followRedirects }
  }),
  setHttpVersion: (httpVersion) => set((state) => {
    const next = { ...state, httpVersion }
    persistRequestSettings({
      followRedirects: next.followRedirects,
      httpVersion: next.httpVersion,
      requestTimeoutMs: next.requestTimeoutMs,
      maxResponseSizeMB: next.maxResponseSizeMB,
      sslCertificateVerification: next.sslCertificateVerification,
      sslTlsKeyLog: next.sslTlsKeyLog,
      disableCookies: next.disableCookies,
      responseFormatDetection: next.responseFormatDetection,
    })
    return { httpVersion }
  }),
  setRequestTimeoutMs: (requestTimeoutMs) => set((state) => {
    const normalized = clampSettingNumber(Math.round(requestTimeoutMs), 0, 3_600_000)
    const next = { ...state, requestTimeoutMs: normalized }
    persistRequestSettings({
      followRedirects: next.followRedirects,
      httpVersion: next.httpVersion,
      requestTimeoutMs: next.requestTimeoutMs,
      maxResponseSizeMB: next.maxResponseSizeMB,
      sslCertificateVerification: next.sslCertificateVerification,
      sslTlsKeyLog: next.sslTlsKeyLog,
      disableCookies: next.disableCookies,
      responseFormatDetection: next.responseFormatDetection,
    })
    return { requestTimeoutMs: normalized }
  }),
  setMaxResponseSizeMB: (maxResponseSizeMB) => set((state) => {
    const normalized = clampSettingNumber(Math.round(maxResponseSizeMB), 0, 2048)
    const next = { ...state, maxResponseSizeMB: normalized }
    persistRequestSettings({
      followRedirects: next.followRedirects,
      httpVersion: next.httpVersion,
      requestTimeoutMs: next.requestTimeoutMs,
      maxResponseSizeMB: next.maxResponseSizeMB,
      sslCertificateVerification: next.sslCertificateVerification,
      sslTlsKeyLog: next.sslTlsKeyLog,
      disableCookies: next.disableCookies,
      responseFormatDetection: next.responseFormatDetection,
    })
    return { maxResponseSizeMB: normalized }
  }),
  setSSLCertificateVerification: (sslCertificateVerification) => set((state) => {
    const next = { ...state, sslCertificateVerification }
    persistRequestSettings({
      followRedirects: next.followRedirects,
      httpVersion: next.httpVersion,
      requestTimeoutMs: next.requestTimeoutMs,
      maxResponseSizeMB: next.maxResponseSizeMB,
      sslCertificateVerification: next.sslCertificateVerification,
      sslTlsKeyLog: next.sslTlsKeyLog,
      disableCookies: next.disableCookies,
      responseFormatDetection: next.responseFormatDetection,
    })
    return { sslCertificateVerification }
  }),
  setSSLTlsKeyLog: (sslTlsKeyLog) => set((state) => {
    const next = { ...state, sslTlsKeyLog }
    persistRequestSettings({
      followRedirects: next.followRedirects,
      httpVersion: next.httpVersion,
      requestTimeoutMs: next.requestTimeoutMs,
      maxResponseSizeMB: next.maxResponseSizeMB,
      sslCertificateVerification: next.sslCertificateVerification,
      sslTlsKeyLog: next.sslTlsKeyLog,
      disableCookies: next.disableCookies,
      responseFormatDetection: next.responseFormatDetection,
    })
    return { sslTlsKeyLog }
  }),
  setDisableCookies: (disableCookies) => set((state) => {
    const next = { ...state, disableCookies }
    persistRequestSettings({
      followRedirects: next.followRedirects,
      httpVersion: next.httpVersion,
      requestTimeoutMs: next.requestTimeoutMs,
      maxResponseSizeMB: next.maxResponseSizeMB,
      sslCertificateVerification: next.sslCertificateVerification,
      sslTlsKeyLog: next.sslTlsKeyLog,
      disableCookies: next.disableCookies,
      responseFormatDetection: next.responseFormatDetection,
    })
    return { disableCookies }
  }),
  setResponseFormatDetection: (responseFormatDetection) => set((state) => {
    const next = { ...state, responseFormatDetection }
    persistRequestSettings({
      followRedirects: next.followRedirects,
      httpVersion: next.httpVersion,
      requestTimeoutMs: next.requestTimeoutMs,
      maxResponseSizeMB: next.maxResponseSizeMB,
      sslCertificateVerification: next.sslCertificateVerification,
      sslTlsKeyLog: next.sslTlsKeyLog,
      disableCookies: next.disableCookies,
      responseFormatDetection: next.responseFormatDetection,
    })
    return { responseFormatDetection }
  }),
  setEditingEnvironmentId: (editingEnvironmentId) => set((state) => {
    if (!editingEnvironmentId) {
      return { editingEnvironmentId: null }
    }
    if (state.openEnvironmentTabIds.includes(editingEnvironmentId)) {
      return { editingEnvironmentId }
    }
    return {
      editingEnvironmentId,
      openEnvironmentTabIds: [...state.openEnvironmentTabIds, editingEnvironmentId],
    }
  }),
  closeEnvironmentTab: (id) => set((state) => {
    const index = state.openEnvironmentTabIds.indexOf(id)
    const openEnvironmentTabIds = state.openEnvironmentTabIds.filter((tabId) => tabId !== id)

    if (state.editingEnvironmentId !== id) {
      return index === -1 ? {} : { openEnvironmentTabIds }
    }

    const nextEditingEnvironmentId = index === -1
      ? null
      : openEnvironmentTabIds[index] ?? openEnvironmentTabIds[index - 1] ?? null

    return {
      editingEnvironmentId: nextEditingEnvironmentId,
      openEnvironmentTabIds,
    }
  }),
  closeActiveEnvironmentTab: () => set((state) => {
    const activeId = state.editingEnvironmentId
    if (!activeId) return {}

    const index = state.openEnvironmentTabIds.indexOf(activeId)
    const openEnvironmentTabIds = state.openEnvironmentTabIds.filter((tabId) => tabId !== activeId)
    const nextEditingEnvironmentId = index === -1
      ? null
      : openEnvironmentTabIds[index] ?? openEnvironmentTabIds[index - 1] ?? null

    return {
      editingEnvironmentId: nextEditingEnvironmentId,
      openEnvironmentTabIds,
    }
  }),
  clearEnvironmentTabs: () => set({
    editingEnvironmentId: null,
    openEnvironmentTabIds: [],
  }),
  addConsoleRequest: (entry) => {
    const id = crypto.randomUUID()
    set((s) => ({
      consoleLogs: [
        ...s.consoleLogs,
        { ...entry, id, timestamp: new Date().toISOString() },
      ].slice(-200),
    }))
    return id
  },
  updateConsoleResponse: (id, data) => set((s) => ({
    consoleLogs: s.consoleLogs.map((log) =>
      log.id === id ? { ...log, ...data } : log
    ),
  })),
  updateConsoleError: (id, error) => set((s) => ({
    consoleLogs: s.consoleLogs.map((log) =>
      log.id === id ? { ...log, error } : log
    ),
  })),
  clearConsoleLogs: () => set({ consoleLogs: [] }),
}))

if (typeof window !== "undefined") {
  applyTheme(getSystemTheme())
  applyScrollbarMode(readScrollbarAutoHide())
  window.matchMedia("(prefers-color-scheme: dark)").addEventListener("change", () => {
    const store = useUIStore.getState()
    if (store.theme === "system") {
      const resolved = getSystemTheme()
      applyTheme(resolved)
      useUIStore.setState({ resolved })
    }
  })
}
