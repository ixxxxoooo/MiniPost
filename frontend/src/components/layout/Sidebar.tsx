import { useState, useRef, useEffect, useMemo, useCallback, useLayoutEffect, type DragEvent as ReactDragEvent } from "react"
import { createPortal } from "react-dom"
import { AppIcon } from "@/components/ui/icon"
import { useI18n } from "@/hooks/useI18n"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { cn } from "@/lib/utils"
import { writeClipboardText } from "@/lib/clipboard"
import { METHOD_COLORS, type HttpMethod } from "@/lib/constants"
import { buildDraftRequestFromCurl } from "@/lib/curlImportDraft"
import { info } from "@/lib/logger"
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
type ImportMode = "file" | "url" | "curl"
type DetectedImportType = "postman" | "postman-environment" | "swagger" | "unknown"

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

interface DeleteConfirmState {
  id: string
  type: "folder" | "request"
  name: string
  nextRequestId?: string
}

const AUTO_EXPAND_DELAY = 600

function detectImportType(content: string, t: (zh: string, en: string) => string): {
  type: DetectedImportType
  title: string
  description: string
} {
  const trimmed = content.trim()
  if (!trimmed) {
    return { type: "unknown", title: t("未选择文件", "No file selected"), description: t("请先选择导入文件", "Please select an import file first") }
  }

  try {
    const parsed = JSON.parse(trimmed) as Record<string, unknown>
    if (parsed && typeof parsed === "object") {
      if ("info" in parsed && "item" in parsed) {
        return {
          type: "postman",
          title: "Postman Collection",
          description: t("检测到 Postman v2.x 集合结构", "Detected Postman v2.x collection structure"),
        }
      }
      if ("values" in parsed && !("item" in parsed)) {
        return {
          type: "postman-environment",
          title: "Postman Environment",
          description: t("检测到 Postman 环境变量结构", "Detected Postman environment structure"),
        }
      }
      if ("paths" in parsed && ("openapi" in parsed || "swagger" in parsed)) {
        return {
          type: "swagger",
          title: "OpenAPI / Swagger",
          description: t("检测到 OpenAPI/Swagger 结构", "Detected OpenAPI/Swagger structure"),
        }
      }
      if ("paths" in parsed) {
        return {
          type: "swagger",
          title: "OpenAPI / Swagger",
          description: t("检测到 paths 字段，可按 OpenAPI 导入", "Detected paths field, can import as OpenAPI"),
        }
      }
    }
  } catch {
    const normalized = trimmed.toLowerCase()
    if (normalized.includes("openapi:") || normalized.includes("swagger:")) {
      return {
        type: "swagger",
        title: "OpenAPI / Swagger (YAML)",
        description: t("检测到 YAML OpenAPI/Swagger 结构", "Detected YAML OpenAPI/Swagger structure"),
      }
    }
  }

  return {
    type: "unknown",
    title: t("未识别格式", "Unrecognized format"),
    description: t("支持 Postman Collection / Environment / OpenAPI / Swagger（JSON 或 YAML）", "Supports Postman Collection / Environment / OpenAPI / Swagger (JSON or YAML)"),
  }
}

function getImportErrorMessage(err: unknown, fallback: string): string {
  if (err instanceof Error && err.message.trim()) return err.message
  if (typeof err === "string" && err.trim()) return err
  if (err && typeof err === "object") {
    const maybeMessage = Reflect.get(err, "message")
    if (typeof maybeMessage === "string" && maybeMessage.trim()) return maybeMessage
    try {
      const serialized = JSON.stringify(err)
      if (serialized && serialized !== "{}") return serialized
    } catch {
      // ignore serialization errors
    }
  }
  return fallback
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
    params: (request.params ?? []).map((p: { key: string; value: string; description?: string }) => ({
      id: crypto.randomUUID(), key: p.key, value: p.value, enabled: true, description: p.description ?? "",
    })),
    headers: (request.headers ?? []).map((h: { key: string; value: string; description?: string }) => ({
      id: crypto.randomUUID(), key: h.key, value: h.value, enabled: true, description: h.description ?? "",
    })),
    body: request.body
      ? {
          type: request.body.type as "none" | "raw" | "json" | "form-urlencoded" | "form-data",
          raw: request.body.raw,
          json: request.body.json,
          formUrlEncoded: (request.body.formUrlEncoded ?? []).map((f: { key: string; value: string; description?: string }) => ({
            id: crypto.randomUUID(), key: f.key, value: f.value, enabled: true, description: f.description ?? "",
          })),
          formData: (request.body.formData ?? []).map((f: { key: string; value: string; description?: string; type?: string; filePath?: string; fileName?: string }) => ({
            id: crypto.randomUUID(),
            key: f.key,
            value: f.value ?? "",
            description: f.description ?? "",
            enabled: true,
            type: (f.type as "text" | "file") || "text",
            filePath: f.filePath,
            fileName: f.fileName,
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

interface SidebarProps {
  forceOpen?: boolean
}

export function Sidebar({ forceOpen = false }: SidebarProps) {
  const { t } = useI18n()
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
    importFromURL,
  } = useProjectStore()
  const { openRequestTab, addTab } = useTabStore()
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
  const [detectedImport, setDetectedImport] = useState(() => detectImportType("", t))
  const [importLoading, setImportLoading] = useState(false)
  const [importProgress, setImportProgress] = useState(0)
  const [importError, setImportError] = useState("")
  const [importSuccess, setImportSuccess] = useState("")
  const [isImportFileDragActive, setIsImportFileDragActive] = useState(false)
  const [urlImportInput, setUrlImportInput] = useState("")
  const [curlImportInput, setCurlImportInput] = useState("")
  const [deleteConfirm, setDeleteConfirm] = useState<DeleteConfirmState | null>(null)
  const [deleteConfirmLoading, setDeleteConfirmLoading] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)
  const resizingRef = useRef(false)
  const searchInputRef = useRef<HTMLInputElement>(null)
  const autoExpandTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const autoExpandTargetRef = useRef<string | null>(null)
  const importProgressTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const importFileDragDepthRef = useRef(0)

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

  const requestDeleteConfirm = useCallback((target: { id: string; type: "folder" | "request"; name: string }, fromShortcut = false) => {
    let nextRequestId = ""
    if (fromShortcut && target.type === "request") {
      const requestIds = sortedTreeNodes
        .filter((node) => node.nodeType === "request" && (!filteredNodeIds || filteredNodeIds.has(node.nodeId)))
        .map((node) => node.nodeId)
      const currentIndex = requestIds.indexOf(target.id)
      nextRequestId = currentIndex >= 0 ? (requestIds[currentIndex + 1] ?? requestIds[currentIndex - 1] ?? "") : ""
    }
    setDeleteConfirm({ ...target, nextRequestId: nextRequestId || undefined })
    setDropdownMenu(null)
  }, [filteredNodeIds, sortedTreeNodes])

  const handleConfirmDelete = useCallback(async () => {
    if (!deleteConfirm || deleteConfirmLoading) return
    setDeleteConfirmLoading(true)
    try {
      if (deleteConfirm.type === "folder") {
        await deleteFolder(deleteConfirm.id)
      } else {
        await deleteRequest(deleteConfirm.id)
        if (deleteConfirm.nextRequestId) {
          const nextRequest = requestMap.get(deleteConfirm.nextRequestId)
          if (nextRequest) {
            setSelectedNode({ type: "request", id: nextRequest.id, name: nextRequest.name })
            handleOpenRequest(nextRequest)
          } else {
            setSelectedNode(null)
          }
        } else if (selectedNode?.id === deleteConfirm.id) {
          setSelectedNode(null)
        }
      }
      setDeleteConfirm(null)
    } finally {
      setDeleteConfirmLoading(false)
    }
  }, [deleteConfirm, deleteConfirmLoading, deleteFolder, deleteRequest, handleOpenRequest, requestMap, selectedNode?.id])

  useEffect(() => {
    if (!deleteConfirm) return
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault()
        if (!deleteConfirmLoading) setDeleteConfirm(null)
        return
      }
      if (event.key === "Enter") {
        event.preventDefault()
        if (!deleteConfirmLoading) {
          void handleConfirmDelete()
        }
      }
    }
    window.addEventListener("keydown", onKeyDown)
    return () => window.removeEventListener("keydown", onKeyDown)
  }, [deleteConfirm, deleteConfirmLoading, handleConfirmDelete])

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
    void writeClipboardText(curl)
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
    setDetectedImport(detectImportType("", t))
    setImportError("")
    setImportSuccess("")
    setImportProgress(0)
    setIsImportFileDragActive(false)
    importFileDragDepthRef.current = 0
    setUrlImportInput("")
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
      setDetectedImport(detectImportType(content, t))
    } catch (err) {
      setImportError(getImportErrorMessage(err, t("选择文件失败", "Failed to choose file")))
    }
  }

  const isFileDragEvent = (event: ReactDragEvent<HTMLElement>) => (
    Array.from(event.dataTransfer?.types ?? []).includes("Files")
  )

  const handleImportFileDragEnter = (event: ReactDragEvent<HTMLElement>) => {
    if (importLoading || importMode !== "file" || !isFileDragEvent(event)) return
    event.preventDefault()
    event.stopPropagation()
    importFileDragDepthRef.current += 1
    event.dataTransfer.dropEffect = "copy"
    setIsImportFileDragActive(true)
  }

  const handleImportFileDragOver = (event: ReactDragEvent<HTMLElement>) => {
    if (importLoading || importMode !== "file" || !isFileDragEvent(event)) return
    event.preventDefault()
    event.stopPropagation()
    event.dataTransfer.dropEffect = "copy"
    if (!isImportFileDragActive) setIsImportFileDragActive(true)
  }

  const handleImportFileDragLeave = (event: ReactDragEvent<HTMLElement>) => {
    if (importLoading || importMode !== "file" || !isFileDragEvent(event)) return
    event.preventDefault()
    event.stopPropagation()
    importFileDragDepthRef.current = Math.max(0, importFileDragDepthRef.current - 1)
    if (importFileDragDepthRef.current === 0) {
      setIsImportFileDragActive(false)
    }
  }

  const handleImportFileDrop = async (event: ReactDragEvent<HTMLElement>) => {
    if (importLoading || importMode !== "file" || !isFileDragEvent(event)) return
    event.preventDefault()
    event.stopPropagation()
    importFileDragDepthRef.current = 0
    setIsImportFileDragActive(false)
    const file = event.dataTransfer.files?.[0]
    if (!file) return
    try {
      setImportError("")
      setImportSuccess("")
      const content = await file.text()
      if (!content) {
        setImportError(t("文件内容为空", "The file is empty"))
        return
      }
      setImportContent(content)
      setDetectedImport(detectImportType(content, t))
    } catch (err) {
      setImportError(getImportErrorMessage(err, t("读取拖拽文件失败", "Failed to read dropped file")))
    }
  }

  useEffect(() => {
    if (importMode === "file" && showImportDialog) return
    setIsImportFileDragActive(false)
    importFileDragDepthRef.current = 0
  }, [importMode, showImportDialog])

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
      setImportError(t("请先选择导入文件", "Please select an import file first"))
      return
    }

    setImportLoading(true)
    setImportError("")
    setImportSuccess("")
    startImportProgress()

    try {
      await importFromFile("auto", importContent)
      finishImportProgress()
      setImportSuccess(t("导入完成", "Import completed"))
      setActiveTab("requests")
      setSearchQuery("")
    } catch (err) {
      if (importProgressTimerRef.current) clearInterval(importProgressTimerRef.current)
      importProgressTimerRef.current = null
      setImportProgress(0)
      setImportError(getImportErrorMessage(err, t("导入失败", "Import failed")))
    } finally {
      setImportLoading(false)
    }
  }

  const handleCurlImportConfirm = async () => {
    if (!currentProjectId) return
    const command = curlImportInput.trim()
    if (!command) {
      setImportError(t("请输入 cURL 命令", "Please enter a cURL command"))
      return
    }

    setImportLoading(true)
    setImportError("")
    setImportSuccess("")
    startImportProgress()

    try {
      const parsed = await ImportCurl(command)
      const draftRequest = buildDraftRequestFromCurl(parsed, {
        projectId: currentProjectId,
        name: "Imported cURL",
      })
      info("CurlImport", "cURL imported into draft tab", {
        source: "sidebar",
        commandLength: command.length,
        method: draftRequest.method,
        hasUrl: Boolean(draftRequest.url),
        headerCount: draftRequest.headers.filter((item) => item.key.trim()).length,
        bodyType: draftRequest.body.type,
      })
      addTab({
        title: draftRequest.name || "Imported cURL",
        projectId: currentProjectId,
        closable: true,
        dirty: true,
        request: draftRequest,
        response: null,
        responseError: null,
      })
      finishImportProgress()
      setImportSuccess(t("cURL 已导入为未保存草稿", "cURL imported as an unsaved draft"))
      setActiveTab("requests")
      setSearchQuery("")
    } catch (err) {
      if (importProgressTimerRef.current) clearInterval(importProgressTimerRef.current)
      importProgressTimerRef.current = null
      setImportProgress(0)
      setImportError(getImportErrorMessage(err, t("cURL 导入失败", "cURL import failed")))
    } finally {
      setImportLoading(false)
    }
  }

  const handleUrlImportConfirm = async () => {
    const sourceURL = urlImportInput.trim()
    if (!sourceURL) {
      setImportError(t("请输入导入地址", "Please enter an import URL"))
      return
    }
    if (!/^https?:\/\//i.test(sourceURL)) {
      setImportError(t("导入地址需以 http:// 或 https:// 开头", "Import URL must start with http:// or https://"))
      return
    }

    setImportLoading(true)
    setImportError("")
    setImportSuccess("")
    startImportProgress()

    try {
      await importFromURL("auto", sourceURL)
      finishImportProgress()
      setImportSuccess(t("已从地址导入完成", "Imported successfully from URL"))
      setActiveTab("requests")
      setSearchQuery("")
    } catch (err) {
      if (importProgressTimerRef.current) clearInterval(importProgressTimerRef.current)
      importProgressTimerRef.current = null
      setImportProgress(0)
      setImportError(getImportErrorMessage(err, t("地址导入失败", "URL import failed")))
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

      const node = selectedNode
      if (!node || node.type !== "request") return

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
        requestDeleteConfirm(node, true)
      }
    }

    window.addEventListener("keydown", handleKeydown)
    return () => window.removeEventListener("keydown", handleKeydown)
  }, [activeTab, renamingId, requestDeleteConfirm, selectedNode])

  const sidebarTabs: { key: SidebarTab; label: string; icon: "commandLine" | "clock" | "globe" }[] = [
    { key: "requests", label: t("请求", "Requests"), icon: "commandLine" },
    { key: "history", label: t("历史", "History"), icon: "clock" },
    { key: "environments", label: t("环境", "Environments"), icon: "globe" },
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
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <button
                        className={MENU_BTN}
                        onClick={(e) => { e.stopPropagation(); handleNewRequest(folder.id) }}
                      >
                        <AppIcon name="add" size={12} className="text-[var(--fg-muted)]" />
                      </button>
                    </TooltipTrigger>
                    <TooltipContent>{t("新建请求", "New request")}</TooltipContent>
                  </Tooltip>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <button
                        className={MENU_BTN}
                        onClick={(e) => { e.stopPropagation(); openDropdownMenu(e, "folder", folder.id, folder.name) }}
                      >
                        <AppIcon name="more" size={12} className="text-[var(--fg-muted)]" />
                      </button>
                    </TooltipTrigger>
                    <TooltipContent>{t("更多操作", "More actions")}</TooltipContent>
                  </Tooltip>
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
              )}>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span className="block truncate">{request.name}</span>
                  </TooltipTrigger>
                  <TooltipContent>{request.name}</TooltipContent>
                </Tooltip>
              </span>
              <div className="absolute right-1 top-1/2 -translate-y-1/2 opacity-0 pointer-events-none group-hover:opacity-100 group-hover:pointer-events-auto flex items-center gap-0.5 transition-opacity">
                <Tooltip>
                  <TooltipTrigger asChild>
                    <button
                      className={MENU_BTN}
                      onClick={(e) => { e.stopPropagation(); handleNewRequest(request.folderId || "") }}
                    >
                      <AppIcon name="add" size={12} className="text-[var(--fg-muted)]" />
                    </button>
                  </TooltipTrigger>
                  <TooltipContent>{t("新建请求", "New request")}</TooltipContent>
                </Tooltip>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <button
                      className={MENU_BTN}
                      onClick={(e) => { e.stopPropagation(); openDropdownMenu(e, "request", request.id, request.name) }}
                    >
                      <AppIcon name="more" size={12} className="text-[var(--fg-muted)]" />
                    </button>
                  </TooltipTrigger>
                  <TooltipContent>{t("更多操作", "More actions")}</TooltipContent>
                </Tooltip>
              </div>
            </>
          )}
        </div>
        {renderDropLine(request.id, "after", depth)}
      </div>
    )
  }

  const rootNodes = getChildren("")
  const rootFolderName = t("根目录", "Root")

  if (sidebarCollapsed && !forceOpen) return null

  return (
    <div
      className={cn(
        "relative flex h-full min-h-0 flex-col overflow-y-hidden overflow-x-visible border-r border-[var(--sidebar-border)] bg-[var(--sidebar-bg)]"
      )}
      style={{ width: sidebarWidth }}
    >
      <div
        className="absolute -right-[3px] top-0 h-full w-[6px] cursor-col-resize z-10 group"
        onMouseDown={handleSidebarResizeStart}
      >
        <div className="absolute inset-y-0 left-1/2 -translate-x-1/2 w-px bg-transparent group-hover:bg-[var(--accent)]/40 transition-colors duration-200" />
      </div>

      {currentProjectId && (
        <div className="flex items-center h-[var(--size-tab)] border-b border-[var(--sidebar-border)] flex-shrink-0">
          {sidebarTabs.map((tab) => (
            <button
              key={tab.key}
              className={cn(
                "flex-1 h-[calc(var(--size-tab)-2px)] transition-colors",
                "text-[length:var(--size-font-2xs)]",
                activeTab === tab.key
                  ? "font-medium text-[var(--fg)] border-b-2 border-[var(--accent)]"
                  : "font-normal text-[var(--fg-muted)] hover:text-[var(--fg-secondary)] border-b-2 border-transparent"
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
              placeholder={t("搜索请求...", "Search requests...")}
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
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                className="h-[var(--size-btn-sm)] w-[var(--size-btn-sm)] flex items-center justify-center rounded-[var(--radius-btn)] hover:bg-[var(--sidebar-hover)] transition-colors flex-shrink-0"
                onClick={openImportDialog}
              >
                <AppIcon name="fileImport" size={14} className="text-[var(--fg-secondary)]" />
              </button>
            </TooltipTrigger>
            <TooltipContent>{t("导入", "Import")}</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                className="h-[var(--size-btn-sm)] w-[var(--size-btn-sm)] flex items-center justify-center rounded-[var(--radius-btn)] hover:bg-[var(--sidebar-hover)] transition-colors flex-shrink-0"
                onClick={() => handleNewFolder()}
              >
                <AppIcon name="folderAdd" size={14} className="text-[var(--fg-secondary)]" />
              </button>
            </TooltipTrigger>
            <TooltipContent>{t("新建文件夹", "New folder")}</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                className="h-[var(--size-btn-sm)] w-[var(--size-btn-sm)] flex items-center justify-center rounded-[var(--radius-btn)] hover:bg-[var(--sidebar-hover)] transition-colors flex-shrink-0"
                onClick={() => handleNewRequest()}
              >
                <AppIcon name="add" size={14} className="text-[var(--fg-secondary)]" />
              </button>
            </TooltipTrigger>
            <TooltipContent>{t("新建请求", "New request")} (⌘N)</TooltipContent>
          </Tooltip>
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
              <p className="text-[length:var(--size-font-xs)] text-[var(--fg-secondary)] font-medium">{t("无项目", "No project")}</p>
              <p className="text-2xs text-[var(--fg-muted)] mt-1">{t("请在顶部左侧创建或选择项目", "Create or select a project from the top-left")}</p>
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
                <Tooltip>
                  <TooltipTrigger asChild>
                    <button
                      className={MENU_BTN}
                      onClick={(e) => { e.stopPropagation(); void handleNewRequest("") }}
                    >
                      <AppIcon name="add" size={12} className="text-[var(--fg-muted)]" />
                    </button>
                  </TooltipTrigger>
                  <TooltipContent>{t("在根目录新建请求", "New request in root")}</TooltipContent>
                </Tooltip>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <button
                      className={MENU_BTN}
                      onClick={(e) => { e.stopPropagation(); void handleNewFolder("") }}
                    >
                      <AppIcon name="folderAdd" size={12} className="text-[var(--fg-muted)]" />
                    </button>
                  </TooltipTrigger>
                  <TooltipContent>{t("在根目录新建文件夹", "New folder in root")}</TooltipContent>
                </Tooltip>
              </div>
            </div>
            {rootNodes.map((node) => renderNode(node, 0))}
            {rootNodes.length === 0 && (
              <div className="text-center py-8">
                <p className="text-2xs text-[var(--fg-muted)]">
                  {isSearching ? t("无匹配结果", "No results") : t("暂无请求", "No requests")}
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
                <AppIcon name="add" size={12} /> {t("添加请求", "Add request")}
              </button>
              <button className={MENU_ITEM} onClick={() => { handleNewFolder(dropdownMenu.id); setDropdownMenu(null) }}>
                <AppIcon name="folderAdd" size={12} /> {t("添加文件夹", "Add folder")}
              </button>
              <div className="h-px bg-[var(--border-subtle)] my-0.5" />
            </>
          )}
          <button className={MENU_ITEM} onClick={() => startRename(dropdownMenu.id, dropdownMenu.type, dropdownMenu.name)}>
            <AppIcon name="pencil" size={12} />
            <span className="flex-1">{t("重命名", "Rename")}</span>
            <span className={MENU_ITEM_HOTKEY}>⌘E</span>
          </button>
          <button className={MENU_ITEM} onClick={() => handleDuplicate(dropdownMenu.type, dropdownMenu.id)}>
            <AppIcon name="copy" size={12} />
            <span className="flex-1">{t("复制", "Copy")}</span>
            <span className={MENU_ITEM_HOTKEY}>⌘C</span>
          </button>
          <button className={MENU_ITEM} onClick={() => handleDuplicate(dropdownMenu.type, dropdownMenu.id)}>
            <AppIcon name="copy" size={12} />
            <span className="flex-1">{t("Duplicate", "Duplicate")}</span>
            <span className={MENU_ITEM_HOTKEY}>⌘D</span>
          </button>
          {dropdownMenu.type === "request" && (
            <button className={MENU_ITEM} onClick={() => { const req = requestMap.get(dropdownMenu.id); if (req) handleCopyCurl(req) }}>
              <AppIcon name="commandLine" size={12} /> {t("复制 cURL", "Copy cURL")}
            </button>
          )}
          {dropdownMenu.type === "request" && (
            <button className={MENU_ITEM} onClick={() => { const req = requestMap.get(dropdownMenu.id); if (req) void handleExportCurl(req) }}>
              <AppIcon name="download" size={12} /> {t("导出 cURL", "Export cURL")}
            </button>
          )}
          <button className={MENU_ITEM} onClick={() => handleExportNode(dropdownMenu.type, dropdownMenu.id)}>
            <AppIcon name="download" size={12} /> {t("导出", "Export")}
          </button>
          <div className="h-px bg-[var(--border-subtle)] my-0.5" />
          <button
            className={cn(MENU_ITEM, "!text-[var(--danger)]")}
            onClick={() => {
              requestDeleteConfirm({ id: dropdownMenu.id, type: dropdownMenu.type, name: dropdownMenu.name })
            }}
          >
            <AppIcon name="delete" size={12} />
            <span className="flex-1">{t("删除", "Delete")}</span>
            <span className={cn(MENU_ITEM_HOTKEY, "!text-[var(--danger)]/80")}>⌫</span>
          </button>
        </div>,
        document.body
      )}
      {deleteConfirm && createPortal(
        <div className="fixed inset-0 z-[330] flex items-center justify-center" onClick={() => { if (!deleteConfirmLoading) setDeleteConfirm(null) }}>
          <div className="absolute inset-0 bg-black/45 backdrop-blur-[1.5px]" />
          <div
            className="relative z-[331] w-[440px] rounded-[12px] border border-[var(--border-color)] bg-[var(--surface)] shadow-[var(--shadow-lg)]"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="px-4 py-3 border-b border-[var(--border-subtle)]">
              <div className="text-[16px] font-semibold text-[var(--fg)]">{t("确认删除", "Confirm deletion")}</div>
            </div>
            <div className="px-4 py-4 text-[13px] text-[var(--fg-secondary)] leading-[1.6]">
              {t("确定删除", "Are you sure you want to delete")}
              {deleteConfirm.type === "folder" ? t("文件夹", "folder") : t("请求", "request")}
              <span className="mx-1 text-[var(--fg)] font-medium">“{deleteConfirm.name}”</span>{t("吗？", "?")}
            </div>
            <div className="px-4 py-3 border-t border-[var(--border-subtle)] flex items-center justify-end gap-2">
              <button
                type="button"
                className="h-[32px] px-3 rounded-[8px] border border-[var(--button-border)] text-[12px] text-[var(--fg)] hover:bg-[var(--button-bg)]"
                onClick={() => setDeleteConfirm(null)}
                disabled={deleteConfirmLoading}
              >
                {t("取消", "Cancel")}
              </button>
              <button
                type="button"
                className="h-[32px] px-3 rounded-[8px] bg-[var(--danger)] text-white text-[12px] font-medium hover:opacity-95 disabled:opacity-60"
                onClick={() => void handleConfirmDelete()}
                disabled={deleteConfirmLoading}
              >
                {deleteConfirmLoading ? t("删除中...", "Deleting...") : t("删除", "Delete")}
              </button>
            </div>
          </div>
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
                  <div className="text-[13px] font-semibold text-[var(--fg)]">{t("导入", "Import")}</div>
                  <div className="text-[11px] text-[var(--fg-muted)]">{t("支持文件、URL 与 cURL 导入", "Supports file, URL, and cURL import")}</div>
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
                  {t("文件导入", "File import")}
                </button>
                <button
                  type="button"
                  disabled={importLoading}
                  className={cn(
                    "h-7 px-3 rounded-[7px] text-[12px] font-medium transition-colors",
                    importMode === "url"
                      ? "bg-[var(--surface)] text-[var(--fg)] shadow-sm"
                      : "text-[var(--fg-secondary)] hover:text-[var(--fg)]",
                  )}
                  onClick={() => { setImportMode("url"); setImportError(""); setImportSuccess("") }}
                >
                  {t("URL 导入", "URL import")}
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
                  {t("cURL 导入", "cURL import")}
                </button>
              </div>
            </div>

            <div className="px-4 py-3">
              {importMode === "file" ? (
                <div
                  className="space-y-3"
                  onDragEnter={handleImportFileDragEnter}
                  onDragOver={handleImportFileDragOver}
                  onDragLeave={handleImportFileDragLeave}
                  onDrop={(event) => { void handleImportFileDrop(event) }}
                >
                  <button
                    type="button"
                    disabled={importLoading}
                    onClick={() => void pickImportFile()}
                    className={cn(
                      "w-full h-[88px] rounded-[10px] border border-dashed transition-colors",
                      isImportFileDragActive
                        ? "border-[var(--accent)] bg-[var(--accent)]/10"
                        : "border-[var(--border-color)] bg-[var(--surface-secondary)] hover:bg-[var(--button-bg)]",
                      "flex flex-col items-center justify-center gap-1.5"
                    )}
                  >
                    <AppIcon name="upload" size={16} className="text-[var(--fg-secondary)]" />
                    <span className="text-[12px] text-[var(--fg)]">
                      {isImportFileDragActive ? t("松开即可导入文件", "Drop file to import") : t("选择导入文件", "Select import file")}
                    </span>
                    <span className="text-[10px] text-[var(--fg-muted)]">Postman Collection / Environment / OpenAPI / Swagger（JSON/YAML）</span>
                  </button>

                  <div className="rounded-[10px] border border-[var(--border-subtle)] bg-[var(--surface-secondary)] px-3 py-2.5">
                    <div className="text-[11px] text-[var(--fg-muted)]">{t("识别结果", "Detection result")}</div>
                    <div className="mt-1 text-[12px] font-medium text-[var(--fg)]">{detectedImport.title}</div>
                    <div className="mt-0.5 text-[11px] text-[var(--fg-secondary)]">{detectedImport.description}</div>
                    {importContent && (
                      <div className="mt-1 text-[10px] text-[var(--fg-muted)]">
                        {t("已读取", "Read")} {new Blob([importContent]).size} bytes
                      </div>
                    )}
                  </div>
                </div>
              ) : importMode === "url" ? (
                <div className="space-y-3">
                  <input
                    type="url"
                    value={urlImportInput}
                    disabled={importLoading}
                    onChange={(event) => setUrlImportInput(event.target.value)}
                    onKeyDown={(event) => {
                      if (event.key === "Enter") {
                        event.preventDefault()
                        void handleUrlImportConfirm()
                      }
                    }}
                    className={cn(
                      "w-full h-10 rounded-[10px] border border-[var(--border-color)]",
                      "bg-[var(--surface-secondary)] px-3 text-[12px] text-[var(--fg)]",
                      "placeholder:text-[var(--fg-muted)] focus:outline-none focus:border-[var(--accent)]"
                    )}
                    placeholder={t("粘贴 Swagger / OpenAPI / Postman 文档地址", "Paste a Swagger / OpenAPI / Postman document URL")}
                  />
                  <div className="rounded-[10px] border border-[var(--border-subtle)] bg-[var(--surface-secondary)] px-3 py-2.5">
                    <div className="text-[11px] text-[var(--fg-muted)]">{t("导入方式", "Import method")}</div>
                    <div className="mt-1 text-[12px] font-medium text-[var(--fg)]">{t("自动拉取并识别远程文档", "Fetch and auto-detect remote document")}</div>
                    <div className="mt-0.5 text-[11px] text-[var(--fg-secondary)]">
                      {t("支持 http/https 地址，内容会按 Postman / OpenAPI / Swagger 自动识别。", "Supports http/https URLs and auto-detects Postman / OpenAPI / Swagger content.")}
                    </div>
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
                    placeholder={t("粘贴 cURL 命令，按 Enter 可直接导入", "Paste a cURL command, press Enter to import")}
                  />
                  <div className="text-[10px] text-[var(--fg-muted)]">{t("Enter 导入，Shift+Enter 换行", "Enter to import, Shift+Enter for newline")}</div>
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
                  <div className="mt-1 text-[10px] text-[var(--fg-muted)]">{t("正在导入，请稍候...", "Importing, please wait...")}</div>
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
                {t("取消", "Cancel")}
              </button>
              <button
                type="button"
                disabled={
                  importLoading || (
                    importMode === "file"
                      ? !importContent.trim()
                      : importMode === "url"
                        ? !urlImportInput.trim()
                        : !curlImportInput.trim()
                  )
                }
                className={cn(
                  "h-8 px-4 rounded-[8px] text-[12px] font-medium text-white",
                  "bg-[var(--accent)] hover:brightness-105 disabled:opacity-40 disabled:pointer-events-none"
                )}
                onClick={() => {
                  if (importMode === "file") {
                    void handleFileImportConfirm()
                    return
                  }
                  if (importMode === "url") {
                    void handleUrlImportConfirm()
                    return
                  }
                  void handleCurlImportConfirm()
                }}
              >
                {importMode === "file"
                  ? t("导入文件", "Import file")
                  : importMode === "url"
                    ? t("导入地址", "Import URL")
                    : t("导入 cURL", "Import cURL")}
              </button>
            </div>
          </div>
        </div>,
        document.body
      )}
    </div>
  )
}
