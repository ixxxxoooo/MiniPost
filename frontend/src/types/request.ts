import type { HttpMethod, BodyType, AuthType } from "@/lib/constants"

export interface KeyValuePair {
  id: string
  key: string
  value: string
  enabled: boolean
  description?: string
}

export interface RequestBody {
  type: BodyType
  raw?: string
  json?: string
  formUrlEncoded?: KeyValuePair[]
}

export interface BasicAuthConfig {
  username: string
  password: string
}

export interface BearerAuthConfig {
  token: string
}

export interface ApiKeyAuthConfig {
  key: string
  value: string
  addTo: "header" | "query"
}

export interface AuthConfig {
  type: AuthType
  basic?: BasicAuthConfig
  bearer?: BearerAuthConfig
  apiKey?: ApiKeyAuthConfig
}

export interface RequestData {
  id: string
  name: string
  method: HttpMethod
  url: string
  params: KeyValuePair[]
  headers: KeyValuePair[]
  body: RequestBody
  auth: AuthConfig
  folderId?: string
  projectId?: string
  createdAt: string
  updatedAt: string
}

export function createDefaultRequest(partial?: Partial<RequestData>): RequestData {
  return {
    id: crypto.randomUUID(),
    name: "New Request",
    method: "GET",
    url: "",
    params: [],
    headers: [],
    body: { type: "none" },
    auth: { type: "none" },
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    ...partial,
  }
}

export function createKeyValuePair(partial?: Partial<KeyValuePair>): KeyValuePair {
  return {
    id: crypto.randomUUID(),
    key: "",
    value: "",
    enabled: true,
    ...partial,
  }
}
