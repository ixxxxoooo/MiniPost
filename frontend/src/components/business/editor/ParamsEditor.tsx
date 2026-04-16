import { useCallback } from "react"
import { KeyValueEditor } from "./KeyValueEditor"
import { useI18n } from "@/hooks/useI18n"
import { useTabStore, getProjectActiveTabFromState } from "@/stores/tabStore"
import { buildUrlWithParams } from "@/lib/urlQuerySync"
import type { KeyValuePair } from "@/types/request"

export function ParamsEditor() {
  const { t } = useI18n()
  const activeTab = useTabStore(getProjectActiveTabFromState)
  const updateTabRequest = useTabStore((s) => s.updateTabRequest)
  const activeTabId = activeTab?.id
  const activeUrl = activeTab?.request.url ?? ""

  const handleParamsChange = useCallback((params: KeyValuePair[]) => {
    if (!activeTabId) return
    updateTabRequest(activeTabId, {
      params,
      url: buildUrlWithParams(activeUrl, params),
    })
  }, [activeTabId, activeUrl, updateTabRequest])

  if (!activeTab) return null

  return (
    <KeyValueEditor
      items={activeTab.request.params}
      onChange={handleParamsChange}
      keyPlaceholder={t("参数名", "Param name")}
      valuePlaceholder={t("参数值", "Param value")}
    />
  )
}
