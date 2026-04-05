import { useState, useRef, useEffect, useMemo, useCallback } from "react"
import { createPortal } from "react-dom"
import {
  Plus, FolderPlus, Search, X, Folder, FolderOpen, ChevronRight,
  Trash2, Pencil, Copy, MoreHorizontal,
} from "lucide-react"
import { cn } from "@/lib/utils"
import { METHOD_COLORS, type HttpMethod } from "@/lib/constants"
import { useProjectStore } from "@/stores/projectStore"
import { useTabStore } from "@/stores/tabStore"
import { useUIStore } from "@/stores/uiStore"
import { HistoryPanel } from "@/components/business/history/HistoryPanel"
import { EnvironmentManager } from "@/components/business/environment/EnvironmentManager"
import type { model } from "../../../wailsjs/go/models"

type SidebarTab = "requests" | "history" | "environments"

interface ContextMenuState {
  x: number
  y: number
  type: "folder" | "request"
  id: string
  name: string
}

function convertRequestToData(request: model.RequestItem) {
  return {
    id: request.id,
    name: request.name,
    method: request.method as HttpMethod,
    url: request.url,
    params: (request.params ?? []).map((p: { key: string; value: string }) => ({
      id: crypto.randomUUID(), key: p.key, value: p.value, enabled: true,
    })),
    headers: (request.headers ?? []).map((h: { key: string; value: string }) => ({
      id: crypto.randomUUID(), key: h.key, value: h.value, enabled: true,
    })),
    body: request.body
      ? {
          type: request.body.type as "none" | "raw" | "json" | "form-urlencoded",
          raw: request.body.raw,
          json: request.body.json,
          formUrlEncoded: (request.body.formUrlEncoded ?? []).map((f: { key: string; value: string }) => ({
            id: crypto.randomUUID(), key: f.key, value: f.value, enabled: true,
          })),
        }
      : { type: "none" as const },
    auth: request.auth
      ? {
          type: request.auth.type as "none" | "basic" | "bearer" | "api-key",
          basic: request.auth.basic ? { username: request.auth.basic.username, password: request.auth.basic.password } : undefined,
          bearer: request.auth.bearer ? { token: request.auth.bearer.token } : undefined,
          apiKey: request.auth.apiKey
            ? { key: request.auth.apiKey.key, value: request.auth.apiKey.value, addTo: (request.auth.apiKey.addTo as "header" | "query") || "header" }
            : undefined,
        }
      : { type: "none" as const },
    folderId: request.folderId,
    projectId: request.projectId,
    createdAt: request.createdAt,
    updatedAt: request.updatedAt,
  }
}

export function Sidebar() {
  const { sidebarWidth, setSidebarWidth, sidebarCollapsed } = useUIStore()
  const {
    currentProjectId, projects, folders, requests,
    createFolder, createRequest, deleteFolder, deleteRequest, renameFolder,
    selectProject, loadProjects, createProject,
  } = useProjectStore()
  const { openRequestTab, tabs, activeTabId } = useTabStore()
  const [activeTab, setActiveTab] = useState<SidebarTab>("requests")
  const [searchQuery, setSearchQuery] = useState("")
  const [expandedFolders, setExpandedFolders] = useState<Set<string>>(new Set())
  const [contextMenu, setContextMenu] = useState<ContextMenuState | null>(null)
  const [renamingId, setRenamingId] = useState<string | null>(null)
  const [renameValue, setRenameValue] = useState("")
  const [showNewProject, setShowNewProject] = useState(false)
  const [newProjectName, setNewProjectName] = useState("")
  const menuRef = useRef<HTMLDivElement>(null)
  const resizingRef = useRef(false)
  const searchInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    loadProjects()
  }, [loadProjects])

  useEffect(() => {
    if (!contextMenu) return
    const handler = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) setContextMenu(null)
    }
    document.addEventListener("mousedown", handler)
    return () => document.removeEventListener("mousedown", handler)
  }, [contextMenu])

  const handleSidebarResizeStart = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    resizingRef.current = true
    const startX = e.clientX
    const startW = sidebarWidth
    const onMove = (ev: MouseEvent) => {
      if (!resizingRef.current) return
      setSidebarWidth(Math.max(180, Math.min(500, startW + ev.clientX - startX)))
    }
    const onUp = () => {
      resizingRef.current = false
      document.removeEventListener("mousemove", onMove)
      document.removeEventListener("mouseup", onUp)
    }
    document.addEventListener("mousemove", onMove)
    document.addEventListener("mouseup", onUp)
  }, [sidebarWidth, setSidebarWidth])

  const toggleFolder = (folderId: string) => {
    setExpandedFolders((prev) => {
      const next = new Set(prev)
      if (next.has(folderId)) next.delete(folderId)
      else next.add(folderId)
      return next
    })
  }

  const activeTabRequestId = tabs.find((t) => t.id === activeTabId)?.requestId

  const filteredRequests = useMemo(() => {
    if (!searchQuery.trim()) return requests
    const q = searchQuery.toLowerCase()
    return requests.filter((r) =>
      r.name.toLowerCase().includes(q) || r.url?.toLowerCase().includes(q)
    )
  }, [requests, searchQuery])

  const rootFolders = folders.filter((f) => !f.parentId || f.parentId === "")
  const rootRequests = filteredRequests.filter((r) => !r.folderId || r.folderId === "")

  const getChildFolders = (parentId: string) => folders.filter((f) => f.parentId === parentId)
  const getChildRequests = (folderId: string) => filteredRequests.filter((r) => r.folderId === folderId)

  const handleOpenRequest = (request: model.RequestItem) => {
    if (!currentProjectId) return
    openRequestTab(currentProjectId, convertRequestToData(request))
  }

  const handleContextMenu = (e: React.MouseEvent, type: "folder" | "request", id: string, name: string) => {
    e.preventDefault()
    e.stopPropagation()
    setContextMenu({ x: e.clientX, y: e.clientY, type, id, name })
  }

  const handleNewRequest = async (folderId: string = "") => {
    if (!currentProjectId) return
    const req = await createRequest(folderId, "New Request")
    if (req) handleOpenRequest(req)
  }

  const handleNewFolder = async () => {
    if (!currentProjectId) return
    await createFolder("", "New Folder")
  }

  const handleCreateProject = async () => {
    if (!newProjectName.trim()) return
    await createProject(newProjectName.trim())
    setNewProjectName("")
    setShowNewProject(false)
    const { projects: updatedProjects } = useProjectStore.getState()
    const newP = updatedProjects[updatedProjects.length - 1]
    if (newP) await selectProject(newP.id)
  }

  const startRename = (id: string, currentName: string) => {
    setRenamingId(id)
    setRenameValue(currentName)
    setContextMenu(null)
  }

  const handleRenameSubmit = async () => {
    if (!renamingId || !renameValue.trim()) {
      setRenamingId(null)
      return
    }
    if (contextMenu?.type === "folder" || folders.find((f) => f.id === renamingId)) {
      await renameFolder(renamingId, renameValue.trim())
    }
    setRenamingId(null)
  }

  const sidebarTabs: { key: SidebarTab; label: string }[] = [
    { key: "requests", label: "请求" },
    { key: "history", label: "历史" },
    { key: "environments", label: "环境" },
  ]

  if (sidebarCollapsed) return null

  const renderFolderNode = (folder: model.Folder, depth: number = 0) => {
    const isExpanded = expandedFolders.has(folder.id)
    const childFolders = getChildFolders(folder.id)
    const childRequests = getChildRequests(folder.id)

    return (
      <div key={folder.id}>
        <div
          className={cn(
            "flex items-center h-[24px] px-2 rounded-[var(--radius-btn)] cursor-pointer group",
            "hover:bg-[var(--sidebar-hover)]"
          )}
          style={{ paddingLeft: `${8 + depth * 16}px` }}
          onClick={() => toggleFolder(folder.id)}
          onContextMenu={(e) => handleContextMenu(e, "folder", folder.id, folder.name)}
        >
          <ChevronRight
            className={cn(
              "h-3 w-3 mr-1 flex-shrink-0 transition-transform text-[var(--fg-muted)]",
              isExpanded && "rotate-90"
            )}
          />
          {isExpanded ? (
            <FolderOpen className="h-3.5 w-3.5 mr-1.5 flex-shrink-0 text-[var(--fg-muted)]" />
          ) : (
            <Folder className="h-3.5 w-3.5 mr-1.5 flex-shrink-0 text-[var(--fg-muted)]" />
          )}
          {renamingId === folder.id ? (
            <input
              className="flex-1 bg-transparent text-[length:var(--size-font-2xs)] text-[var(--fg)] border-b border-[var(--accent)] outline-none"
              value={renameValue}
              onChange={(e) => setRenameValue(e.target.value)}
              onKeyDown={(e) => { if (e.key === "Enter") handleRenameSubmit(); if (e.key === "Escape") setRenamingId(null) }}
              onBlur={handleRenameSubmit}
              autoFocus
            />
          ) : (
            <span className="text-[length:var(--size-font-2xs)] truncate flex-1 text-[var(--sidebar-fg)]">{folder.name}</span>
          )}
          <button
            className="opacity-0 group-hover:opacity-100 h-4 w-4 flex items-center justify-center rounded-[var(--radius-sm)] hover:bg-[var(--sidebar-hover)] transition-opacity"
            onClick={(e) => { e.stopPropagation(); handleNewRequest(folder.id) }}
            title="在此文件夹新建请求"
          >
            <Plus className="h-3 w-3 text-[var(--fg-muted)]" />
          </button>
        </div>
        {isExpanded && (
          <div>
            {childFolders.map((cf) => renderFolderNode(cf, depth + 1))}
            {childRequests.map((req) => renderRequestNode(req, depth + 1))}
          </div>
        )}
      </div>
    )
  }

  const renderRequestNode = (request: model.RequestItem, depth: number = 0) => {
    const isSelected = activeTabRequestId === request.id
    return (
      <div
        key={request.id}
        className={cn(
          "flex items-center h-[24px] px-2 rounded-[var(--radius-btn)] cursor-pointer group",
          isSelected ? "bg-[var(--sidebar-active)] text-[var(--sidebar-accent)]" : "hover:bg-[var(--sidebar-hover)]"
        )}
        style={{ paddingLeft: `${8 + depth * 16 + 16}px` }}
        onPointerDown={(e) => {
          if (e.button !== 0) return
          e.preventDefault()
          handleOpenRequest(request)
        }}
        onContextMenu={(e) => handleContextMenu(e, "request", request.id, request.name)}
      >
        <span className={cn(
          "text-[9px] font-mono font-bold mr-1.5 w-[32px] text-right flex-shrink-0 uppercase",
          METHOD_COLORS[request.method as HttpMethod] || "text-[var(--fg-muted)]"
        )}>
          {request.method?.substring(0, 3) || "GET"}
        </span>
        <span className={cn(
          "text-[length:var(--size-font-2xs)] truncate flex-1",
          isSelected ? "text-[var(--sidebar-accent)] font-medium" : "text-[var(--sidebar-fg)]"
        )} title={request.name}>
          {request.name}
        </span>
      </div>
    )
  }

  return (
    <div
      className={cn(
        "relative flex min-h-0 flex-col overflow-hidden border-r border-[var(--sidebar-border)] bg-[var(--sidebar-bg)]"
      )}
      style={{ width: sidebarWidth }}
    >
      {/* 自绘拖拽条 */}
      <div
        className="absolute right-0 top-0 h-full w-[5px] cursor-col-resize z-10 group"
        onMouseDown={handleSidebarResizeStart}
      >
        <div className="absolute inset-y-0 right-1/2 translate-x-1/2 w-px bg-transparent group-hover:bg-[var(--accent)]/40 transition-colors duration-200" />
      </div>

      {/* 项目选择区 */}
      <div className="flex items-center h-[28px] px-2 gap-1 border-b border-[var(--sidebar-border)] flex-shrink-0">
        <Select
          currentProjectId={currentProjectId}
          projects={projects}
          onSelect={(id) => selectProject(id)}
        />
        <button
          className="h-5 w-5 flex items-center justify-center rounded-[var(--radius-sm)] hover:bg-[var(--sidebar-hover)] transition-colors flex-shrink-0"
          onClick={() => setShowNewProject(true)}
          title="新建项目"
        >
          <Plus className="h-3 w-3 text-[var(--fg-muted)]" />
        </button>
      </div>

      {showNewProject && (
        <div className="px-2 py-1.5 border-b border-[var(--sidebar-border)] flex-shrink-0">
          <input
            className="w-full h-[var(--size-btn-sm)] px-2 text-[length:var(--size-font-2xs)] rounded-[var(--radius-input)] border border-[var(--border-color)] bg-[var(--surface)] text-[var(--fg)] outline-none focus:border-[var(--accent)]"
            placeholder="项目名称..."
            value={newProjectName}
            onChange={(e) => setNewProjectName(e.target.value)}
            onKeyDown={(e) => { if (e.key === "Enter") handleCreateProject(); if (e.key === "Escape") setShowNewProject(false) }}
            autoFocus
          />
        </div>
      )}

      {/* Tab 切换栏 */}
      {currentProjectId && (
        <div className="flex items-center border-b border-[var(--sidebar-border)] flex-shrink-0">
          {sidebarTabs.map((tab) => (
            <button
              key={tab.key}
              className={cn(
                "flex-1 text-center font-medium transition-colors py-1.5",
                "text-[length:var(--size-font-2xs)]",
                activeTab === tab.key
                  ? "text-[var(--fg)] border-b-2 border-[var(--accent)]"
                  : "text-[var(--fg-muted)] hover:text-[var(--fg-secondary)] border-b-2 border-transparent"
              )}
              onClick={() => setActiveTab(tab.key)}
            >
              {tab.label}
            </button>
          ))}
        </div>
      )}

      {/* 搜索 + 操作栏 */}
      {currentProjectId && activeTab === "requests" && (
        <div className="px-2 pt-1.5 pb-1 flex items-center gap-1 flex-shrink-0">
          <div className="relative flex-1">
            <Search className="absolute left-1.5 top-1/2 -translate-y-1/2 h-3 w-3 text-[var(--fg-muted)]" />
            <input
              ref={searchInputRef}
              type="text"
              className={cn(
                "w-full h-[var(--size-btn-sm)] pl-6 pr-6 text-[length:var(--size-font-2xs)] rounded-[var(--radius-input)]",
                "border border-[var(--border-color)] bg-[var(--surface)] text-[var(--fg)]",
                "placeholder:text-[var(--fg-muted)] focus:outline-none focus:border-[var(--accent)]"
              )}
              placeholder="搜索请求..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
            {searchQuery && (
              <button
                className="absolute right-1 top-1/2 -translate-y-1/2 h-4 w-4 flex items-center justify-center"
                onClick={() => setSearchQuery("")}
              >
                <X className="h-2.5 w-2.5 text-[var(--fg-muted)]" />
              </button>
            )}
          </div>
          <button
            className="h-[var(--size-btn-sm)] w-[var(--size-btn-sm)] flex items-center justify-center rounded-[var(--radius-btn)] hover:bg-[var(--sidebar-hover)] transition-colors flex-shrink-0"
            onClick={handleNewFolder}
            title="新建文件夹"
          >
            <FolderPlus className="h-3.5 w-3.5 text-[var(--fg-secondary)]" />
          </button>
          <button
            className="h-[var(--size-btn-sm)] w-[var(--size-btn-sm)] flex items-center justify-center rounded-[var(--radius-btn)] hover:bg-[var(--sidebar-hover)] transition-colors flex-shrink-0"
            onClick={() => handleNewRequest()}
            title="新建请求 (⌘N)"
          >
            <Plus className="h-3.5 w-3.5 text-[var(--fg-secondary)]" />
          </button>
        </div>
      )}

      {/* 主内容区 */}
      <div className="flex-1 min-h-0 overflow-y-auto py-0.5 px-1">
        {!currentProjectId ? (
          <div className="flex items-center justify-center h-full">
            <div className="text-center px-4">
              <p className="text-[length:var(--size-font-xs)] text-[var(--fg-secondary)] font-medium">无项目</p>
              <p className="text-2xs text-[var(--fg-muted)] mt-1">创建一个项目开始使用</p>
              <button
                className="mt-3 px-3 py-1 text-[length:var(--size-font-2xs)] rounded-[var(--radius-btn)] bg-[var(--accent)] text-[var(--accent-fg)] hover:bg-[var(--accent-hover)] transition-colors"
                onClick={() => setShowNewProject(true)}
              >
                新建项目
              </button>
            </div>
          </div>
        ) : activeTab === "requests" ? (
          <div>
            {rootFolders.map((f) => renderFolderNode(f))}
            {rootRequests.map((r) => renderRequestNode(r))}
            {rootFolders.length === 0 && rootRequests.length === 0 && (
              <div className="text-center py-8">
                <p className="text-2xs text-[var(--fg-muted)]">
                  {searchQuery ? "无匹配结果" : "暂无请求"}
                </p>
              </div>
            )}
          </div>
        ) : activeTab === "history" ? (
          <HistoryPanel />
        ) : (
          <EnvironmentManager />
        )}
      </div>

      {/* 右键菜单 */}
      {contextMenu && createPortal(
        <div
          ref={menuRef}
          className={cn(
            "fixed z-[100] min-w-[160px] py-1 rounded-[var(--radius-menu)] shadow-lg border animate-fade-in",
            "bg-[var(--surface-elevated)] border-[var(--border-color)]"
          )}
          style={{ left: contextMenu.x, top: contextMenu.y }}
        >
          <button
            className="w-full px-2.5 py-1 text-[length:var(--size-font-2xs)] text-left hover:bg-[var(--sidebar-hover)] text-[var(--fg)] flex items-center gap-2"
            onClick={() => startRename(contextMenu.id, contextMenu.name)}
          >
            <Pencil className="h-3 w-3" /> 重命名
          </button>
          {contextMenu.type === "folder" && (
            <button
              className="w-full px-2.5 py-1 text-[length:var(--size-font-2xs)] text-left hover:bg-[var(--sidebar-hover)] text-[var(--fg)] flex items-center gap-2"
              onClick={() => { handleNewRequest(contextMenu.id); setContextMenu(null) }}
            >
              <Plus className="h-3 w-3" /> 在此新建请求
            </button>
          )}
          <div className="h-px bg-[var(--border-subtle)] my-0.5" />
          <button
            className="w-full px-2.5 py-1 text-[length:var(--size-font-2xs)] text-left hover:bg-[var(--sidebar-hover)] text-[var(--danger)] flex items-center gap-2"
            onClick={async () => {
              if (contextMenu.type === "folder") await deleteFolder(contextMenu.id)
              else await deleteRequest(contextMenu.id)
              setContextMenu(null)
            }}
          >
            <Trash2 className="h-3 w-3" /> 删除
          </button>
        </div>,
        document.body
      )}
    </div>
  )
}

function Select({ currentProjectId, projects, onSelect }: {
  currentProjectId: string | null
  projects: model.Project[]
  onSelect: (id: string) => void
}) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  const current = projects.find((p) => p.id === currentProjectId)

  useEffect(() => {
    if (!open) return
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener("mousedown", handler)
    return () => document.removeEventListener("mousedown", handler)
  }, [open])

  return (
    <div className="relative flex-1" ref={ref}>
      <button
        className="w-full h-5 flex items-center px-1.5 rounded-[var(--radius-sm)] hover:bg-[var(--sidebar-hover)] transition-colors text-left"
        onClick={() => setOpen(!open)}
      >
        <span className="text-[length:var(--size-font-2xs)] font-medium text-[var(--fg)] truncate flex-1">
          {current?.name || "选择项目"}
        </span>
        <ChevronRight className={cn("h-2.5 w-2.5 text-[var(--fg-muted)] transition-transform", open && "rotate-90")} />
      </button>
      {open && projects.length > 0 && (
        <div className={cn(
          "absolute left-0 top-full mt-0.5 z-50 min-w-[160px] py-0.5 rounded-[var(--radius-menu)] shadow-lg border",
          "bg-[var(--surface-elevated)] border-[var(--border-color)]"
        )}>
          {projects.map((p) => (
            <button
              key={p.id}
              className={cn(
                "w-full px-2.5 py-1 text-[length:var(--size-font-2xs)] text-left transition-colors",
                p.id === currentProjectId
                  ? "bg-[var(--sidebar-active)] text-[var(--accent)] font-medium"
                  : "text-[var(--fg)] hover:bg-[var(--sidebar-hover)]"
              )}
              onClick={() => { onSelect(p.id); setOpen(false) }}
            >
              {p.name}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
