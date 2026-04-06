export const PROJECT_THEME_COLORS = [
  "#0A84FF",
  "#30D158",
  "#FF9F0A",
  "#FF453A",
  "#BF5AF2",
  "#64D2FF",
  "#5E5CE6",
  "#FF375F",
  "#34C759",
  "#FFD60A",
  "#FF6B35",
  "#00C7BE",
] as const

type RGB = { r: number; g: number; b: number }

function normalizeHexColor(input?: string | null): string | null {
  if (!input) return null
  const value = input.trim()
  if (!value) return null
  const hex = value.startsWith("#") ? value : `#${value}`
  if (!/^#[\dA-Fa-f]{6}$/.test(hex)) return null
  return hex.toUpperCase()
}

function hexToRgb(hex: string): RGB {
  return {
    r: parseInt(hex.slice(1, 3), 16),
    g: parseInt(hex.slice(3, 5), 16),
    b: parseInt(hex.slice(5, 7), 16),
  }
}

function clampByte(value: number): number {
  return Math.max(0, Math.min(255, Math.round(value)))
}

function mixColor(hex: string, target: RGB, ratio: number): string {
  const source = hexToRgb(hex)
  const t = Math.max(0, Math.min(1, ratio))
  const r = clampByte(source.r + (target.r - source.r) * t)
  const g = clampByte(source.g + (target.g - source.g) * t)
  const b = clampByte(source.b + (target.b - source.b) * t)
  return `#${r.toString(16).padStart(2, "0").toUpperCase()}${g.toString(16).padStart(2, "0").toUpperCase()}${b.toString(16).padStart(2, "0").toUpperCase()}`
}

function withAlpha(hex: string, alpha: number): string {
  const { r, g, b } = hexToRgb(hex)
  const normalizedAlpha = Math.max(0, Math.min(1, alpha))
  return `rgba(${r}, ${g}, ${b}, ${normalizedAlpha})`
}

export function applyProjectThemeColor(color: string | null | undefined, resolvedTheme: "light" | "dark") {
  if (typeof document === "undefined") return
  const root = document.documentElement
  const normalized = normalizeHexColor(color)

  if (!normalized) {
    root.style.removeProperty("--accent")
    root.style.removeProperty("--accent-hover")
    root.style.removeProperty("--sidebar-accent")
    root.style.removeProperty("--tab-active-border")
    root.style.removeProperty("--sidebar-active")
    root.style.removeProperty("--selected-bg")
    return
  }

  const hover = resolvedTheme === "dark"
    ? mixColor(normalized, { r: 255, g: 255, b: 255 }, 0.16)
    : mixColor(normalized, { r: 0, g: 0, b: 0 }, 0.14)
  const selectedBg = withAlpha(normalized, resolvedTheme === "dark" ? 0.2 : 0.13)
  const sidebarActive = withAlpha(normalized, resolvedTheme === "dark" ? 0.24 : 0.14)

  root.style.setProperty("--accent", normalized)
  root.style.setProperty("--accent-hover", hover)
  root.style.setProperty("--sidebar-accent", normalized)
  root.style.setProperty("--tab-active-border", normalized)
  root.style.setProperty("--selected-bg", selectedBg)
  root.style.setProperty("--sidebar-active", sidebarActive)
}
