export function mergeCookieHeaders(manual: string, fromJar: string): string {
  const cookies = new Map<string, string>()
  const append = (source: string, overwrite: boolean) => {
    source.split(";").forEach((pair) => {
      const trimmed = pair.trim()
      if (!trimmed) return
      const separator = trimmed.indexOf("=")
      if (separator <= 0) return
      const name = trimmed.slice(0, separator).trim()
      if (!name) return
      const value = trimmed.slice(separator + 1).trim()
      if (overwrite || !cookies.has(name)) cookies.set(name, value)
    })
  }

  append(fromJar, false)
  append(manual, true)
  return Array.from(cookies.entries()).map(([name, value]) => `${name}=${value}`).join("; ")
}
