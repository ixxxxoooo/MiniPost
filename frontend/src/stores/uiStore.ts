import { create } from "zustand"

export type Theme = "light" | "dark" | "system"
export type LayoutDirection = "vertical" | "horizontal"

interface UIState {
  theme: Theme
  resolved: "light" | "dark"
  sidebarWidth: number
  sidebarCollapsed: boolean
  layoutDirection: LayoutDirection
  isSending: boolean

  setTheme: (theme: Theme) => void
  setSidebarWidth: (width: number) => void
  toggleSidebar: () => void
  setLayoutDirection: (d: LayoutDirection) => void
  setIsSending: (v: boolean) => void
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

export const useUIStore = create<UIState>((set) => ({
  theme: "system",
  resolved: getSystemTheme(),
  sidebarWidth: 240,
  sidebarCollapsed: false,
  layoutDirection: "vertical",
  isSending: false,

  setTheme: (theme) => {
    const resolved = theme === "system" ? getSystemTheme() : theme
    applyTheme(resolved)
    set({ theme, resolved })
  },
  setSidebarWidth: (sidebarWidth) => set({ sidebarWidth }),
  toggleSidebar: () => set((s) => ({ sidebarCollapsed: !s.sidebarCollapsed })),
  setLayoutDirection: (layoutDirection) => set({ layoutDirection }),
  setIsSending: (isSending) => set({ isSending }),
}))

// 初始化时应用系统主题
if (typeof window !== "undefined") {
  applyTheme(getSystemTheme())
  window.matchMedia("(prefers-color-scheme: dark)").addEventListener("change", () => {
    const store = useUIStore.getState()
    if (store.theme === "system") {
      const resolved = getSystemTheme()
      applyTheme(resolved)
      useUIStore.setState({ resolved })
    }
  })
}
