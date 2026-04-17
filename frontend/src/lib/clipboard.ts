import { ClipboardGetText, ClipboardSetText } from "../../wailsjs/runtime/runtime"

export async function writeClipboardText(text: string): Promise<boolean> {
  try {
    const copied = await ClipboardSetText(text)
    if (copied) return true
  } catch {
    // Fall back to the browser clipboard API when the Wails bridge is unavailable.
  }

  if (typeof navigator !== "undefined" && navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch {
      return false
    }
  }

  return false
}

export async function readClipboardText(): Promise<string | null> {
  try {
    const text = await ClipboardGetText()
    if (typeof text === "string") return text
  } catch {
    // Fall back to the browser clipboard API when the Wails bridge is unavailable.
  }

  if (typeof navigator !== "undefined" && navigator.clipboard?.readText) {
    try {
      const text = await navigator.clipboard.readText()
      return typeof text === "string" ? text : null
    } catch {
      return null
    }
  }

  return null
}
