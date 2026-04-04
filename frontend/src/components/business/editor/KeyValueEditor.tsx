import { Plus, Trash2, GripVertical } from "lucide-react"
import { cn } from "@/lib/utils"
import type { KeyValuePair } from "@/types/request"
import { createKeyValuePair } from "@/types/request"

interface KeyValueEditorProps {
  items: KeyValuePair[]
  onChange: (items: KeyValuePair[]) => void
  keyPlaceholder?: string
  valuePlaceholder?: string
}

export function KeyValueEditor({
  items,
  onChange,
  keyPlaceholder = "Key",
  valuePlaceholder = "Value",
}: KeyValueEditorProps) {
  const handleAdd = () => {
    onChange([...items, createKeyValuePair()])
  }

  const handleUpdate = (id: string, field: keyof KeyValuePair, value: string | boolean) => {
    onChange(items.map((item) => (item.id === id ? { ...item, [field]: value } : item)))
  }

  const handleDelete = (id: string) => {
    onChange(items.filter((item) => item.id !== id))
  }

  return (
    <div className="p-[var(--size-padding-sm)]">
      {/* 表头 */}
      <div className="flex items-center gap-1 px-1 mb-1">
        <div className="w-5" />
        <div className="w-6" />
        <span className="flex-1 text-2xs font-medium text-[var(--fg-muted)] uppercase">{keyPlaceholder}</span>
        <span className="flex-1 text-2xs font-medium text-[var(--fg-muted)] uppercase">{valuePlaceholder}</span>
        <div className="w-6" />
      </div>

      {items.map((item) => (
        <div key={item.id} className="flex items-center gap-1 group mb-0.5">
          <GripVertical className="h-3 w-3 text-[var(--fg-muted)] opacity-0 group-hover:opacity-40 flex-shrink-0 cursor-grab" />

          {/* 启用/禁用 */}
          <input
            type="checkbox"
            checked={item.enabled}
            onChange={(e) => handleUpdate(item.id, "enabled", e.target.checked)}
            className="w-3.5 h-3.5 rounded accent-[var(--accent)] flex-shrink-0 cursor-pointer"
          />

          <input
            value={item.key}
            onChange={(e) => handleUpdate(item.id, "key", e.target.value)}
            placeholder={keyPlaceholder}
            className={cn(
              "flex-1 h-[var(--size-btn-sm)] px-2 rounded-[var(--radius-sm)]",
              "border border-transparent bg-transparent text-[var(--fg)]",
              "text-[length:var(--size-font-2xs)] font-mono",
              "placeholder:text-[var(--fg-muted)]",
              "focus:outline-none focus:border-[var(--border-color)] focus:bg-[var(--surface)]",
              !item.enabled && "opacity-40"
            )}
          />
          <input
            value={item.value}
            onChange={(e) => handleUpdate(item.id, "value", e.target.value)}
            placeholder={valuePlaceholder}
            className={cn(
              "flex-1 h-[var(--size-btn-sm)] px-2 rounded-[var(--radius-sm)]",
              "border border-transparent bg-transparent text-[var(--fg)]",
              "text-[length:var(--size-font-2xs)] font-mono",
              "placeholder:text-[var(--fg-muted)]",
              "focus:outline-none focus:border-[var(--border-color)] focus:bg-[var(--surface)]",
              !item.enabled && "opacity-40"
            )}
          />

          <button
            className="h-5 w-5 flex items-center justify-center rounded-[var(--radius-sm)] opacity-0 group-hover:opacity-100 hover:bg-[var(--sidebar-hover)] text-[var(--fg-muted)] hover:text-[var(--danger)] transition-all flex-shrink-0"
            onClick={() => handleDelete(item.id)}
          >
            <Trash2 className="h-3 w-3" />
          </button>
        </div>
      ))}

      <button
        className="flex items-center gap-1 mt-1 px-2 py-1 text-2xs text-[var(--fg-muted)] hover:text-[var(--accent)] transition-colors rounded-[var(--radius-sm)] hover:bg-[var(--sidebar-hover)]"
        onClick={handleAdd}
      >
        <Plus className="h-3 w-3" /> 添加
      </button>
    </div>
  )
}
