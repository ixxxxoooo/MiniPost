import { useState, useEffect, useRef, useCallback } from "react"
import { AppIcon } from "@/components/ui/icon"
import { cn } from "@/lib/utils"
import type { KeyValuePair } from "@/types/request"
import { createKeyValuePair } from "@/types/request"

interface KeyValueEditorProps {
  items: KeyValuePair[]
  onChange: (items: KeyValuePair[]) => void
  keyPlaceholder?: string
  valuePlaceholder?: string
  showDescription?: boolean
  supportFile?: boolean
}

function parseBulkText(text: string): KeyValuePair[] {
  return text
    .split("\n")
    .filter((line) => line.trim())
    .map((line) => {
      const colonIdx = line.indexOf(":")
      if (colonIdx === -1) {
        return createKeyValuePair({ key: line.trim(), value: "" })
      }
      const key = line.slice(0, colonIdx).trim()
      const value = line.slice(colonIdx + 1).trim()
      return createKeyValuePair({ key, value })
    })
}

function serializeBulkText(items: KeyValuePair[]): string {
  return items
    .filter((item) => item.key)
    .map((item) => `${item.key}: ${item.value}`)
    .join("\n")
}

export function KeyValueEditor({
  items,
  onChange,
  keyPlaceholder = "Key",
  valuePlaceholder = "Value",
  showDescription = true,
  supportFile = false,
}: KeyValueEditorProps) {
  const [bulkMode, setBulkMode] = useState(false)
  const [bulkText, setBulkText] = useState("")
  const newKeyRef = useRef<HTMLInputElement>(null)

  // 当最后一行有内容时，自动在底部保持一行空行占位
  // items 为空时默认显示一行空行
  const hasEmptyRow = items.length > 0 && items[items.length - 1].key === "" && items[items.length - 1].value === ""

  const handleUpdate = (id: string, field: keyof KeyValuePair, value: string | boolean) => {
    const updated = items.map((item) => (item.id === id ? { ...item, [field]: value } : item))
    onChange(updated)
  }

  const handleDelete = (id: string) => {
    onChange(items.filter((item) => item.id !== id))
  }

  // 当用户在最后一行空行的 key 输入框聚焦 / 输入时，添加新行
  const handleNewRowKeyChange = (id: string, value: string) => {
    const updated = items.map((item) => (item.id === id ? { ...item, key: value } : item))
    // 如果这一行不再是空的，追加一行空行
    if (value) {
      updated.push(createKeyValuePair())
    }
    onChange(updated)
  }

  // 没有任何 items 时，自动提供一行空行
  useEffect(() => {
    if (items.length === 0) {
      onChange([createKeyValuePair()])
    }
  }, [items.length, onChange])

  const enterBulkMode = () => {
    setBulkText(serializeBulkText(items))
    setBulkMode(true)
  }

  const exitBulkMode = () => {
    const parsed = parseBulkText(bulkText)
    if (parsed.length === 0 || parsed[parsed.length - 1].key !== "") {
      parsed.push(createKeyValuePair())
    }
    onChange(parsed)
    setBulkMode(false)
  }

  if (bulkMode) {
    return (
      <div className="p-[var(--size-padding-sm)]">
        <div className="flex items-center justify-between mb-2">
          <span className="text-[10px] text-[var(--fg-muted)]">
            每行一条，格式：key: value（兼容 Chrome 复制的带空格格式）
          </span>
          <button
            className="text-[10px] text-[var(--accent)] hover:underline"
            onClick={exitBulkMode}
          >
            退出批量编辑
          </button>
        </div>
        <textarea
          value={bulkText}
          onChange={(e) => setBulkText(e.target.value)}
          className={cn(
            "w-full min-h-[200px] p-3 rounded-[var(--radius-input)]",
            "border border-[var(--border-color)] bg-[var(--surface)]",
            "text-[var(--fg)] font-mono text-[11px]",
            "placeholder:text-[var(--fg-muted)] resize-y",
            "focus:outline-none focus:border-[var(--accent)]"
          )}
          placeholder={"Content-Type: application/json\nAuthorization: Bearer token"}
          autoFocus
        />
      </div>
    )
  }

  return (
    <div className="px-[var(--size-padding-sm)] pt-2">
      <table className="w-full border-collapse border border-[var(--border-color)]" style={{ fontSize: "11px" }}>
        <thead>
          <tr>
            <th className="w-7 border border-[var(--border-color)] px-1 py-1" />
            <th className="text-left border border-[var(--border-color)] px-2 py-1 font-semibold text-[var(--fg-secondary)]">{keyPlaceholder}</th>
            <th className="text-left border border-[var(--border-color)] px-2 py-1 font-semibold text-[var(--fg-secondary)]">{valuePlaceholder}</th>
            {showDescription && (
              <th className="text-left border border-[var(--border-color)] px-2 py-1 font-semibold text-[var(--fg-secondary)]">Description</th>
            )}
            <th className="w-16 border border-[var(--border-color)] px-1 py-1 text-right">
              <button
                className="text-[10px] text-[var(--fg-muted)] hover:text-[var(--accent)] transition-colors whitespace-nowrap"
                onClick={enterBulkMode}
                title="批量编辑"
              >
                Bulk Edit
              </button>
            </th>
          </tr>
        </thead>
        <tbody>
          {items.map((item, idx) => {
            const isLastEmpty = idx === items.length - 1 && !item.key && !item.value
            return (
              <tr key={item.id} className="group hover:bg-[var(--surface-secondary)]/50 transition-colors">
                <td className="border border-[var(--border-color)] px-1 py-0 text-center">
                  <input
                    type="checkbox"
                    checked={item.enabled}
                    onChange={(e) => handleUpdate(item.id, "enabled", e.target.checked)}
                    className="w-3 h-3 rounded accent-[var(--accent)] cursor-pointer"
                    disabled={isLastEmpty}
                    style={isLastEmpty ? { opacity: 0.3 } : {}}
                  />
                </td>
                <td className="border border-[var(--border-color)] px-0 py-0">
                  <input
                    ref={isLastEmpty ? newKeyRef : undefined}
                    value={item.key}
                    onChange={(e) => {
                      if (isLastEmpty) {
                        handleNewRowKeyChange(item.id, e.target.value)
                      } else {
                        handleUpdate(item.id, "key", e.target.value)
                      }
                    }}
                    placeholder={keyPlaceholder}
                    className={cn(
                      "w-full h-[24px] px-2 bg-transparent text-[var(--fg)] font-mono",
                      "text-[11px] placeholder:text-[var(--fg-muted)] placeholder:italic",
                      "focus:outline-none",
                      !item.enabled && !isLastEmpty && "opacity-40"
                    )}
                  />
                </td>
                <td className="border border-[var(--border-color)] px-0 py-0">
                  <input
                    value={item.value}
                    onChange={(e) => handleUpdate(item.id, "value", e.target.value)}
                    placeholder={valuePlaceholder}
                    className={cn(
                      "w-full h-[24px] px-2 bg-transparent text-[var(--fg)] font-mono",
                      "text-[11px] placeholder:text-[var(--fg-muted)] placeholder:italic",
                      "focus:outline-none",
                      !item.enabled && !isLastEmpty && "opacity-40"
                    )}
                    disabled={isLastEmpty}
                  />
                </td>
                {showDescription && (
                  <td className="border border-[var(--border-color)] px-0 py-0">
                    <input
                      value={item.description ?? ""}
                      onChange={(e) => handleUpdate(item.id, "description", e.target.value)}
                      placeholder="Description"
                      className={cn(
                        "w-full h-[24px] px-2 bg-transparent text-[var(--fg-secondary)]",
                        "text-[11px] placeholder:text-[var(--fg-muted)] placeholder:italic",
                        "focus:outline-none",
                        !item.enabled && !isLastEmpty && "opacity-40"
                      )}
                      disabled={isLastEmpty}
                    />
                  </td>
                )}
                <td className="border border-[var(--border-color)] px-1 py-0 text-center">
                  {!isLastEmpty && (
                    <button
                      className="h-4 w-4 inline-flex items-center justify-center rounded-[2px] opacity-0 group-hover:opacity-100 hover:bg-[var(--sidebar-hover)] text-[var(--fg-muted)] hover:text-[var(--danger)] transition-all"
                      onClick={() => handleDelete(item.id)}
                    >
                      <AppIcon name="delete" size={10} />
                    </button>
                  )}
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}
