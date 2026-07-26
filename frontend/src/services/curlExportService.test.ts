import { afterEach, beforeEach, describe, expect, it } from "vitest"
import { buildAutoHeaderDisabledMarkerKey } from "@/lib/autoHeaders"
import type { CurlRequest } from "@/lib/curlCommand"
import { useCookieStore, type CookieItem } from "@/stores/cookieStore"
import { useUIStore } from "@/stores/uiStore"
import { buildRuntimeCurlCommand } from "./curlExportService"

const baseRequest: CurlRequest = {
  method: "GET",
  url: "{{baseUrl}}/private/resource",
  params: [],
  headers: [],
  body: { type: "none" },
  auth: { type: "none" },
}

function cookie(partial: Partial<CookieItem>): CookieItem {
  return {
    id: partial.id ?? crypto.randomUUID(),
    domain: partial.domain ?? "api.example.com",
    name: partial.name ?? "session",
    value: partial.value ?? "secret",
    path: partial.path ?? "/",
    expires: partial.expires ?? "",
    secure: partial.secure ?? false,
    httpOnly: partial.httpOnly ?? false,
    enabled: partial.enabled ?? true,
  }
}

describe("buildRuntimeCurlCommand", () => {
  beforeEach(() => {
    useUIStore.setState({ disableCookies: false })
    useCookieStore.setState({
      cookies: [
        cookie({ name: "session", value: "abc", secure: true, path: "/private" }),
        cookie({ name: "wrong-path", value: "no", path: "/admin" }),
      ],
    })
  })

  afterEach(() => {
    useCookieStore.setState({ cookies: [] })
    useUIStore.setState({ disableCookies: false })
  })

  it("matches Cookie Jar entries against the environment-resolved URL", () => {
    const result = buildRuntimeCurlCommand(baseRequest, [
      { key: "baseUrl", value: "https://api.example.com" },
    ])

    expect(result.cookieIncluded).toBe(true)
    expect(result.command).toContain("-H 'Cookie: session=abc'")
    expect(result.command).not.toContain("wrong-path")
  })

  it("does not include Cookie Jar values when cookies are globally disabled", () => {
    useUIStore.setState({ disableCookies: true })

    const result = buildRuntimeCurlCommand({
      ...baseRequest,
      headers: [{ key: "Cookie", value: "manual=kept", enabled: true }],
    }, [
      { key: "baseUrl", value: "https://api.example.com" },
    ])

    expect(result.cookieIncluded).toBe(false)
    expect(result.command).toContain("-H 'Cookie: manual=kept'")
    expect(result.command).not.toContain("session=abc")
  })

  it("does not include Cookie Jar values when the request suppresses the Cookie auto header", () => {
    const request: CurlRequest = {
      ...baseRequest,
      headers: [{
        key: buildAutoHeaderDisabledMarkerKey("Cookie"),
        value: "",
        enabled: false,
      }],
    }

    const result = buildRuntimeCurlCommand(request, [
      { key: "baseUrl", value: "https://api.example.com" },
    ])

    expect(result.cookieIncluded).toBe(false)
    expect(result.command).not.toContain("Cookie:")
  })
})
