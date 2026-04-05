import { useState, useRef, useEffect, useMemo, useCallback, useLayoutEffect } from "react"
import { createPortal } from "react-dom"
import { AppIcon } from "@/components/ui/icon"
import { cn } from "@/lib/utils"
import { METHOD_COLORS, type HttpMethod } from "@/lib/constants"
import { useProjectStore } from "@/stores/projectStore"
import { useTabStore, getProjectActiveTabIdFromState, getProjectTabsFromState } from "@/stores/tabStore"
import { useUIStore } from "@/stores/uiStore"
import { HistoryPanel } from "@/components/business/history/HistoryPanel"
import { EnvironmentManager } from "@/components/business/environment/EnvironmentManager"
import type { model } from "../../../wailsjs/go/models"
import { ImportCurl, OpenFileDialogJSON, SaveTextFile } from "../../../wailsjs/go/main/App"

type SidebarTab = "requests" | "history" | "environments"
const TREE_INDENT_BASE = 8
const TREE_INDENT_STEP = 12
const REQUEST_METHOD_SPACER = 12
type CollectionNodeType = "folder" | "request"
type DropPosition = "before" | "after" | "inside"
type ImportMode = "file" | "curl"
type DetectedImportType = "postman" | "swagger" | "unknown"

interface DropdownMenuState {
  x: number
  y: number
  anchorX: number
  anchorY: number
  type: "folder" | "request"
  id: string
  name: string
}

interface SelectedNodeState {
  type: "folder" | "request"
  id: string
  name: string
}

interface DraggingState {
  id: string
  type: CollectionNodeType
}

interface DropIndicator {
  targetId: string | null
  targetType: CollectionNodeType | "root"
  position: DropPosition
}

const AUTO_EXPAND_DELAY = 600

function detectImportType(content: string): {
  type: DetectedImportType
  title: string
  description: string
} {
  const trimmed = content.trim()
  if (!trimmed) {
    return { type: "unknown", title: "未选择文件", description: "请先选择导入文件" }
  }

  try {
    const parsed = JSON.parse(trimmed) as Record<string, unknown>
    if (parsed && typeof parsed === "object") {
      if ("info" in parsed && "item" in parsed) {
        return {
          type: "postman",
          title: "Postman Collection",
          description: "检测到 Postman v2.x 集合结构",
        }
      }
      if ("paths" in parsed && ("openapi" in parsed || "swagger" in parsed)) {
        return {
          type: "swagger",
          title: "OpenAPI / Swagger",
          description: "检测到 OpenAPI/Swagger 结构",
        }
      }
      if ("paths" in parsed) {
        return {
          type: "swagger",
          title: "OpenAPI / Swagger",
          description: "检测到 paths 字段，可按 OpenAPI 导入",
        }
      }
    }
  } catch {
    const normalized = trimmed.toLowerCase()
    if (normalized.includes("openapi:") || normalized.includes("swagger:")) {
      return {
        type: "swagger",
        title: "OpenAPI / Swagger (YAML)",
        description: "检测到 YAML OpenAPI/Swagger 结构",
      }
    }
  }

  return {
    type: "unknown",
    title: "未识别格式",
    description: "支持 Postman Collection / OpenAPI / Swagger（JSON 或 YAML）",
  }
}

function buildCurlCommand(request: model.RequestItem): string {
  const parts: string[] = ["curl"]
  const method = (request.method || "GET").toUpperCase()
  if (method !== "GET") parts.push(`-X ${method}`)
  const url = request.url || ""
  const params = request.params?.filter((p) => p.key)
  let fullUrl = url
  if (params && params.length > 0) {
    const qs = params.map((p) => `${encodeURIComponent(p.key)}=${encodeURIComponent(p.value || "")}`).join("&")
    fullUrl += (url.includes("?") ? "&" : "?") + qs
  }
  parts.push(`'${fullUrl}'`)
  const headers = request.headers?.filter((h) => h.key)
  if (headers) {
    for (const h of headers) {
      parts.push(`-H '${h.key}: ${h.value || ""}'`)
    }
  }
  if (request.auth) {
    if (request.auth.type === "bearer" && request.auth.bearer?.token) {
      parts.push(`-H 'Authorization: Bearer ${request.auth.bearer.token}'`)
    } else if (request.auth.type === "basic" && request.auth.basic) {
      parts.push(`-u '${request.auth.basic.username || ""}:${request.auth.basic.password || ""}'`)
    } else if (request.auth.type === "api-key" && request.auth.apiKey) {
      if ((request.auth.apiKey.addTo || "header") === "header") {
        parts.push(`-H '${request.auth.apiKey.key}: ${request.auth.apiKey.value || ""}'`)
      }
    }
  }
  if (request.body) {
    if (request.body.type === "json" && request.body.json) {
      parts.push(`-H 'Content-Type: application/json'`)
      parts.push(`-d '${request.body.json}'`)
    } else if (request.body.type === "raw" && request.body.raw) {
      parts.push(`-d '${request.body.raw}'`)
    } else if (request.body.type === "form-urlencoded" && request.body.formUrlEncoded) {
      parts.push(`-H 'Content-Type: application/x-www-form-urlencoded'`)
      const formData = request.body.formUrlEncoded
        .filter((f: { key: string }) => f.key)
        .map((f: { key: string; value: string }) => `${encodeURIComponent(f.key)}=${encodeURIComponent(f.value || "")}`)
        .join("&")
      if (formData) parts.push(`-d '${formData}'`)
    }
  }
  return parts.join(" \\\n  ")
}

function normalizeFileName(name: string): string {
  const trimmed = name.trim()
  if (!trimmed) return "request"
  return trimmed
    .replace(/[\\/:*?"<>|]/g, "-")
    .replace(/\s+/g, "-")
    .replace(/-+/g, "-")
    .replace(/^-|-$/g, "")
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

const MENU_BTN = "h-5 w-5 flex items-center justify-center rounded-[var(--radius-sm)] hover:bg-[var(--sidebar-hover)] transition-all"
const MENU_ITEM = "w-full whitespace-nowrap px-2.5 py-1.5 rounded-[7px] text-[length:var(--size-font-2xs)] text-left hover:bg-[var(--sidebar-hover)] text-[var(--fg)] flex items-center gap-2"
const MENU_ITEM_HOTKEY = "text-[10px] text-[var(--fg-muted)] font-mono ml-4"

export function Sidebar() {
  const { sidebarWidth, setSidebarWidth, sidebarCollapsed, editingEnvironmentId, setEditingEnvironmentId } = useUIStore()
  const {
    currentProjectId,
    projects,
    folders,
    requests,
    treeNodes,
    createFolder,
    createRequest,
    deleteFolder,
    deleteRequest,
    renameFolder,
    renameRequest,
    duplicateRequest,
    duplicateFolder,
    moveCollectionNode,
    exportProjectJSON,
    importFromFile,
    saveRequestToBackend,
  } = useProjectStore()
  const { openRequestTab } = useTabStore()
  const tabs = useTabStore(getProjectTabsFromState)
  const activeTabId = useTabStore(getProjectActiveTabIdFromState)
  const [activeTab, setActiveTab] = useState<SidebarTab>("requests")
  const [searchQuery, setSearchQuery] = useState("")
  const [expandedFolders, setExpandedFolders] = useState<Set<string>>(new Set())
  const [dropdownMenu, setDropdownMenu] = useState<DropdownMenuState | null>(null)
  const [selectedNode, setSelectedNode] = useState<SelectedNodeState | null>(null)
  const [renamingId, setRenamingId] = useState<string | null>(null)
  const [renamingType, setRenamingType] = useState<"folder" | "request">("folder")
  const [renameValue, setRenameValue] = useState("")
  const [dragging, setDragging] = useState<DraggingState | null>(null)
  const [dropIndicator, setDropIndicator] = useState<DropIndicator | null>(null)
  const [showImportDialog, setShowImportDialog] = useState(false)
  const [importMode, setImportMode] = useState<ImportMode>("file")
  const [importContent, setImportContent] = useState("")
  const [detectedImport, setDetectedImport] = useState(() => detectImportType(""))
  const [importLoading, setImportLoading] = useState(false)
  const [importProgress, setImportProgress] = useState(0)
  const [importError, setImportError] = useState("")
  const [importSuccess, setImportSuccess] = useState("")
  const [curlImportInput, setCurlImportInput] = useState("")
  const menuRef = useRef<HTMLDivElement>(null)
  const resizingRef = useRef(false)
  const searchInputRef = useRef<HTMLInputElement>(null)
  const autoExpandTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const autoExpandTargetRef = useRef<string | null>(null)
  const importProgressTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const isDragging = dragging !== null
  const isSearching = searchQuery.trim().length > 0

  useEffect(() => {
    if (!dropdownMenu) return
    const handler = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) setDropdownMenu(null)
    }
    document.addEventListener("mousedown", handler)
    return () => document.removeEventListener("mousedown", handler)
  }, [dropdownMenu])

  useLayoutEffect(() => {
    if (!dropdownMenu || !menuRef.current) return

    const menuEl = menuRef.current
    const menuWidth = menuEl.offsetWidth
    const menuHeight = menuEl.offsetHeight
    const viewportPadding = 8

    let nextX = dropdownMenu.anchorX + 2
    if (nextX + menuWidth > window.innerWidth - viewportPadding) {
      nextX = Math.max(viewportPadding, window.innerWidth - menuWidth - viewportPadding)
    }

    const spaceBelow = window.innerHeight - dropdownMenu.anchorY - viewportPadding
    const shouldOpenUp = spaceBelow < menuHeight + 4
    let nextY = shouldOpenUp ? dropdownMenu.anchorY - menuHeight - 4 : dropdownMenu.anchorY + 2
    if (nextY < viewportPadding) {
      nextY = viewportPadding
    }
    if (nextY + menuHeight > window.innerHeight - viewportPadding) {
      nextY = Math.max(viewportPadding, window.innerHeight - menuHeight - viewportPadding)
    }

    setDropdownMenu((prev) => {
      if (!prev) return prev
      if (prev.x === nextX && prev.y === nextY) return prev
      return { ...prev, x: nextX, y: nextY }
    })
  }, [dropdownMenu])

  useEffect(() => {
    return () => {
      if (autoExpandTimerRef.current) clearTimeout(autoExpandTimerRef.current)
      if (importProgressTimerRef.current) clearInterval(importProgressTimerRef.current)
    }
  }, [])

  useEffect(() => {
    if (editingEnvironmentId) {
      setActiveTab("environments")
    }
  }, [editingEnvironmentId])

  useEffect(() => {
    if (!showImportDialog) return
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault()
        if (!importLoading) {
          setShowImportDialog(false)
        }
      }
    }
    window.addEventListener("keydown", onKeyDown)
    return () => window.removeEventListener("keydown", onKeyDown)
  }, [importLoading, showImportDialog])

  useEffect(() => {
    const handler = () => {
      openImportDialog()
    }
    window.addEventListener("minipost:open-import", handler as EventListener)
    return () => window.removeEventListener("minipost:open-import", handler as EventListener)
  }, [])

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

  const folderMap = useMemo(() => new Map(folders.map((folder) => [folder.id, folder])), [folders])
  const requestMap = useMemo(() => new Map(requests.map((request) => [request.id, request])), [requests])

  const filteredNodeIds = useMemo(() => {
    if (!isSearching) return null
    const q = searchQuery.toLowerCase()
    const matchedRequestIds = new Set(
      requests
        .filter((request) => request.name.toLowerCase().includes(q) || request.url?.toLowerCase().includes(q))
        .map((request) => request.id)
    )
    const matchedFolderIds = new Set<string>()
    treeNodes.forEach((node) => {
      if (node.nodeType === "request" && matchedRequestIds.has(node.nodeId)) {
        let currentParentId = node.parentFolderId
        while (currentParentId) {
          matchedFolderIds.add(currentParentId)
          currentParentId = folderMap.get(currentParentId)?.parentId || ""
        }
      }
    })
    return new Set<string>([...matchedRequestIds, ...matchedFolderIds])
  }, [searchQuery, isSearching, requests, treeNodes, folderMap])

  const sortedTreeNodes = useMemo(
    () => [...treeNodes].sort((a, b) => a.sortOrder - b.sortOrder || a.nodeId.localeCompare(b.nodeId)),
    [treeNodes]
  )

  const getChildren = useCallback((parentFolderId: string) => {
    const children = sortedTreeNodes.filter((node) => (node.parentFolderId || "") === parentFolderId)
    if (!filteredNodeIds) return children
    return children.filter((node) => filteredNodeIds.has(node.nodeId))
  }, [sortedTreeNodes, filteredNodeIds])

  const isDescendantFolder = useCallback((folderId: string, potentialParentId: string) => {
    if (!potentialParentId) return false
    let currentId: string | undefined = potentialParentId
    while (currentId) {
      if (currentId === folderId) return true
      currentId = folderMap.get(currentId)?.parentId
    }
    return false
  }, [folderMap])

  const handleOpenRequest = (request: model.RequestItem) => {
    if (!currentProjectId) return
    setEditingEnvironmentId(null)
    openRequestTab(currentProjectId, convertRequestToData(request))
  }

  const handleNewRequest = async (folderId: string = "") => {
    if (!currentProjectId) return
    const req = await createRequest(folderId, "New Request")
    if (req) handleOpenRequest(req)
  }

  const handleNewFolder = async (parentId: string = "") => {
    if (!currentProjectId) return
    await createFolder(parentId, "New Folder")
    if (parentId) {
      setExpandedFolders((prev) => new Set(prev).add(parentId))
    }
  }

  const startRename = (id: string, type: "folder" | "request", currentName: string) => {
    setSelectedNode({ id, type, name: currentName })
    setRenamingId(id)
    setRenamingType(type)
    setRenameValue(currentName)
    setDropdownMenu(null)
  }

  const handleRenameSubmit = async () => {
    if (!renamingId || !renameValue.trim()) {
      setRenamingId(null)
      return
    }
    if (renamingType === "folder") {
      await renameFolder(renamingId, renameValue.trim())
    } else {
      await renameRequest(renamingId, renameValue.trim())
    }
    setRenamingId(null)
  }

  const handleCopyCurl = (request: model.RequestItem) => {
    const curl = buildCurlCommand(request)
    navigator.clipboard.writeText(curl)
    setDropdownMenu(null)
  }

  const handleExportCurl = async (request: model.RequestItem) => {
    const curl = buildCurlCommand(request)
    const filename = `${normalizeFileName(request.name || "request")}.curl.sh`
    await SaveTextFile(filename, curl)
    setDropdownMenu(null)
  }

  const handleDuplicate = async (type: "folder" | "request", id: string) => {
    setDropdownMenu(null)
    if (type === "folder") {
      await duplicateFolder(id)
    } else {
      await duplicateRequest(id)
    }
  }

  const handleExportNode = async (type: "folder" | "request", id: string) => {
    setDropdownMenu(null)
    const data = type === "folder"
      ? JSON.stringify(folderMap.get(id), null, 2)
      : JSON.stringify(requestMap.get(id), null, 2)
    if (data) {
      const { SaveFileDialogJSON } = await import("../../../wailsjs/go/main/App")
      const name = type === "folder" ? folderMap.get(id)?.name : requestMap.get(id)?.name
      await SaveFileDialogJSON(`${name || id}.json`, data)
    }
  }

  const handleExportProject = async () => {
    const json = await exportProjectJSON()
    if (json) {
      const { SaveFileDialogJSON } = await import("../../../wailsjs/go/main/App")
      await SaveFileDialogJSON("project.postman_collection.json", json)
    }
  }

  const resetImportDialogState = () => {
    setImportContent("")
    setDetectedImport(detectImportType(""))
    setImportError("")
    setImportSuccess("")
    setImportProgress(0)
    setCurlImportInput("")
  }

  const openImportDialog = () => {
    resetImportDialogState()
    setImportMode("file")
    setShowImportDialog(true)
  }

  const pickImportFile = async () => {
    try {
      setImportError("")
      setImportSuccess("")
      const content = await OpenFileDialogJSON()
      if (!content) return
      setImportContent(content)
      setDetectedImport(detectImportType(content))
    } catch (err) {
      setImportError(err instanceof Error ? err.message : "选择文件失败")
    }
  }

  const startImportProgress = () => {
    setImportProgress(8)
    if (importProgressTimerRef.current) clearInterval(importProgressTimerRef.current)
    importProgressTimerRef.current = setInterval(() => {
      setImportProgress((prev) => (prev >= 88 ? prev : prev + 7))
    }, 140)
  }

  const finishImportProgress = () => {
    if (importProgressTimerRef.current) clearInterval(importProgressTimerRef.current)
    importProgressTimerRef.current = null
    setImportProgress(100)
  }

  const handleFileImportConfirm = async () => {
    if (!importContent.trim()) {
      setImportError("请先选择导入文件")
      return
    }

    setImportLoading(true)
    setImportError("")
    setImportSuccess("")
    startImportProgress()

    try {
      await importFromFile("auto", importContent)
      finishImportProgress()
      setImportSuccess("导入完成")
      setActiveTab("requests")
      setSearchQuery("")
    } catch (err) {
      if (importProgressTimerRef.current) clearInterval(importProgressTimerRef.current)
      importProgressTimerRef.current = null
      setImportProgress(0)
      setImportError(err instanceof Error ? err.message : "导入失败")
    } finally {
      setImportLoading(false)
    }
  }

  const handleCurlImportConfirm = async () => {
    if (!currentProjectId) return
    const command = curlImportInput.trim()
    if (!command) {
      setImportError("请输入 cURL 命令")
      return
    }

    setImportLoading(true)
    setImportError("")
    setImportSuccess("")
    startImportProgress()

    try {
      const parsed = await ImportCurl(command)
      const createdRequest = await createRequest("", "Imported cURL")
      if (!createdRequest) throw new Error("新建请求失败")

      const requestToSave = {
        ...createdRequest,
        method: parsed.method || createdRequest.method,
        url: parsed.url || createdRequest.url,
        params: (parsed.params ?? []).map((p) => ({ key: p.key ?? "", value: p.value ?? "" })),
        headers: (parsed.headers ?? []).map((h) => ({ key: h.key ?? "", value: h.value ?? "" })),
        body: parsed.body
          ? {
              type: parsed.body.type ?? "none",
              raw: parsed.body.raw ?? "",
              json: parsed.body.json ?? "",
              formUrlEncoded: (parsed.body.formUrlEncoded ?? []).map((f) => ({
                key: f.key ?? "",
                value: f.value ?? "",
              })),
            }
          : createdRequest.body,
        auth: parsed.auth ?? createdRequest.auth,
        updatedAt: new Date().toISOString(),
      }
      await saveRequestToBackend(requestToSave as model.RequestItem)
      openRequestTab(currentProjectId, convertRequestToData(requestToSave as model.RequestItem))
      finishImportProgress()
      setImportSuccess("cURL 导入完成")
      setActiveTab("requests")
      setSearchQuery("")
    } catch (err) {
      if (importProgressTimerRef.current) clearInterval(importProgressTimerRef.current)
      importProgressTimerRef.current = null
      setImportProgress(0)
      setImportError(err instanceof Error ? err.message : "cURL 导入失败")
    } finally {
      setImportLoading(false)
    }
  }

  const openDropdownMenu = (e: React.MouseEvent, type: "folder" | "request", id: string, name: string) => {
    e.preventDefault()
    e.stopPropagation()
    const selection = window.getSelection()
    if (selection && selection.rangeCount > 0) {
      selection.removeAllRanges()
    }
    const anchorX = e.clientX
    const anchorY = e.clientY
    setSelectedNode({ type, id, name })
    setDropdownMenu({
      x: anchorX + 2,
      y: anchorY + 2,
      anchorX,
      anchorY,
      type,
      id,
      name,
    })
  }

  useEffect(() => {
    const handleKeydown = (event: KeyboardEvent) => {
      if (activeTab !== "requests") return
      if (renamingId) return
      const target = event.target as HTMLElement | null
      if (target && (target.tagName === "INPUT" || target.tagName === "TEXTAREA" || target.isContentEditable)) {
        return
      }

      let node = selectedNode
      if (!node && activeTabRequestId) {
        const activeReq = requestMap.get(activeTabRequestId)
        if (activeReq) {
          node = { type: "request", id: activeReq.id, name: activeReq.name }
        }
      }
      if (!node) return

      const withMeta = event.metaKey || event.ctrlKey
      const key = event.key.toLowerCase()

      if (withMeta && key === "e") {
        event.preventDefault()
        startRename(node.id, node.type, node.name)
        return
      }

      if (withMeta && key === "c") {
        event.preventDefault()
        void handleDuplicate(node.type, node.id)
        return
      }

      if (withMeta && key === "d") {
        event.preventDefault()
        void handleDuplicate(node.type, node.id)
        return
      }

      if (event.key === "Delete" || event.key === "Backspace") {
        event.preventDefault()
        if (node.type === "folder") {
          void deleteFolder(node.id)
        } else {
          void deleteRequest(node.id)
        }
      }
    }

    window.addEventListener("keydown", handleKeydown)
    return () => window.removeEventListener("keydown", handleKeydown)
  }, [activeTab, activeTabRequestId, deleteFolder, deleteRequest, renamingId, requestMap, selectedNode])

  const sidebarTabs: { key: SidebarTab; label: string; icon: "commandLine" | "clock" | "globe" }[] = [
    { key: "requests", label: "请求", icon: "commandLine" },
    { key: "history", label: "历史", icon: "clock" },
    { key: "environments", label: "环境", icon: "globe" },
  ]

  const clearDragState = () => {
    setDragging(null)
    setDropIndicator(null)
    if (autoExpandTimerRef.current) {
      clearTimeout(autoExpandTimerRef.current)
      autoExpandTimerRef.current = null
    }
    autoExpandTargetRef.current = null
  }

  const scheduleAutoExpand = (folderId: string) => {
    if (autoExpandTargetRef.current === folderId) return
    if (autoExpandTimerRef.current) clearTimeout(autoExpandTimerRef.current)
    autoExpandTargetRef.current = folderId
    autoExpandTimerRef.current = setTimeout(() => {
      setExpandedFolders((prev) => {
        if (prev.has(folderId)) return prev
        return new Set(prev).add(folderId)
      })
      autoExpandTargetRef.current = null
      autoExpandTimerRef.current = null
    }, AUTO_EXPAND_DELAY)
  }

  const cancelAutoExpand = () => {
    if (autoExpandTimerRef.current) {
      clearTimeout(autoExpandTimerRef.current)
      autoExpandTimerRef.current = null
    }
    autoExpandTargetRef.current = null
  }

  const getDropPosition = (event: React.DragEvent<HTMLElement>, allowInside: boolean): DropPosition => {
    const rect = event.currentTarget.getBoundingClientRect()
    const offsetY = event.clientY - rect.top
    const ratio = rect.height > 0 ? offsetY / rect.height : 0.5
    if (allowInside && ratio >= 0.25 && ratio <= 0.75) return "inside"
    return ratio < 0.5 ? "before" : "after"
  }

  const moveNode = async (nodeId: string, nodeType: CollectionNodeType, targetParentFolderId: string, targetIndex: number) => {
    await moveCollectionNode(nodeId, nodeType, targetParentFolderId, targetIndex)
  }

  const handleDragStart = (type: CollectionNodeType, id: string) => (event: React.DragEvent<HTMLElement>) => {
    if (isSearching) {
      event.preventDefault()
      return
    }
    event.stopPropagation()
    event.dataTransfer.effectAllowed = "move"
    event.dataTransfer.setData("text/plain", `${type}:${id}`)
    requestAnimationFrame(() => {
      setDragging({ type, id })
    })
  }

  const handleDragEnd = () => {
    clearDragState()
  }

  const handleNodeDragOver = (node: model.CollectionNode) => (event: React.DragEvent<HTMLDivElement>) => {
    if (!dragging || dragging.id === node.nodeId || isSearching) return
    event.preventDefault()
    event.stopPropagation()
    event.dataTransfer.dropEffect = "move"

    if (dragging.type === "folder" && node.nodeType === "folder" && isDescendantFolder(dragging.id, node.nodeId)) {
      setDropIndicator(null)
      cancelAutoExpand()
      return
    }

    const isFolder = node.nodeType === "folder"
    const position = getDropPosition(event, isFolder)
    setDropIndicator({ targetId: node.nodeId, targetType: node.nodeType as CollectionNodeType, position })

    if (isFolder && position === "inside") {
      scheduleAutoExpand(node.nodeId)
    } else {
      cancelAutoExpand()
    }
  }

  const handleNodeDragLeave = () => {
    cancelAutoExpand()
  }

  const handleRootDragOver = (event: React.DragEvent<HTMLDivElement>) => {
    if (!dragging || isSearching) return
    event.preventDefault()
    event.dataTransfer.dropEffect = "move"
    setDropIndicator({ targetId: null, targetType: "root", position: "after" })
    cancelAutoExpand()
  }

  const handleNodeDrop = (node: model.CollectionNode) => async (event: React.DragEvent<HTMLDivElement>) => {
    if (!dragging || isSearching) return
    event.preventDefault()
    event.stopPropagation()

    if (dragging.type === "folder" && node.nodeType === "folder" && isDescendantFolder(dragging.id, node.nodeId)) {
      clearDragState()
      return
    }

    const isFolder = node.nodeType === "folder"
    const position = getDropPosition(event, isFolder)
    if (position === "inside" && !isFolder) {
      clearDragState()
      return
    }

    if (position === "inside") {
      await moveNode(dragging.id, dragging.type, node.nodeId, getChildren(node.nodeId).length)
      setExpandedFolders((prev) => new Set(prev).add(node.nodeId))
      clearDragState()
      return
    }

    const siblings = getChildren(node.parentFolderId || "")
    const targetIndex = siblings.findIndex((item) => item.nodeId === node.nodeId) + (position === "after" ? 1 : 0)
    await moveNode(dragging.id, dragging.type, node.parentFolderId || "", Math.max(0, targetIndex))
    clearDragState()
  }

  const handleRootDrop = async (event: React.DragEvent<HTMLDivElement>) => {
    if (!dragging || isSearching) return
    event.preventDefault()
    event.stopPropagation()
    await moveNode(dragging.id, dragging.type, "", getChildren("").length)
    clearDragState()
  }

  const renderDropLine = (id: string, position: "before" | "after", depth: number) => {
    const isActive = dropIndicator?.targetId === id && dropIndicator.position === position
    if (!isActive) return null
    return (
      <div
        className="absolute left-0 right-1 h-[2px] bg-[var(--accent)] rounded-full pointer-events-none z-[5]"
        style={{
          marginLeft: `${TREE_INDENT_BASE + depth * TREE_INDENT_STEP}px`,
          top: position === "before" ? -1 : undefined,
          bottom: position === "after" ? -1 : undefined,
        }}
      />
    )
  }

  const renderNode = (node: model.CollectionNode, depth: number = 0): React.ReactNode => {
    if (node.nodeType === "folder") {
      const folder = folderMap.get(node.nodeId)
      if (!folder) return null
      const isExpanded = expandedFolders.has(folder.id)
      const childNodes = getChildren(folder.id)
      const isDropInside = dropIndicator?.targetId === folder.id && dropIndicator.position === "inside"
      const isDraggingSelf = dragging?.type === "folder" && dragging.id === folder.id

      return (
        <div key={`folder:${node.nodeId}`} className="relative">
          {renderDropLine(folder.id, "before", depth)}
          <div
            draggable={renamingId !== folder.id && !isSearching}
            className={cn(
              "relative flex items-center h-[28px] px-2 rounded-[var(--radius-btn)] cursor-pointer group transition-colors duration-100 select-none",
              !isDragging && "hover:bg-[var(--sidebar-hover)]",
              isDropInside && "bg-[var(--accent)]/10 ring-1 ring-[var(--accent)]",
              isDraggingSelf && "opacity-40"
            )}
            style={{ paddingLeft: `${TREE_INDENT_BASE + depth * TREE_INDENT_STEP}px` }}
            onClick={() => toggleFolder(folder.id)}
            onMouseDown={() => setSelectedNode({ type: "folder", id: folder.id, name: folder.name })}
            onContextMenu={(e) => { e.preventDefault(); e.stopPropagation(); openDropdownMenu(e, "folder", folder.id, folder.name) }}
            onDragStart={handleDragStart("folder", folder.id)}
            onDragEnd={handleDragEnd}
            onDragOver={handleNodeDragOver(node)}
            onDragLeave={handleNodeDragLeave}
            onDrop={handleNodeDrop(node)}
          >
            <AppIcon
              name="arrowRight"
              size={12}
              strokeWidth={1.9}
              className={cn(
                "mr-1 flex-shrink-0 transition-transform text-[var(--fg-muted)]",
                isExpanded && "rotate-90"
              )}
            />
            {isExpanded ? (
              <AppIcon name="folderOpen" size={14} className="mr-1.5 flex-shrink-0 text-[var(--fg-muted)]" />
            ) : (
              <AppIcon name="folder" size={14} className="mr-1.5 flex-shrink-0 text-[var(--fg-muted)]" />
            )}
            {renamingId === folder.id ? (
              <input
                className="flex-1 bg-transparent text-[length:var(--size-font-2xs)] text-[var(--fg)] border-b border-[var(--accent)] outline-none"
                value={renameValue}
                onChange={(e) => setRenameValue(e.target.value)}
                onKeyDown={(e) => { if (e.key === "Enter") handleRenameSubmit(); if (e.key === "Escape") setRenamingId(null) }}
                onBlur={handleRenameSubmit}
                autoFocus
                onClick={(e) => e.stopPropagation()}
              />
            ) : (
              <>
                <span className="text-[length:var(--size-font-2xs)] truncate min-w-0 flex-1 pr-2 group-hover:pr-11 transition-[padding] duration-150 text-[var(--sidebar-fg)]">{folder.name}</span>
                <div className="absolute right-1 top-1/2 -translate-y-1/2 opacity-0 pointer-events-none group-hover:opacity-100 group-hover:pointer-events-auto flex items-center gap-0.5 transition-opacity">
                  <button
                    className={MENU_BTN}
                    onClick={(e) => { e.stopPropagation(); handleNewRequest(folder.id) }}
                    title="新建请求"
                  >
                    <AppIcon name="add" size={12} className="text-[var(--fg-muted)]" />
                  </button>
                  <button
                    className={MENU_BTN}
                    onClick={(e) => { e.stopPropagation(); openDropdownMenu(e, "folder", folder.id, folder.name) }}
                    title="更多操作"
                  >
                    <AppIcon name="more" size={12} className="text-[var(--fg-muted)]" />
                  </button>
                </div>
              </>
            )}
          </div>
          {renderDropLine(folder.id, "after", depth)}
          {isExpanded && childNodes.length > 0 && (
            <div>{childNodes.map((childNode) => renderNode(childNode, depth + 1))}</div>
          )}
        </div>
      )
    }

    const request = requestMap.get(node.nodeId)
    if (!request) return null
    const isSelected = activeTabRequestId === request.id
    const isDraggingSelf = dragging?.type === "request" && dragging.id === request.id

    return (
      <div key={`request:${node.nodeId}`} className="relative">
        {renderDropLine(request.id, "before", depth)}
        <div
          draggable={!isSearching && renamingId !== request.id}
          className={cn(
            "flex items-center h-[28px] px-2 rounded-[var(--radius-btn)] cursor-pointer group transition-colors duration-100 select-none",
            isSelected ? "bg-[var(--sidebar-active)] text-[var(--sidebar-accent)]" : !isDragging && "hover:bg-[var(--sidebar-hover)]",
            isDraggingSelf && "opacity-40"
          )}
          style={{ paddingLeft: `${TREE_INDENT_BASE + depth * TREE_INDENT_STEP + REQUEST_METHOD_SPACER}px` }}
          onClick={() => { if (renamingId !== request.id) handleOpenRequest(request) }}
          onMouseDown={() => setSelectedNode({ type: "request", id: request.id, name: request.name })}
          onContextMenu={(e) => { e.preventDefault(); e.stopPropagation(); openDropdownMenu(e, "request", request.id, request.name) }}
          onDragStart={handleDragStart("request", request.id)}
          onDragEnd={handleDragEnd}
          onDragOver={handleNodeDragOver(node)}
          onDragLeave={handleNodeDragLeave}
          onDrop={handleNodeDrop(node)}
        >
          <span className={cn(
            "text-[9px] font-mono font-bold mr-1 min-w-[28px] text-left flex-shrink-0 uppercase",
            METHOD_COLORS[request.method as HttpMethod] || "text-[var(--fg-muted)]"
          )}>
            {request.method || "GET"}
          </span>
          {renamingId === request.id ? (
            <input
              className="flex-1 bg-transparent text-[length:var(--size-font-2xs)] text-[var(--fg)] border-b border-[var(--accent)] outline-none"
              value={renameValue}
              onChange={(e) => setRenameValue(e.target.value)}
              onKeyDown={(e) => { if (e.key === "Enter") handleRenameSubmit(); if (e.key === "Escape") setRenamingId(null) }}
              onBlur={handleRenameSubmit}
              autoFocus
              onClick={(e) => e.stopPropagation()}
            />
          ) : (
            <>
              <span className={cn(
                "text-[length:var(--size-font-2xs)] truncate min-w-0 flex-1 pr-2 group-hover:pr-11 transition-[padding] duration-150",
                isSelected ? "text-[var(--sidebar-accent)] font-medium" : "text-[var(--sidebar-fg)]"
              )} title={request.name}>
                {request.name}
              </span>
              <div className="absolute right-1 top-1/2 -translate-y-1/2 opacity-0 pointer-events-none group-hover:opacity-100 group-hover:pointer-events-auto flex items-center gap-0.5 transition-opacity">
                <button
                  className={MENU_BTN}
                  onClick={(e) => { e.stopPropagation(); handleNewRequest(request.folderId || "") }}
                  title="新建请求"
                >
                  <AppIcon name="add" size={12} className="text-[var(--fg-muted)]" />
                </button>
                <button
                  className={MENU_BTN}
                  onClick={(e) => { e.stopPropagation(); openDropdownMenu(e, "request", request.id, request.name) }}
                  title="更多操作"
                >
                  <AppIcon name="more" size={12} className="text-[var(--fg-muted)]" />
                </button>
              </div>
            </>
          )}
        </div>
        {renderDropLine(request.id, "after", depth)}
      </div>
    )
  }

  const rootNodes = getChildren("")
  const rootFolderName = "根目录"

  if (sidebarCollapsed) return null

  return (
    <div
      className={cn(
        "relative flex min-h-0 flex-col overflow-hidden border-r border-[var(--sidebar-border)] bg-[var(--sidebar-bg)]"
      )}
      style={{ width: sidebarWidth }}
    >
      <div
        className="absolute right-0 top-0 h-full w-[5px] cursor-col-resize z-10 group"
        onMouseDown={handleSidebarResizeStart}
      >
        <div className="absolute inset-y-0 right-1/2 translate-x-1/2 w-px bg-transparent group-hover:bg-[var(--accent)]/40 transition-colors duration-200" />
      </div>

      {currentProjectId && (
        <div className="flex items-center h-[var(--size-tab)] border-b border-[var(--sidebar-border)] flex-shrink-0">
          {sidebarTabs.map((tab) => (
            <button
              key={tab.key}
              className={cn(
                "flex-1 h-[calc(var(--size-tab)-2px)] font-medium transition-colors",
                "text-[length:var(--size-font-2xs)]",
                activeTab === tab.key
                  ? "text-[var(--fg)] border-b-2 border-[var(--accent)]"
                  : "text-[var(--fg-muted)] hover:text-[var(--fg-secondary)] border-b-2 border-transparent"
              )}
              onClick={() => setActiveTab(tab.key)}
            >
              <span className="inline-flex h-full items-center gap-1.5">
                <AppIcon name={tab.icon} size={12} />
                <span>{tab.label}</span>
              </span>
            </button>
          ))}
        </div>
      )}

      {currentProjectId && activeTab === "requests" && (
        <div className="px-2 pt-1.5 pb-1 flex items-center gap-1 flex-shrink-0">
          <div className="relative flex-1">
            <AppIcon name="search" size={12} className="absolute left-1.5 top-1/2 -translate-y-1/2 text-[var(--fg-muted)]" />
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
                <AppIcon name="clear" size={10} className="text-[var(--fg-muted)]" />
              </button>
            )}
          </div>
          <button
            className="h-[var(--size-btn-sm)] w-[var(--size-btn-sm)] flex items-center justify-center rounded-[var(--radius-btn)] hover:bg-[var(--sidebar-hover)] transition-colors flex-shrink-0"
            onClick={openImportDialog}
            title="导入"
          >
            <AppIcon name="fileImport" size={14} className="text-[var(--fg-secondary)]" />
          </button>
          <button
            className="h-[var(--size-btn-sm)] w-[var(--size-btn-sm)] flex items-center justify-center rounded-[var(--radius-btn)] hover:bg-[var(--sidebar-hover)] transition-colors flex-shrink-0"
            onClick={() => handleNewFolder()}
            title="新建文件夹"
          >
            <AppIcon name="folderAdd" size={14} className="text-[var(--fg-secondary)]" />
          </button>
          <button
            className="h-[var(--size-btn-sm)] w-[var(--size-btn-sm)] flex items-center justify-center rounded-[var(--radius-btn)] hover:bg-[var(--sidebar-hover)] transition-colors flex-shrink-0"
            onClick={() => handleNewRequest()}
            title="新建请求 (⌘N)"
          >
            <AppIcon name="add" size={14} className="text-[var(--fg-secondary)]" />
          </button>
        </div>
      )}

      <div
        className="flex-1 min-h-0 overflow-y-auto py-0.5 px-1"
        onDragOver={activeTab === "requests" && !isSearching ? handleRootDragOver : undefined}
        onDrop={activeTab === "requests" && !isSearching ? handleRootDrop : undefined}
      >
        {!currentProjectId ? (
          <div className="flex items-center justify-center h-full">
            <div className="text-center px-4">
              <p className="text-[length:var(--size-font-xs)] text-[var(--fg-secondary)] font-medium">无项目</p>
              <p className="text-2xs text-[var(--fg-muted)] mt-1">请在顶部左侧创建或选择项目</p>
            </div>
          </div>
        ) : activeTab === "requests" ? (
          <div>
            <div
              className={cn(
                "relative flex items-center h-[28px] px-2 rounded-[var(--radius-btn)] group transition-colors duration-100 select-none",
                !isDragging && "hover:bg-[var(--sidebar-hover)]"
              )}
            >
              <AppIcon name="folderOpen" size={14} className="mr-1.5 flex-shrink-0 text-[var(--fg-muted)]" />
              <span className="text-[length:var(--size-font-2xs)] truncate flex-1 text-[var(--sidebar-fg)] font-medium">
                {rootFolderName}
              </span>
              <div className="opacity-0 group-hover:opacity-100 flex items-center gap-0.5 flex-shrink-0 transition-opacity ml-1">
                <button
                  className={MENU_BTN}
                  onClick={(e) => { e.stopPropagation(); void handleNewRequest("") }}
                  title="在根目录新建请求"
                >
                  <AppIcon name="add" size={12} className="text-[var(--fg-muted)]" />
                </button>
                <button
                  className={MENU_BTN}
                  onClick={(e) => { e.stopPropagation(); void handleNewFolder("") }}
                  title="在根目录新建文件夹"
                >
                  <AppIcon name="folderAdd" size={12} className="text-[var(--fg-muted)]" />
                </button>
              </div>
            </div>
            {rootNodes.map((node) => renderNode(node, 0))}
            {rootNodes.length === 0 && (
              <div className="text-center py-8">
                <p className="text-2xs text-[var(--fg-muted)]">
                  {isSearching ? "无匹配结果" : "暂无请求"}
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

      {/* 节点操作下拉菜单 */}
      {dropdownMenu && createPortal(
        <div
          ref={menuRef}
          className={cn(
            "fixed z-[100] w-max p-1 rounded-[10px] shadow-lg border animate-fade-in",
            "bg-[var(--surface-elevated)] border-[var(--border-color)]"
          )}
          style={{ left: dropdownMenu.x, top: dropdownMenu.y }}
        >
          {dropdownMenu.type === "folder" && (
            <>
              <button className={MENU_ITEM} onClick={() => { handleNewRequest(dropdownMenu.id); setDropdownMenu(null) }}>
                <AppIcon name="add" size={12} /> 添加请求
              </button>
              <button className={MENU_ITEM} onClick={() => { handleNewFolder(dropdownMenu.id); setDropdownMenu(null) }}>
                <AppIcon name="folderAdd" size={12} /> 添加文件夹
              </button>
              <div className="h-px bg-[var(--border-subtle)] my-0.5" />
            </>
          )}
          <button className={MENU_ITEM} onClick={() => startRename(dropdownMenu.id, dropdownMenu.type, dropdownMenu.name)}>
            <AppIcon name="pencil" size={12} />
            <span className="flex-1">重命名</span>
            <span className={MENU_ITEM_HOTKEY}>⌘E</span>
          </button>
          <button className={MENU_ITEM} onClick={() => handleDuplicate(dropdownMenu.type, dropdownMenu.id)}>
            <AppIcon name="copy" size={12} />
            <span className="flex-1">复制</span>
            <span className={MENU_ITEM_HOTKEY}>⌘C</span>
          </button>
          <button className={MENU_ITEM} onClick={() => handleDuplicate(dropdownMenu.type, dropdownMenu.id)}>
            <AppIcon name="copy" size={12} />
            <span className="flex-1">Duplicate</span>
            <span className={MENU_ITEM_HOTKEY}>⌘D</span>
          </button>
          {dropdownMenu.type === "request" && (
            <button className={MENU_ITEM} onClick={() => { const req = requestMap.get(dropdownMenu.id); if (req) handleCopyCurl(req) }}>
              <AppIcon name="commandLine" size={12} /> 复制 cURL
            </button>
          )}
          {dropdownMenu.type === "request" && (
            <button className={MENU_ITEM} onClick={() => { const req = requestMap.get(dropdownMenu.id); if (req) void handleExportCurl(req) }}>
              <AppIcon name="download" size={12} /> 导出 cURL
            </button>
          )}
          <button className={MENU_ITEM} onClick={() => handleExportNode(dropdownMenu.type, dropdownMenu.id)}>
            <AppIcon name="download" size={12} /> 导出
          </button>
          <div className="h-px bg-[var(--border-subtle)] my-0.5" />
          <button
            className={cn(MENU_ITEM, "!text-[var(--danger)]")}
            onClick={async () => {
              if (dropdownMenu.type === "folder") await deleteFolder(dropdownMenu.id)
              else await deleteRequest(dropdownMenu.id)
              setDropdownMenu(null)
            }}
          >
            <AppIcon name="delete" size={12} />
            <span className="flex-1">删除</span>
            <span className={cn(MENU_ITEM_HOTKEY, "!text-[var(--danger)]/80")}>⌫</span>
          </button>
        </div>,
        document.body
      )}

      {showImportDialog && createPortal(
        <div
          className="fixed inset-0 z-[320] flex items-center justify-center"
          onClick={() => {
            if (!importLoading) setShowImportDialog(false)
          }}
        >
          <div className="absolute inset-0 bg-black/45 backdrop-blur-[2px]" />
          <div
            className={cn(
              "relative z-[321] w-[620px] rounded-[12px] border shadow-[var(--shadow-lg)]",
              "bg-[var(--surface)] border-[var(--border-color)] overflow-hidden"
            )}
            onClick={(event) => event.stopPropagation()}
          >
            <div className="flex items-center justify-between px-4 py-3 border-b border-[var(--border-subtle)]">
              <div className="flex items-center gap-2">
                <div className="h-7 w-7 rounded-[8px] bg-[var(--accent)]/12 flex items-center justify-center">
                  <AppIcon name="fileImport" size={14} className="text-[var(--accent)]" />
                </div>
                <div>
                  <div className="text-[13px] font-semibold text-[var(--fg)]">导入</div>
                  <div className="text-[11px] text-[var(--fg-muted)]">支持文件导入与 cURL 导入</div>
                </div>
              </div>
              <button
                type="button"
                disabled={importLoading}
                className="h-6 w-6 flex items-center justify-center rounded-[6px] hover:bg-[var(--surface-secondary)] disabled:opacity-40"
                onClick={() => setShowImportDialog(false)}
              >
                <AppIcon name="clear" size={12} className="text-[var(--fg-muted)]" />
              </button>
            </div>

            <div className="px-4 pt-3">
              <div className="inline-flex rounded-[8px] bg-[var(--surface-secondary)] p-0.5">
                <button
                  type="button"
                  disabled={importLoading}
                  className={cn(
                    "h-7 px-3 rounded-[7px] text-[12px] font-medium transition-colors",
                    importMode === "file"
                      ? "bg-[var(--surface)] text-[var(--fg)] shadow-sm"
                      : "text-[var(--fg-secondary)] hover:text-[var(--fg)]",
                  )}
                  onClick={() => { setImportMode("file"); setImportError(""); setImportSuccess("") }}
                >
                  文件导入
                </button>
                <button
                  type="button"
                  disabled={importLoading}
                  className={cn(
                    "h-7 px-3 rounded-[7px] text-[12px] font-medium transition-colors",
                    importMode === "curl"
                      ? "bg-[var(--surface)] text-[var(--fg)] shadow-sm"
                      : "text-[var(--fg-secondary)] hover:text-[var(--fg)]",
                  )}
                  onClick={() => { setImportMode("curl"); setImportError(""); setImportSuccess("") }}
                >
                  cURL 导入
                </button>
              </div>
            </div>

            <div className="px-4 py-3">
              {importMode === "file" ? (
                <div className="space-y-3">
                  <button
                    type="button"
                    disabled={importLoading}
                    onClick={() => void pickImportFile()}
                    className={cn(
                      "w-full h-[88px] rounded-[10px] border border-dashed transition-colors",
                      "border-[var(--border-color)] bg-[var(--surface-secondary)] hover:bg-[var(--button-bg)]",
                      "flex flex-col items-center justify-center gap-1.5"
                    )}
                  >
                    <AppIcon name="upload" size={16} className="text-[var(--fg-secondary)]" />
                    <span className="text-[12px] text-[var(--fg)]">选择导入文件</span>
                    <span className="text-[10px] text-[var(--fg-muted)]">Postman / OpenAPI / Swagger（JSON/YAML）</span>
                  </button>

                  <div className="rounded-[10px] border border-[var(--border-subtle)] bg-[var(--surface-secondary)] px-3 py-2.5">
                    <div className="text-[11px] text-[var(--fg-muted)]">识别结果</div>
                    <div className="mt-1 text-[12px] font-medium text-[var(--fg)]">{detectedImport.title}</div>
                    <div className="mt-0.5 text-[11px] text-[var(--fg-secondary)]">{detectedImport.description}</div>
                    {importContent && (
                      <div className="mt-1 text-[10px] text-[var(--fg-muted)]">
                        已读取 {new Blob([importContent]).size} bytes
                      </div>
                    )}
                  </div>
                </div>
              ) : (
                <div className="space-y-2">
                  <textarea
                    value={curlImportInput}
                    disabled={importLoading}
                    onChange={(event) => setCurlImportInput(event.target.value)}
                    onKeyDown={(event) => {
                      if (event.key === "Enter" && !event.shiftKey) {
                        event.preventDefault()
                        void handleCurlImportConfirm()
                      }
                    }}
                    className={cn(
                      "w-full min-h-[180px] rounded-[10px] border border-[var(--border-color)]",
                      "bg-[var(--surface-secondary)] p-3 font-mono text-[12px] text-[var(--fg)]",
                      "placeholder:text-[var(--fg-muted)] focus:outline-none focus:border-[var(--accent)]"
                    )}
                    placeholder="粘贴 cURL 命令，按 Enter 可直接导入"
                  />
                  <div className="text-[10px] text-[var(--fg-muted)]">Enter 导入，Shift+Enter 换行</div>
                </div>
              )}

              {importLoading && (
                <div className="mt-3">
                  <div className="h-1.5 w-full rounded-full bg-[var(--surface-secondary)] overflow-hidden">
                    <div
                      className="h-full bg-[var(--accent)] transition-all duration-150"
                      style={{ width: `${importProgress}%` }}
                    />
                  </div>
                  <div className="mt-1 text-[10px] text-[var(--fg-muted)]">正在导入，请稍候...</div>
                </div>
              )}

              {importError && (
                <div className="mt-3 rounded-[8px] border border-[var(--danger)]/25 bg-[var(--danger)]/8 px-2.5 py-2 text-[11px] text-[var(--danger)]">
                  {importError}
                </div>
              )}
              {importSuccess && (
                <div className="mt-3 rounded-[8px] border border-[var(--success)]/25 bg-[var(--success)]/10 px-2.5 py-2 text-[11px] text-[var(--success)]">
                  {importSuccess}
                </div>
              )}
            </div>

            <div className="flex items-center justify-end gap-2 px-4 py-3 border-t border-[var(--border-subtle)]">
              <button
                type="button"
                disabled={importLoading}
                className={cn(
                  "h-8 px-4 rounded-[8px] text-[12px] font-medium",
                  "border border-[var(--border-color)] text-[var(--fg-secondary)] hover:bg-[var(--surface-secondary)] disabled:opacity-40"
                )}
                onClick={() => setShowImportDialog(false)}
              >
                取消
              </button>
              <button
                type="button"
                disabled={importLoading || (importMode === "file" ? !importContent.trim() : !curlImportInput.trim())}
                className={cn(
                  "h-8 px-4 rounded-[8px] text-[12px] font-medium text-white",
                  "bg-[var(--accent)] hover:brightness-105 disabled:opacity-40 disabled:pointer-events-none"
                )}
                onClick={() => {
                  if (importMode === "file") {
                    void handleFileImportConfirm()
                    return
                  }
                  void handleCurlImportConfirm()
                }}
              >
                {importMode === "file" ? "导入文件" : "导入 cURL"}
              </button>
            </div>
          </div>
        </div>,
        document.body
      )}
    </div>
  )
}
