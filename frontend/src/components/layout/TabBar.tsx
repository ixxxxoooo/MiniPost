import { useState, useEffect, useRef, useCallback, useMemo } from "react"
import { createPortal } from "react-dom"
import { AppIcon } from "@/components/ui/icon"
import { cn } from "@/lib/utils"
import { METHOD_COLORS, type HttpMethod } from "@/lib/constants"
import { useTabStore, getProjectActiveTabIdFromState, getProjectTabsFromState } from "@/stores/tabStore"
import { useProjectStore } from "@/stores/projectStore"
import { useEnvironmentStore } from "@/stores/environmentStore"
import { useUIStore } from "@/stores/uiStore"

interface TabContextMenuState {
  x: number
  y: number
  tabId: string
}

const DROPDOWN_CHAR_WIDTH = 7
const DROPDOWN_PANEL_CLASS = "rounded-[10px] border border-[var(--border-color)] bg-[var(--surface-elevated)] shadow-[var(--shadow-lg)]"
const DROPDOWN_ITEM_CLASS = "w-full whitespace-nowrap px-3 py-1.5 rounded-[7px] text-[11px] text-left transition-colors flex items-center gap-2"

export function TabBar() {
  const tabs = useTabStore(getProjectTabsFromState)
  const activeTabId = useTabStore(getProjectActiveTabIdFromState)
  const setActiveTab = useTabStore((s) => s.setActiveTab)
  const removeTab = useTabStore((s) => s.removeTab)
  const updateTab = useTabStore((s) => s.updateTab)
  const closeOtherTabs = useTabStore((s) => s.closeOtherTabs)
  const closeAllTabs = useTabStore((s) => s.closeAllTabs)
  const addNewUnsavedTab = useTabStore((s) => s.addNewUnsavedTab)
  const currentProjectId = useProjectStore((s) => s.currentProjectId)
  const renameRequest = useProjectStore((s) => s.renameRequest)
  const editingEnvironmentId = useUIStore((s) => s.editingEnvironmentId)
  const openEnvironmentTabIds = useUIStore((s) => s.openEnvironmentTabIds)
  const setEditingEnvironmentId = useUIStore((s) => s.setEditingEnvironmentId)
  const closeEnvironmentTab = useUIStore((s) => s.closeEnvironmentTab)

  const { environments, activeEnvironmentId, setActiveEnvironment, loadEnvironments, createEnvironment } = useEnvironmentStore()

  const [contextMenu, setContextMenu] = useState<TabContextMenuState | null>(null)
  const [renamingTabId, setRenamingTabId] = useState<string | null>(null)
  const [renameValue, setRenameValue] = useState("")
  const [hasOverflow, setHasOverflow] = useState(false)
  const [showTabList, setShowTabList] = useState(false)
  const [showEnvDropdown, setShowEnvDropdown] = useState(false)
  const [creatingEnvironment, setCreatingEnvironment] = useState(false)
  const scrollContainerRef = useRef<HTMLDivElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)
  const tabListRef = useRef<HTMLDivElement>(null)
  const envDropdownRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (currentProjectId) loadEnvironments(currentProjectId)
  }, [currentProjectId, loadEnvironments])

  const checkOverflow = useCallback(() => {
    const container = scrollContainerRef.current
    if (container) {
      setHasOverflow(container.scrollWidth > container.clientWidth)
      return
    }
    setHasOverflow(false)
  }, [])

  useEffect(() => {
    checkOverflow()
  }, [checkOverflow, currentProjectId, tabs, editingEnvironmentId, openEnvironmentTabIds])

  useEffect(() => {
    const container = scrollContainerRef.current
    if (!container) {
      setHasOverflow(false)
      return
    }

    checkOverflow()
    window.addEventListener("resize", checkOverflow)

    const observer = new ResizeObserver(() => checkOverflow())
    observer.observe(container)

    return () => {
      window.removeEventListener("resize", checkOverflow)
      observer.disconnect()
    }
  }, [checkOverflow, currentProjectId, tabs, editingEnvironmentId, openEnvironmentTabIds])

  useEffect(() => {
    const targetTabDomId = editingEnvironmentId ? `env-${editingEnvironmentId}` : activeTabId
    if (!targetTabDomId || !scrollContainerRef.current) return
    const activeEl = scrollContainerRef.current.querySelector(`[data-tab-id="${targetTabDomId}"]`)
    if (activeEl) activeEl.scrollIntoView({ behavior: "auto", block: "nearest", inline: "nearest" })
  }, [activeTabId, editingEnvironmentId])

  useEffect(() => {
    if (!contextMenu) return
    const handler = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) setContextMenu(null)
    }
    document.addEventListener("mousedown", handler)
    return () => document.removeEventListener("mousedown", handler)
  }, [contextMenu])

  useEffect(() => {
    if (!showTabList) return
    const handler = (e: MouseEvent) => {
      if (tabListRef.current && !tabListRef.current.contains(e.target as Node)) setShowTabList(false)
    }
    document.addEventListener("mousedown", handler)
    return () => document.removeEventListener("mousedown", handler)
  }, [showTabList])

  useEffect(() => {
    if (!showEnvDropdown) return
    const handler = (e: MouseEvent) => {
      if (envDropdownRef.current && !envDropdownRef.current.contains(e.target as Node)) setShowEnvDropdown(false)
    }
    document.addEventListener("mousedown", handler)
    return () => document.removeEventListener("mousedown", handler)
  }, [showEnvDropdown])

  const startRename = (tabId: string, currentTitle: string) => {
    setRenamingTabId(tabId)
    setRenameValue(currentTitle)
    setContextMenu(null)
  }

  const handleRenameSubmit = async () => {
    if (!renamingTabId || !renameValue.trim()) {
      setRenamingTabId(null)
      return
    }
    const tab = tabs.find((t) => t.id === renamingTabId)
    if (tab) {
      updateTab(renamingTabId, { title: renameValue.trim() })
      if (tab.requestId) {
        await renameRequest(tab.requestId, renameValue.trim())
      }
    }
    setRenamingTabId(null)
  }

  const handleAddTab = () => {
    if (!currentProjectId) return
    setEditingEnvironmentId(null)
    addNewUnsavedTab(currentProjectId)
  }

  const buildNewEnvironmentName = useCallback(() => {
    const existing = new Set(environments.map((env) => env.name))
    const base = "New Environment"
    if (!existing.has(base)) return base
    let index = 2
    while (existing.has(`${base} ${index}`)) {
      index += 1
    }
    return `${base} ${index}`
  }, [environments])

  const handleQuickCreateEnvironment = useCallback(async () => {
    if (!currentProjectId || creatingEnvironment) return
    setCreatingEnvironment(true)
    try {
      const created = await createEnvironment(currentProjectId, buildNewEnvironmentName())
      if (created?.id) {
        setEditingEnvironmentId(created.id)
        setShowEnvDropdown(false)
      }
    } finally {
      setCreatingEnvironment(false)
    }
  }, [buildNewEnvironmentName, createEnvironment, creatingEnvironment, currentProjectId, setEditingEnvironmentId])

  const handleEditEnvironment = useCallback((envId: string) => {
    setEditingEnvironmentId(envId)
    setShowEnvDropdown(false)
  }, [setEditingEnvironmentId])

  const activeEnvName = useMemo(() => {
    if (!activeEnvironmentId) return "No Environment"
    const env = environments.find((e) => e.id === activeEnvironmentId)
    return env?.name || "No Environment"
  }, [activeEnvironmentId, environments])

  const openEnvironmentTabs = useMemo(
    () => openEnvironmentTabIds.map((id) => ({
      id,
      name: environments.find((env) => env.id === id)?.name || "Environment",
    })),
    [environments, openEnvironmentTabIds]
  )

  const hasAnyTab = tabs.length > 0 || openEnvironmentTabs.length > 0
  const envDropdownWidth = useMemo(() => {
    const labels = [
      "No Environment",
      creatingEnvironment ? "创建中..." : "新建环境",
      ...environments.map((env) => env.name),
    ]
    const longest = labels.reduce((max, label) => Math.max(max, label.length), 0)
    return Math.max(172, Math.min(360, 76 + longest * DROPDOWN_CHAR_WIDTH))
  }, [creatingEnvironment, environments])
  const tabListWidth = useMemo(() => {
    const tabLabels = tabs.map((tab) => `${tab.request.method || "GET"} ${tab.title || "Untitled"}`)
    const envLabels = openEnvironmentTabs.map((tab) => `Environment ${tab.name}`)
    const longest = [...tabLabels, ...envLabels, "No Environment"].reduce((max, label) => Math.max(max, label.length), 0)
    return Math.max(240, Math.min(460, 86 + longest * DROPDOWN_CHAR_WIDTH))
  }, [openEnvironmentTabs, tabs])

  if (!hasAnyTab) {
    return (
      <div
        className={cn(
          "flex items-center h-[var(--size-tab)] border-b flex-shrink-0",
          "bg-[var(--surface-secondary)] border-[var(--border-color)]"
        )}
      >
        <div className="flex items-center gap-1 px-2 flex-shrink-0">
          <button
            className="h-[22px] w-[22px] flex items-center justify-center rounded-[var(--radius-sm)] text-[var(--fg-muted)] hover:text-[var(--fg)] hover:bg-[var(--tab-hover-bg)] transition-colors"
            onClick={handleAddTab}
            title="新建请求"
          >
            <AppIcon name="add" size={13} strokeWidth={2} />
          </button>
        </div>
        <div className="flex-1" />
        <div className="flex items-center gap-1 px-2 flex-shrink-0">
          {/* 环境选择 */}
          <div className="relative" ref={envDropdownRef}>
            <button
              className={cn(
                "h-[22px] px-2 flex items-center gap-1 rounded-[6px] text-[10px] transition-colors",
                "text-[var(--fg-muted)] hover:text-[var(--fg)] hover:bg-[var(--tab-hover-bg)]",
                activeEnvironmentId && "text-[var(--accent)]"
              )}
              onClick={() => setShowEnvDropdown(!showEnvDropdown)}
              title="切换环境"
            >
              <AppIcon name="globe" size={11} strokeWidth={1.8} />
              <span className="max-w-[100px] truncate">{activeEnvName}</span>
              <AppIcon name="arrowDown" size={8} strokeWidth={2} />
            </button>
            {showEnvDropdown && (
              <div
                className={cn("absolute right-0 top-full mt-1 z-[250] p-1", DROPDOWN_PANEL_CLASS)}
                style={{ width: `${envDropdownWidth}px` }}
              >
                <button
                  className={cn(
                    DROPDOWN_ITEM_CLASS,
                    !activeEnvironmentId ? "bg-[rgb(237,237,237)] text-[var(--fg)] font-medium" : "text-[var(--fg)]"
                  )}
                  onClick={() => { setActiveEnvironment(null); setShowEnvDropdown(false) }}
                >
                  <AppIcon name="globe" size={11} /> No Environment
                </button>
                <button
                  className={cn(DROPDOWN_ITEM_CLASS, "text-[var(--fg)]")}
                  onClick={() => void handleQuickCreateEnvironment()}
                  disabled={creatingEnvironment}
                >
                  <AppIcon name="add" size={11} />
                  {creatingEnvironment ? "创建中..." : "新建环境"}
                </button>
                <div className="h-px bg-[var(--border-subtle)] my-0.5" />
                {environments.map((env) => (
                  <div key={env.id} className="group relative">
                    <button
                      className={cn(
                        "w-full whitespace-nowrap px-3 pr-8 py-1.5 rounded-[7px] text-[11px] text-left transition-colors flex items-center gap-2 hover:bg-[var(--sidebar-hover)]",
                        activeEnvironmentId === env.id ? "bg-[rgb(237,237,237)] text-[var(--fg)] font-medium" : "text-[var(--fg)]"
                      )}
                      onClick={() => { setActiveEnvironment(env.id); setShowEnvDropdown(false) }}
                    >
                      <AppIcon name="globe" size={11} />
                      <span className="truncate">{env.name}</span>
                    </button>
                    <button
                      className="absolute right-2 top-1/2 -translate-y-1/2 h-5 w-5 flex items-center justify-center rounded-[4px] text-[var(--fg-muted)] hover:text-[var(--fg)] hover:bg-[var(--sidebar-hover)] opacity-0 group-hover:opacity-100 transition-opacity"
                      onClick={(e) => { e.stopPropagation(); handleEditEnvironment(env.id) }}
                      title="编辑环境"
                    >
                      <AppIcon name="pencil" size={10} />
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    )
  }

  const contextTab = contextMenu ? tabs.find((t) => t.id === contextMenu.tabId) : null
  const closableTabs = tabs.filter((t) => t.closable)

  return (
    <div
      className={cn(
        "flex items-end h-[var(--size-tab)] border-b flex-shrink-0",
        "bg-[var(--surface-secondary)] border-[var(--border-color)]"
      )}
    >
      {hasOverflow && (
        <div className="flex items-center flex-shrink-0 h-[calc(var(--size-tab)-2px)] px-0.5">
          <button
            className="h-full px-1 flex items-center justify-center text-[var(--fg-secondary)] hover:bg-[var(--tab-hover-bg)] transition-colors"
            onClick={() => scrollContainerRef.current?.scrollBy({ left: -200, behavior: "smooth" })}
          >
            <AppIcon name="arrowLeft" size={12} />
          </button>
          <button
            className="h-full px-1 flex items-center justify-center text-[var(--fg-secondary)] hover:bg-[var(--tab-hover-bg)] transition-colors"
            onClick={() => scrollContainerRef.current?.scrollBy({ left: 200, behavior: "smooth" })}
          >
            <AppIcon name="arrowRight" size={12} />
          </button>
        </div>
      )}

      <div
        ref={scrollContainerRef}
        className="flex items-end flex-1 min-w-0 overflow-x-hidden"
      >
        {tabs.map((tab) => {
          const isActive = !editingEnvironmentId && tab.id === activeTabId
          const isRenaming = renamingTabId === tab.id
          return (
            <div
              key={tab.id}
              data-tab-id={tab.id}
              className={cn(
                "flex items-center gap-[var(--size-gap-sm)] px-2.5 h-[calc(var(--size-tab)-2px)]",
                "text-[length:var(--size-font-2xs)] cursor-pointer select-none",
                "border-r border-[var(--border-color)] transition-colors group min-w-0 flex-shrink-0",
                isActive
                  ? "bg-[var(--surface)] text-[var(--fg)] border-b-2 border-b-[var(--accent)]"
                  : "text-[var(--fg)] opacity-80 hover:opacity-100 hover:bg-[var(--tab-hover-bg)]"
              )}
              title={`${tab.request.method} ${tab.title}`}
              onPointerDown={(e) => {
                if (isRenaming) return
                if (e.button === 0) {
                  e.preventDefault()
                  setEditingEnvironmentId(null)
                  setActiveTab(tab.id)
                }
                if (e.button === 1 && tab.closable) { e.preventDefault(); removeTab(tab.id) }
              }}
              onDoubleClick={() => startRename(tab.id, tab.title)}
              onContextMenu={(e) => { e.preventDefault(); setContextMenu({ x: e.clientX, y: e.clientY, tabId: tab.id }) }}
            >
              <span className={cn(
                "text-[9px] font-mono font-bold uppercase flex-shrink-0",
                METHOD_COLORS[tab.request.method as HttpMethod] || "text-[var(--fg-muted)]"
              )}>
                {tab.request.method?.substring(0, 3) || "GET"}
              </span>
              {isRenaming ? (
                <input
                  className="bg-transparent text-[length:var(--size-font-2xs)] text-[var(--fg)] border-b border-[var(--accent)] outline-none w-[80px]"
                  value={renameValue}
                  onChange={(e) => setRenameValue(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") void handleRenameSubmit()
                    if (e.key === "Escape") setRenamingTabId(null)
                  }}
                  onBlur={() => void handleRenameSubmit()}
                  autoFocus
                  onClick={(e) => e.stopPropagation()}
                  onPointerDown={(e) => e.stopPropagation()}
                />
              ) : (
                <span className="truncate max-w-[100px]">{tab.title}</span>
              )}
              {tab.dirty && (
                <span className="w-1 h-1 rounded-full bg-[var(--accent)] flex-shrink-0" />
              )}
              {tab.closable && !isRenaming && (
                <button
                  className={cn(
                    "flex items-center justify-center flex-shrink-0 transition-opacity",
                    "opacity-0 group-hover:opacity-100",
                    "text-[var(--fg-muted)] hover:text-[var(--fg)]"
                  )}
                  onClick={(e) => { e.stopPropagation(); removeTab(tab.id) }}
                >
                  <AppIcon name="clear" size={10} />
                </button>
              )}
            </div>
          )
        })}
        {openEnvironmentTabs.map((environmentTab) => {
          const isEnvironmentTabActive = editingEnvironmentId === environmentTab.id
          return (
            <div
              key={environmentTab.id}
              data-tab-id={`env-${environmentTab.id}`}
              className={cn(
                "flex items-center gap-[var(--size-gap-sm)] px-2.5 h-[calc(var(--size-tab)-2px)]",
                "text-[length:var(--size-font-2xs)] cursor-pointer select-none",
                "border-r border-[var(--border-color)] transition-colors group min-w-0 flex-shrink-0",
                isEnvironmentTabActive
                  ? "bg-[var(--surface)] text-[var(--fg)] border-b-2 border-b-[var(--accent)]"
                  : "text-[var(--fg)] opacity-80 hover:opacity-100 hover:bg-[var(--tab-hover-bg)]"
              )}
              title={`Environment ${environmentTab.name}`}
              onPointerDown={(e) => {
                if (e.button === 0) {
                  e.preventDefault()
                  setEditingEnvironmentId(environmentTab.id)
                }
                if (e.button === 1) {
                  e.preventDefault()
                  closeEnvironmentTab(environmentTab.id)
                }
              }}
            >
              <AppIcon
                name="globe"
                size={11}
                className={cn(
                  "flex-shrink-0",
                  isEnvironmentTabActive ? "text-[var(--accent)]" : "text-[var(--fg-muted)]"
                )}
              />
              <span className="truncate max-w-[120px]">{environmentTab.name}</span>
              <button
                className={cn(
                  "flex items-center justify-center flex-shrink-0 transition-opacity",
                  "opacity-0 group-hover:opacity-100",
                  "text-[var(--fg-muted)] hover:text-[var(--fg)]"
                )}
                onClick={(e) => {
                  e.stopPropagation()
                  closeEnvironmentTab(environmentTab.id)
                }}
                title="关闭环境标签"
              >
                <AppIcon name="clear" size={10} />
              </button>
            </div>
          )
        })}
        {!hasOverflow && (
          <button
            className="h-[22px] w-[22px] flex items-center justify-center rounded-[var(--radius-sm)] text-[var(--fg-muted)] hover:text-[var(--fg)] hover:bg-[var(--tab-hover-bg)] transition-colors flex-shrink-0 ml-0.5 self-center"
            onClick={handleAddTab}
            title="新建请求"
          >
            <AppIcon name="add" size={13} strokeWidth={2} />
          </button>
        )}
      </div>

      {/* 右侧工具区：Tab下拉列表、环境选择 */}
      <div className="flex items-center gap-0.5 px-1.5 flex-shrink-0 h-[calc(var(--size-tab)-2px)]">
        {hasOverflow && (
          <button
            className="h-[22px] w-[22px] flex items-center justify-center rounded-[var(--radius-sm)] text-[var(--fg-muted)] hover:text-[var(--fg)] hover:bg-[var(--tab-hover-bg)] transition-colors"
            onClick={handleAddTab}
            title="新建请求"
          >
            <AppIcon name="add" size={13} strokeWidth={2} />
          </button>
        )}

        {/* Tab 下拉列表 */}
        <div className="relative" ref={tabListRef}>
          <button
            className="h-[22px] w-[22px] flex items-center justify-center rounded-[var(--radius-sm)] text-[var(--fg-muted)] hover:text-[var(--fg)] hover:bg-[var(--tab-hover-bg)] transition-colors"
            onClick={() => setShowTabList(!showTabList)}
            title="显示所有标签"
          >
            <AppIcon name="arrowDown" size={11} strokeWidth={2} />
          </button>

          {showTabList && (
            <div
              className={cn("absolute right-0 top-full mt-1 z-[250] max-h-[400px] overflow-y-auto p-1", DROPDOWN_PANEL_CLASS)}
              style={{ width: `${tabListWidth}px` }}
            >
              {openEnvironmentTabs.map((environmentTab) => (
                <button
                  key={`list-env-${environmentTab.id}`}
                  className={cn(
                    DROPDOWN_ITEM_CLASS,
                    editingEnvironmentId === environmentTab.id ? "bg-[rgb(237,237,237)]" : ""
                  )}
                  onClick={() => {
                    setEditingEnvironmentId(environmentTab.id)
                    setShowTabList(false)
                  }}
                >
                  <AppIcon
                    name="globe"
                    size={11}
                    className={cn(
                      "flex-shrink-0",
                      editingEnvironmentId === environmentTab.id ? "text-[var(--accent)]" : "text-[var(--fg-muted)]"
                    )}
                  />
                  <span className="truncate text-[var(--fg)]">{environmentTab.name}</span>
                </button>
              ))}
              {tabs.map((tab) => (
                <button
                  key={tab.id}
                  className={cn(
                    DROPDOWN_ITEM_CLASS,
                    !editingEnvironmentId && tab.id === activeTabId ? "bg-[rgb(237,237,237)]" : ""
                  )}
                  onClick={() => {
                    setEditingEnvironmentId(null)
                    setActiveTab(tab.id)
                    setShowTabList(false)
                  }}
                >
                  <span className={cn(
                    "text-[9px] font-mono font-bold uppercase flex-shrink-0 w-[28px]",
                    METHOD_COLORS[tab.request.method as HttpMethod] || "text-[var(--fg-muted)]"
                  )}>
                    {tab.request.method?.substring(0, 3) || "GET"}
                  </span>
                  <span className="truncate text-[var(--fg)]">{tab.title}</span>
                  {tab.dirty && <span className="w-1.5 h-1.5 rounded-full bg-[var(--accent)] flex-shrink-0" />}
                </button>
              ))}
            </div>
          )}
        </div>

        {/* 分隔线 */}
        <div className="w-px h-3 bg-[var(--border-color)] mx-0.5" />

        {/* 环境选择 */}
        <div className="relative" ref={envDropdownRef}>
          <button
            className={cn(
              "h-[22px] px-2 flex items-center gap-1 rounded-[6px] text-[10px] transition-colors",
              "text-[var(--fg-muted)] hover:text-[var(--fg)] hover:bg-[var(--tab-hover-bg)]",
              activeEnvironmentId && "text-[var(--accent)]"
            )}
            onClick={() => setShowEnvDropdown(!showEnvDropdown)}
            title="切换环境"
          >
            <AppIcon name="globe" size={11} strokeWidth={1.8} />
            <span className="max-w-[100px] truncate">{activeEnvName}</span>
            <AppIcon name="arrowDown" size={8} strokeWidth={2} />
          </button>
          {showEnvDropdown && (
            <div
              className={cn("absolute right-0 top-full mt-1 z-[250] p-1", DROPDOWN_PANEL_CLASS)}
              style={{ width: `${envDropdownWidth}px` }}
            >
              <button
                className={cn(
                  DROPDOWN_ITEM_CLASS,
                  !activeEnvironmentId ? "bg-[rgb(237,237,237)] text-[var(--fg)] font-medium" : "text-[var(--fg)]"
                )}
                onClick={() => { setActiveEnvironment(null); setShowEnvDropdown(false) }}
              >
                <AppIcon name="globe" size={11} /> No Environment
              </button>
              <button
                className={cn(DROPDOWN_ITEM_CLASS, "text-[var(--fg)]")}
                onClick={() => void handleQuickCreateEnvironment()}
                disabled={creatingEnvironment}
              >
                <AppIcon name="add" size={11} />
                {creatingEnvironment ? "创建中..." : "新建环境"}
              </button>
              <div className="h-px bg-[var(--border-subtle)] my-0.5" />
              {environments.map((env) => (
                <div key={env.id} className="group relative">
                  <button
                    className={cn(
                      "w-full whitespace-nowrap px-3 pr-8 py-1.5 rounded-[7px] text-[11px] text-left transition-colors flex items-center gap-2 hover:bg-[var(--sidebar-hover)]",
                      activeEnvironmentId === env.id ? "bg-[rgb(237,237,237)] text-[var(--fg)] font-medium" : "text-[var(--fg)]"
                    )}
                    onClick={() => { setActiveEnvironment(env.id); setShowEnvDropdown(false) }}
                  >
                    <AppIcon name="globe" size={11} />
                    <span className="truncate">{env.name}</span>
                  </button>
                  <button
                    className="absolute right-2 top-1/2 -translate-y-1/2 h-5 w-5 flex items-center justify-center rounded-[4px] text-[var(--fg-muted)] hover:text-[var(--fg)] hover:bg-[var(--sidebar-hover)] opacity-0 group-hover:opacity-100 transition-opacity"
                    onClick={(e) => { e.stopPropagation(); handleEditEnvironment(env.id) }}
                    title="编辑环境"
                  >
                    <AppIcon name="pencil" size={10} />
                  </button>
                </div>
              ))}
              {environments.length === 0 && (
                <div className="px-3 py-2 text-[10px] text-[var(--fg-muted)]">
                  暂无环境，请在侧边栏环境管理中创建
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {contextMenu && createPortal(
        <div
          ref={menuRef}
          className={cn(
            "fixed z-[260] w-max p-1 animate-fade-in",
            DROPDOWN_PANEL_CLASS
          )}
          style={{ left: contextMenu.x, top: contextMenu.y }}
        >
          <button
            className="w-full whitespace-nowrap px-2.5 py-1.5 rounded-[7px] text-[length:var(--size-font-2xs)] text-left hover:bg-[var(--sidebar-hover)] text-[var(--fg)] flex items-center gap-2"
            onClick={() => { if (contextTab) startRename(contextTab.id, contextTab.title) }}
          >
            <AppIcon name="pencil" size={12} /> 重命名
          </button>
          {contextTab?.closable && (
            <button
              className="w-full whitespace-nowrap px-2.5 py-1.5 rounded-[7px] text-[length:var(--size-font-2xs)] text-left hover:bg-[var(--sidebar-hover)] text-[var(--fg)] flex items-center gap-2"
              onClick={() => { removeTab(contextMenu.tabId); setContextMenu(null) }}
            >
              <AppIcon name="clear" size={12} /> 关闭
            </button>
          )}
          <button
            className="w-full whitespace-nowrap px-2.5 py-1.5 rounded-[7px] text-[length:var(--size-font-2xs)] text-left hover:bg-[var(--sidebar-hover)] text-[var(--fg)] disabled:opacity-40 flex items-center gap-2"
            disabled={closableTabs.length <= 1}
            onClick={() => { closeOtherTabs(contextMenu.tabId); setContextMenu(null) }}
          >
            <AppIcon name="clear" size={12} /> 关闭其他
          </button>
          <button
            className="w-full whitespace-nowrap px-2.5 py-1.5 rounded-[7px] text-[length:var(--size-font-2xs)] text-left hover:bg-[var(--sidebar-hover)] text-[var(--fg)] disabled:opacity-40 flex items-center gap-2"
            disabled={closableTabs.length === 0}
            onClick={() => { closeAllTabs(currentProjectId || undefined); setContextMenu(null) }}
          >
            <AppIcon name="clear" size={12} /> 关闭全部
          </button>
        </div>,
        document.body
      )}
    </div>
  )
}
