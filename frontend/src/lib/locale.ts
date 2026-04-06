export type Locale = "zh-CN" | "en-US"

export function normalizeLocale(input: unknown): Locale {
  if (typeof input !== "string") return "zh-CN"
  const normalized = input.trim().toLowerCase()
  if (normalized.startsWith("en")) return "en-US"
  if (normalized.startsWith("zh")) return "zh-CN"
  return "zh-CN"
}

export function detectSystemLocale(): Locale {
  if (typeof navigator === "undefined") return "zh-CN"
  return normalizeLocale(navigator.language)
}

export function isChineseLocale(locale: Locale): boolean {
  return locale === "zh-CN"
}

export function applyDocumentLocale(locale: Locale) {
  if (typeof document === "undefined") return
  document.documentElement.lang = locale
}
