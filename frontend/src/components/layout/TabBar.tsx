import { useState, useEffect, useRef, useCallback, useMemo } from "react"
import { createPortal } from "react-dom"
import { AppIcon } from "@/components/ui/icon"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { useI18n } from "@/hooks/useI18n"
import { cn } from "@/lib/utils"
import { METHOD_COLORS, type HttpMethod } from "@/lib/constants"
import { useTabStore, getProjectActiveTabIdFromState, getProjectTabsFromState, type RequestTab } from "@/stores/tabStore"
import { useProjectStore } from "@/stores/projectStore"
import { useEnvironmentStore } from "@/stores/environmentStore"
import { useUIStore } from "@/stores/uiStore"
import { isAutoHeaderDisabledMarkerKey } from "@/lib/autoHeaders"

interface TabContextMenuState {
  x: number
  y: number
  tabId: string
}

const DROPDOWN_CHAR_WIDTH = 7
const DROPDOWN_PANEL_CLASS = "rounded-[10px] border border-[var(--border-color)] bg-[var(--surface-elevated)] shadow-[var(--shadow-lg)]"
const DROPDOWN_ITEM_CLASS = "w-full whitespace-nowrap px-3 py-1.5 rounded-[7px] text-[11px] text-left transition-colors flex items-center gap-2"
const TAB_BAR_CLASS = "relative z-[40] flex h-[var(--size-tab)] flex-shrink-0 items-stretch border-b bg-[var(--surface-secondary)] border-[var(--tab-divider)]"
const TAB_ITEM_CLASS = "group relative flex h-[calc(var(--size-tab)-2px)] min-w-[148px] max-w-[230px] flex-shrink-0 cursor-pointer select-none items-center gap-1 self-end border-r border-[var(--tab-divider)] px-2 text-[length:var(--size-font-2xs)] transition-colors"
const TAB_ITEM_ACTIVE_CLASS = "bg-[var(--tab-active-bg)] font-medium text-[var(--fg)]"
const TAB_ITEM_INACTIVE_CLASS = "text-[var(--fg-secondary)] hover:bg-[var(--tab-hover-bg)] hover:text-[var(--fg)]"
const TAB_ICON_CLASS = "h-[15px] w-[15px] flex-shrink-0"
const TAB_METHOD_CLASS = "flex-shrink-0 text-left font-mono text-[9px] font-bold uppercase leading-none"
const TAB_ACTION_BUTTON_CLASS = "flex h-[22px] w-[22px] flex-shrink-0 items-center justify-center rounded-[var(--radius-sm)] text-[var(--fg-muted)] transition-colors hover:bg-[var(--tab-hover-bg)] hover:text-[var(--fg)]"

export function TabBar() {
  const { t } = useI18n()
  const tabs = useTabStore(getProjectTabsFromState)
  const activeTabId = useTabStore(getProjectActiveTabIdFromState)
  const setActiveTab = useTabStore((s) => s.setActiveTab)
  const removeTab = useTabStore((s) => s.removeTab)
  const closeOtherTabs = useTabStore((s) => s.closeOtherTabs)
  const closeAllTabs = useTabStore((s) => s.closeAllTabs)
  const addNewUnsavedTab = useTabStore((s) => s.addNewUnsavedTab)
  const currentProjectId = useProjectStore((s) => s.currentProjectId)
  const saveRequestToBackend = useProjectStore((s) => s.saveRequestToBackend)
  const editingEnvironmentId = useUIStore((s) => s.editingEnvironmentId)
  const openEnvironmentTabIds = useUIStore((s) => s.openEnvironmentTabIds)
  const setEditingEnvironmentId = useUIStore((s) => s.setEditingEnvironmentId)
  const closeEnvironmentTab = useUIStore((s) => s.closeEnvironmentTab)
  const alwaysDiscardUnsavedOnClose = useUIStore((s) => s.alwaysDiscardUnsavedOnClose)
  const alwaysSaveUnsavedOnClose = useUIStore((s) => s.alwaysSaveUnsavedOnClose)
  const setAlwaysDiscardUnsavedOnClose = useUIStore((s) => s.setAlwaysDiscardUnsavedOnClose)
  const setAlwaysSaveUnsavedOnClose = useUIStore((s) => s.setAlwaysSaveUnsavedOnClose)

  const { environments, activeEnvironmentId, setActiveEnvironment, loadEnvironments, createEnvironment } = useEnvironmentStore()

  const [contextMenu, setContextMenu] = useState<TabContextMenuState | null>(null)
  const [hasOverflow, setHasOverflow] = useState(false)
  const [showTabList, setShowTabList] = useState(false)
  const [showEnvDropdown, setShowEnvDropdown] = useState(false)
  const [envSearchQuery, setEnvSearchQuery] = useState("")
  const [creatingEnvironment, setCreatingEnvironment] = useState(false)
  const [closeConfirmTabId, setCloseConfirmTabId] = useState<string | null>(null)
  const [closeConfirmSaving, setCloseConfirmSaving] = useState(false)
  const scrollContainerRef = useRef<HTMLDivElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)
  const tabListRef = useRef<HTMLDivElement>(null)
  const envTriggerRef = useRef<HTMLButtonElement>(null)
  const envDropdownPanelRef = useRef<HTMLDivElement>(null)
  const [envDropdownPosition, setEnvDropdownPosition] = useState<{ top: number; left: number } | null>(null)

  const closeConfirmTab = useMemo(
    () => (closeConfirmTabId ? tabs.find((tab) => tab.id === closeConfirmTabId) ?? null : null),
    [closeConfirmTabId, tabs]
  )

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

  const saveTabToBackend = useCallback(async (tab: RequestTab) => {
    if (!currentProjectId) return
    const req = tab.request
    const requestItem = {
      id: req.id,
      name: req.name,
      method: req.method,
      url: req.url,
      params: req.params.filter((p) => p.key).map((p) => ({ key: p.key, value: p.value, description: p.description ?? "" })),
      headers: req.headers
        .filter((h) => h.key && !isAutoHeaderDisabledMarkerKey(h.key))
        .map((h) => ({ key: h.key, value: h.value, description: h.description ?? "" })),
      body: {
        type: req.body.type,
        raw: req.body.raw ?? "",
        json: req.body.json ?? "",
        formUrlEncoded: (req.body.formUrlEncoded ?? [])
          .filter((f) => f.key)
          .map((f) => ({ key: f.key, value: f.value, description: f.description ?? "" })),
        formData: (req.body.formData ?? [])
          .filter((f) => f.key)
          .map((f) => ({
            key: f.key,
            value: f.value,
            description: f.description ?? "",
            type: f.type,
            filePath: f.filePath ?? "",
            fileName: f.fileName ?? "",
          })),
      },
      auth: {
        type: req.auth.type,
        basic: req.auth.basic ?? { username: "", password: "" },
        bearer: req.auth.bearer ?? { token: "" },
        apiKey: req.auth.apiKey ?? { key: "", value: "", addTo: "header" },
      },
      folderId: req.folderId ?? "",
      projectId: currentProjectId,
      createdAt: req.createdAt,
      updatedAt: new Date().toISOString(),
    }
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    await saveRequestToBackend(requestItem as any)
  }, [currentProjectId, saveRequestToBackend])

  const saveDirtyTabsBeforeClosing = useCallback(async (targetTabs: RequestTab[]) => {
    const dirtyTabs = targetTabs.filter((tab) => tab.dirty)
    if (dirtyTabs.length === 0) return true

    try {
      await Promise.all(dirtyTabs.map(async (tab) => {
        await saveTabToBackend(tab)
        useTabStore.getState().markTabDirty(tab.id, false)
      }))
      return true
    } catch {
      window.alert(t("保存失败，请重试。", "Save failed. Please try again."))
      return false
    }
  }, [saveTabToBackend, t])

  const requestTabClose = useCallback(async (tabId: string) => {
    const tab = tabs.find((candidate) => candidate.id === tabId)
    if (!tab || !tab.closable) return
    if (!tab.dirty) {
      removeTab(tabId)
      return
    }
    if (alwaysDiscardUnsavedOnClose) {
      removeTab(tabId)
      return
    }
    if (alwaysSaveUnsavedOnClose) {
      const saved = await saveDirtyTabsBeforeClosing([tab])
      if (saved) removeTab(tabId)
      return
    }
    setCloseConfirmTabId(tabId)
  }, [alwaysDiscardUnsavedOnClose, alwaysSaveUnsavedOnClose, removeTab, saveDirtyTabsBeforeClosing, tabs])

  const handleCloseOtherTabs = useCallback(async (tabId: string) => {
    const targetToClose = tabs.filter((tab) => tab.id !== tabId && tab.closable)
    if (alwaysSaveUnsavedOnClose) {
      const saved = await saveDirtyTabsBeforeClosing(targetToClose)
      if (!saved) return
      closeOtherTabs(tabId)
      return
    }
    if (!alwaysDiscardUnsavedOnClose && targetToClose.some((tab) => tab.dirty)) {
      const confirmed = window.confirm(t("即将关闭其他标签页，其中包含未保存修改。确定继续并丢弃这些修改吗？", "You're about to close other tabs with unsaved changes. Continue and discard them?"))
      if (!confirmed) return
    }
    closeOtherTabs(tabId)
  }, [alwaysDiscardUnsavedOnClose, alwaysSaveUnsavedOnClose, closeOtherTabs, saveDirtyTabsBeforeClosing, t, tabs])

  const handleCloseAllTabs = useCallback(async () => {
    const targetToClose = tabs.filter((tab) => tab.closable)
    if (alwaysSaveUnsavedOnClose) {
      const saved = await saveDirtyTabsBeforeClosing(targetToClose)
      if (!saved) return
      closeAllTabs(currentProjectId || undefined)
      return
    }
    if (!alwaysDiscardUnsavedOnClose && targetToClose.some((tab) => tab.dirty)) {
      const confirmed = window.confirm(t("即将关闭全部标签页，其中包含未保存修改。确定继续并丢弃这些修改吗？", "You're about to close all tabs with unsaved changes. Continue and discard them?"))
      if (!confirmed) return
    }
    closeAllTabs(currentProjectId || undefined)
  }, [alwaysDiscardUnsavedOnClose, alwaysSaveUnsavedOnClose, closeAllTabs, currentProjectId, saveDirtyTabsBeforeClosing, t, tabs])

  const handleConfirmCloseWithoutSave = useCallback(() => {
    if (!closeConfirmTabId) return
    removeTab(closeConfirmTabId)
    setCloseConfirmTabId(null)
  }, [closeConfirmTabId, removeTab])

  const handleConfirmCloseWithSave = useCallback(async () => {
    if (!closeConfirmTab || closeConfirmSaving) return
    setCloseConfirmSaving(true)
    try {
      await saveTabToBackend(closeConfirmTab)
      useTabStore.getState().markTabDirty(closeConfirmTab.id, false)
      removeTab(closeConfirmTab.id)
      setCloseConfirmTabId(null)
    } finally {
      setCloseConfirmSaving(false)
    }
  }, [closeConfirmSaving, closeConfirmTab, removeTab, saveTabToBackend])

  useEffect(() => {
    const handler = () => {
      if (editingEnvironmentId || !activeTabId) return
      requestTabClose(activeTabId)
    }
    window.addEventListener("minipost:close-active-request-tab", handler as EventListener)
    return () => window.removeEventListener("minipost:close-active-request-tab", handler as EventListener)
  }, [activeTabId, editingEnvironmentId, requestTabClose])

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
    if (!activeEnvironmentId) return t("无环境", "No Environment")
    const env = environments.find((e) => e.id === activeEnvironmentId)
    return env?.name || t("无环境", "No Environment")
  }, [activeEnvironmentId, environments, t])

  const openEnvironmentTabs = useMemo(
    () => openEnvironmentTabIds.map((id) => ({
      id,
      name: environments.find((env) => env.id === id)?.name || t("环境", "Environment"),
    })),
    [environments, openEnvironmentTabIds, t]
  )

  const hasAnyTab = tabs.length > 0 || openEnvironmentTabs.length > 0
  const envDropdownWidth = useMemo(() => {
    const labels = [
      t("无环境", "No Environment"),
      creatingEnvironment ? t("创建中...", "Creating...") : t("新建环境", "New environment"),
      ...environments.map((env) => env.name),
    ]
    const longest = labels.reduce((max, label) => Math.max(max, label.length), 0)
    return Math.max(172, Math.min(360, 76 + longest * DROPDOWN_CHAR_WIDTH))
  }, [creatingEnvironment, environments, t])
  const filteredEnvironments = useMemo(() => {
    const query = envSearchQuery.trim().toLowerCase()
    if (!query) return environments
    return environments.filter((env) => env.name.toLowerCase().includes(query))
  }, [envSearchQuery, environments])
  const updateEnvDropdownPosition = useCallback(() => {
    const trigger = envTriggerRef.current
    if (!trigger || typeof window === "undefined") return

    const rect = trigger.getBoundingClientRect()
    const margin = 8
    const left = Math.max(
      margin,
      Math.min(rect.right - envDropdownWidth, window.innerWidth - envDropdownWidth - margin)
    )
    const top = Math.max(margin, rect.bottom + 4)
    setEnvDropdownPosition({ top, left })
  }, [envDropdownWidth])

  useEffect(() => {
    if (!showEnvDropdown) return
    updateEnvDropdownPosition()

    const syncPosition = () => updateEnvDropdownPosition()
    const handler = (e: MouseEvent) => {
      const target = e.target as Node
      const clickedTrigger = envTriggerRef.current?.contains(target)
      const clickedPanel = envDropdownPanelRef.current?.contains(target)
      if (!clickedTrigger && !clickedPanel) setShowEnvDropdown(false)
    }
    window.addEventListener("resize", syncPosition)
    window.addEventListener("scroll", syncPosition, true)
    document.addEventListener("mousedown", handler)
    return () => {
      window.removeEventListener("resize", syncPosition)
      window.removeEventListener("scroll", syncPosition, true)
      document.removeEventListener("mousedown", handler)
    }
  }, [showEnvDropdown, updateEnvDropdownPosition])
  const tabListWidth = useMemo(() => {
    const tabLabels = tabs.map((tab) => `${tab.request.method || "GET"} ${tab.title || "Untitled"}`)
    const envLabels = openEnvironmentTabs.map((tab) => `Environment ${tab.name}`)
    const longest = [...tabLabels, ...envLabels, t("无环境", "No Environment")].reduce((max, label) => Math.max(max, label.length), 0)
    return Math.max(240, Math.min(460, 86 + longest * DROPDOWN_CHAR_WIDTH))
  }, [openEnvironmentTabs, t, tabs])
  const envDropdownPortal = showEnvDropdown && envDropdownPosition && createPortal(
    <div
      ref={envDropdownPanelRef}
      className={cn("fixed z-[460] p-1", DROPDOWN_PANEL_CLASS)}
      style={{
        left: `${envDropdownPosition.left}px`,
        top: `${envDropdownPosition.top}px`,
        width: `${envDropdownWidth}px`,
      }}
    >
      <div className="mb-1 flex items-center gap-1 border-b border-[var(--border-subtle)] pb-1">
        <div className="relative flex-1">
          <AppIcon
            name="search"
            size={10}
            className="pointer-events-none absolute left-2 top-1/2 -translate-y-1/2 text-[var(--fg-muted)]"
          />
          <input
            value={envSearchQuery}
            onChange={(event) => setEnvSearchQuery(event.target.value)}
            placeholder={t("搜索环境...", "Search environments...")}
            className={cn(
              "h-7 w-full rounded-[7px] border border-[var(--border-color)] bg-[var(--surface)] pl-6 pr-2",
              "text-[11px] text-[var(--fg)] outline-none placeholder:text-[var(--fg-muted)] focus:border-[var(--accent)]"
            )}
          />
        </div>
        <button
          className={cn(
            "h-7 rounded-[7px] border px-2.5 text-[11px] transition-colors",
            "border-[var(--border-color)] bg-[var(--surface)] text-[var(--fg)] hover:bg-[var(--surface-secondary)]"
          )}
          onClick={() => void handleQuickCreateEnvironment()}
          disabled={creatingEnvironment}
        >
          {creatingEnvironment ? t("创建中", "Creating") : t("新建", "New")}
        </button>
      </div>
      <button
        className={cn(
          DROPDOWN_ITEM_CLASS,
          !activeEnvironmentId ? "bg-[var(--selected-bg)] text-[var(--fg)] font-medium" : "text-[var(--fg)]"
        )}
        onClick={() => { setActiveEnvironment(null); setShowEnvDropdown(false) }}
      >
        <AppIcon name="globe" size={11} /> {t("无环境", "No Environment")}
      </button>
      <div className="h-px bg-[var(--border-subtle)] my-0.5" />
      {filteredEnvironments.map((env) => (
        <div key={env.id} className="group relative">
          <button
            className={cn(
              "w-full whitespace-nowrap px-3 pr-8 py-1.5 rounded-[7px] text-[11px] text-left transition-colors flex items-center gap-2 hover:bg-[var(--sidebar-hover)]",
              activeEnvironmentId === env.id ? "bg-[var(--selected-bg)] text-[var(--fg)] font-medium" : "text-[var(--fg)]"
            )}
            onClick={() => { setActiveEnvironment(env.id); setShowEnvDropdown(false) }}
          >
            <AppIcon name="globe" size={11} />
            <span className="truncate">{env.name}</span>
          </button>
          <button
            className="absolute right-2 top-1/2 -translate-y-1/2 h-5 w-5 flex items-center justify-center rounded-[4px] text-[var(--fg-muted)] hover:text-[var(--fg)] hover:bg-[var(--sidebar-hover)] opacity-0 group-hover:opacity-100 transition-opacity"
            onClick={(e) => { e.stopPropagation(); handleEditEnvironment(env.id) }}
            title={t("编辑环境", "Edit environment")}
          >
            <AppIcon name="pencil" size={10} />
          </button>
        </div>
      ))}
      {filteredEnvironments.length === 0 && (
        <div className="px-3 py-2 text-[10px] text-[var(--fg-muted)]">
          {t("未找到匹配环境", "No matching environments")}
        </div>
      )}
    </div>,
    document.body
  )

  if (!hasAnyTab) {
    return (
      <>
        <div
          className={TAB_BAR_CLASS}
        >
          <div className="flex flex-shrink-0 items-center border-r border-[var(--tab-divider)] px-1.5">
            <button
              className={TAB_ACTION_BUTTON_CLASS}
              onClick={handleAddTab}
              title={t("新建请求", "New request")}
            >
              <AppIcon name="add" size={13} strokeWidth={2} />
            </button>
          </div>
          <div className="flex-1" />
          <div className="flex flex-shrink-0 items-center border-l border-[var(--tab-divider)] px-1.5">
            {/* 环境选择 */}
            <div className="relative">
              <button
                ref={envTriggerRef}
                className={cn(
                  "flex h-[22px] items-center gap-1 rounded-[6px] px-2 text-[10px] transition-colors",
                  "text-[var(--fg-muted)] hover:text-[var(--fg)] hover:bg-[var(--tab-hover-bg)]",
                  activeEnvironmentId && "text-[var(--accent)]"
                )}
                onClick={() => {
                  setShowEnvDropdown((prev) => {
                    const next = !prev
                    if (next) setEnvSearchQuery("")
                    return next
                  })
                }}
                title={t("切换环境", "Switch environment")}
              >
                <AppIcon name="globe" size={11} strokeWidth={1.8} />
                <span className="max-w-[100px] truncate">{activeEnvName}</span>
                <AppIcon name="arrowDown" size={8} strokeWidth={2} />
              </button>
            </div>
          </div>
        </div>
        {envDropdownPortal}
      </>
    )
  }

  const contextTab = contextMenu ? tabs.find((t) => t.id === contextMenu.tabId) : null
  const closableTabs = tabs.filter((t) => t.closable)

  return (
    <div
      className={TAB_BAR_CLASS}
    >
      {hasOverflow && (
        <div className="flex h-[calc(var(--size-tab)-2px)] flex-shrink-0 items-center self-end border-r border-[var(--tab-divider)] px-0.5">
          <button
            className="flex h-full px-1 items-center justify-center text-[var(--fg-secondary)] transition-colors hover:bg-[var(--tab-hover-bg)]"
            onClick={() => scrollContainerRef.current?.scrollBy({ left: -200, behavior: "smooth" })}
          >
            <AppIcon name="arrowLeft" size={12} />
          </button>
          <button
            className="flex h-full px-1 items-center justify-center text-[var(--fg-secondary)] transition-colors hover:bg-[var(--tab-hover-bg)]"
            onClick={() => scrollContainerRef.current?.scrollBy({ left: 200, behavior: "smooth" })}
          >
            <AppIcon name="arrowRight" size={12} />
          </button>
        </div>
      )}

      <div
        ref={scrollContainerRef}
        className="flex min-w-0 flex-1 items-stretch overflow-x-hidden"
      >
        {tabs.map((tab) => {
          const isActive = !editingEnvironmentId && tab.id === activeTabId
          const tabUrl = tab.request.url?.trim() || t("未设置请求地址", "Request URL not set")
          return (
            <Tooltip key={tab.id}>
              <TooltipTrigger asChild>
                <div
                  data-tab-id={tab.id}
                  className={cn(
                    TAB_ITEM_CLASS,
                    isActive ? TAB_ITEM_ACTIVE_CLASS : TAB_ITEM_INACTIVE_CLASS
                  )}
                  onPointerDown={(e) => {
                    if (e.button === 0) {
                      e.preventDefault()
                      setEditingEnvironmentId(null)
                      setActiveTab(tab.id)
                    }
                    if (e.button === 1 && tab.closable) { e.preventDefault(); requestTabClose(tab.id) }
                  }}
                  onContextMenu={(e) => { e.preventDefault(); setContextMenu({ x: e.clientX, y: e.clientY, tabId: tab.id }) }}
                >
                  {isActive && <span className="absolute inset-x-0 bottom-[-1px] h-0.5 bg-[var(--tab-active-border)]" />}
                  <span className={cn(
                    TAB_METHOD_CLASS,
                    METHOD_COLORS[tab.request.method as HttpMethod] || "text-[var(--fg-muted)]"
                  )}>
                    {tab.request.method || "GET"}
                  </span>
                  <span className={cn("min-w-0 flex-1 truncate", isActive ? "text-[var(--fg)]" : "text-[var(--fg-secondary)]")}>
                    {tab.title}
                  </span>
                  {tab.dirty && (
                    <span className="h-1.5 w-1.5 flex-shrink-0 rounded-full bg-[var(--accent)]" />
                  )}
                  {tab.closable && (
                    <button
                      className={cn(
                        "flex h-4 w-4 flex-shrink-0 items-center justify-center rounded-[4px] transition-opacity",
                        "text-[var(--fg-muted)] opacity-0 hover:bg-[var(--tab-hover-bg)] hover:text-[var(--fg)] group-hover:opacity-100"
                      )}
                      onClick={(e) => { e.stopPropagation(); requestTabClose(tab.id) }}
                    >
                      <AppIcon name="clear" size={10} />
                    </button>
                  )}
                </div>
              </TooltipTrigger>
              <TooltipContent side="bottom" align="start" sideOffset={6} className="max-w-[460px] px-2.5 py-2">
                <div className="text-[13px] leading-[1.35] text-[var(--fg)] break-words">
                  {tab.title}
                </div>
                <div className="mt-1 text-[12px] leading-[1.35] text-[var(--fg-muted)] break-all">
                  {tabUrl}
                </div>
              </TooltipContent>
            </Tooltip>
          )
        })}
        {openEnvironmentTabs.map((environmentTab) => {
          const isEnvironmentTabActive = editingEnvironmentId === environmentTab.id
          return (
            <div
              key={environmentTab.id}
              data-tab-id={`env-${environmentTab.id}`}
              className={cn(
                TAB_ITEM_CLASS,
                isEnvironmentTabActive ? TAB_ITEM_ACTIVE_CLASS : TAB_ITEM_INACTIVE_CLASS
              )}
              title={`${t("环境", "Environment")} ${environmentTab.name}`}
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
              {isEnvironmentTabActive && <span className="absolute inset-x-0 bottom-[-1px] h-0.5 bg-[var(--tab-active-border)]" />}
              <AppIcon
                name="globe"
                size={15}
                strokeWidth={1.75}
                className={cn(
                  TAB_ICON_CLASS,
                  isEnvironmentTabActive ? "text-[var(--fg)]" : "text-[var(--fg-secondary)]"
                )}
              />
              <span className={cn("min-w-0 flex-1 truncate", isEnvironmentTabActive ? "text-[var(--fg)]" : "text-[var(--fg-secondary)]")}>
                {environmentTab.name}
              </span>
              <button
                className={cn(
                  "flex h-4 w-4 flex-shrink-0 items-center justify-center rounded-[4px] transition-opacity",
                  "text-[var(--fg-muted)] opacity-0 hover:bg-[var(--tab-hover-bg)] hover:text-[var(--fg)] group-hover:opacity-100"
                )}
                onClick={(e) => {
                  e.stopPropagation()
                  closeEnvironmentTab(environmentTab.id)
                }}
                title={t("关闭环境标签", "Close environment tab")}
              >
                <AppIcon name="clear" size={10} />
              </button>
            </div>
          )
        })}
        {!hasOverflow && (
          <button
            className={cn(TAB_ACTION_BUTTON_CLASS, "ml-1 self-center")}
            onClick={handleAddTab}
            title={t("新建请求", "New request")}
          >
            <AppIcon name="add" size={13} strokeWidth={2} />
          </button>
        )}
      </div>

      {/* 右侧工具区：Tab下拉列表、环境选择 */}
      <div className="flex h-full flex-shrink-0 items-center gap-0.5 border-l border-[var(--tab-divider)] px-1.5">
        {hasOverflow && (
          <button
            className={TAB_ACTION_BUTTON_CLASS}
            onClick={handleAddTab}
            title={t("新建请求", "New request")}
          >
            <AppIcon name="add" size={13} strokeWidth={2} />
          </button>
        )}

        {/* Tab 下拉列表 */}
        <div className="relative" ref={tabListRef}>
          <button
            className={TAB_ACTION_BUTTON_CLASS}
            onClick={() => setShowTabList(!showTabList)}
            title={t("显示所有标签", "Show all tabs")}
          >
            <AppIcon name="arrowDown" size={11} strokeWidth={2} />
          </button>

          {showTabList && (
            <div
              className={cn("absolute right-0 top-full mt-1 z-[360] max-h-[400px] overflow-y-auto p-1", DROPDOWN_PANEL_CLASS)}
              style={{ width: `${tabListWidth}px` }}
            >
              {openEnvironmentTabs.map((environmentTab) => (
                <button
                  key={`list-env-${environmentTab.id}`}
                  className={cn(
                    DROPDOWN_ITEM_CLASS,
                    editingEnvironmentId === environmentTab.id ? "bg-[var(--selected-bg)]" : ""
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
                    !editingEnvironmentId && tab.id === activeTabId ? "bg-[var(--selected-bg)]" : ""
                  )}
                  onClick={() => {
                    setEditingEnvironmentId(null)
                    setActiveTab(tab.id)
                    setShowTabList(false)
                  }}
                >
                  <span className={cn(
                    "text-[9px] font-mono font-bold uppercase flex-shrink-0 w-[40px] text-right",
                    METHOD_COLORS[tab.request.method as HttpMethod] || "text-[var(--fg-muted)]"
                  )}>
                    {tab.request.method || "GET"}
                  </span>
                  <span className="truncate text-[var(--fg)]">{tab.title}</span>
                  {tab.dirty && <span className="w-1.5 h-1.5 rounded-full bg-[var(--accent)] flex-shrink-0" />}
                </button>
              ))}
            </div>
          )}
        </div>

        {/* 分隔线 */}
        <div className="w-px h-3 bg-[var(--tab-divider)] mx-0.5" />

        {/* 环境选择 */}
        <div className="relative">
          <button
            ref={envTriggerRef}
            className={cn(
              "flex h-[22px] items-center gap-1 rounded-[6px] px-2 text-[10px] transition-colors",
              "text-[var(--fg-muted)] hover:text-[var(--fg)] hover:bg-[var(--tab-hover-bg)]",
              activeEnvironmentId && "text-[var(--accent)]"
            )}
            onClick={() => {
              setShowEnvDropdown((prev) => {
                const next = !prev
                if (next) setEnvSearchQuery("")
                return next
              })
            }}
            title={t("切换环境", "Switch environment")}
          >
            <AppIcon name="globe" size={11} strokeWidth={1.8} />
            <span className="max-w-[100px] truncate">{activeEnvName}</span>
            <AppIcon name="arrowDown" size={8} strokeWidth={2} />
          </button>
        </div>
      </div>

      {envDropdownPortal}

      {contextMenu && createPortal(
        <div
          ref={menuRef}
          className={cn(
            "fixed z-[260] w-max p-1 animate-fade-in",
            DROPDOWN_PANEL_CLASS
          )}
          style={{ left: contextMenu.x, top: contextMenu.y }}
        >
          {contextTab?.closable && (
            <button
              className="w-full whitespace-nowrap px-2.5 py-1.5 rounded-[7px] text-[length:var(--size-font-2xs)] text-left hover:bg-[var(--sidebar-hover)] text-[var(--fg)] flex items-center gap-2"
              onClick={() => { requestTabClose(contextMenu.tabId); setContextMenu(null) }}
            >
              <AppIcon name="clear" size={12} /> {t("关闭", "Close")}
            </button>
          )}
          <button
            className="w-full whitespace-nowrap px-2.5 py-1.5 rounded-[7px] text-[length:var(--size-font-2xs)] text-left hover:bg-[var(--sidebar-hover)] text-[var(--fg)] disabled:opacity-40 flex items-center gap-2"
            disabled={closableTabs.length <= 1}
            onClick={() => { handleCloseOtherTabs(contextMenu.tabId); setContextMenu(null) }}
          >
            <AppIcon name="clear" size={12} /> {t("关闭其他", "Close others")}
          </button>
          <button
            className="w-full whitespace-nowrap px-2.5 py-1.5 rounded-[7px] text-[length:var(--size-font-2xs)] text-left hover:bg-[var(--sidebar-hover)] text-[var(--fg)] disabled:opacity-40 flex items-center gap-2"
            disabled={closableTabs.length === 0}
            onClick={() => { handleCloseAllTabs(); setContextMenu(null) }}
          >
            <AppIcon name="clear" size={12} /> {t("关闭全部", "Close all")}
          </button>
        </div>,
        document.body
      )}
      {closeConfirmTab && createPortal(
        <div className="fixed inset-0 z-[320] flex items-center justify-center" onClick={() => { if (!closeConfirmSaving) setCloseConfirmTabId(null) }}>
          <div className="absolute inset-0 bg-black/40 backdrop-blur-[1.5px]" />
          <div
            className="relative z-[321] w-[460px] rounded-[12px] border border-[var(--border-color)] bg-[var(--surface)] shadow-[var(--shadow-lg)]"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="flex items-center justify-between px-4 py-3 border-b border-[var(--border-subtle)]">
              <div className="text-[16px] font-semibold text-[var(--fg)]">{t("保存更改", "Save changes")}</div>
              <button
                type="button"
                className="h-6 w-6 inline-flex items-center justify-center rounded-[6px] text-[var(--fg-muted)] hover:bg-[var(--button-bg)] hover:text-[var(--fg)]"
                onClick={() => setCloseConfirmTabId(null)}
                disabled={closeConfirmSaving}
              >
                <AppIcon name="clear" size={12} />
              </button>
            </div>
            <div className="px-4 py-3.5">
              <p className="text-[13px] text-[var(--fg)] leading-[1.6]">
                {t("请求", "Request")} <span className="font-medium">“{closeConfirmTab.title}”</span> {t("有未保存的修改。", "has unsaved changes.")}
              </p>
              <p className="mt-1 text-[12px] text-[var(--fg-secondary)]">{t("关闭后这些修改将会丢失，是否先保存？", "Closing will discard these changes. Save first?")}</p>
              <label className="mt-3 flex items-center gap-2 text-[12px] text-[var(--fg)] cursor-pointer select-none">
                <input
                  type="checkbox"
                  checked={alwaysSaveUnsavedOnClose}
                  onChange={(event) => setAlwaysSaveUnsavedOnClose(event.target.checked)}
                  className="h-4 w-4 rounded-[4px] border border-[var(--border-color)] accent-[var(--accent)]"
                />
                <span>{t("关闭 Tab 时默认保存未保存修改", "Save unsaved changes by default when closing tabs")}</span>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span className="inline-flex h-4 w-4 items-center justify-center rounded-[4px] text-[var(--fg-muted)] hover:bg-[var(--button-bg)]">
                      <AppIcon name="info" size={12} className="text-[var(--fg-muted)]" />
                    </span>
                  </TooltipTrigger>
                  <TooltipContent side="top" align="start" className="max-w-[320px] text-[12px] leading-5">
                    {t("勾选后，关闭标签页将自动保存再关闭。开启它会自动关闭“默认丢弃”。", "When checked, closing tabs will save before closing. Enabling this turns off default discard.")}
                  </TooltipContent>
                </Tooltip>
              </label>
              <label className="mt-2 flex items-center gap-2 text-[12px] text-[var(--fg)] cursor-pointer select-none">
                <input
                  type="checkbox"
                  checked={alwaysDiscardUnsavedOnClose}
                  onChange={(event) => setAlwaysDiscardUnsavedOnClose(event.target.checked)}
                  className="h-4 w-4 rounded-[4px] border border-[var(--border-color)] accent-[var(--accent)]"
                />
                <span>{t("关闭 Tab 时默认丢弃未保存修改", "Discard unsaved changes by default when closing tabs")}</span>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span className="inline-flex h-4 w-4 items-center justify-center rounded-[4px] text-[var(--fg-muted)] hover:bg-[var(--button-bg)]">
                      <AppIcon name="info" size={12} className="text-[var(--fg-muted)]" />
                    </span>
                  </TooltipTrigger>
                  <TooltipContent side="top" align="start" className="max-w-[320px] text-[12px] leading-5">
                    {t("勾选后，关闭标签页将不再提示是否保存。你可以随时在设置中修改此行为。", "When checked, closing tabs won't prompt for save. You can change this in Settings anytime.")}
                  </TooltipContent>
                </Tooltip>
              </label>
            </div>
            <div className="px-4 py-3 border-t border-[var(--border-subtle)] flex items-center justify-end gap-2">
              <button
                type="button"
                className="h-[32px] px-3 rounded-[8px] border border-[var(--button-border)] text-[12px] text-[var(--fg)] hover:bg-[var(--button-bg)]"
                onClick={handleConfirmCloseWithoutSave}
                disabled={closeConfirmSaving}
              >
                {t("不保存", "Don't save")}
              </button>
              <button
                type="button"
                className="h-[32px] px-3 rounded-[8px] border border-[var(--button-border)] text-[12px] text-[var(--fg)] hover:bg-[var(--button-bg)]"
                onClick={() => setCloseConfirmTabId(null)}
                disabled={closeConfirmSaving}
              >
                {t("取消", "Cancel")}
              </button>
              <button
                type="button"
                className="h-[32px] px-3 rounded-[8px] bg-[var(--accent)] text-white text-[12px] font-medium hover:opacity-95 disabled:opacity-60"
                onClick={() => void handleConfirmCloseWithSave()}
                disabled={closeConfirmSaving}
              >
                {closeConfirmSaving ? t("保存中...", "Saving...") : t("保存并关闭", "Save and close")}
              </button>
            </div>
          </div>
        </div>,
        document.body
      )}
    </div>
  )
}
