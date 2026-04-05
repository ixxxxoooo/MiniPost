import { useState } from "react"
import { BODY_TYPES, type BodyType } from "@/lib/constants"
import { useTabStore, getProjectActiveTabFromState } from "@/stores/tabStore"
import { useUIStore } from "@/stores/uiStore"
import { KeyValueEditor } from "./KeyValueEditor"
import { FormDataEditor } from "./FormDataEditor"
import { CodeEditor, formatJsonWithComments } from "@/components/ui/CodeEditor"
import { AppIcon } from "@/components/ui/icon"
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
  const activeTab = useTabStore(getProjectActiveTabFromState)
  const updateTabRequest = useTabStore((s) => s.updateTabRequest)
  const { resolved } = useUIStore()
  const isDark = resolved === "dark"

  if (!activeTab) return null

  const { body } = activeTab.request
  const tabId = activeTab.id

  const setBodyType = (type: BodyType) => {
    updateTabRequest(tabId, { body: { ...body, type } })
  }

  const handleFormatJson = () => {
    const content = body.json ?? ""
    const formatted = formatJsonWithComments(content)
    updateTabRequest(tabId, { body: { ...body, json: formatted } })
  }

  const handleFormatRaw = () => {
    const content = body.raw ?? ""
    try {
      const parsed = JSON.parse(content)
      updateTabRequest(tabId, { body: { ...body, raw: JSON.stringify(parsed, null, 2) } })
    } catch {
      // 非 JSON 格式则不处理
    }
  }

  return (
    <div className="flex flex-col h-full">
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
              className="text-2xs text-[var(--accent)] hover:underline px-1.5 py-0.5"
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
              className="text-2xs text-[var(--accent)] hover:underline px-1.5 py-0.5"
              onClick={handleFormatRaw}
            >
              Beautify
            </button>
          </div>
        )}
      </div>

      {/* Body 内容 */}
      <div className="flex-1 overflow-auto min-h-0">
        {body.type === "none" && (
          <div className="text-center py-8 text-2xs text-[var(--fg-muted)]">
            此请求没有请求体
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
                placeholder="请求体内容..."
                isDark={isDark}
                fillParent
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
