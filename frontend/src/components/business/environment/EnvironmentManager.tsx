import { useState, useEffect } from "react"
import { AppIcon } from "@/components/ui/icon"
import { cn } from "@/lib/utils"
import { useProjectStore } from "@/stores/projectStore"
import { useEnvironmentStore } from "@/stores/environmentStore"
import { useUIStore } from "@/stores/uiStore"

export function EnvironmentManager() {
  const { currentProjectId } = useProjectStore()
  const {
    environments, activeEnvironmentId,
    createEnvironment, deleteEnvironment, setActiveEnvironment,
  } = useEnvironmentStore()
  const { editingEnvironmentId, setEditingEnvironmentId, closeEnvironmentTab } = useUIStore()

  const [newEnvName, setNewEnvName] = useState("")
  const [showNewInput, setShowNewInput] = useState(false)

  const handleCreateEnv = async () => {
    if (!currentProjectId || !newEnvName.trim()) return
    const name = newEnvName.trim()
    const created = await createEnvironment(currentProjectId, name)
    setNewEnvName("")
    setShowNewInput(false)
    if (created?.id) {
      setEditingEnvironmentId(created.id)
      return
    }
    // 兜底：极端情况下后端未返回完整对象时，从最新列表反查
    const allEnvs = useEnvironmentStore.getState().environments
    const fallback = allEnvs.filter((e) => e.name === name).at(-1)
    if (fallback?.id) {
      setEditingEnvironmentId(fallback.id)
    }
  }

  const handleDeleteEnv = async (envId: string) => {
    if (!currentProjectId) return
    await deleteEnvironment(currentProjectId, envId)
    closeEnvironmentTab(envId)
  }

  if (!currentProjectId) return null

  return (
    <div className="flex flex-col h-full">
      <div className="flex-shrink-0">
        <div className="flex items-center justify-between px-2 py-1">
          <span className="text-2xs text-[var(--fg-muted)] font-medium uppercase">环境变量</span>
          <button
            className="h-5 w-5 flex items-center justify-center rounded-[var(--radius-sm)] hover:bg-[var(--sidebar-hover)] transition-colors"
            onClick={() => setShowNewInput(true)}
            title="新建环境"
          >
            <AppIcon name="add" size={12} className="text-[var(--fg-muted)]" />
          </button>
        </div>

        {showNewInput && (
          <div className="px-2 pb-1">
            <input
              className="w-full h-[var(--size-btn-sm)] px-2 text-[length:var(--size-font-2xs)] rounded-[var(--radius-input)] border border-[var(--border-color)] bg-[var(--surface)] text-[var(--fg)] outline-none focus:border-[var(--accent)]"
              placeholder="环境名称..."
              value={newEnvName}
              onChange={(e) => setNewEnvName(e.target.value)}
              onKeyDown={(e) => { if (e.key === "Enter") void handleCreateEnv(); if (e.key === "Escape") setShowNewInput(false) }}
              autoFocus
            />
          </div>
        )}

        {environments.map((env) => (
          <div
            key={env.id}
            className={cn(
              "flex items-center h-[24px] px-2 mx-1 rounded-[var(--radius-btn)] cursor-pointer group",
              editingEnvironmentId === env.id ? "bg-[var(--sidebar-active)]" : "hover:bg-[var(--sidebar-hover)]"
            )}
            onClick={() => setEditingEnvironmentId(env.id)}
          >
            <button
              className={cn(
                "w-3 h-3 rounded-full border mr-2 flex items-center justify-center flex-shrink-0 transition-colors",
                activeEnvironmentId === env.id
                  ? "bg-[var(--accent)] border-[var(--accent)]"
                  : "border-[var(--border-color)] hover:border-[var(--accent)]"
              )}
              onClick={(e) => {
                e.stopPropagation()
                setActiveEnvironment(activeEnvironmentId === env.id ? null : env.id)
              }}
              title={activeEnvironmentId === env.id ? "取消激活" : "激活此环境"}
            >
              {activeEnvironmentId === env.id && (
                <div className="w-1.5 h-1.5 rounded-full bg-white" />
              )}
            </button>
            <span className={cn(
              "text-[length:var(--size-font-2xs)] truncate flex-1",
              editingEnvironmentId === env.id ? "text-[var(--accent)] font-medium" : "text-[var(--sidebar-fg)]"
            )}>
              {env.name}
            </span>
            <button
              className="opacity-0 group-hover:opacity-100 h-4 w-4 flex items-center justify-center rounded-[var(--radius-sm)] hover:bg-[var(--sidebar-hover)] text-[var(--fg-muted)] hover:text-[var(--danger)] transition-all flex-shrink-0"
              onClick={(e) => { e.stopPropagation(); void handleDeleteEnv(env.id) }}
            >
              <AppIcon name="delete" size={10} />
            </button>
          </div>
        ))}

        {environments.length === 0 && (
          <div className="text-center py-6 text-2xs text-[var(--fg-muted)]">
            暂无环境，点击 + 创建
          </div>
        )}
      </div>
    </div>
  )
}
