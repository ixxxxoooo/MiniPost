import type { KeyValuePair } from "@/types/request"

export function isPlaceholderKeyValueRow(item: Pick<KeyValuePair, "key" | "value" | "description">): boolean {
  return item.key === "" && item.value === "" && (item.description ?? "") === ""
}

export function ensureTrailingKeyValuePlaceholder<T extends KeyValuePair>(
  items: T[],
  createPlaceholder: () => T
): T[] {
  if (items.length === 0) return [createPlaceholder()]
  const last = items[items.length - 1]
  if (isPlaceholderKeyValueRow(last)) return items
  return [...items, createPlaceholder()]
}

export function applyPlaceholderKeyChange<T extends KeyValuePair>(
  items: T[],
  placeholderId: string,
  value: string,
  createItem: (partial?: Partial<T>) => T
): T[] {
  let matchedExisting = false
  const updated = items.map((item) => {
    if (item.id !== placeholderId) return item
    matchedExisting = true
    return { ...item, key: value }
  })
  if (!matchedExisting) {
    updated.push(createItem({ id: placeholderId, key: value } as Partial<T>))
  }
  if (value) updated.push(createItem())
  return updated
}
