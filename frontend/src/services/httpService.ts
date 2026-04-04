import { SendRequest, SendRequestWithEnv } from "../../wailsjs/go/main/App"
import type { HttpResponse } from "@/types/response"
import type { RequestData } from "@/types/request"

export interface SendRequestPayload {
  method: string
  url: string
  params: { key: string; value: string }[]
  headers: { key: string; value: string }[]
  body: {
    type: string
    raw: string
    json: string
    formUrlEncoded: { key: string; value: string }[]
  }
  auth: {
    type: string
    basic: { username: string; password: string }
    bearer: { token: string }
    apiKey: { key: string; value: string; addTo: string }
  }
}

function buildPayload(request: RequestData): SendRequestPayload {
  return {
    method: request.method,
    url: request.url,
    params: request.params
      .filter((p) => p.enabled && p.key)
      .map((p) => ({ key: p.key, value: p.value })),
    headers: request.headers
      .filter((h) => h.enabled && h.key)
      .map((h) => ({ key: h.key, value: h.value })),
    body: {
      type: request.body.type,
      raw: request.body.raw ?? "",
      json: request.body.json ?? "",
      formUrlEncoded: (request.body.formUrlEncoded ?? [])
        .filter((f) => f.enabled && f.key)
        .map((f) => ({ key: f.key, value: f.value })),
    },
    auth: {
      type: request.auth.type,
      basic: request.auth.basic ?? { username: "", password: "" },
      bearer: request.auth.bearer ?? { token: "" },
      apiKey: request.auth.apiKey ?? { key: "", value: "", addTo: "header" },
    },
  }
}

export async function sendHttpRequest(
  request: RequestData,
  projectId?: string,
  envId?: string
): Promise<HttpResponse> {
  const payload = buildPayload(request)

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const result = (projectId && envId)
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    ? await SendRequestWithEnv(payload as any, projectId, envId)
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    : await SendRequest(payload as any)

  return {
    statusCode: result.statusCode,
    statusText: result.statusText,
    headers: result.headers,
    body: result.body,
    duration: result.duration,
    size: result.size,
    contentType: result.contentType,
  }
}
