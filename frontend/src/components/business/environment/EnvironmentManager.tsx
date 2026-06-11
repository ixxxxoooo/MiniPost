import { useState } from "react"
import { createPortal } from "react-dom"
import { AppIcon } from "@/components/ui/icon"
import { useI18n } from "@/hooks/useI18n"
import { getEnvironmentDeleteConfirmMessage, getEnvironmentDeleteConfirmTitle } from "@/lib/environmentDeleteConfirm"
import * as logger from "@/lib/logger"
import { cn } from "@/lib/utils"
import { useProjectStore } from "@/stores/projectStore"
import { useEnvironmentStore } from "@/stores/environmentStore"
import { useUIStore } from "@/stores/uiStore"

interface DeleteConfirmState {
  id: string
  name: string
}

export function EnvironmentManager() {
  const { t } = useI18n()
  const { currentProjectId } = useProjectStore()
  const {
    environments, activeEnvironmentId,
    createEnvironment, deleteEnvironment, setActiveEnvironment,
  } = useEnvironmentStore()
  const { editingEnvironmentId, setEditingEnvironmentId, closeEnvironmentTab } = useUIStore()

  const [newEnvName, setNewEnvName] = useState("")
  const [showNewInput, setShowNewInput] = useState(false)
  const [deleteConfirm, setDeleteConfirm] = useState<DeleteConfirmState | null>(null)
  const [deleteConfirmLoading, setDeleteConfirmLoading] = useState(false)

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

  const requestDeleteEnv = (env: DeleteConfirmState) => {
    logger.info("EnvironmentManager", "请求删除环境确认", { envID: env.id, projectID: currentProjectId })
    setDeleteConfirm(env)
  }

  const cancelDeleteEnv = () => {
    if (deleteConfirmLoading) return
    if (deleteConfirm) {
      logger.debug("EnvironmentManager", "取消删除环境", { envID: deleteConfirm.id, projectID: currentProjectId })
    }
    setDeleteConfirm(null)
  }

  const handleConfirmDeleteEnv = async () => {
    if (!currentProjectId || !deleteConfirm || deleteConfirmLoading) return
    setDeleteConfirmLoading(true)
    try {
      await deleteEnvironment(currentProjectId, deleteConfirm.id)
      closeEnvironmentTab(deleteConfirm.id)
      logger.info("EnvironmentManager", "环境删除确认完成", { envID: deleteConfirm.id, projectID: currentProjectId })
      setDeleteConfirm(null)
    } catch (error) {
      logger.error("EnvironmentManager", "环境删除失败", {
        envID: deleteConfirm.id,
        projectID: currentProjectId,
        error: error instanceof Error ? error.message : String(error),
      })
    } finally {
      setDeleteConfirmLoading(false)
    }
  }

  if (!currentProjectId) return null

  return (
    <div className="flex flex-col h-full">
      <div className="flex-shrink-0">
        <div className="flex items-center justify-between px-2 py-1">
          <span className="text-2xs text-[var(--fg-muted)] font-medium uppercase">{t("环境变量", "Environment variables")}</span>
          <button
            className="h-5 w-5 flex items-center justify-center rounded-[var(--radius-sm)] hover:bg-[var(--sidebar-hover)] transition-colors"
            onClick={() => setShowNewInput(true)}
            title={t("新建环境", "New environment")}
          >
            <AppIcon name="add" size={12} className="text-[var(--fg-muted)]" />
          </button>
        </div>

        {showNewInput && (
          <div className="px-2 pb-1">
            <input
              className="w-full h-[var(--size-btn-sm)] px-2 text-[length:var(--size-font-2xs)] rounded-[var(--radius-input)] border border-[var(--border-color)] bg-[var(--surface)] text-[var(--fg)] outline-none focus:border-[var(--accent)]"
              placeholder={t("环境名称...", "Environment name...")}
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
              title={activeEnvironmentId === env.id ? t("取消激活", "Deactivate") : t("激活此环境", "Activate this environment")}
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
              onClick={(e) => { e.stopPropagation(); requestDeleteEnv({ id: env.id, name: env.name }) }}
              title={t("删除环境", "Delete environment")}
            >
              <AppIcon name="delete" size={10} />
            </button>
          </div>
        ))}

        {environments.length === 0 && (
          <div className="text-center py-6 text-2xs text-[var(--fg-muted)]">
            {t("暂无环境，点击 + 创建", "No environments yet. Click + to create one.")}
          </div>
        )}
      </div>

      {deleteConfirm && createPortal(
        <div
          className="fixed inset-0 z-[330] flex items-center justify-center"
          onClick={cancelDeleteEnv}
        >
          <div className="absolute inset-0 bg-black/45 backdrop-blur-[1.5px]" />
          <div
            className="relative z-[331] w-[400px] rounded-[12px] border border-[var(--border-color)] bg-[var(--surface)] shadow-[var(--shadow-lg)]"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="px-4 py-3 border-b border-[var(--border-subtle)]">
              <div className="text-[16px] font-semibold text-[var(--fg)]">{getEnvironmentDeleteConfirmTitle(t)}</div>
            </div>
            <div className="px-4 py-4 text-[13px] text-[var(--fg-secondary)] leading-[1.6]">
              {getEnvironmentDeleteConfirmMessage(t, deleteConfirm.name)}
            </div>
            <div className="px-4 py-3 border-t border-[var(--border-subtle)] flex items-center justify-end gap-2">
              <button
                type="button"
                className="h-[32px] px-3 rounded-[8px] border border-[var(--button-border)] text-[12px] text-[var(--fg)] hover:bg-[var(--button-bg)] disabled:opacity-60"
                onClick={cancelDeleteEnv}
                disabled={deleteConfirmLoading}
              >
                {t("取消", "Cancel")}
              </button>
              <button
                type="button"
                className="h-[32px] px-3 rounded-[8px] bg-[var(--danger)] text-white text-[12px] font-medium hover:opacity-95 disabled:opacity-60"
                onClick={() => void handleConfirmDeleteEnv()}
                disabled={deleteConfirmLoading}
              >
                {deleteConfirmLoading ? t("删除中...", "Deleting...") : t("确认删除", "Delete")}
              </button>
            </div>
          </div>
        </div>,
        document.body
      )}
    </div>
  )
}
