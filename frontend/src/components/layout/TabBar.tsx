import { useState, useEffect, useRef, useCallback } from "react"
import { createPortal } from "react-dom"
import { X, ChevronLeft, ChevronRight, Globe } from "lucide-react"
import { cn } from "@/lib/utils"
import { METHOD_COLORS, type HttpMethod } from "@/lib/constants"
import { useTabStore, getProjectActiveTabIdFromState, getProjectTabsFromState, type RequestTab } from "@/stores/tabStore"
import { useProjectStore } from "@/stores/projectStore"

interface TabContextMenuState {
  x: number
  y: number
  tabId: string
}

export function TabBar() {
  const tabs = useTabStore(getProjectTabsFromState)
  const activeTabId = useTabStore(getProjectActiveTabIdFromState)
  const setActiveTab = useTabStore((s) => s.setActiveTab)
  const removeTab = useTabStore((s) => s.removeTab)
  const closeOtherTabs = useTabStore((s) => s.closeOtherTabs)
  const closeAllTabs = useTabStore((s) => s.closeAllTabs)
  const currentProjectId = useProjectStore((s) => s.currentProjectId)
  const [contextMenu, setContextMenu] = useState<TabContextMenuState | null>(null)
  const [hasOverflow, setHasOverflow] = useState(false)
  const scrollContainerRef = useRef<HTMLDivElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)

  const checkOverflow = useCallback(() => {
    const container = scrollContainerRef.current
    if (container) setHasOverflow(container.scrollWidth > container.clientWidth)
  }, [])

  useEffect(() => {
    checkOverflow()
    window.addEventListener("resize", checkOverflow)
    return () => window.removeEventListener("resize", checkOverflow)
  }, [checkOverflow, tabs.length])

  useEffect(() => {
    if (!activeTabId || !scrollContainerRef.current) return
    const activeEl = scrollContainerRef.current.querySelector(`[data-tab-id="${activeTabId}"]`)
    if (activeEl) activeEl.scrollIntoView({ behavior: "auto", block: "nearest", inline: "nearest" })
  }, [activeTabId])

  useEffect(() => {
    if (!contextMenu) return
    const handler = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) setContextMenu(null)
    }
    document.addEventListener("mousedown", handler)
    return () => document.removeEventListener("mousedown", handler)
  }, [contextMenu])

  if (tabs.length === 0) return null

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
            <ChevronLeft className="h-3 w-3" />
          </button>
          <button
            className="h-full px-1 flex items-center justify-center text-[var(--fg-secondary)] hover:bg-[var(--tab-hover-bg)] transition-colors"
            onClick={() => scrollContainerRef.current?.scrollBy({ left: 200, behavior: "smooth" })}
          >
            <ChevronRight className="h-3 w-3" />
          </button>
        </div>
      )}

      <div
        ref={scrollContainerRef}
        className="flex items-end flex-1 min-w-0 overflow-x-hidden"
      >
        {tabs.map((tab) => {
          const isActive = tab.id === activeTabId
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
                if (e.button === 0) { e.preventDefault(); setActiveTab(tab.id) }
                if (e.button === 1 && tab.closable) { e.preventDefault(); removeTab(tab.id) }
              }}
              onContextMenu={(e) => { e.preventDefault(); setContextMenu({ x: e.clientX, y: e.clientY, tabId: tab.id }) }}
            >
              <span className={cn(
                "text-[9px] font-mono font-bold uppercase flex-shrink-0",
                METHOD_COLORS[tab.request.method as HttpMethod] || "text-[var(--fg-muted)]"
              )}>
                {tab.request.method?.substring(0, 3) || "GET"}
              </span>
              <span className="truncate max-w-[100px]">{tab.title}</span>
              {tab.dirty && (
                <span className="w-1 h-1 rounded-full bg-[var(--accent)] flex-shrink-0" />
              )}
              {tab.closable && (
                <button
                  className={cn(
                    "flex items-center justify-center flex-shrink-0 transition-opacity",
                    "opacity-0 group-hover:opacity-100",
                    "text-[var(--fg-muted)] hover:text-[var(--fg)]"
                  )}
                  onClick={(e) => { e.stopPropagation(); removeTab(tab.id) }}
                >
                  <X className="h-3 w-3" />
                </button>
              )}
            </div>
          )
        })}
      </div>

      {contextMenu && createPortal(
        <div
          ref={menuRef}
          className={cn(
            "fixed z-[260] min-w-[140px] py-0.5 rounded-[var(--radius-menu)] shadow-lg border",
            "bg-[var(--surface-elevated)] border-[var(--border-color)]"
          )}
          style={{ left: contextMenu.x, top: contextMenu.y }}
        >
          {contextTab?.closable && (
            <button
              className="w-full px-2.5 py-1 text-[length:var(--size-font-2xs)] text-left hover:bg-[var(--sidebar-hover)] text-[var(--fg)]"
              onClick={() => { removeTab(contextMenu.tabId); setContextMenu(null) }}
            >
              关闭
            </button>
          )}
          <button
            className="w-full px-2.5 py-1 text-[length:var(--size-font-2xs)] text-left hover:bg-[var(--sidebar-hover)] text-[var(--fg)] disabled:opacity-40"
            disabled={closableTabs.length <= 1}
            onClick={() => { closeOtherTabs(contextMenu.tabId); setContextMenu(null) }}
          >
            关闭其他
          </button>
          <div className="h-px bg-[var(--border-subtle)] my-0.5" />
          <button
            className="w-full px-2.5 py-1 text-[length:var(--size-font-2xs)] text-left hover:bg-[var(--sidebar-hover)] text-[var(--fg)] disabled:opacity-40"
            disabled={closableTabs.length === 0}
            onClick={() => { closeAllTabs(currentProjectId || undefined); setContextMenu(null) }}
          >
            关闭全部
          </button>
        </div>,
        document.body
      )}
    </div>
  )
}
