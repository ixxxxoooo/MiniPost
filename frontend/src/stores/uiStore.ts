import { create } from "zustand"
import { applyDocumentLocale, detectSystemLocale, normalizeLocale, type Locale } from "@/lib/locale"

export type Theme = "light" | "dark" | "system"
export type LayoutDirection = "vertical" | "horizontal"
export type HttpVersion = "auto" | "http1" | "http2"
export type ResponseFormatDetection = "auto" | "json"
export type WorkspaceView = "project" | "home"

export interface ConsoleEntry {
  id: string
  timestamp: string
  method: string
  url: string
  requestBody?: string
  requestProtocol?: string
  status?: number
  statusText?: string
  duration?: number
  size?: number
  responseProtocol?: string
  warnings?: string[]
  error?: string
  requestHeaders?: Record<string, string>
  responseHeaders?: Record<string, string[]>
  responseBody?: string
}

interface UIState {
  locale: Locale
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
  sendNoCacheHeader: boolean
  sendPostmanTokenHeader: boolean
  responseFormatDetection: ResponseFormatDetection
  alwaysDiscardUnsavedOnClose: boolean
  alwaysSaveUnsavedOnClose: boolean
  autoBackupEnabled: boolean
  autoBackupIntervalMinutes: number
  editingEnvironmentId: string | null
  openEnvironmentTabIds: string[]
  workspaceView: WorkspaceView

  setTheme: (theme: Theme) => void
  setLocale: (locale: Locale) => void
  setSidebarWidth: (width: number) => void
  setSidebarCollapsed: (collapsed: boolean) => void
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
  setSendNoCacheHeader: (v: boolean) => void
  setSendPostmanTokenHeader: (v: boolean) => void
  setResponseFormatDetection: (v: ResponseFormatDetection) => void
  setAlwaysDiscardUnsavedOnClose: (v: boolean) => void
  setAlwaysSaveUnsavedOnClose: (v: boolean) => void
  setAutoBackupEnabled: (v: boolean) => void
  setAutoBackupIntervalMinutes: (v: number) => void
  setEditingEnvironmentId: (id: string | null) => void
  closeEnvironmentTab: (id: string) => void
  closeActiveEnvironmentTab: () => void
  clearEnvironmentTabs: () => void
  setWorkspaceView: (view: WorkspaceView) => void
  addConsoleRequest: (entry: Pick<ConsoleEntry, "method" | "url" | "requestHeaders" | "requestBody" | "requestProtocol">) => string
  updateConsoleResponse: (id: string, data: Pick<ConsoleEntry, "status" | "statusText" | "duration" | "size" | "responseHeaders" | "responseBody" | "responseProtocol" | "warnings">) => void
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
const THEME_STORAGE_KEY = "minipost:theme"
const LOCALE_STORAGE_KEY = "minipost:locale"
const FONT_SIZE_STORAGE_KEY = "minipost:font-size"
const BACKUP_SETTINGS_STORAGE_KEY = "minipost:backup-settings"
let scrollbarActivityTeardown: (() => void) | null = null
let scrollbarHideTimer: number | null = null

type PersistedRequestSettings = {
  followRedirects: boolean
  httpVersion: HttpVersion
  requestTimeoutMs: number
  maxResponseSizeMB: number
  sslCertificateVerification: boolean
  sslTlsKeyLog: boolean
  disableCookies: boolean
  sendNoCacheHeader: boolean
  sendPostmanTokenHeader: boolean
  responseFormatDetection: ResponseFormatDetection
  alwaysDiscardUnsavedOnClose: boolean
  alwaysSaveUnsavedOnClose: boolean
}

type PersistedBackupSettings = {
  autoBackupEnabled: boolean
  autoBackupIntervalMinutes: number
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
    sendNoCacheHeader: false,
    sendPostmanTokenHeader: false,
    responseFormatDetection: "auto",
    alwaysDiscardUnsavedOnClose: false,
    alwaysSaveUnsavedOnClose: false,
  }
}

type PersistedRequestSettingsSource = Pick<
  UIState,
  | "followRedirects"
  | "httpVersion"
  | "requestTimeoutMs"
  | "maxResponseSizeMB"
  | "sslCertificateVerification"
  | "sslTlsKeyLog"
  | "disableCookies"
  | "sendNoCacheHeader"
  | "sendPostmanTokenHeader"
  | "responseFormatDetection"
  | "alwaysDiscardUnsavedOnClose"
  | "alwaysSaveUnsavedOnClose"
>

function toPersistedRequestSettings(state: PersistedRequestSettingsSource): PersistedRequestSettings {
  return {
    followRedirects: state.followRedirects,
    httpVersion: state.httpVersion,
    requestTimeoutMs: state.requestTimeoutMs,
    maxResponseSizeMB: state.maxResponseSizeMB,
    sslCertificateVerification: state.sslCertificateVerification,
    sslTlsKeyLog: state.sslTlsKeyLog,
    disableCookies: state.disableCookies,
    sendNoCacheHeader: state.sendNoCacheHeader,
    sendPostmanTokenHeader: state.sendPostmanTokenHeader,
    responseFormatDetection: state.responseFormatDetection,
    alwaysDiscardUnsavedOnClose: state.alwaysDiscardUnsavedOnClose,
    alwaysSaveUnsavedOnClose: state.alwaysSaveUnsavedOnClose,
  }
}

function defaultBackupSettings(): PersistedBackupSettings {
  return {
    autoBackupEnabled: false,
    autoBackupIntervalMinutes: 30,
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

    const alwaysSaveUnsavedOnClose = parsed.alwaysSaveUnsavedOnClose ?? defaults.alwaysSaveUnsavedOnClose

    return {
      followRedirects: parsed.followRedirects ?? defaults.followRedirects,
      httpVersion,
      requestTimeoutMs: clampSettingNumber(parsed.requestTimeoutMs ?? defaults.requestTimeoutMs, 0, 3_600_000),
      maxResponseSizeMB: clampSettingNumber(parsed.maxResponseSizeMB ?? defaults.maxResponseSizeMB, 0, 2048),
      sslCertificateVerification: parsed.sslCertificateVerification ?? defaults.sslCertificateVerification,
      sslTlsKeyLog: parsed.sslTlsKeyLog ?? defaults.sslTlsKeyLog,
      disableCookies: parsed.disableCookies ?? defaults.disableCookies,
      sendNoCacheHeader: parsed.sendNoCacheHeader ?? defaults.sendNoCacheHeader,
      sendPostmanTokenHeader: parsed.sendPostmanTokenHeader ?? defaults.sendPostmanTokenHeader,
      responseFormatDetection,
      alwaysDiscardUnsavedOnClose: alwaysSaveUnsavedOnClose
        ? false
        : parsed.alwaysDiscardUnsavedOnClose ?? defaults.alwaysDiscardUnsavedOnClose,
      alwaysSaveUnsavedOnClose,
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

function readBackupSettings(): PersistedBackupSettings {
  const defaults = defaultBackupSettings()
  if (typeof window === "undefined") return defaults
  try {
    const raw = window.localStorage.getItem(BACKUP_SETTINGS_STORAGE_KEY)
    if (!raw) return defaults
    const parsed = JSON.parse(raw) as Partial<PersistedBackupSettings>
    return {
      autoBackupEnabled: parsed.autoBackupEnabled ?? defaults.autoBackupEnabled,
      autoBackupIntervalMinutes: clampSettingNumber(
        Math.round(parsed.autoBackupIntervalMinutes ?? defaults.autoBackupIntervalMinutes),
        5,
        1440
      ),
    }
  } catch {
    return defaults
  }
}

function persistBackupSettings(settings: PersistedBackupSettings) {
  if (typeof window === "undefined") return
  try {
    window.localStorage.setItem(BACKUP_SETTINGS_STORAGE_KEY, JSON.stringify(settings))
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
  const root = document.documentElement
  root.classList.toggle("scrollbar-auto-hide", autoHide)
  root.classList.remove("scrollbar-active")

  if (!autoHide) {
    if (scrollbarHideTimer !== null) {
      window.clearTimeout(scrollbarHideTimer)
      scrollbarHideTimer = null
    }
    if (scrollbarActivityTeardown) {
      scrollbarActivityTeardown()
      scrollbarActivityTeardown = null
    }
    return
  }

  if (scrollbarActivityTeardown) return

  const reveal = () => {
    root.classList.add("scrollbar-active")
    if (scrollbarHideTimer !== null) window.clearTimeout(scrollbarHideTimer)
    scrollbarHideTimer = window.setTimeout(() => {
      root.classList.remove("scrollbar-active")
      scrollbarHideTimer = null
    }, 750)
  }

  const onScrollActivity = () => reveal()
  window.addEventListener("scroll", onScrollActivity, true)
  window.addEventListener("wheel", onScrollActivity, { passive: true, capture: true })
  window.addEventListener("touchmove", onScrollActivity, { passive: true, capture: true })

  scrollbarActivityTeardown = () => {
    window.removeEventListener("scroll", onScrollActivity, true)
    window.removeEventListener("wheel", onScrollActivity, true)
    window.removeEventListener("touchmove", onScrollActivity, true)
  }
}

function applyUIFontSize(fontSize: number) {
  if (typeof document === "undefined") return
  const root = document.documentElement
  const normalized = Math.max(10, Math.min(16, Math.round(fontSize)))
  root.style.setProperty("--size-font-2xs", `${normalized}px`)
  root.style.setProperty("--size-font-xs", `${normalized + 1}px`)
  root.style.setProperty("--size-font-sm", `${normalized + 2}px`)
  root.style.setProperty("--size-font-base", `${normalized + 3}px`)
}

function readTheme(): Theme {
  if (typeof window === "undefined") return "system"
  try {
    const raw = window.localStorage.getItem(THEME_STORAGE_KEY)
    if (raw === "light" || raw === "dark" || raw === "system") {
      return raw
    }
  } catch {
    // ignore persistence errors
  }
  return "system"
}

function persistTheme(theme: Theme) {
  if (typeof window === "undefined") return
  try {
    window.localStorage.setItem(THEME_STORAGE_KEY, theme)
  } catch {
    // ignore persistence errors
  }
}

function readLocale(): Locale {
  if (typeof window === "undefined") return detectSystemLocale()
  try {
    const raw = window.localStorage.getItem(LOCALE_STORAGE_KEY)
    if (raw) return normalizeLocale(raw)
  } catch {
    // ignore persistence errors
  }
  return detectSystemLocale()
}

function persistLocale(locale: Locale) {
  if (typeof window === "undefined") return
  try {
    window.localStorage.setItem(LOCALE_STORAGE_KEY, locale)
  } catch {
    // ignore persistence errors
  }
}

function readFontSize(): number {
  if (typeof window === "undefined") return 12
  try {
    const raw = window.localStorage.getItem(FONT_SIZE_STORAGE_KEY)
    if (!raw) return 12
    const parsed = Number(raw)
    if (!Number.isFinite(parsed)) return 12
    return Math.max(10, Math.min(16, Math.round(parsed)))
  } catch {
    return 12
  }
}

function persistFontSize(fontSize: number) {
  if (typeof window === "undefined") return
  try {
    window.localStorage.setItem(FONT_SIZE_STORAGE_KEY, String(fontSize))
  } catch {
    // ignore persistence errors
  }
}

const initialTheme = readTheme()
const initialResolvedTheme = initialTheme === "system" ? getSystemTheme() : initialTheme
const initialLocale = readLocale()
const initialFontSize = readFontSize()

export const useUIStore = create<UIState>((set) => ({
  ...readRequestSettings(),
  ...readBackupSettings(),
  locale: initialLocale,
  theme: initialTheme,
  resolved: initialResolvedTheme,
  sidebarWidth: 240,
  sidebarCollapsed: false,
  layoutDirection: "vertical",
  isSending: false,
  consoleOpen: false,
  consoleHeight: 220,
  consoleLogs: [],
  settingsOpen: false,
  fontSize: initialFontSize,
  scrollbarAutoHide: readScrollbarAutoHide(),
  editingEnvironmentId: null,
  openEnvironmentTabIds: [],
  workspaceView: "project",

  setTheme: (theme) => {
    const resolved = theme === "system" ? getSystemTheme() : theme
    applyTheme(resolved)
    persistTheme(theme)
    set({ theme, resolved })
  },
  setLocale: (locale) => {
    applyDocumentLocale(locale)
    persistLocale(locale)
    set({ locale })
  },
  setSidebarWidth: (sidebarWidth) => set({ sidebarWidth }),
  setSidebarCollapsed: (sidebarCollapsed) => set({ sidebarCollapsed }),
  toggleSidebar: () => set((s) => ({ sidebarCollapsed: !s.sidebarCollapsed })),
  setLayoutDirection: (layoutDirection) => set({ layoutDirection }),
  setIsSending: (isSending) => set({ isSending }),
  toggleConsole: () => set((s) => ({ consoleOpen: !s.consoleOpen })),
  setConsoleHeight: (consoleHeight) => set({ consoleHeight: Math.max(80, Math.min(600, consoleHeight)) }),
  setSettingsOpen: (settingsOpen) => set({ settingsOpen }),
  setFontSize: (fontSize) => {
    const normalized = Math.max(10, Math.min(16, Math.round(fontSize)))
    applyUIFontSize(normalized)
    persistFontSize(normalized)
    set({ fontSize: normalized })
  },
  setScrollbarAutoHide: (scrollbarAutoHide) => {
    applyScrollbarMode(scrollbarAutoHide)
    persistScrollbarAutoHide(scrollbarAutoHide)
    set({ scrollbarAutoHide })
  },
  setFollowRedirects: (followRedirects) => set((state) => {
    const next = { ...state, followRedirects }
    persistRequestSettings(toPersistedRequestSettings(next))
    return { followRedirects }
  }),
  setHttpVersion: (httpVersion) => set((state) => {
    const next = { ...state, httpVersion }
    persistRequestSettings(toPersistedRequestSettings(next))
    return { httpVersion }
  }),
  setRequestTimeoutMs: (requestTimeoutMs) => set((state) => {
    const normalized = clampSettingNumber(Math.round(requestTimeoutMs), 0, 3_600_000)
    const next = { ...state, requestTimeoutMs: normalized }
    persistRequestSettings(toPersistedRequestSettings(next))
    return { requestTimeoutMs: normalized }
  }),
  setMaxResponseSizeMB: (maxResponseSizeMB) => set((state) => {
    const normalized = clampSettingNumber(Math.round(maxResponseSizeMB), 0, 2048)
    const next = { ...state, maxResponseSizeMB: normalized }
    persistRequestSettings(toPersistedRequestSettings(next))
    return { maxResponseSizeMB: normalized }
  }),
  setSSLCertificateVerification: (sslCertificateVerification) => set((state) => {
    const next = { ...state, sslCertificateVerification }
    persistRequestSettings(toPersistedRequestSettings(next))
    return { sslCertificateVerification }
  }),
  setSSLTlsKeyLog: (sslTlsKeyLog) => set((state) => {
    const next = { ...state, sslTlsKeyLog }
    persistRequestSettings(toPersistedRequestSettings(next))
    return { sslTlsKeyLog }
  }),
  setDisableCookies: (disableCookies) => set((state) => {
    const next = { ...state, disableCookies }
    persistRequestSettings(toPersistedRequestSettings(next))
    return { disableCookies }
  }),
  setSendNoCacheHeader: (sendNoCacheHeader) => set((state) => {
    const next = { ...state, sendNoCacheHeader }
    persistRequestSettings(toPersistedRequestSettings(next))
    return { sendNoCacheHeader }
  }),
  setSendPostmanTokenHeader: (sendPostmanTokenHeader) => set((state) => {
    const next = { ...state, sendPostmanTokenHeader }
    persistRequestSettings(toPersistedRequestSettings(next))
    return { sendPostmanTokenHeader }
  }),
  setResponseFormatDetection: (responseFormatDetection) => set((state) => {
    const next = { ...state, responseFormatDetection }
    persistRequestSettings(toPersistedRequestSettings(next))
    return { responseFormatDetection }
  }),
  setAlwaysDiscardUnsavedOnClose: (alwaysDiscardUnsavedOnClose) => set((state) => {
    const next = {
      ...state,
      alwaysDiscardUnsavedOnClose,
      alwaysSaveUnsavedOnClose: alwaysDiscardUnsavedOnClose ? false : state.alwaysSaveUnsavedOnClose,
    }
    persistRequestSettings(toPersistedRequestSettings(next))
    return {
      alwaysDiscardUnsavedOnClose: next.alwaysDiscardUnsavedOnClose,
      alwaysSaveUnsavedOnClose: next.alwaysSaveUnsavedOnClose,
    }
  }),
  setAlwaysSaveUnsavedOnClose: (alwaysSaveUnsavedOnClose) => set((state) => {
    const next = {
      ...state,
      alwaysSaveUnsavedOnClose,
      alwaysDiscardUnsavedOnClose: alwaysSaveUnsavedOnClose ? false : state.alwaysDiscardUnsavedOnClose,
    }
    persistRequestSettings(toPersistedRequestSettings(next))
    return {
      alwaysDiscardUnsavedOnClose: next.alwaysDiscardUnsavedOnClose,
      alwaysSaveUnsavedOnClose: next.alwaysSaveUnsavedOnClose,
    }
  }),
  setAutoBackupEnabled: (autoBackupEnabled) => set((state) => {
    const next = { ...state, autoBackupEnabled }
    persistBackupSettings({
      autoBackupEnabled: next.autoBackupEnabled,
      autoBackupIntervalMinutes: next.autoBackupIntervalMinutes,
    })
    return { autoBackupEnabled }
  }),
  setAutoBackupIntervalMinutes: (autoBackupIntervalMinutes) => set((state) => {
    const normalized = clampSettingNumber(Math.round(autoBackupIntervalMinutes), 5, 1440)
    const next = { ...state, autoBackupIntervalMinutes: normalized }
    persistBackupSettings({
      autoBackupEnabled: next.autoBackupEnabled,
      autoBackupIntervalMinutes: next.autoBackupIntervalMinutes,
    })
    return { autoBackupIntervalMinutes: normalized }
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
  setWorkspaceView: (workspaceView) => set({ workspaceView }),
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
  const store = useUIStore.getState()
  applyDocumentLocale(store.locale)
  applyTheme(store.resolved)
  applyUIFontSize(store.fontSize)
  applyScrollbarMode(store.scrollbarAutoHide)
  window.matchMedia("(prefers-color-scheme: dark)").addEventListener("change", () => {
    const store = useUIStore.getState()
    if (store.theme === "system") {
      const resolved = getSystemTheme()
      applyTheme(resolved)
      useUIStore.setState({ resolved })
    }
  })
}
