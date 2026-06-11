import { describe, expect, it } from "vitest"
import { shouldAutoFormatJsonOnEntry } from "./bodyEditorAutoFormat"

describe("body editor JSON auto format", () => {
  it("formats non-empty JSON only once when entering the JSON editor", () => {
    expect(shouldAutoFormatJsonOnEntry('{"name":"MiniPost"}', false)).toBe(true)
    expect(shouldAutoFormatJsonOnEntry('{"name":"MiniPost"}', true)).toBe(false)
  })

  it("does not format empty JSON body on entry", () => {
    expect(shouldAutoFormatJsonOnEntry("", false)).toBe(false)
    expect(shouldAutoFormatJsonOnEntry("   \n", false)).toBe(false)
  })
})
