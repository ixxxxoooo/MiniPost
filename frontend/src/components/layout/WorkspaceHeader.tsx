import { useEffect, useMemo, useRef, useState } from "react"
import { createPortal } from "react-dom"
import { cn } from "@/lib/utils"
import { AppIcon } from "@/components/ui/icon"
import { useProjectStore } from "@/stores/projectStore"
import { useTabStore, getProjectTabsFromState, getProjectActiveTabIdFromState } from "@/stores/tabStore"
import { useUIStore } from "@/stores/uiStore"

const BUTTON_MIN_WIDTH = 92
const BUTTON_MAX_WIDTH = 180
const PANEL_MIN_WIDTH = 220
const PANEL_MAX_WIDTH = 320
const BUTTON_BASE_WIDTH = 54
const PANEL_BASE_WIDTH = 84
const CHAR_WIDTH = 11

type NavigationTarget =
  | { type: "request"; id: string }
  | { type: "environment"; id: string }

interface NavigationState {
  history: NavigationTarget[]
  index: number
}

function isSameTarget(a: NavigationTarget | undefined, b: NavigationTarget | undefined): boolean {
  if (!a || !b) return false
  return a.type === b.type && a.id === b.id
}

export function WorkspaceHeader() {
  const { currentProjectId, projects, selectProject, createProject, deleteProject, exportProjectJSON } = useProjectStore()
  const projectTabs = useTabStore(getProjectTabsFromState)
  const activeTabId = useTabStore(getProjectActiveTabIdFromState)
  const setActiveTab = useTabStore((s) => s.setActiveTab)
  const { editingEnvironmentId, setEditingEnvironmentId } = useUIStore()
  const [open, setOpen] = useState(false)
  const [searchQuery, setSearchQuery] = useState("")
  const [deleteConfirm, setDeleteConfirm] = useState<{ id: string; name: string } | null>(null)
  const [switchConfirm, setSwitchConfirm] = useState<{ id: string; name: string } | null>(null)
  const [navigation, setNavigation] = useState<NavigationState>({ history: [], index: -1 })
  const ref = useRef<HTMLDivElement>(null)
  const navigatingRef = useRef(false)

  const current = projects.find((project) => project.id === currentProjectId)
  const currentLabel = current?.name || "选择项目"
  const longestProjectNameLength = useMemo(() => {
    return projects.reduce((max, project) => Math.max(max, project.name.length), currentLabel.length)
  }, [projects, currentLabel])

  const triggerWidth = useMemo(() => {
    return Math.min(BUTTON_MAX_WIDTH, Math.max(BUTTON_MIN_WIDTH, BUTTON_BASE_WIDTH + currentLabel.length * CHAR_WIDTH))
  }, [currentLabel])

  const panelWidth = useMemo(() => {
    return Math.min(PANEL_MAX_WIDTH, Math.max(PANEL_MIN_WIDTH, PANEL_BASE_WIDTH + longestProjectNameLength * CHAR_WIDTH))
  }, [longestProjectNameLength])
  const requestTabIdSet = useMemo(() => new Set(projectTabs.map((tab) => tab.id)), [projectTabs])
  const currentTarget = useMemo<NavigationTarget | null>(() => {
    if (editingEnvironmentId) return { type: "environment", id: editingEnvironmentId }
    if (activeTabId) return { type: "request", id: activeTabId }
    return null
  }, [activeTabId, editingEnvironmentId])

  const hasNavigableAt = useMemo(() => {
    return (idx: number) => {
      const target = navigation.history[idx]
      if (!target) return false
      if (target.type === "environment") return true
      return requestTabIdSet.has(target.id)
    }
  }, [navigation.history, requestTabIdSet])

  const canGoBack = useMemo(() => {
    for (let i = navigation.index - 1; i >= 0; i -= 1) {
      if (hasNavigableAt(i)) return true
    }
    return false
  }, [hasNavigableAt, navigation.index])

  const canGoForward = useMemo(() => {
    for (let i = navigation.index + 1; i < navigation.history.length; i += 1) {
      if (hasNavigableAt(i)) return true
    }
    return false
  }, [hasNavigableAt, navigation.history.length, navigation.index])

  useEffect(() => {
    if (!currentProjectId) {
      setNavigation({ history: [], index: -1 })
      navigatingRef.current = false
      return
    }
    setNavigation({ history: [], index: -1 })
    navigatingRef.current = false
  }, [currentProjectId])

  useEffect(() => {
    if (!currentTarget || !currentProjectId) return
    setNavigation((prev) => {
      if (navigatingRef.current) {
        navigatingRef.current = false
        return prev
      }
      const currentInHistory = prev.history[prev.index]
      if (isSameTarget(currentInHistory, currentTarget)) return prev
      const history = [...prev.history.slice(0, prev.index + 1), currentTarget]
      return { history, index: history.length - 1 }
    })
  }, [currentProjectId, currentTarget])

  const jumpToTarget = (target: NavigationTarget) => {
    navigatingRef.current = true
    if (target.type === "environment") {
      setEditingEnvironmentId(target.id)
      return
    }
    setEditingEnvironmentId(null)
    setActiveTab(target.id)
  }

  const handleGoBack = () => {
    setNavigation((prev) => {
      if (prev.index <= 0) return prev
      for (let i = prev.index - 1; i >= 0; i -= 1) {
        const target = prev.history[i]
        if (!target) continue
        if (target.type === "request" && !requestTabIdSet.has(target.id)) continue
        jumpToTarget(target)
        return { ...prev, index: i }
      }
      return prev
    })
  }

  const handleGoForward = () => {
    setNavigation((prev) => {
      if (prev.index >= prev.history.length - 1) return prev
      for (let i = prev.index + 1; i < prev.history.length; i += 1) {
        const target = prev.history[i]
        if (!target) continue
        if (target.type === "request" && !requestTabIdSet.has(target.id)) continue
        jumpToTarget(target)
        return { ...prev, index: i }
      }
      return prev
    })
  }

  useEffect(() => {
    if (!open) return

    const handleOutside = (event: MouseEvent) => {
      if (ref.current && !ref.current.contains(event.target as Node)) {
        setOpen(false)
      }
    }

    document.addEventListener("mousedown", handleOutside)
    return () => document.removeEventListener("mousedown", handleOutside)
  }, [open])

  useEffect(() => {
    if (!open && !deleteConfirm && !switchConfirm) return
    const handler = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return
      event.preventDefault()
      if (deleteConfirm) {
        setDeleteConfirm(null)
        return
      }
      if (switchConfirm) {
        setSwitchConfirm(null)
        return
      }
      setOpen(false)
    }
    window.addEventListener("keydown", handler)
    return () => window.removeEventListener("keydown", handler)
  }, [deleteConfirm, open, switchConfirm])

  const handleCreate = async () => {
    const name = searchQuery.trim()
    if (!name) return

    await createProject(name)
    const { projects: updatedProjects } = useProjectStore.getState()
    const project = updatedProjects[updatedProjects.length - 1]
    if (project) {
      await selectProject(project.id)
    }

    setSearchQuery("")
    setOpen(false)
  }

  const handleSelectProject = async (projectId: string) => {
    if (projectId === currentProjectId) {
      setOpen(false)
      return
    }

    const hasDirtyTabs = projectTabs.some((tab) => tab.dirty)
    if (hasDirtyTabs) {
      const project = projects.find((p) => p.id === projectId)
      setSwitchConfirm({ id: projectId, name: project?.name || projectId })
      return
    }

    await selectProject(projectId)
    setOpen(false)
  }

  const handleConfirmSwitchProject = async () => {
    if (!switchConfirm) return
    await selectProject(switchConfirm.id)
    setSwitchConfirm(null)
    setOpen(false)
  }

  const handleExportProject = async (projectId: string) => {
    const prevProjectId = currentProjectId
    if (projectId !== currentProjectId) {
      await selectProject(projectId)
    }
    const json = await exportProjectJSON()
    if (json) {
      const project = projects.find((p) => p.id === projectId)
      const { SaveFileDialogJSON } = await import("../../../wailsjs/go/main/App")
      await SaveFileDialogJSON(`${project?.name || projectId}-export.json`, json)
    }
    if (prevProjectId && prevProjectId !== projectId) {
      await selectProject(prevProjectId)
    }
  }

  const handleDeleteProject = async (id: string) => {
    await deleteProject(id)
    setDeleteConfirm(null)
    setOpen(false)
  }

  return (
    <>
      <div className="titlebar-no-drag flex items-center gap-1" ref={ref} onMouseDown={(event) => event.stopPropagation()}>
        <button
          className={cn(
            "flex h-[24px] w-[24px] items-center justify-center rounded-[7px] border transition-colors",
            "border-transparent bg-[var(--surface-secondary)] text-[var(--fg-secondary)] hover:bg-[var(--button-bg)]"
          )}
          title="主页"
          type="button"
        >
          <AppIcon name="home" size={15} strokeWidth={1.65} />
        </button>

        <button
          className={cn(
            "flex h-[24px] w-[24px] items-center justify-center rounded-[7px] border transition-colors",
            "border-transparent bg-[var(--surface-secondary)] text-[var(--fg-secondary)]",
            canGoBack ? "hover:bg-[var(--button-bg)]" : "opacity-40 cursor-not-allowed"
          )}
          title="后退"
          type="button"
          onClick={handleGoBack}
          disabled={!canGoBack}
        >
          <AppIcon name="arrowLeft" size={14} strokeWidth={1.9} />
        </button>

        <button
          className={cn(
            "flex h-[24px] w-[24px] items-center justify-center rounded-[7px] border transition-colors",
            "border-transparent bg-[var(--surface-secondary)] text-[var(--fg-secondary)]",
            canGoForward ? "hover:bg-[var(--button-bg)]" : "opacity-40 cursor-not-allowed"
          )}
          title="前进"
          type="button"
          onClick={handleGoForward}
          disabled={!canGoForward}
        >
          <AppIcon name="arrowRight" size={14} strokeWidth={1.9} />
        </button>

        <div className="relative" style={{ width: `${triggerWidth}px`, maxWidth: `${BUTTON_MAX_WIDTH}px` }}>
          <button
            className={cn(
              "flex h-[28px] w-full items-center gap-1.5 rounded-[8px] px-2 text-left transition-colors",
              "bg-[var(--surface-secondary)] text-[var(--fg)] hover:bg-[var(--button-bg)]"
            )}
            onClick={() => setOpen((prev) => !prev)}
            type="button"
          >
            <AppIcon name="folderShared" size={15} strokeWidth={1.65} className="text-[var(--fg-secondary)]" />
            <span className="min-w-0 flex-1 truncate text-[11px] font-medium text-[var(--fg)]">
              {current?.name || "选择项目"}
            </span>
            <AppIcon
              name="arrowDown"
              size={11}
              strokeWidth={1.9}
              className={cn("text-[var(--fg-muted)] transition-transform", open && "rotate-180")}
            />
          </button>

          {open && (
            <div
              className={cn(
                "absolute left-0 top-[calc(100%+5px)] z-[120] overflow-hidden rounded-[9px] border shadow-[var(--shadow-lg)]",
                "border-[var(--button-border)] bg-[var(--surface-elevated)]"
              )}
              style={{ width: `${Math.max(triggerWidth, panelWidth)}px`, maxWidth: `${PANEL_MAX_WIDTH}px` }}
            >
              <div className="flex items-center gap-1.5 border-b border-[var(--border-subtle)] p-1.5">
                <div className="relative flex-1">
                  <AppIcon
                    name="search"
                    size={10}
                    strokeWidth={1.9}
                    className="absolute left-2.5 top-1/2 -translate-y-1/2 text-[var(--fg-muted)]"
                  />
                  <input
                    className={cn(
                      "h-7 w-full rounded-[7px] border border-[var(--button-border)] bg-[var(--surface)] pl-7 pr-2.5",
                      "text-[10px] text-[var(--fg)] outline-none placeholder:text-[var(--fg-muted)] focus:border-[var(--accent)]"
                    )}
                    placeholder="搜索或新建项目..."
                    value={searchQuery}
                    onChange={(event) => setSearchQuery(event.target.value)}
                    onKeyDown={(e) => { if (e.key === "Enter") void handleCreate() }}
                  />
                </div>
                <button
                  className={cn(
                    "h-7 rounded-[7px] border px-2.5 text-[10px] font-medium transition-colors",
                    "border-[var(--button-border)] bg-[var(--surface)] text-[var(--fg)] hover:bg-[var(--surface-secondary)]"
                  )}
                  onClick={() => {
                    void handleCreate()
                  }}
                  type="button"
                >
                  新建
                </button>
              </div>

              <div className="max-h-[240px] overflow-y-auto p-1">
                {projects
                  .filter((project) => project.name.toLowerCase().includes(searchQuery.toLowerCase()))
                  .map((project) => {
                  const selected = project.id === currentProjectId
                  return (
                    <div
                      key={project.id}
                      className={cn(
                        "flex w-full items-center gap-2 rounded-[7px] px-2 py-1.5 text-left transition-colors group",
                        selected
                          ? "bg-[var(--selected-bg)] text-[var(--fg)]"
                          : "text-[var(--fg)] hover:bg-[var(--surface-secondary)]"
                      )}
                    >
                      <button
                        className="flex flex-1 items-center gap-2 min-w-0"
                        onClick={() => {
                          void handleSelectProject(project.id)
                        }}
                        type="button"
                      >
                        <div className="flex h-5 w-5 items-center justify-center rounded-[6px] bg-[var(--surface-secondary)] text-[var(--fg-secondary)] flex-shrink-0">
                          <AppIcon name="lock" size={11} strokeWidth={1.9} />
                        </div>
                        <div className="min-w-0 flex-1 text-left">
                          <div className="truncate text-[12px] font-medium">{project.name}</div>
                        </div>
                      </button>
                      {selected && (
                        <span className="text-[9px] font-medium text-[var(--fg-muted)] flex-shrink-0">当前</span>
                      )}
                      <div className="opacity-0 group-hover:opacity-100 flex items-center gap-0.5 flex-shrink-0 transition-opacity">
                        <button
                          className="h-5 w-5 flex items-center justify-center rounded-[4px] hover:bg-[var(--button-bg)] transition-colors"
                          onClick={(e) => { e.stopPropagation(); void handleExportProject(project.id) }}
                          title="导出项目"
                          type="button"
                        >
                          <AppIcon name="download" size={11} className="text-[var(--fg-muted)]" />
                        </button>
                        <button
                          className="h-5 w-5 flex items-center justify-center rounded-[4px] hover:bg-[var(--button-bg)] transition-colors"
                          onClick={(e) => { e.stopPropagation(); setDeleteConfirm({ id: project.id, name: project.name }) }}
                          title="删除项目"
                          type="button"
                        >
                          <AppIcon name="delete" size={11} className="text-[var(--fg-muted)]" />
                        </button>
                      </div>
                    </div>
                  )
                })}
              </div>
            </div>
          )}
        </div>
      </div>

      {/* 删除项目确认弹窗 */}
      {deleteConfirm && createPortal(
        <div className="fixed inset-0 z-[300] flex items-center justify-center" onClick={() => setDeleteConfirm(null)}>
          <div className="absolute inset-0 bg-black/40" />
          <div
            className={cn(
              "relative z-[301] w-[360px] rounded-[var(--radius-panel)] border shadow-[var(--shadow-lg)] p-5",
              "bg-[var(--surface)] border-[var(--border-color)]"
            )}
            onClick={(e) => e.stopPropagation()}
          >
            <h3 className="text-[length:var(--size-font-sm)] font-semibold text-[var(--fg)] mb-2">确认删除项目</h3>
            <p className="text-[length:var(--size-font-2xs)] text-[var(--fg-secondary)] mb-4">
              确定要删除项目 <span className="font-semibold text-[var(--fg)]">「{deleteConfirm.name}」</span> 吗？该操作不可撤销，项目下的所有请求和文件夹都将被永久删除。
            </p>
            <div className="flex justify-end gap-2">
              <button
                className={cn(
                  "h-8 px-4 rounded-[var(--radius-btn)] text-[length:var(--size-font-2xs)] font-medium transition-colors",
                  "border border-[var(--border-color)] bg-[var(--surface)] text-[var(--fg)] hover:bg-[var(--surface-secondary)]"
                )}
                onClick={() => setDeleteConfirm(null)}
              >
                取消
              </button>
              <button
                className={cn(
                  "h-8 px-4 rounded-[var(--radius-btn)] text-[length:var(--size-font-2xs)] font-medium transition-colors",
                  "bg-[var(--danger)] text-white hover:opacity-90"
                )}
                onClick={() => void handleDeleteProject(deleteConfirm.id)}
              >
                确认删除
              </button>
            </div>
          </div>
        </div>,
        document.body
      )}

      {switchConfirm && createPortal(
        <div className="fixed inset-0 z-[310] flex items-center justify-center" onClick={() => setSwitchConfirm(null)}>
          <div className="absolute inset-0 bg-black/40" />
          <div
            className={cn(
              "relative z-[311] w-[420px] rounded-[var(--radius-panel)] border shadow-[var(--shadow-lg)] p-5",
              "bg-[var(--surface)] border-[var(--border-color)]"
            )}
            onClick={(e) => e.stopPropagation()}
          >
            <h3 className="text-[length:var(--size-font-sm)] font-semibold text-[var(--fg)] mb-2">切换项目确认</h3>
            <p className="text-[length:var(--size-font-2xs)] text-[var(--fg-secondary)] mb-4">
              当前工作区存在未保存的标签。切换到
              <span className="font-semibold text-[var(--fg)]">「{switchConfirm.name}」</span>
              后，这些修改仍会保留在原项目标签中，但你将离开当前编辑上下文。确定继续吗？
            </p>
            <div className="flex justify-end gap-2">
              <button
                className={cn(
                  "h-8 px-4 rounded-[var(--radius-btn)] text-[length:var(--size-font-2xs)] font-medium transition-colors",
                  "border border-[var(--border-color)] bg-[var(--surface)] text-[var(--fg)] hover:bg-[var(--surface-secondary)]"
                )}
                onClick={() => setSwitchConfirm(null)}
              >
                取消
              </button>
              <button
                className={cn(
                  "h-8 px-4 rounded-[var(--radius-btn)] text-[length:var(--size-font-2xs)] font-medium transition-colors",
                  "bg-[var(--accent)] text-white hover:opacity-90"
                )}
                onClick={() => void handleConfirmSwitchProject()}
              >
                继续切换
              </button>
            </div>
          </div>
        </div>,
        document.body
      )}
    </>
  )
}
