import { createKeyValuePair, type KeyValuePair } from "@/types/request"

function isPlaceholderParam(item: KeyValuePair): boolean {
  return item.key === "" && item.value === "" && (item.description ?? "") === ""
}

function splitUrl(input: string): { head: string; query: string; hash: string } {
  const hashIndex = input.indexOf("#")
  const hash = hashIndex >= 0 ? input.slice(hashIndex) : ""
  const beforeHash = hashIndex >= 0 ? input.slice(0, hashIndex) : input
  const queryIndex = beforeHash.indexOf("?")
  if (queryIndex < 0) {
    return { head: beforeHash, query: "", hash }
  }
  return {
    head: beforeHash.slice(0, queryIndex),
    query: beforeHash.slice(queryIndex + 1),
    hash,
  }
}

function readQueryPairs(query: string): Array<{ key: string; value: string }> {
  if (!query) return []
  const params = new URLSearchParams(query)
  const pairs: Array<{ key: string; value: string }> = []
  params.forEach((value, key) => {
    pairs.push({ key, value })
  })
  return pairs
}

function meaningfulParams(items: KeyValuePair[]): KeyValuePair[] {
  return items.filter((item) => !isPlaceholderParam(item))
}

export function areParamsEquivalent(left: KeyValuePair[], right: KeyValuePair[]): boolean {
  const a = meaningfulParams(left)
  const b = meaningfulParams(right)
  if (a.length !== b.length) return false
  for (let i = 0; i < a.length; i += 1) {
    if (a[i].key !== b[i].key || a[i].value !== b[i].value || Boolean(a[i].enabled) !== Boolean(b[i].enabled)) {
      return false
    }
  }
  return true
}

export function syncParamsWithUrlQuery(url: string, currentParams: KeyValuePair[]): KeyValuePair[] {
  const { query } = splitUrl(url)
  const queryPairs = readQueryPairs(query)
  const source = meaningfulParams(currentParams)
  const trailingPlaceholder = currentParams.length > 0 && isPlaceholderParam(currentParams[currentParams.length - 1])
    ? currentParams[currentParams.length - 1]
    : null

  const byKey = new Map<string, KeyValuePair[]>()
  source.forEach((item) => {
    const list = byKey.get(item.key) ?? []
    list.push(item)
    byKey.set(item.key, list)
  })

  const next = queryPairs.map(({ key, value }) => {
    const list = byKey.get(key) ?? []
    let matchedIndex = list.findIndex((item) => item.value === value)
    if (matchedIndex < 0 && list.length > 0) matchedIndex = 0
    if (matchedIndex < 0) {
      return createKeyValuePair({
        key,
        value,
        enabled: true,
      })
    }
    const reuse = list.splice(matchedIndex, 1)[0]
    return {
      ...reuse,
      key,
      value,
      enabled: true,
    }
  })

  if (trailingPlaceholder) {
    return [...next, trailingPlaceholder]
  }
  return next
}

export function buildUrlWithParams(url: string, params: KeyValuePair[]): string {
  const { head, hash } = splitUrl(url)
  const query = new URLSearchParams()
  meaningfulParams(params)
    .filter((item) => item.enabled && item.key.trim())
    .forEach((item) => {
      query.append(item.key, item.value)
    })

  const encoded = query.toString()
  if (!encoded) return `${head}${hash}`
  return `${head}?${encoded}${hash}`
}
