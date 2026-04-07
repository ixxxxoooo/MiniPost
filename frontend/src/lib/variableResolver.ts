type VariableLike = {
  key: string
  value: string
}

function buildVariableMap(variables: VariableLike[]): Map<string, string> {
  const map = new Map<string, string>()
  variables.forEach((item) => {
    const key = item.key?.trim()
    if (!key) return
    map.set(key, item.value ?? "")
  })
  return map
}

export function resolveTemplateVariables(input: string, variables: VariableLike[]): string {
  if (!input || !variables.length) return input
  const map = buildVariableMap(variables)
  return input.replace(/\{\{([^{}]+)\}\}/g, (token, rawKey: string) => {
    const key = rawKey.trim()
    if (!key) return token
    return map.has(key) ? map.get(key)! : token
  })
}

export function ensureRequestProtocol(input: string): string {
  const value = input.trim()
  if (!value) return input
  if (value.includes("{{")) return input
  if (/^[A-Za-z][A-Za-z\d+\-.]*:\/\//.test(value)) return input
  if (value.startsWith("//")) return `http:${value}`
  return `http://${value}`
}
