import { useCallback } from "react"
import { isChineseLocale } from "@/lib/locale"
import { useUIStore } from "@/stores/uiStore"

export function useI18n() {
  const locale = useUIStore((s) => s.locale)
  const setLocale = useUIStore((s) => s.setLocale)
  const isZh = isChineseLocale(locale)

  const t = useCallback((zh: string, en: string) => (isZh ? zh : en), [isZh])

  return {
    locale,
    setLocale,
    isZh,
    t,
  }
}
