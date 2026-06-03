import { describe, expect, it } from "vitest"
import { applyPlaceholderKeyChange, ensureTrailingKeyValuePlaceholder } from "./keyValueRows"
import type { KeyValuePair } from "@/types/request"

function row(partial: Partial<KeyValuePair> = {}): KeyValuePair {
  return {
    id: partial.id ?? "row",
    key: partial.key ?? "",
    value: partial.value ?? "",
    enabled: partial.enabled ?? true,
    description: partial.description,
  }
}

describe("keyValueRows", () => {
  it("derives a trailing placeholder without mutating imported rows", () => {
    const imported = [
      row({ id: "accept", key: "accept", value: "application/json" }),
      row({ id: "content-type", key: "content-type", value: "application/json" }),
    ]

    const displayRows = ensureTrailingKeyValuePlaceholder(imported, () => row({ id: "placeholder" }))

    expect(imported).toHaveLength(2)
    expect(displayRows).toHaveLength(3)
    expect(displayRows[0]).toBe(imported[0])
    expect(displayRows[2]).toMatchObject({ id: "placeholder", key: "", value: "" })
  })

  it("reuses an existing trailing placeholder array", () => {
    const rows = [
      row({ id: "accept", key: "accept", value: "application/json" }),
      row({ id: "placeholder" }),
    ]

    expect(ensureTrailingKeyValuePlaceholder(rows, () => row({ id: "new" }))).toBe(rows)
  })

  it("turns a derived placeholder into a real row when the user types", () => {
    const imported = [
      row({ id: "accept", key: "accept", value: "application/json" }),
    ]

    const next = applyPlaceholderKeyChange(imported, "derived-placeholder", "origin", (partial) => row(partial))

    expect(next).toHaveLength(3)
    expect(next[1]).toMatchObject({ id: "derived-placeholder", key: "origin" })
    expect(next[2]).toMatchObject({ key: "", value: "" })
  })
})
