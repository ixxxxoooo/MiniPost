import { ClipboardSetText } from "../../wailsjs/runtime/runtime"

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
