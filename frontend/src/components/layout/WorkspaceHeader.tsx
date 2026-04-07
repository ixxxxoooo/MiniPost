import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { createPortal } from "react-dom"
import { cn } from "@/lib/utils"
import { AppIcon } from "@/components/ui/icon"
import { useI18n } from "@/hooks/useI18n"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
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
  const { t } = useI18n()
  const { currentProjectId, projects, selectProject, createProject, deleteProject, exportProjectJSON } = useProjectStore()
  const projectTabs = useTabStore(getProjectTabsFromState)
  const activeTabId = useTabStore(getProjectActiveTabIdFromState)
  const setActiveTab = useTabStore((s) => s.setActiveTab)
  const { editingEnvironmentId, setEditingEnvironmentId, workspaceView, setWorkspaceView } = useUIStore()
  const [open, setOpen] = useState(false)
  const [searchQuery, setSearchQuery] = useState("")
  const [deleteConfirm, setDeleteConfirm] = useState<{ id: string; name: string } | null>(null)
  const [switchConfirm, setSwitchConfirm] = useState<{ id: string; name: string } | null>(null)
  const [navigation, setNavigation] = useState<NavigationState>({ history: [], index: -1 })
  const ref = useRef<HTMLDivElement>(null)
  const triggerRef = useRef<HTMLDivElement>(null)
  const panelRef = useRef<HTMLDivElement>(null)
  const navigatingRef = useRef(false)
  const [dropdownPosition, setDropdownPosition] = useState<{ top: number; left: number; width: number; maxHeight: number } | null>(null)

  const current = projects.find((project) => project.id === currentProjectId)
  const currentLabel = current?.name || t("选择项目", "Select project")
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

  const updateDropdownPosition = useCallback(() => {
    const trigger = triggerRef.current
    if (!trigger || typeof window === "undefined") return

    const rect = trigger.getBoundingClientRect()
    const margin = 8
    const width = Math.min(Math.max(triggerWidth, panelWidth), window.innerWidth - margin * 2)
    const left = Math.max(margin, Math.min(rect.left, window.innerWidth - width - margin))
    const spaceBelow = window.innerHeight - rect.bottom - margin
    const spaceAbove = rect.top - margin
    const preferAbove = spaceBelow < 190 && spaceAbove > spaceBelow
    const maxHeight = Math.min(320, Math.max(140, (preferAbove ? spaceAbove : spaceBelow) - 6))
    const top = preferAbove
      ? Math.max(margin, rect.top - maxHeight - 6)
      : Math.max(margin, rect.bottom + 6)

    setDropdownPosition({ top, left, width, maxHeight })
  }, [panelWidth, triggerWidth])

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
    setWorkspaceView("project")
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

    updateDropdownPosition()

    const handleOutside = (event: MouseEvent) => {
      const target = event.target as Node
      const clickedTrigger = triggerRef.current?.contains(target)
      const clickedPanel = panelRef.current?.contains(target)
      if (!clickedTrigger && !clickedPanel) {
        setOpen(false)
      }
    }

    const syncPosition = () => updateDropdownPosition()
    document.addEventListener("mousedown", handleOutside)
    window.addEventListener("resize", syncPosition)
    window.addEventListener("scroll", syncPosition, true)
    return () => {
      document.removeEventListener("mousedown", handleOutside)
      window.removeEventListener("resize", syncPosition)
      window.removeEventListener("scroll", syncPosition, true)
    }
  }, [open, updateDropdownPosition])

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

    const project = await createProject(name)
    if (project?.id) {
      await selectProject(project.id)
    }

    setSearchQuery("")
    setOpen(false)
  }

  const handleSelectProject = async (projectId: string) => {
    if (projectId === currentProjectId) {
      setWorkspaceView("project")
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
      const safeName = (project?.name || projectId).replace(/[\\/:*?"<>|]/g, "-").trim() || projectId
      await SaveFileDialogJSON(`${safeName}.postman_collection.json`, json)
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

  const projectDropdownPortal = open && dropdownPosition && createPortal(
    <>
      <div className="fixed inset-0 z-[118]" aria-hidden="true" />
      <div
        ref={panelRef}
        className={cn(
          "fixed z-[120] overflow-hidden rounded-[9px] border shadow-[var(--shadow-lg)]",
          "border-[var(--button-border)] bg-[var(--surface-elevated)]"
        )}
        style={{
          left: `${dropdownPosition.left}px`,
          top: `${dropdownPosition.top}px`,
          width: `${dropdownPosition.width}px`,
        }}
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
              placeholder={t("搜索或新建项目...", "Search or create project...")}
              value={searchQuery}
              onChange={(event) => setSearchQuery(event.target.value)}
              onKeyDown={(e) => { if (e.key === "Enter") void handleCreate() }}
              autoFocus
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
            {t("新建", "New")}
          </button>
        </div>

        <div className="overflow-y-auto p-1" style={{ maxHeight: `${dropdownPosition.maxHeight}px` }}>
          {projects
            .filter((project) => project.name.toLowerCase().includes(searchQuery.toLowerCase()))
            .map((project) => {
              const selected = project.id === currentProjectId
              return (
                <div
                  key={project.id}
                  className={cn(
                    "group flex w-full items-center gap-2 rounded-[7px] px-2 py-1.5 text-left transition-colors",
                    selected
                      ? "bg-[var(--selected-bg)] text-[var(--fg)]"
                      : "text-[var(--fg)] hover:bg-[var(--surface-secondary)]"
                  )}
                >
                  <button
                    className="flex min-w-0 flex-1 items-center gap-2"
                    onClick={() => {
                      void handleSelectProject(project.id)
                    }}
                    type="button"
                  >
                    <span
                      className="h-2.5 w-2.5 flex-shrink-0 rounded-full"
                      style={{ backgroundColor: project.themeColor || "var(--accent)" }}
                    />
                    <div className="min-w-0 flex-1 text-left">
                      <div className="truncate text-[12px] font-medium">{project.name}</div>
                    </div>
                  </button>
                  {selected && (
                    <span className="flex-shrink-0 text-[9px] font-medium text-[var(--fg-muted)]">{t("当前", "Current")}</span>
                  )}
                  <div className="flex flex-shrink-0 items-center gap-0.5 opacity-0 transition-opacity group-hover:opacity-100">
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <button
                          className="flex h-5 w-5 items-center justify-center rounded-[4px] transition-colors hover:bg-[var(--button-bg)]"
                          onClick={(e) => { e.stopPropagation(); void handleExportProject(project.id) }}
                          type="button"
                        >
                          <AppIcon name="download" size={11} className="text-[var(--fg-muted)]" />
                        </button>
                      </TooltipTrigger>
                      <TooltipContent>{t("导出项目", "Export project")}</TooltipContent>
                    </Tooltip>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <button
                          className="flex h-5 w-5 items-center justify-center rounded-[4px] transition-colors hover:bg-[var(--button-bg)]"
                          onClick={(e) => { e.stopPropagation(); setDeleteConfirm({ id: project.id, name: project.name }) }}
                          type="button"
                        >
                          <AppIcon name="delete" size={11} className="text-[var(--fg-muted)]" />
                        </button>
                      </TooltipTrigger>
                      <TooltipContent>{t("删除项目", "Delete project")}</TooltipContent>
                    </Tooltip>
                  </div>
                </div>
              )
            })}
        </div>
      </div>
    </>,
    document.body
  )

  return (
    <>
      <div className="titlebar-no-drag flex items-center gap-1" ref={ref} onMouseDown={(event) => event.stopPropagation()}>
        <Tooltip>
          <TooltipTrigger asChild>
            <button
              className={cn(
                "flex h-[24px] w-[24px] items-center justify-center rounded-[7px] border transition-colors",
                "border-transparent bg-[var(--surface-secondary)] text-[var(--fg-secondary)]",
                canGoBack ? "hover:bg-[var(--button-bg)]" : "opacity-40 cursor-not-allowed"
              )}
              type="button"
              onClick={handleGoBack}
              disabled={!canGoBack}
            >
              <AppIcon name="arrowLeft" size={14} strokeWidth={1.9} />
            </button>
          </TooltipTrigger>
          <TooltipContent>{t("后退", "Back")}</TooltipContent>
        </Tooltip>

        <Tooltip>
          <TooltipTrigger asChild>
            <button
              className={cn(
                "flex h-[24px] w-[24px] items-center justify-center rounded-[7px] border transition-colors",
                "border-transparent bg-[var(--surface-secondary)] text-[var(--fg-secondary)]",
                canGoForward ? "hover:bg-[var(--button-bg)]" : "opacity-40 cursor-not-allowed"
              )}
              type="button"
              onClick={handleGoForward}
              disabled={!canGoForward}
            >
              <AppIcon name="arrowRight" size={14} strokeWidth={1.9} />
            </button>
          </TooltipTrigger>
          <TooltipContent>{t("前进", "Forward")}</TooltipContent>
        </Tooltip>

        <Tooltip>
          <TooltipTrigger asChild>
            <button
              className={cn(
                "flex h-[24px] w-[24px] items-center justify-center rounded-[7px] border transition-colors",
                workspaceView === "home"
                  ? "border-[var(--button-border)] bg-[var(--selected-bg)] text-[var(--accent)]"
                  : "border-transparent bg-[var(--surface-secondary)] text-[var(--fg-secondary)] hover:bg-[var(--button-bg)]"
              )}
              type="button"
              onClick={() => {
                setEditingEnvironmentId(null)
                if (workspaceView === "home") {
                  setWorkspaceView(currentProjectId ? "project" : "home")
                  return
                }
                setWorkspaceView("home")
              }}
            >
              <AppIcon name="home" size={15} strokeWidth={1.65} />
            </button>
          </TooltipTrigger>
          <TooltipContent>{workspaceView === "home" ? t("关闭主页", "Close home") : t("打开主页", "Open home")}</TooltipContent>
        </Tooltip>

        <div ref={triggerRef} className="relative" style={{ width: `${triggerWidth}px`, maxWidth: `${BUTTON_MAX_WIDTH}px` }}>
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
              {current?.name || t("选择项目", "Select project")}
            </span>
            <AppIcon
              name="arrowDown"
              size={11}
              strokeWidth={1.9}
              className={cn("text-[var(--fg-muted)] transition-transform", open && "rotate-180")}
            />
          </button>
        </div>
      </div>
      {projectDropdownPortal}

      {/* delete project confirm modal */}
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
            <h3 className="text-[length:var(--size-font-sm)] font-semibold text-[var(--fg)] mb-2">{t("确认删除项目", "Confirm project deletion")}</h3>
            <p className="text-[length:var(--size-font-2xs)] text-[var(--fg-secondary)] mb-4">
              {t("确定要删除项目", "Are you sure you want to delete project")} <span className="font-semibold text-[var(--fg)]">「{deleteConfirm.name}」</span>{t("吗？该操作不可撤销，项目下的所有请求和文件夹都将被永久删除。", "? This action cannot be undone. All requests and folders under this project will be permanently deleted.")}
            </p>
            <div className="flex justify-end gap-2">
              <button
                className={cn(
                  "h-8 px-4 rounded-[var(--radius-btn)] text-[length:var(--size-font-2xs)] font-medium transition-colors",
                  "border border-[var(--border-color)] bg-[var(--surface)] text-[var(--fg)] hover:bg-[var(--surface-secondary)]"
                )}
                onClick={() => setDeleteConfirm(null)}
              >
                {t("取消", "Cancel")}
              </button>
              <button
                className={cn(
                  "h-8 px-4 rounded-[var(--radius-btn)] text-[length:var(--size-font-2xs)] font-medium transition-colors",
                  "bg-[var(--danger)] text-white hover:opacity-90"
                )}
                onClick={() => void handleDeleteProject(deleteConfirm.id)}
              >
                {t("确认删除", "Delete")}
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
            <h3 className="text-[length:var(--size-font-sm)] font-semibold text-[var(--fg)] mb-2">{t("切换项目确认", "Confirm project switch")}</h3>
            <p className="text-[length:var(--size-font-2xs)] text-[var(--fg-secondary)] mb-4">
              {t("当前工作区存在未保存的标签。切换到", "There are unsaved tabs in current workspace. When switching to")}
              <span className="font-semibold text-[var(--fg)]">「{switchConfirm.name}」</span>
              {t("后，这些修改仍会保留在原项目标签中，但你将离开当前编辑上下文。确定继续吗？", ", those changes remain in original project tabs, but you'll leave the current editing context. Continue?")}
            </p>
            <div className="flex justify-end gap-2">
              <button
                className={cn(
                  "h-8 px-4 rounded-[var(--radius-btn)] text-[length:var(--size-font-2xs)] font-medium transition-colors",
                  "border border-[var(--border-color)] bg-[var(--surface)] text-[var(--fg)] hover:bg-[var(--surface-secondary)]"
                )}
                onClick={() => setSwitchConfirm(null)}
              >
                {t("取消", "Cancel")}
              </button>
              <button
                className={cn(
                  "h-8 px-4 rounded-[var(--radius-btn)] text-[length:var(--size-font-2xs)] font-medium transition-colors",
                  "bg-[var(--accent)] text-white hover:opacity-90"
                )}
                onClick={() => void handleConfirmSwitchProject()}
              >
                {t("继续切换", "Continue")}
              </button>
            </div>
          </div>
        </div>,
        document.body
      )}
    </>
  )
}
