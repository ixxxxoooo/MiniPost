import { useState, useEffect } from "react"
import { Plus, Trash2, Save, Eye, EyeOff, Check } from "lucide-react"
import { cn } from "@/lib/utils"
import { useProjectStore } from "@/stores/projectStore"
import { useEnvironmentStore } from "@/stores/environmentStore"

interface EditableEnv {
  id: string
  name: string
  projectId: string
  variables: { id: string; key: string; value: string; enabled: boolean; isSecret: boolean }[]
}

export function EnvironmentManager() {
  const { currentProjectId } = useProjectStore()
  const {
    environments, activeEnvironmentId, loadEnvironments,
    createEnvironment, saveEnvironment, deleteEnvironment, setActiveEnvironment,
  } = useEnvironmentStore()

  const [selectedEnvId, setSelectedEnvId] = useState<string | null>(null)
  const [editingEnv, setEditingEnv] = useState<EditableEnv | null>(null)
  const [newEnvName, setNewEnvName] = useState("")
  const [showNewInput, setShowNewInput] = useState(false)

  useEffect(() => {
    if (selectedEnvId) {
      const env = environments.find((e) => e.id === selectedEnvId)
      if (env) setEditingEnv(JSON.parse(JSON.stringify(env)))
    }
  }, [selectedEnvId, environments])

  const handleCreateEnv = async () => {
    if (!currentProjectId || !newEnvName.trim()) return
    await createEnvironment(currentProjectId, newEnvName.trim())
    setNewEnvName("")
    setShowNewInput(false)
  }

  const handleSaveEnv = async () => {
    if (!editingEnv) return
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    await saveEnvironment(editingEnv as any)
  }

  const handleDeleteEnv = async (envId: string) => {
    if (!currentProjectId) return
    await deleteEnvironment(currentProjectId, envId)
    if (selectedEnvId === envId) {
      setSelectedEnvId(null)
      setEditingEnv(null)
    }
  }

  const addVariable = () => {
    if (!editingEnv) return
    setEditingEnv({
      ...editingEnv,
      variables: [
        ...(editingEnv.variables ?? []),
        { id: crypto.randomUUID(), key: "", value: "", enabled: true, isSecret: false },
      ],
    })
  }

  const updateVariable = (varId: string, field: string, value: string | boolean) => {
    if (!editingEnv) return
    setEditingEnv({
      ...editingEnv,
      variables: (editingEnv.variables ?? []).map((v) =>
        v.id === varId ? { ...v, [field]: value } : v
      ),
    })
  }

  const removeVariable = (varId: string) => {
    if (!editingEnv) return
    setEditingEnv({
      ...editingEnv,
      variables: (editingEnv.variables ?? []).filter((v) => v.id !== varId),
    })
  }

  if (!currentProjectId) return null

  return (
    <div className="flex flex-col h-full">
      {/* 环境列表 */}
      <div className="flex-shrink-0">
        <div className="flex items-center justify-between px-2 py-1">
          <span className="text-2xs text-[var(--fg-muted)] font-medium uppercase">环境变量</span>
          <button
            className="h-5 w-5 flex items-center justify-center rounded-[var(--radius-sm)] hover:bg-[var(--sidebar-hover)] transition-colors"
            onClick={() => setShowNewInput(true)}
            title="新建环境"
          >
            <Plus className="h-3 w-3 text-[var(--fg-muted)]" />
          </button>
        </div>

        {showNewInput && (
          <div className="px-2 pb-1">
            <input
              className="w-full h-[var(--size-btn-sm)] px-2 text-[length:var(--size-font-2xs)] rounded-[var(--radius-input)] border border-[var(--border-color)] bg-[var(--surface)] text-[var(--fg)] outline-none focus:border-[var(--accent)]"
              placeholder="环境名称..."
              value={newEnvName}
              onChange={(e) => setNewEnvName(e.target.value)}
              onKeyDown={(e) => { if (e.key === "Enter") handleCreateEnv(); if (e.key === "Escape") setShowNewInput(false) }}
              autoFocus
            />
          </div>
        )}

        {environments.map((env) => (
          <div
            key={env.id}
            className={cn(
              "flex items-center h-[24px] px-2 mx-1 rounded-[var(--radius-btn)] cursor-pointer group",
              selectedEnvId === env.id ? "bg-[var(--sidebar-active)]" : "hover:bg-[var(--sidebar-hover)]"
            )}
            onClick={() => setSelectedEnvId(env.id)}
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
              {activeEnvironmentId === env.id && <Check className="h-2 w-2 text-white" />}
            </button>
            <span className={cn(
              "text-[length:var(--size-font-2xs)] truncate flex-1",
              selectedEnvId === env.id ? "text-[var(--accent)] font-medium" : "text-[var(--sidebar-fg)]"
            )}>
              {env.name}
            </span>
            <button
              className="opacity-0 group-hover:opacity-100 h-4 w-4 flex items-center justify-center rounded-[var(--radius-sm)] hover:bg-[var(--sidebar-hover)] text-[var(--fg-muted)] hover:text-[var(--danger)] transition-all flex-shrink-0"
              onClick={(e) => { e.stopPropagation(); handleDeleteEnv(env.id) }}
            >
              <Trash2 className="h-2.5 w-2.5" />
            </button>
          </div>
        ))}
      </div>

      {/* 变量编辑区 */}
      {editingEnv && (
        <div className="flex-1 overflow-y-auto mt-2 border-t border-[var(--border-subtle)] pt-2">
          <div className="flex items-center justify-between px-2 mb-1">
            <span className="text-2xs text-[var(--fg-secondary)] font-medium">{editingEnv.name} 变量</span>
            <div className="flex items-center gap-1">
              <button
                className="h-5 flex items-center gap-0.5 px-1.5 rounded-[var(--radius-sm)] text-2xs text-[var(--fg-muted)] hover:bg-[var(--sidebar-hover)] transition-colors"
                onClick={addVariable}
              >
                <Plus className="h-2.5 w-2.5" /> 添加
              </button>
              <button
                className="h-5 flex items-center gap-0.5 px-1.5 rounded-[var(--radius-sm)] text-2xs text-[var(--accent)] hover:bg-[var(--sidebar-hover)] transition-colors"
                onClick={handleSaveEnv}
              >
                <Save className="h-2.5 w-2.5" /> 保存
              </button>
            </div>
          </div>

          {(editingEnv.variables ?? []).map((v) => (
            <div key={v.id} className="flex items-center gap-1 px-2 mb-0.5 group">
              <input
                type="checkbox"
                checked={v.enabled}
                onChange={(e) => updateVariable(v.id, "enabled", e.target.checked)}
                className="w-3 h-3 rounded accent-[var(--accent)] flex-shrink-0"
              />
              <input
                value={v.key}
                onChange={(e) => updateVariable(v.id, "key", e.target.value)}
                placeholder="变量名"
                className={cn(
                  "flex-1 h-[20px] px-1.5 rounded-[var(--radius-sm)] bg-transparent",
                  "text-[length:var(--size-font-2xs)] font-mono text-[var(--fg)]",
                  "border border-transparent focus:border-[var(--border-color)] focus:bg-[var(--surface)] outline-none",
                  "placeholder:text-[var(--fg-muted)]",
                  !v.enabled && "opacity-40"
                )}
              />
              <input
                value={v.value}
                onChange={(e) => updateVariable(v.id, "value", e.target.value)}
                placeholder="变量值"
                type={v.isSecret ? "password" : "text"}
                className={cn(
                  "flex-1 h-[20px] px-1.5 rounded-[var(--radius-sm)] bg-transparent",
                  "text-[length:var(--size-font-2xs)] font-mono text-[var(--fg)]",
                  "border border-transparent focus:border-[var(--border-color)] focus:bg-[var(--surface)] outline-none",
                  "placeholder:text-[var(--fg-muted)]",
                  !v.enabled && "opacity-40"
                )}
              />
              <button
                className="h-4 w-4 flex items-center justify-center text-[var(--fg-muted)] opacity-0 group-hover:opacity-100 transition-opacity"
                onClick={() => updateVariable(v.id, "isSecret", !v.isSecret)}
                title={v.isSecret ? "显示" : "隐藏"}
              >
                {v.isSecret ? <EyeOff className="h-2.5 w-2.5" /> : <Eye className="h-2.5 w-2.5" />}
              </button>
              <button
                className="h-4 w-4 flex items-center justify-center text-[var(--fg-muted)] hover:text-[var(--danger)] opacity-0 group-hover:opacity-100 transition-all"
                onClick={() => removeVariable(v.id)}
              >
                <Trash2 className="h-2.5 w-2.5" />
              </button>
            </div>
          ))}

          {(editingEnv.variables ?? []).length === 0 && (
            <div className="text-center py-4 text-2xs text-[var(--fg-muted)]">
              暂无变量，点击"添加"创建
            </div>
          )}
        </div>
      )}
    </div>
  )
}
