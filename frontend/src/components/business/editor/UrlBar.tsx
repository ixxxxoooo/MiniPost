import { useState, useRef, useEffect, useMemo, useCallback } from "react"
import { HTTP_METHODS, METHOD_COLORS, type HttpMethod } from "@/lib/constants"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { AppIcon } from "@/components/ui/icon"
import { useTabStore, getProjectActiveTabFromState } from "@/stores/tabStore"
import { useUIStore } from "@/stores/uiStore"
import { useProjectStore } from "@/stores/projectStore"
import { cn } from "@/lib/utils"

interface UrlBarProps {
  onSend: (downloadAfter?: boolean) => void
  onCancel: () => void
  onSave: () => void
}

export function UrlBar({ onSend, onCancel, onSave }: UrlBarProps) {
  const activeTab = useTabStore(getProjectActiveTabFromState)
  const updateTabRequest = useTabStore((s) => s.updateTabRequest)
  const updateTab = useTabStore((s) => s.updateTab)
  const { isSending } = useUIStore()
  const projects = useProjectStore((s) => s.projects)
  const currentProjectId = useProjectStore((s) => s.currentProjectId)
  const folders = useProjectStore((s) => s.folders)
  const renameRequest = useProjectStore((s) => s.renameRequest)
  const [showSendMenu, setShowSendMenu] = useState(false)
  const [isEditingName, setIsEditingName] = useState(false)
  const [nameDraft, setNameDraft] = useState("")
  const sendMenuRef = useRef<HTMLDivElement>(null)
  const nameInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (!showSendMenu) return
    const handler = (e: MouseEvent) => {
      if (sendMenuRef.current && !sendMenuRef.current.contains(e.target as Node)) {
        setShowSendMenu(false)
      }
    }
    document.addEventListener("mousedown", handler)
    return () => document.removeEventListener("mousedown", handler)
  }, [showSendMenu])

  if (!activeTab) return null

  const { request } = activeTab
  const requestName = request.name?.trim() || activeTab.title || "Untitled"

  const folderPath = useMemo(() => {
    const folderMap = new Map(folders.map((folder) => [folder.id, folder]))
    const segments: string[] = []
    let currentFolderId = request.folderId
    const visited = new Set<string>()

    while (currentFolderId && !visited.has(currentFolderId)) {
      visited.add(currentFolderId)
      const folder = folderMap.get(currentFolderId)
      if (!folder) break
      segments.unshift(folder.name || "Untitled Folder")
      currentFolderId = folder.parentId
    }

    return segments
  }, [folders, request.folderId])

  const projectName = useMemo(() => {
    if (!currentProjectId) return "Project"
    return projects.find((project) => project.id === currentProjectId)?.name || "Project"
  }, [currentProjectId, projects])

  const breadcrumbSegments = useMemo(() => [...(projectName ? [projectName] : []), ...folderPath], [folderPath, projectName])

  useEffect(() => {
    setIsEditingName(false)
    setNameDraft(requestName)
  }, [activeTab.id, requestName])

  useEffect(() => {
    if (!isEditingName) return
    nameInputRef.current?.focus()
    nameInputRef.current?.select()
  }, [isEditingName])

  const commitRename = useCallback(async () => {
    const nextName = nameDraft.trim()
    setIsEditingName(false)

    if (!nextName || nextName === requestName) {
      setNameDraft(requestName)
      return
    }

    const nextRequest = {
      ...activeTab.request,
      name: nextName,
      updatedAt: new Date().toISOString(),
    }

    updateTab(activeTab.id, {
      title: nextName,
      request: nextRequest,
    })

    if (activeTab.requestId) {
      await renameRequest(activeTab.requestId, nextName)
    }
  }, [activeTab.id, activeTab.request, activeTab.requestId, nameDraft, renameRequest, requestName, updateTab])

  return (
    <div className="flex flex-col flex-shrink-0 bg-[var(--surface)]">
      <div className="px-[var(--size-padding)] pt-1">
        <div className={cn("h-7 max-w-full rounded-[8px] px-2", "flex items-center gap-1 overflow-hidden")}>
          <span className="inline-flex h-4 min-w-[24px] items-center justify-center rounded-[4px] bg-sky-500/15 px-1 text-[9px] font-mono font-semibold text-sky-600 dark:text-sky-400">
            HTTP
          </span>

          {breadcrumbSegments.map((segment, index) => (
            <div key={`${segment}-${index}`} className="flex min-w-0 items-center">
              <span
                className={cn(
                  "inline-flex h-6 max-w-[220px] items-center rounded-[6px] px-1.5",
                  "truncate text-[12px] text-[var(--fg-muted)]",
                  "hover:bg-[var(--selected-bg)] transition-colors"
                )}
                title={segment}
              >
                {segment}
              </span>
              <AppIcon name="arrowRight" size={10} className="mx-0.5 text-[var(--fg-muted)]" />
            </div>
          ))}

          {isEditingName ? (
            <input
              ref={nameInputRef}
              value={nameDraft}
              onChange={(event) => setNameDraft(event.target.value)}
              onBlur={() => void commitRename()}
              onKeyDown={(event) => {
                if (event.key === "Enter") void commitRename()
                if (event.key === "Escape") {
                  setNameDraft(requestName)
                  setIsEditingName(false)
                }
              }}
              className={cn(
                "h-6 rounded-[6px] bg-transparent px-1.5 text-[12px] font-semibold text-[var(--fg)]",
                "outline-none ring-1 ring-[var(--accent)]/40"
              )}
              style={{ width: `${Math.max(6, Math.min(56, nameDraft.trim().length || 8))}ch` }}
            />
          ) : (
            <button
              type="button"
              onClick={() => setIsEditingName(true)}
              className={cn(
                "inline-flex h-6 max-w-[560px] items-center rounded-[6px] px-1.5",
                "text-left text-[12px] font-semibold text-[var(--fg)]",
                "hover:bg-[var(--selected-bg)] transition-colors"
              )}
              title="单击编辑请求名称"
            >
              <span className="block truncate">{requestName}</span>
            </button>
          )}
        </div>
      </div>

      <div
        className={cn(
          "flex items-center gap-[var(--size-gap)] px-[var(--size-padding)] py-1",
          "bg-[var(--surface)]"
        )}
      >
        <div className="flex flex-1 items-center overflow-hidden rounded-[10px] border border-[var(--button-border)] bg-[var(--surface)] focus-within:border-[var(--accent)] focus-within:ring-1 focus-within:ring-[var(--accent)]/20">
          <Select value={request.method} onValueChange={(value) => updateTabRequest(activeTab.id, { method: value as HttpMethod })}>
            <SelectTrigger
              className={cn(
                "h-[30px] w-[96px] rounded-none border-0 border-r border-[var(--button-border)] bg-transparent px-3",
                "text-[11px] font-mono font-semibold shadow-none focus:ring-0",
                "justify-between",
                METHOD_COLORS[request.method as HttpMethod]
              )}
            >
              <SelectValue placeholder="方法" />
            </SelectTrigger>
            <SelectContent>
              {HTTP_METHODS.map((method) => (
                <SelectItem
                  key={method}
                  value={method}
                  className={cn(
                    "rounded-[8px] py-1.5 px-2 text-[11px] font-mono font-semibold",
                    METHOD_COLORS[method]
                  )}
                >
                  {method}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          <input
            value={request.url}
            onChange={(e) => updateTabRequest(activeTab.id, { url: e.target.value })}
            placeholder="输入请求 URL..."
            className={cn(
              "h-[30px] min-w-0 flex-1 bg-transparent px-3",
              "border-0 text-[var(--fg)] font-mono text-[length:var(--size-font-xs)]",
              "placeholder:text-[var(--fg-muted)] focus:outline-none focus:ring-0"
            )}
            onKeyDown={(e) => {
              if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) onSend()
            }}
          />
        </div>

        <button
          className={cn(
            "h-[var(--size-btn)] px-3 flex items-center justify-center gap-1 rounded-[var(--radius-btn)]",
            "text-[length:var(--size-font-2xs)] font-medium transition-colors",
            "hover:bg-[var(--sidebar-hover)]",
            activeTab.dirty ? "text-[var(--fg-secondary)]" : "text-[var(--fg-muted)]"
          )}
          onClick={onSave}
          disabled={!activeTab.dirty}
          title="保存 (⌘S)"
        >
          <AppIcon name="save" size={12} strokeWidth={1.9} />
          Save
        </button>

        {/* 发送/取消按钮组 */}
        <div className="relative flex items-center flex-shrink-0 w-[112px]" ref={sendMenuRef}>
          {isSending ? (
            <button
              onClick={onCancel}
              className={cn(
                "no-press-feedback h-[30px] w-full flex items-center justify-center gap-1.5 rounded-[8px]",
                "bg-[rgb(230,230,230)] text-[var(--fg)] text-[13px] font-medium",
                "hover:bg-[rgb(222,222,222)] active:bg-[rgb(214,214,214)] transition-[background-color,color]"
              )}
            >
              Cancel
            </button>
          ) : (
            <>
              <button
                onClick={() => onSend()}
                disabled={!request.url.trim()}
                className={cn(
                  "no-press-feedback h-[30px] flex-1 flex items-center justify-center gap-1.5 rounded-l-[8px]",
                  "bg-[#3b82f6] text-white text-[13px] font-medium",
                  "hover:bg-[#3477e6] active:bg-[#2f6ed6] transition-[background-color,color]",
                  "disabled:opacity-50 disabled:pointer-events-none"
                )}
              >
                Send
              </button>
              <button
                onClick={() => setShowSendMenu(!showSendMenu)}
                disabled={!request.url.trim()}
                className={cn(
                  "no-press-feedback relative h-[30px] w-[28px] flex items-center justify-center rounded-r-[8px]",
                  "before:pointer-events-none before:absolute before:left-0 before:top-[6px] before:h-[18px] before:w-px before:bg-white/25",
                  "bg-[#3b82f6] text-white",
                  "hover:bg-[#3477e6] active:bg-[#2f6ed6] transition-[background-color,color]",
                  "disabled:opacity-50 disabled:pointer-events-none"
                )}
              >
                <AppIcon name="arrowDown" size={10} />
              </button>
            </>
          )}

          {showSendMenu && (
            <div
              className={cn(
                "absolute right-0 top-full mt-1 z-50 py-1 rounded-[var(--radius-menu)] shadow-lg border",
                "w-max bg-[var(--surface-elevated)] border-[var(--border-color)]"
              )}
            >
              <button
                className="no-press-feedback w-full whitespace-nowrap px-3 py-1.5 text-[11px] text-left hover:bg-[var(--sidebar-hover)] text-[var(--fg)] flex items-center gap-2 transition-[background-color,color] rounded-[7px] mx-1"
                onClick={() => { setShowSendMenu(false); onSend(true) }}
              >
                <AppIcon name="download" size={12} />
                Send and Download
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
