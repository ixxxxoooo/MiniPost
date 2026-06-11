// 请求编辑器默认标签定位规则：
// - GET 等无 body 的请求默认停在 Params
// - POST / PUT / PATCH 等带 body 的请求默认停在 Body
export type RequestEditorTab = "params" | "headers" | "body" | "auth"

const BODY_FIRST_METHODS = new Set(["POST", "PUT", "PATCH"])

export function defaultEditorTabForMethod(method: string): RequestEditorTab {
  const normalized = (method || "").trim().toUpperCase()
  return BODY_FIRST_METHODS.has(normalized) ? "body" : "params"
}

// 打开请求时，POST 类请求若 body 类型还是 none，则默认定位到 JSON 子标签
export function shouldDefaultBodyToJson(method: string, bodyType: string | undefined): boolean {
  const normalized = (method || "").trim().toUpperCase()
  if (!BODY_FIRST_METHODS.has(normalized)) return false
  return !bodyType || bodyType === "none"
}
