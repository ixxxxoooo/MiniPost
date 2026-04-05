import { KeyValueEditor } from "./KeyValueEditor"
import { useTabStore, getProjectActiveTabFromState } from "@/stores/tabStore"

export function HeadersEditor() {
  const activeTab = useTabStore(getProjectActiveTabFromState)
  const updateTabRequest = useTabStore((s) => s.updateTabRequest)

  if (!activeTab) return null

  return (
    <KeyValueEditor
      items={activeTab.request.headers}
      onChange={(headers) => updateTabRequest(activeTab.id, { headers })}
      keyPlaceholder="Header 名"
      valuePlaceholder="Header 值"
    />
  )
}
