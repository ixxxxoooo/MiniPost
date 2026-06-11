import { describe, expect, it } from "vitest"
import { defaultEditorTabForMethod, shouldDefaultBodyToJson } from "./requestEditorTabs"

describe("defaultEditorTabForMethod", () => {
  it("GET 请求默认定位到 Params", () => {
    expect(defaultEditorTabForMethod("GET")).toBe("params")
    expect(defaultEditorTabForMethod("get")).toBe("params")
  })

  it("POST/PUT/PATCH 请求默认定位到 Body", () => {
    expect(defaultEditorTabForMethod("POST")).toBe("body")
    expect(defaultEditorTabForMethod("put")).toBe("body")
    expect(defaultEditorTabForMethod("PATCH")).toBe("body")
  })

  it("其他方法与空值回落到 Params", () => {
    expect(defaultEditorTabForMethod("DELETE")).toBe("params")
    expect(defaultEditorTabForMethod("HEAD")).toBe("params")
    expect(defaultEditorTabForMethod("")).toBe("params")
  })
})

describe("shouldDefaultBodyToJson", () => {
  it("POST 且 body 为 none 时默认切到 JSON", () => {
    expect(shouldDefaultBodyToJson("POST", "none")).toBe(true)
    expect(shouldDefaultBodyToJson("post", undefined)).toBe(true)
  })

  it("已选择 body 类型时保持用户设置", () => {
    expect(shouldDefaultBodyToJson("POST", "json")).toBe(false)
    expect(shouldDefaultBodyToJson("POST", "form-data")).toBe(false)
    expect(shouldDefaultBodyToJson("POST", "raw")).toBe(false)
  })

  it("GET 请求不会改动 body 类型", () => {
    expect(shouldDefaultBodyToJson("GET", "none")).toBe(false)
  })
})
