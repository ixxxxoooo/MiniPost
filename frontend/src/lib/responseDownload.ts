export const BINARY_SAVE_PREFIX = "__MINIPOST_BASE64__:"

const BINARY_MIME_EXACT = new Set([
  "application/octet-stream",
  "application/zip",
  "application/x-zip-compressed",
  "application/x-7z-compressed",
  "application/x-rar-compressed",
  "application/gzip",
  "application/x-gzip",
  "application/pdf",
  "application/msword",
  "application/vnd.ms-excel",
  "application/vnd.ms-powerpoint",
  "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
  "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
  "application/vnd.openxmlformats-officedocument.presentationml.presentation",
  "application/x-binary",
  "application/x-protobuf",
  "application/protobuf",
  "application/wasm",
])

const TEXT_MIME_HINTS = [
  "json",
  "xml",
  "yaml",
  "yml",
  "html",
  "javascript",
  "ecmascript",
  "x-www-form-urlencoded",
  "graphql",
  "csv",
  "plain",
  "svg",
]

function getHeaderValues(headers: Record<string, string[]>, name: string): string[] {
  const target = name.toLowerCase()
  for (const [key, values] of Object.entries(headers)) {
    if (key.toLowerCase() === target) return values
  }
  return []
}

function stripQuotes(value: string): string {
  const trimmed = value.trim()
  if (trimmed.startsWith("\"") && trimmed.endsWith("\"") && trimmed.length >= 2) {
    return trimmed.slice(1, -1).replace(/\\"/g, "\"")
  }
  return trimmed
}

function normalizeMimeType(contentType: string): string {
  return contentType.split(";")[0]?.trim().toLowerCase() ?? ""
}

export function parseContentDispositionFilename(headerValue: string): string | null {
  if (!headerValue) return null

  const filenameStar = headerValue.match(/filename\*\s*=\s*([^;]+)/i)?.[1]
  if (filenameStar) {
    const raw = stripQuotes(filenameStar)
    const encodedPart = raw.split("''", 2)[1]
    if (encodedPart) {
      try {
        return decodeURIComponent(encodedPart)
      } catch {
        return encodedPart
      }
    }
  }

  const filename = headerValue.match(/filename\s*=\s*([^;]+)/i)?.[1]
  if (filename) return stripQuotes(filename)
  return null
}

export function sanitizeFilename(input: string): string {
  const cleaned = input
    .replace(/[/\\?%*:|"<>]/g, "_")
    .replace(/[\u0000-\u001f\u007f]+/g, "")
    .trim()
  return cleaned || "response"
}

export function extensionFromMimeType(contentType: string): string {
  const mime = normalizeMimeType(contentType)
  if (!mime) return ""

  if (mime.includes("json")) return ".json"
  if (mime.includes("xml")) return ".xml"
  if (mime.includes("html")) return ".html"
  if (mime.includes("csv")) return ".csv"
  if (mime.includes("pdf")) return ".pdf"
  if (mime.includes("zip")) return ".zip"
  if (mime.includes("gzip")) return ".gz"
  if (mime.includes("jpeg")) return ".jpg"
  if (mime.includes("png")) return ".png"
  if (mime.includes("gif")) return ".gif"
  if (mime.includes("webp")) return ".webp"
  if (mime.includes("svg")) return ".svg"
  if (mime.includes("audio/mpeg")) return ".mp3"
  if (mime.includes("video/mp4")) return ".mp4"
  if (mime.includes("text/plain")) return ".txt"
  if (mime.includes("octet-stream")) return ".bin"
  return ""
}

function filenameFromUrl(requestUrl?: string): string {
  if (!requestUrl) return ""
  try {
    const pathname = new URL(requestUrl).pathname
    const last = pathname.split("/").filter(Boolean).at(-1) ?? ""
    return decodeURIComponent(last)
  } catch {
    return ""
  }
}

function hasFileExtension(name: string): boolean {
  return /\.[A-Za-z0-9]{1,10}$/.test(name)
}

export function suggestResponseFilename(args: {
  headers: Record<string, string[]>
  contentType: string
  requestUrl?: string
  fallbackBase?: string
}): string {
  const { headers, contentType, requestUrl, fallbackBase = "response" } = args
  const contentDisposition = getHeaderValues(headers, "content-disposition")[0] ?? ""
  const fromHeader = parseContentDispositionFilename(contentDisposition) ?? ""
  const fromUrl = filenameFromUrl(requestUrl)
  const base = sanitizeFilename(fromHeader || fromUrl || fallbackBase)
  const ext = extensionFromMimeType(contentType)
  if (!ext || hasFileExtension(base)) return base
  return `${base}${ext}`
}

export function shouldPreferDownload(args: {
  headers: Record<string, string[]>
  contentType: string
  bodyIsBinary?: boolean
}): boolean {
  const { headers, contentType, bodyIsBinary } = args
  if (bodyIsBinary) return true

  const contentDisposition = (getHeaderValues(headers, "content-disposition")[0] ?? "").toLowerCase()
  if (contentDisposition.includes("attachment")) return true

  const mime = normalizeMimeType(contentType)
  if (!mime) return false
  if (mime.startsWith("text/")) return false
  if (TEXT_MIME_HINTS.some((hint) => mime.includes(hint))) return false
  if (mime.startsWith("image/") || mime.startsWith("audio/") || mime.startsWith("video/") || mime.startsWith("font/")) return true
  if (BINARY_MIME_EXACT.has(mime)) return true
  if (mime.startsWith("application/")) return true
  return false
}

export function decodeBase64ToBytes(base64: string): Uint8Array {
  try {
    const binary = atob(base64)
    const bytes = new Uint8Array(binary.length)
    for (let i = 0; i < binary.length; i += 1) {
      bytes[i] = binary.charCodeAt(i) & 0xff
    }
    return bytes
  } catch {
    return new Uint8Array()
  }
}

export function decodeResponseBodyToText(args: {
  body: string
  bodyBase64?: string
  bodyIsBinary?: boolean
}): string {
  const { body, bodyBase64, bodyIsBinary } = args
  if (bodyIsBinary && bodyBase64) {
    const bytes = decodeBase64ToBytes(bodyBase64)
    return new TextDecoder("utf-8", { fatal: false }).decode(bytes)
  }
  return body ?? ""
}

export function buildSaveResponsePayload(args: {
  body: string
  bodyBase64?: string
  bodyIsBinary?: boolean
}): string {
  const { body, bodyBase64, bodyIsBinary } = args
  if (bodyBase64 && bodyIsBinary) return `${BINARY_SAVE_PREFIX}${bodyBase64}`
  return body ?? ""
}
