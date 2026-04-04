import { useEffect } from "react"

interface ShortcutMap {
  [key: string]: () => void
}

export function useKeyboardShortcuts(shortcuts: ShortcutMap) {
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      const parts: string[] = []
      if (e.metaKey || e.ctrlKey) parts.push("mod")
      if (e.shiftKey) parts.push("shift")
      if (e.altKey) parts.push("alt")
      parts.push(e.key.toLowerCase())

      const combo = parts.join("+")
      const action = shortcuts[combo]

      if (action) {
        e.preventDefault()
        e.stopPropagation()
        action()
      }
    }

    window.addEventListener("keydown", handler)
    return () => window.removeEventListener("keydown", handler)
  }, [shortcuts])
}
