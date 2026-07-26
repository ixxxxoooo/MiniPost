import { describe, expect, it } from "vitest"
import { mergeCookieHeaders } from "./cookieHeader"

describe("mergeCookieHeaders", () => {
  it("adds cookies from the jar and lets manual values override matching names", () => {
    expect(mergeCookieHeaders(
      "session=manual; preference=compact",
      "session=jar; csrf=token"
    )).toBe("session=manual; csrf=token; preference=compact")
  })

  it("ignores malformed cookie segments", () => {
    expect(mergeCookieHeaders("manual=1; malformed", "jar=2; =empty")).toBe("jar=2; manual=1")
  })
})
