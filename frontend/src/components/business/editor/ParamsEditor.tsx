import { KeyValueEditor } from "./KeyValueEditor"
import { useTabStore } from "@/stores/tabStore"

export function ParamsEditor() {
  const activeTab = useTabStore((s) => s.getActiveTab())
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
