export const HTTP_METHODS = [
  "GET",
  "POST",
  "PUT",
  "PATCH",
  "DELETE",
  "HEAD",
  "OPTIONS",
] as const

export type HttpMethod = (typeof HTTP_METHODS)[number]

export const METHOD_COLORS: Record<HttpMethod, string> = {
  GET: "text-[var(--method-get)]",
  POST: "text-[var(--method-post)]",
  PUT: "text-[var(--method-put)]",
  PATCH: "text-[var(--method-patch)]",
  DELETE: "text-[var(--method-delete)]",
  HEAD: "text-[var(--method-head)]",
  OPTIONS: "text-[var(--method-options)]",
}

export const BODY_TYPES = [
  "none",
  "form-data",
  "raw",
  "json",
  "form-urlencoded",
] as const

export type BodyType = (typeof BODY_TYPES)[number]

export const AUTH_TYPES = [
  "none",
  "basic",
  "bearer",
  "api-key",
] as const

export type AuthType = (typeof AUTH_TYPES)[number]
