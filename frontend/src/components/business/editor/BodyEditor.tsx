import { BODY_TYPES, type BodyType } from "@/lib/constants"
import { useTabStore } from "@/stores/tabStore"
import { KeyValueEditor } from "./KeyValueEditor"
import { cn } from "@/lib/utils"

export function BodyEditor() {
  const activeTab = useTabStore((s) => s.getActiveTab())
  const updateTabRequest = useTabStore((s) => s.updateTabRequest)

  if (!activeTab) return null

  const { body } = activeTab.request
  const tabId = activeTab.id

  const setBodyType = (type: BodyType) => {
    updateTabRequest(tabId, { body: { ...body, type } })
  }

  return (
    <div className="p-[var(--size-padding-sm)]">
      {/* Body 类型选择 */}
      <div className="flex items-center gap-1 mb-2">
        {BODY_TYPES.map((t) => (
          <button
            key={t}
            className={cn(
              "px-2 py-1 rounded-[var(--radius-sm)] text-[length:var(--size-font-2xs)] transition-colors",
              body.type === t
                ? "bg-[var(--sidebar-active)] text-[var(--accent)] font-medium"
                : "text-[var(--fg-muted)] hover:text-[var(--fg-secondary)] hover:bg-[var(--sidebar-hover)]"
            )}
            onClick={() => setBodyType(t)}
          >
            {t === "none" ? "None" : t === "json" ? "JSON" : t === "raw" ? "Raw" : "Form"}
          </button>
        ))}
      </div>

      {/* Body 内容 */}
      {body.type === "none" && (
        <div className="text-center py-8 text-2xs text-[var(--fg-muted)]">
          此请求没有请求体
        </div>
      )}

      {body.type === "json" && (
        <textarea
          value={body.json ?? ""}
          onChange={(e) => updateTabRequest(tabId, { body: { ...body, json: e.target.value } })}
          placeholder='{"key": "value"}'
          className={cn(
            "w-full min-h-[200px] p-3 rounded-[var(--radius-input)]",
            "border border-[var(--border-color)] bg-[var(--surface)]",
            "text-[var(--fg)] font-mono text-[length:var(--size-font-2xs)]",
            "placeholder:text-[var(--fg-muted)] resize-y",
            "focus:outline-none focus:border-[var(--accent)]"
          )}
        />
      )}

      {body.type === "raw" && (
        <textarea
          value={body.raw ?? ""}
          onChange={(e) => updateTabRequest(tabId, { body: { ...body, raw: e.target.value } })}
          placeholder="请求体内容..."
          className={cn(
            "w-full min-h-[200px] p-3 rounded-[var(--radius-input)]",
            "border border-[var(--border-color)] bg-[var(--surface)]",
            "text-[var(--fg)] font-mono text-[length:var(--size-font-2xs)]",
            "placeholder:text-[var(--fg-muted)] resize-y",
            "focus:outline-none focus:border-[var(--accent)]"
          )}
        />
      )}

      {body.type === "form-urlencoded" && (
        <KeyValueEditor
          items={body.formUrlEncoded ?? []}
          onChange={(formUrlEncoded) => updateTabRequest(tabId, { body: { ...body, formUrlEncoded } })}
          keyPlaceholder="字段名"
          valuePlaceholder="字段值"
        />
      )}
    </div>
  )
}
