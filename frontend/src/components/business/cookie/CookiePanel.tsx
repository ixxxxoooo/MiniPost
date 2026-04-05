import { useState } from "react"
import { AppIcon } from "@/components/ui/icon"
import { useCookieStore, type CookieItem } from "@/stores/cookieStore"
import { cn } from "@/lib/utils"

export function CookiePanel() {
  const { cookies, cookiePanelOpen, toggleCookiePanel, addCookie, updateCookie, removeCookie, clearCookies } = useCookieStore()
  const [editingId, setEditingId] = useState<string | null>(null)

  if (!cookiePanelOpen) return null

  const handleAdd = () => {
    const newCookie = {
      domain: "",
      name: "",
      value: "",
      path: "/",
    }
    addCookie(newCookie)
    const allCookies = useCookieStore.getState().cookies
    setEditingId(allCookies[allCookies.length - 1]?.id ?? null)
  }

  return (
    <div className="border-t border-[var(--border-color)] bg-[var(--surface)] flex flex-col" style={{ height: 240 }}>
      <div className="flex items-center justify-between h-[28px] px-3 border-b border-[var(--border-color)] bg-[var(--surface-secondary)] flex-shrink-0">
        <div className="flex items-center gap-2">
          <AppIcon name="cookie" size={12} className="text-[var(--fg-secondary)]" />
          <span className="text-[11px] font-medium text-[var(--fg)]">Cookies</span>
          {cookies.length > 0 && (
            <span className="text-[10px] text-[var(--fg-muted)]">({cookies.length})</span>
          )}
        </div>
        <div className="flex items-center gap-1">
          <button
            className="h-5 px-1.5 flex items-center gap-1 rounded-[4px] text-[10px] text-[var(--fg-muted)] hover:bg-[var(--sidebar-hover)] transition-colors"
            onClick={handleAdd}
            title="添加 Cookie"
          >
            <AppIcon name="add" size={10} />
            添加
          </button>
          <button
            className="h-5 px-1.5 flex items-center gap-1 rounded-[4px] text-[10px] text-[var(--fg-muted)] hover:bg-[var(--sidebar-hover)] transition-colors"
            onClick={clearCookies}
            title="清空"
          >
            Clear
          </button>
          <button
            className="h-5 w-5 flex items-center justify-center rounded-[4px] hover:bg-[var(--sidebar-hover)] transition-colors"
            onClick={toggleCookiePanel}
            title="关闭"
          >
            <AppIcon name="clear" size={10} className="text-[var(--fg-muted)]" />
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-auto">
        {cookies.length === 0 ? (
          <div className="text-center py-8 text-[11px] text-[var(--fg-muted)]">
            暂无 Cookie，点击"添加"创建
          </div>
        ) : (
          <table className="w-full text-[11px] border-collapse">
            <thead>
              <tr className="border-b border-[var(--border-color)] bg-[var(--surface-secondary)]/50">
                <th className="text-left px-2 py-1.5 font-medium text-[var(--fg-muted)] w-[24px]" />
                <th className="text-left px-2 py-1.5 font-medium text-[var(--fg-muted)]">Domain</th>
                <th className="text-left px-2 py-1.5 font-medium text-[var(--fg-muted)]">Name</th>
                <th className="text-left px-2 py-1.5 font-medium text-[var(--fg-muted)]">Value</th>
                <th className="text-left px-2 py-1.5 font-medium text-[var(--fg-muted)] w-[60px]">Path</th>
                <th className="w-[50px] px-2 py-1.5" />
              </tr>
            </thead>
            <tbody>
              {cookies.map((cookie) => (
                <CookieRow
                  key={cookie.id}
                  cookie={cookie}
                  isEditing={editingId === cookie.id}
                  onEdit={() => setEditingId(editingId === cookie.id ? null : cookie.id)}
                  onUpdate={(updates) => updateCookie(cookie.id, updates)}
                  onRemove={() => removeCookie(cookie.id)}
                />
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}

function CookieRow({
  cookie,
  isEditing,
  onEdit,
  onUpdate,
  onRemove,
}: {
  cookie: CookieItem
  isEditing: boolean
  onEdit: () => void
  onUpdate: (updates: Partial<CookieItem>) => void
  onRemove: () => void
}) {
  return (
    <tr
      className={cn(
        "border-b border-[var(--border-color)]/50 group transition-colors cursor-pointer",
        isEditing ? "bg-[var(--sidebar-active)]" : "hover:bg-[var(--surface-secondary)]/50"
      )}
      onClick={onEdit}
    >
      <td className="px-2 py-1">
        <input
          type="checkbox"
          checked={cookie.enabled}
          onChange={(e) => { e.stopPropagation(); onUpdate({ enabled: e.target.checked }) }}
          className="w-3 h-3 rounded accent-[var(--accent)]"
        />
      </td>
      <td className="px-2 py-1">
        {isEditing ? (
          <input
            value={cookie.domain}
            onChange={(e) => onUpdate({ domain: e.target.value })}
            onClick={(e) => e.stopPropagation()}
            placeholder="例如 .example.com"
            className="w-full bg-transparent text-[var(--fg)] text-[11px] outline-none border-b border-[var(--accent)]"
            autoFocus
          />
        ) : (
          <span className={cn("text-[var(--fg)]", !cookie.domain && "text-[var(--fg-muted)]")}>
            {cookie.domain || "-"}
          </span>
        )}
      </td>
      <td className="px-2 py-1">
        {isEditing ? (
          <input
            value={cookie.name}
            onChange={(e) => onUpdate({ name: e.target.value })}
            onClick={(e) => e.stopPropagation()}
            placeholder="cookie名称"
            className="w-full bg-transparent text-[var(--fg)] text-[11px] outline-none border-b border-[var(--accent)]"
          />
        ) : (
          <span className="text-[var(--fg)] font-medium">{cookie.name || "-"}</span>
        )}
      </td>
      <td className="px-2 py-1">
        {isEditing ? (
          <input
            value={cookie.value}
            onChange={(e) => onUpdate({ value: e.target.value })}
            onClick={(e) => e.stopPropagation()}
            placeholder="cookie值"
            className="w-full bg-transparent text-[var(--fg)] text-[11px] outline-none border-b border-[var(--accent)]"
          />
        ) : (
          <span className="text-[var(--fg-secondary)] truncate max-w-[200px] block">{cookie.value || "-"}</span>
        )}
      </td>
      <td className="px-2 py-1">
        {isEditing ? (
          <input
            value={cookie.path}
            onChange={(e) => onUpdate({ path: e.target.value })}
            onClick={(e) => e.stopPropagation()}
            placeholder="/"
            className="w-full bg-transparent text-[var(--fg)] text-[11px] outline-none border-b border-[var(--accent)]"
          />
        ) : (
          <span className="text-[var(--fg-muted)]">{cookie.path}</span>
        )}
      </td>
      <td className="px-2 py-1">
        <button
          className="opacity-0 group-hover:opacity-100 h-4 w-4 flex items-center justify-center rounded text-[var(--fg-muted)] hover:text-[var(--danger)] transition-all"
          onClick={(e) => { e.stopPropagation(); onRemove() }}
          title="删除"
        >
          <AppIcon name="delete" size={10} />
        </button>
      </td>
    </tr>
  )
}
