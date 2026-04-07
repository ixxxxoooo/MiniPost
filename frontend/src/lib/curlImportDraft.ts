import type { HttpMethod } from "@/lib/constants"
import { createDefaultRequest, createFormDataItem, createKeyValuePair, type RequestData } from "@/types/request"
import type { model } from "../../wailsjs/go/models"

type RequestBodyType = "none" | "raw" | "json" | "form-urlencoded" | "form-data"
type RequestAuthType = "none" | "basic" | "bearer" | "api-key"

const SUPPORTED_BODY_TYPES = new Set<RequestBodyType>(["none", "raw", "json", "form-urlencoded", "form-data"])
const SUPPORTED_AUTH_TYPES = new Set<RequestAuthType>(["none", "basic", "bearer", "api-key"])

function normalizeBodyType(raw: string | undefined): RequestBodyType {
  if (!raw) return "none"
  if (SUPPORTED_BODY_TYPES.has(raw as RequestBodyType)) {
    return raw as RequestBodyType
  }
  return "none"
}

function normalizeAuthType(raw: string | undefined): RequestAuthType {
  if (!raw) return "none"
  if (SUPPORTED_AUTH_TYPES.has(raw as RequestAuthType)) {
    return raw as RequestAuthType
  }
  return "none"
}

export function buildDraftRequestFromCurl(
  parsed: model.SendRequestInput,
  options: {
    projectId: string
    name?: string
    folderId?: string
  }
): RequestData {
  const bodyType = normalizeBodyType(parsed.body?.type)
  const authType = normalizeAuthType(parsed.auth?.type)

  return createDefaultRequest({
    name: options.name || "Imported cURL",
    method: (parsed.method || "GET") as HttpMethod,
    url: parsed.url || "",
    params: (parsed.params ?? []).map((item) => createKeyValuePair({
      key: item.key ?? "",
      value: item.value ?? "",
      enabled: true,
    })),
    headers: (parsed.headers ?? []).map((item) => createKeyValuePair({
      key: item.key ?? "",
      value: item.value ?? "",
      enabled: true,
    })),
    body: parsed.body
      ? {
          type: bodyType,
          raw: parsed.body.raw ?? "",
          json: parsed.body.json ?? "",
          formUrlEncoded: (parsed.body.formUrlEncoded ?? []).map((item) => createKeyValuePair({
            key: item.key ?? "",
            value: item.value ?? "",
            enabled: true,
          })),
          formData: (parsed.body.formData ?? []).map((item) => createFormDataItem({
            key: item.key ?? "",
            value: item.value ?? "",
            enabled: true,
            type: item.type === "file" ? "file" : "text",
            filePath: item.filePath ?? "",
            fileName: item.fileName ?? "",
          })),
        }
      : { type: "none" },
    auth: parsed.auth
      ? {
          type: authType,
          basic: parsed.auth.basic
            ? {
                username: parsed.auth.basic.username ?? "",
                password: parsed.auth.basic.password ?? "",
              }
            : undefined,
          bearer: parsed.auth.bearer
            ? { token: parsed.auth.bearer.token ?? "" }
            : undefined,
          apiKey: parsed.auth.apiKey
            ? {
                key: parsed.auth.apiKey.key ?? "",
                value: parsed.auth.apiKey.value ?? "",
                addTo: parsed.auth.apiKey.addTo === "query" ? "query" : "header",
              }
            : undefined,
        }
      : { type: "none" },
    folderId: options.folderId ?? "",
    projectId: options.projectId,
  })
}
