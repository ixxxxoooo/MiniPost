import { describe, expect, it } from "vitest"
import { getEditorValueSyncDecision, shouldApplyPendingEditorValue } from "./codeEditorValueSync"

describe("code editor value sync", () => {
  it("does nothing when the editor already has the incoming value", () => {
    expect(getEditorValueSyncDecision("hello", "hello", false)).toEqual({ type: "none" })
    expect(getEditorValueSyncDecision("中文", "中文", true)).toEqual({ type: "none" })
  })

  it("applies incoming values immediately outside IME composition", () => {
    expect(getEditorValueSyncDecision("old", "new", false)).toEqual({ type: "apply" })
  })

  it("defers incoming values while IME composition is active", () => {
    expect(getEditorValueSyncDecision("li", "李", true)).toEqual({
      type: "defer",
      pendingValue: "李",
    })
  })

  it("applies deferred values only when the editor still differs", () => {
    expect(shouldApplyPendingEditorValue("old", "new")).toBe(true)
    expect(shouldApplyPendingEditorValue("new", "new")).toBe(false)
    expect(shouldApplyPendingEditorValue("new", null)).toBe(false)
  })
})
