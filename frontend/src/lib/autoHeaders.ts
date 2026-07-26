import type { KeyValuePair } from "@/types/request"

export const AUTO_HEADER_DISABLED_PREFIX = "__minipost_auto_disabled__:"

export function normalizeHeaderName(name: string): string {
  return name.trim().toLowerCase()
}

export function buildAutoHeaderDisabledMarkerKey(name: string): string {
  return `${AUTO_HEADER_DISABLED_PREFIX}${normalizeHeaderName(name)}`
}

export function isAutoHeaderDisabledMarkerKey(key: string): boolean {
  return key.trim().toLowerCase().startsWith(AUTO_HEADER_DISABLED_PREFIX)
}

export function extractDisabledHeaderNameFromMarker(key: string): string | null {
  const trimmed = key.trim().toLowerCase()
  if (!trimmed.startsWith(AUTO_HEADER_DISABLED_PREFIX)) return null
  const name = trimmed.slice(AUTO_HEADER_DISABLED_PREFIX.length).trim()
  return name || null
}

export function getSuppressedAutoHeaders(headers: Array<Pick<KeyValuePair, "key">>): Set<string> {
  const suppressed = new Set<string>()
  headers.forEach((header) => {
    const name = extractDisabledHeaderNameFromMarker(header.key)
    if (name) {
      suppressed.add(name)
    }
  })
  return suppressed
}
