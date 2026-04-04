import { useCallback, useEffect, useMemo } from "react"
import { TooltipProvider } from "@/components/ui/tooltip"
import { AppLayout } from "@/components/layout/AppLayout"
import { useKeyboardShortcuts } from "@/hooks/useKeyboardShortcuts"
import { useUIStore } from "@/stores/uiStore"
import { useProjectStore } from "@/stores/projectStore"
import { useTabStore } from "@/stores/tabStore"

function App() {
  const { toggleSidebar } = useUIStore()
  const { currentProjectId, createRequest } = useProjectStore()
  const { addNewUnsavedTab } = useTabStore()

  useEffect(() => {
    useProjectStore.getState().loadProjects()
  }, [])

  const handleNewRequest = useCallback(async () => {
    if (!currentProjectId) return
    const req = await createRequest("", "New Request")
    if (req) {
      addNewUnsavedTab(currentProjectId)
    }
  }, [currentProjectId, createRequest, addNewUnsavedTab])

  const handleSave = useCallback(() => {
    window.dispatchEvent(new CustomEvent("minipost:save"))
  }, [])

  const shortcuts = useMemo(
    () => ({
      "mod+b": toggleSidebar,
      "mod+n": handleNewRequest,
      "mod+s": handleSave,
    }),
    [toggleSidebar, handleNewRequest, handleSave]
  )

  useKeyboardShortcuts(shortcuts)

  return (
    <TooltipProvider delayDuration={300}>
      <div className="window-frame fixed inset-0 bg-transparent">
        <div className="window-shell h-full w-full bg-[var(--surface)]">
          <AppLayout />
        </div>
      </div>
    </TooltipProvider>
  )
}

export default App
