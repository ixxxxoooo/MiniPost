import { KeyValueEditor } from "./KeyValueEditor"
import { useTabStore, getProjectActiveTabFromState } from "@/stores/tabStore"

export function ParamsEditor() {
  const activeTab = useTabStore(getProjectActiveTabFromState)
  const updateTabRequest = useTabStore((s) => s.updateTabRequest)

  if (!activeTab) return null

  return (
    <KeyValueEditor
      items={activeTab.request.params}
      onChange={(params) => updateTabRequest(activeTab.id, { params })}
      keyPlaceholder="参数名"
      valuePlaceholder="参数值"
    />
  )
}
