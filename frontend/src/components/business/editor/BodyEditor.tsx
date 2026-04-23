import { useCallback, useEffect, useRef, useState } from "react"
import { BODY_TYPES, type BodyType } from "@/lib/constants"
import { useTabStore, getProjectActiveTabFromState } from "@/stores/tabStore"
import { useUIStore } from "@/stores/uiStore"
import { KeyValueEditor } from "./KeyValueEditor"
import { FormDataEditor } from "./FormDataEditor"
import { CodeEditor } from "@/components/ui/CodeEditor"
import { AppIcon } from "@/components/ui/icon"
import { useI18n } from "@/hooks/useI18n"
import { cn } from "@/lib/utils"

const BODY_TYPE_LABELS: Record<string, string> = {
  none: "none",
  raw: "raw",
  json: "JSON",
  "form-urlencoded": "x-www-form-urlencoded",
  "form-data": "form-data",
}

const ALL_BODY_TYPES = BODY_TYPES

export function BodyEditor() {
  const { t } = useI18n()
  const activeTab = useTabStore(getProjectActiveTabFromState)
  const updateTabRequest = useTabStore((s) => s.updateTabRequest)
  const { resolved } = useUIStore()
  const isDark = resolved === "dark"
  const tabId = activeTab?.id ?? ""
  const body = activeTab?.request.body ?? { type: "none" as const }
  const isCodeBody = body.type === "json" || body.type === "raw"
  const autoFormattedByTabRef = useRef<Record<string, boolean>>({})
  const [jsonFormatSignalByTab, setJsonFormatSignalByTab] = useState<Record<string, number>>({})

  const triggerJsonFormat = useCallback(() => {
    setJsonFormatSignalByTab((prev) => ({
      ...prev,
      [tabId]: (prev[tabId] ?? 0) + 1,
    }))
  }, [tabId])

  const setBodyType = (type: BodyType) => {
    if (!activeTab) return
    updateTabRequest(tabId, { body: { ...body, type } })
  }

  useEffect(() => {
    if (!activeTab) return
    if (body.type !== "json") {
      autoFormattedByTabRef.current[tabId] = false
      return
    }
    if (autoFormattedByTabRef.current[tabId]) return
    autoFormattedByTabRef.current[tabId] = true

    const current = body.json ?? ""
    if (!current.trim()) return
    triggerJsonFormat()
  }, [body.type, body.json, tabId, triggerJsonFormat])

  const handleFormatJson = () => {
    triggerJsonFormat()
  }

  const handleFormatRaw = () => {
    if (!activeTab) return
    const content = body.raw ?? ""
    try {
      const parsed = JSON.parse(content)
      updateTabRequest(tabId, { body: { ...body, raw: JSON.stringify(parsed, null, 2) } })
    } catch {
      // 非 JSON 格式则不处理
    }
  }

  if (!activeTab) return null

  return (
    <div className="flex flex-col h-full min-h-0">
      {/* Body 类型选择 */}
      <div className="flex items-center gap-1 px-[var(--size-padding-sm)] py-1.5 flex-shrink-0">
        {ALL_BODY_TYPES.map((t) => (
          <button
            key={t}
            className={cn(
              "px-2.5 py-1 rounded-[var(--radius-sm)] text-[length:var(--size-font-2xs)] transition-colors",
              body.type === t
                ? "bg-[var(--sidebar-active)] text-[var(--accent)] font-medium"
                : "text-[var(--fg-muted)] hover:text-[var(--fg-secondary)] hover:bg-[var(--sidebar-hover)]"
            )}
            onClick={() => setBodyType(t)}
          >
            {BODY_TYPE_LABELS[t] ?? t}
          </button>
        ))}

        {body.type === "json" && (
          <div className="ml-auto flex items-center gap-1">
            <button
              className="text-2xs text-[var(--accent)] hover:text-[var(--accent)]/90 px-1.5 py-0.5"
              onClick={handleFormatJson}
            >
              Beautify
            </button>
          </div>
        )}
        {body.type === "raw" && (
          <div className="ml-auto flex items-center gap-1">
            <RawFormatSelector />
            <button
              className="text-2xs text-[var(--accent)] hover:text-[var(--accent)]/90 px-1.5 py-0.5"
              onClick={handleFormatRaw}
            >
              Beautify
            </button>
          </div>
        )}
      </div>

      {/* Body 内容 */}
      <div className={cn("flex-1 min-h-0", isCodeBody ? "overflow-hidden" : "overflow-auto")}>
        {body.type === "none" && (
          <div className="text-center py-8 text-2xs text-[var(--fg-muted)]">
            {t("此请求没有请求体", "This request has no body")}
          </div>
        )}

        {body.type === "json" && (
          <div className="h-full flex flex-col">
            <div className="flex-1 min-h-0">
              <CodeEditor
                value={body.json ?? ""}
                onChange={(val) => updateTabRequest(tabId, { body: { ...body, json: val } })}
                language="json"
                placeholder='{"key": "value"}'
                isDark={isDark}
                fillParent
                formatSignal={jsonFormatSignalByTab[tabId]}
                enableSendShortcut
              />
            </div>
          </div>
        )}

        {body.type === "raw" && (
          <div className="h-full flex flex-col">
            <div className="flex-1 min-h-0">
              <CodeEditor
                value={body.raw ?? ""}
                onChange={(val) => updateTabRequest(tabId, { body: { ...body, raw: val } })}
                language="text"
                placeholder={t("请求体内容...", "Request body...")}
                isDark={isDark}
                fillParent
                enableSendShortcut
              />
            </div>
          </div>
        )}

        {body.type === "form-urlencoded" && (
          <KeyValueEditor
            items={body.formUrlEncoded ?? []}
            onChange={(formUrlEncoded) => updateTabRequest(tabId, { body: { ...body, formUrlEncoded } })}
            keyPlaceholder="Key"
            valuePlaceholder="Value"
          />
        )}

        {body.type === "form-data" && (
          <FormDataEditor />
        )}
      </div>
    </div>
  )
}

function RawFormatSelector() {
  return (
    <span className="text-2xs text-[var(--fg-muted)] px-1">
      raw
    </span>
  )
}
