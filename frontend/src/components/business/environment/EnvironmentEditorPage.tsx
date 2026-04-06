import { useState, useEffect, useCallback, useMemo } from "react"
import { AppIcon } from "@/components/ui/icon"
import { useI18n } from "@/hooks/useI18n"
import { cn } from "@/lib/utils"
import { useUIStore } from "@/stores/uiStore"
import { useProjectStore } from "@/stores/projectStore"
import { useEnvironmentStore } from "@/stores/environmentStore"

interface EditableVariable {
  id: string
  key: string
  value: string
  enabled: boolean
  isSecret: boolean
}

function ensureTrailingEmptyRow(rows: EditableVariable[]): EditableVariable[] {
  if (rows.length === 0) {
    return [{ id: crypto.randomUUID(), key: "", value: "", enabled: true, isSecret: false }]
  }

  const last = rows[rows.length - 1]
  if (last.key || last.value) {
    return [...rows, { id: crypto.randomUUID(), key: "", value: "", enabled: true, isSecret: false }]
  }

  return rows
}

function normalizeVariables(rows: EditableVariable[]) {
  return rows
    .map((row) => ({
      id: row.id,
      key: row.key.trim(),
      value: row.value,
      enabled: row.enabled,
      isSecret: row.isSecret,
    }))
    .filter((row) => row.key.length > 0)
}

function createSnapshot(name: string, rows: EditableVariable[], fallbackName = ""): string {
  return JSON.stringify({
    name: name.trim() || fallbackName,
    variables: normalizeVariables(rows).map((row) => ({
      key: row.key,
      value: row.value,
      enabled: row.enabled,
      isSecret: row.isSecret,
    })),
  })
}

export function EnvironmentEditorPage() {
  const { t } = useI18n()
  const editingEnvironmentId = useUIStore((s) => s.editingEnvironmentId)
  const closeActiveEnvironmentTab = useUIStore((s) => s.closeActiveEnvironmentTab)
  const { currentProjectId } = useProjectStore()
  const { environments, saveEnvironment, loadEnvironments } = useEnvironmentStore()

  const [envName, setEnvName] = useState("")
  const [variables, setVariables] = useState<EditableVariable[]>([])
  const [saving, setSaving] = useState(false)
  const [initialSnapshot, setInitialSnapshot] = useState("")

  const env = environments.find((e) => e.id === editingEnvironmentId)

  useEffect(() => {
    if (!editingEnvironmentId || !currentProjectId || env) return
    void loadEnvironments(currentProjectId)
  }, [editingEnvironmentId, currentProjectId, env, loadEnvironments])

  useEffect(() => {
    if (!env) return
    const mapped: EditableVariable[] = (env.variables ?? []).map((v) => ({
      id: v.id || crypto.randomUUID(),
      key: v.key,
      value: v.value,
      enabled: v.enabled,
      isSecret: v.isSecret,
    }))
    setEnvName(env.name)
    setVariables(ensureTrailingEmptyRow(mapped))
    setInitialSnapshot(createSnapshot(env.name, mapped, env.name))
  }, [env?.id])

  const currentSnapshot = useMemo(
    () => createSnapshot(envName, variables, env?.name ?? ""),
    [env?.name, envName, variables]
  )
  const isDirty = !!env && currentSnapshot !== initialSnapshot

  const handleSave = useCallback(async () => {
    if (!env || !currentProjectId || saving || !isDirty) return
    setSaving(true)
    try {
      const payload = {
        id: env.id,
        name: envName.trim() || env.name,
        projectId: currentProjectId,
        variables: normalizeVariables(variables),
      }
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      await saveEnvironment(payload as any)
      setInitialSnapshot(createSnapshot(payload.name, payload.variables, payload.name))
    } finally {
      setSaving(false)
    }
  }, [env, envName, variables, currentProjectId, saveEnvironment, saving, isDirty])

  useEffect(() => {
    const listener = () => { void handleSave() }
    window.addEventListener("minipost:save", listener)
    return () => window.removeEventListener("minipost:save", listener)
  }, [handleSave])

  const updateVariable = (id: string, field: keyof EditableVariable, value: string | boolean) => {
    setVariables((prev) => ensureTrailingEmptyRow(prev.map((v) => (v.id === id ? { ...v, [field]: value } : v))))
  }

  const removeVariable = (id: string) => {
    setVariables((prev) => ensureTrailingEmptyRow(prev.filter((v) => v.id !== id)))
  }

  if (!editingEnvironmentId) return null

  if (!env) {
    return (
      <div className="flex h-full flex-col bg-[var(--surface)]">
        <div className="flex items-center justify-between h-[44px] px-4 flex-shrink-0">
          <h2 className="text-[13px] font-semibold text-[var(--fg)]">{t("环境变量", "Environment variables")}</h2>
          <button
            className="h-6 w-6 flex items-center justify-center rounded-[6px] hover:bg-[var(--sidebar-hover)] text-[var(--fg-muted)] transition-colors"
            onClick={closeActiveEnvironmentTab}
            title={t("关闭", "Close")}
          >
            <AppIcon name="clear" size={14} />
          </button>
        </div>
        <div className="flex-1 flex items-center justify-center text-[12px] text-[var(--fg-muted)]">{t("正在加载环境...", "Loading environment...")}</div>
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col bg-[var(--surface)]">
      <div className="flex items-center justify-between h-[44px] px-4 flex-shrink-0">
        <div className="flex items-center gap-2 min-w-0">
          <AppIcon name="globe" size={14} className="text-[var(--accent)]" />
          <input
            value={envName}
            onChange={(e) => setEnvName(e.target.value)}
            className={cn(
              "h-7 min-w-[220px] max-w-[420px] rounded-[6px] px-2",
              "border border-[var(--border-color)] bg-[var(--surface-secondary)]",
              "text-[13px] font-medium text-[var(--fg)]",
              "focus:outline-none focus:border-[var(--accent)]"
            )}
            placeholder="Environment name"
          />
        </div>

        <div className="flex items-center gap-2">
          <button
            className={cn(
              "h-[var(--size-btn)] px-3 flex items-center justify-center gap-1 rounded-[var(--radius-btn)]",
              "text-[length:var(--size-font-2xs)] font-medium transition-colors",
              "hover:bg-[var(--sidebar-hover)]",
              isDirty ? "text-[var(--fg-secondary)]" : "text-[var(--fg-muted)]",
              "disabled:opacity-60 disabled:pointer-events-none"
            )}
            disabled={!isDirty || saving}
            onClick={() => void handleSave()}
            title={`${t("保存", "Save")} (⌘S)`}
          >
            <AppIcon name="save" size={12} strokeWidth={1.9} />
            {saving ? t("保存中...", "Saving...") : t("保存", "Save")}
          </button>
          <button
            className="h-6 w-6 flex items-center justify-center rounded-[6px] hover:bg-[var(--sidebar-hover)] text-[var(--fg-muted)] transition-colors"
            onClick={closeActiveEnvironmentTab}
            title={t("关闭", "Close")}
          >
            <AppIcon name="clear" size={14} />
          </button>
        </div>
      </div>

      <div className="flex-1 min-h-0 overflow-auto p-4 pt-5">
        <div className="rounded-none border border-[var(--border-color)] overflow-hidden bg-[var(--surface)]">
          <div className="grid grid-cols-[minmax(220px,1fr)_minmax(220px,1fr)_44px] h-[34px] bg-[var(--surface-secondary)] border-b border-[var(--border-color)]">
            <div className="px-3 flex items-center text-[12px] font-semibold text-[var(--fg-secondary)] border-r border-[var(--border-color)]">Variable</div>
            <div className="px-3 flex items-center text-[12px] font-semibold text-[var(--fg-secondary)] border-r border-[var(--border-color)]">Value</div>
            <div className="flex items-center justify-center text-[var(--fg-muted)]">
              <AppIcon name="more" size={13} />
            </div>
          </div>

          {variables.map((v, idx) => {
            const isLast = idx === variables.length - 1 && !v.key && !v.value
            return (
              <div
                key={v.id}
                className={cn(
                  "group grid grid-cols-[minmax(220px,1fr)_minmax(220px,1fr)_44px] min-h-[32px] border-b border-[var(--border-color)] last:border-b-0",
                  !isLast && "hover:bg-[var(--surface-secondary)]/45"
                )}
              >
                <div className="border-r border-[var(--border-color)]">
                  <input
                    value={v.key}
                    onChange={(e) => updateVariable(v.id, "key", e.target.value)}
                    placeholder={isLast ? "Add variable" : "Variable"}
                    className={cn(
                      "w-full h-[32px] px-3 bg-transparent",
                      "text-[12px] text-[var(--fg)] placeholder:text-[var(--fg-muted)]",
                      "focus:outline-none"
                    )}
                  />
                </div>

                <div className="border-r border-[var(--border-color)]">
                  <input
                    value={v.value}
                    onChange={(e) => updateVariable(v.id, "value", e.target.value)}
                    placeholder={isLast ? "" : "Value"}
                    className={cn(
                      "w-full h-[32px] px-3 bg-transparent",
                      "text-[12px] text-[var(--fg)] placeholder:text-[var(--fg-muted)]",
                      "focus:outline-none",
                      isLast && "opacity-50"
                    )}
                    disabled={isLast}
                  />
                </div>

                <div className="flex items-center justify-center">
                  {!isLast && (
                    <button
                      className="h-6 w-6 flex items-center justify-center rounded-[6px] text-[var(--fg-muted)] hover:text-[var(--danger)] hover:bg-[var(--sidebar-hover)] transition-colors opacity-0 group-hover:opacity-100"
                      onClick={() => removeVariable(v.id)}
                      title={t("删除变量", "Delete variable")}
                    >
                      <AppIcon name="delete" size={12} />
                    </button>
                  )}
                </div>
              </div>
            )
          })}
        </div>
      </div>
    </div>
  )
}
