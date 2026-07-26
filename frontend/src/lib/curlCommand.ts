import { ensureRequestProtocol, resolveTemplateVariables } from "@/lib/variableResolver"

type VariableLike = {
  key: string
  value: string
}

export type CurlRequest = {
  id?: string
  name?: string
  method?: string
  url?: string
  params?: Array<{ key: string; value: string; enabled?: boolean; description?: string }>
  headers?: Array<{ key: string; value: string; enabled?: boolean; description?: string }>
  body?: {
    type: string
    raw?: string
    json?: string
    formUrlEncoded?: Array<{ key: string; value: string; enabled?: boolean; description?: string }>
    formData?: Array<{
      key: string
      value: string
      enabled?: boolean
      type: string
      filePath?: string
      fileName?: string
      description?: string
    }>
  }
  auth?: {
    type: string
    basic?: { username: string; password: string }
    bearer?: { token: string }
    apiKey?: { key: string; value: string; addTo: string }
  }
}

function shellQuote(value: string): string {
  return `'${value.replace(/'/g, `'"'"'`)}'`
}

function resolve(value: string | undefined, variables: VariableLike[]): string {
  return resolveTemplateVariables(value ?? "", variables)
}

function appendQueryParameter(url: string, key: string, value: string): string {
  const separator = url.includes("?") ? "&" : "?"
  return `${url}${separator}${encodeURIComponent(key)}=${encodeURIComponent(value)}`
}

export function buildCurlCommand(request: CurlRequest, variables: VariableLike[] = []): string {
  const parts: string[] = ["curl"]
  const method = (request.method || "GET").toUpperCase()
  if (method !== "GET") parts.push(`-X ${method}`)

  let fullUrl = ensureRequestProtocol(resolve(request.url, variables))
  for (const param of request.params ?? []) {
    if (param.enabled === false) continue
    const key = resolve(param.key, variables)
    if (!key) continue
    fullUrl = appendQueryParameter(fullUrl, key, resolve(param.value, variables))
  }

  if (request.auth?.type === "api-key" && request.auth.apiKey?.addTo === "query") {
    const key = resolve(request.auth.apiKey.key, variables)
    if (key) {
      fullUrl = appendQueryParameter(fullUrl, key, resolve(request.auth.apiKey.value, variables))
    }
  }
  parts.push(shellQuote(fullUrl))

  for (const header of request.headers ?? []) {
    if (header.enabled === false) continue
    const key = resolve(header.key, variables)
    if (!key) continue
    parts.push(`-H ${shellQuote(`${key}: ${resolve(header.value, variables)}`)}`)
  }

  if (request.auth) {
    if (request.auth.type === "bearer" && request.auth.bearer?.token) {
      parts.push(`-H ${shellQuote(`Authorization: Bearer ${resolve(request.auth.bearer.token, variables)}`)}`)
    } else if (request.auth.type === "basic" && request.auth.basic) {
      const username = resolve(request.auth.basic.username, variables)
      const password = resolve(request.auth.basic.password, variables)
      parts.push(`-u ${shellQuote(`${username}:${password}`)}`)
    } else if (request.auth.type === "api-key") {
      const apiKey = request.auth.apiKey
      if (apiKey && apiKey.addTo !== "query") {
        const key = resolve(apiKey.key, variables)
        if (key) parts.push(`-H ${shellQuote(`${key}: ${resolve(apiKey.value, variables)}`)}`)
      }
    }
  }

  if (request.body?.type === "json" && request.body.json) {
    parts.push(`-H ${shellQuote("Content-Type: application/json")}`)
    parts.push(`-d ${shellQuote(resolve(request.body.json, variables))}`)
  } else if (request.body?.type === "raw" && request.body.raw) {
    parts.push(`-d ${shellQuote(resolve(request.body.raw, variables))}`)
  } else if (request.body?.type === "form-urlencoded") {
    const fields = (request.body.formUrlEncoded ?? []).flatMap((field) => {
      if (field.enabled === false) return []
      const key = resolve(field.key, variables)
      if (!key) return []
      return [`${encodeURIComponent(key)}=${encodeURIComponent(resolve(field.value, variables))}`]
    })
    if (fields.length > 0) {
      parts.push(`-H ${shellQuote("Content-Type: application/x-www-form-urlencoded")}`)
      parts.push(`-d ${shellQuote(fields.join("&"))}`)
    }
  } else if (request.body?.type === "form-data") {
    for (const field of request.body.formData ?? []) {
      if (field.enabled === false) continue
      const key = resolve(field.key, variables)
      if (!key) continue
      if (field.type === "file") {
        const filePath = resolve(field.filePath || field.value, variables).trim()
        if (filePath) parts.push(`-F ${shellQuote(`${key}=@${filePath}`)}`)
      } else {
        parts.push(`--form-string ${shellQuote(`${key}=${resolve(field.value, variables)}`)}`)
      }
    }
  }

  return parts.join(" " + "\\" + "\n  ")
}
