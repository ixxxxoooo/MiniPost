export type EditorValueSyncDecision =
  | { type: "none" }
  | { type: "apply" }
  | { type: "defer"; pendingValue: string }

export function getEditorValueSyncDecision(
  currentEditorValue: string,
  incomingValue: string,
  isComposing: boolean,
): EditorValueSyncDecision {
  if (currentEditorValue === incomingValue) {
    return { type: "none" }
  }

  if (isComposing) {
    return { type: "defer", pendingValue: incomingValue }
  }

  return { type: "apply" }
}

export function shouldApplyPendingEditorValue(currentEditorValue: string, pendingValue: string | null): pendingValue is string {
  return pendingValue !== null && currentEditorValue !== pendingValue
}

export function shouldEmitComposedEditorValue(composingValue: string | null, appliedPendingExternalValue: boolean): composingValue is string {
  return composingValue !== null && !appliedPendingExternalValue
}
