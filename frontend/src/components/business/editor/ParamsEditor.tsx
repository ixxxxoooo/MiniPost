import { KeyValueEditor } from "./KeyValueEditor"
import { useI18n } from "@/hooks/useI18n"
import { useTabStore, getProjectActiveTabFromState } from "@/stores/tabStore"

export function ParamsEditor() {
  const { t } = useI18n()
  const activeTab = useTabStore(getProjectActiveTabFromState)
  const updateTabRequest = useTabStore((s) => s.updateTabRequest)

  if (!activeTab) return null

  return (
    <KeyValueEditor
      items={activeTab.request.params}
      onChange={(params) => updateTabRequest(activeTab.id, { params })}
      keyPlaceholder={t("参数名", "Param name")}
      valuePlaceholder={t("参数值", "Param value")}
    />
  )
}
