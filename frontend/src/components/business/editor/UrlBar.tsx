import { useState, useRef, useEffect, useMemo, useCallback } from "react"
import { HTTP_METHODS, METHOD_COLORS, type HttpMethod } from "@/lib/constants"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { AppIcon } from "@/components/ui/icon"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { useTabStore, getProjectActiveTabFromState } from "@/stores/tabStore"
import { useProjectStore } from "@/stores/projectStore"
import { useEnvironmentStore } from "@/stores/environmentStore"
import { useI18n } from "@/hooks/useI18n"
import { cn } from "@/lib/utils"

interface UrlBarProps {
  onSend: (downloadAfter?: boolean) => void
  onCancel: () => void
  onSave: () => void
}

type VariableCompletionContext = {
  replaceStart: number
  caret: number
  query: string
}

function getVariableCompletionContext(url: string, caret: number): VariableCompletionContext | null {
  if (caret < 0) return null
  const before = url.slice(0, caret)
  const match = /(?:^|[^A-Za-z0-9_}])(\{\{?)([A-Za-z0-9_.-]*)$/.exec(before)
  if (!match) return null

  const prefix = match[1]
  const query = match[2] ?? ""
  const replaceStart = caret - prefix.length - query.length
  if (replaceStart < 0) return null

  return {
    replaceStart,
    caret,
    query,
  }
}

function tokenizeUrl(url: string): Array<{ type: "text" | "variable"; value: string }> {
  const parts = url.split(/(\{\{[^{}]+\}\})/g).filter(Boolean)
  return parts.map((part) => {
    if (part.startsWith("{{") && part.endsWith("}}")) {
      return { type: "variable" as const, value: part.slice(2, -2).trim() }
    }
    return { type: "text" as const, value: part }
  })
}

export function UrlBar({ onSend, onCancel, onSave }: UrlBarProps) {
  const { t } = useI18n()
  const activeTab = useTabStore(getProjectActiveTabFromState)
  const updateTabRequest = useTabStore((s) => s.updateTabRequest)
  const updateTab = useTabStore((s) => s.updateTab)
  const environments = useEnvironmentStore((s) => s.environments)
  const activeEnvironmentId = useEnvironmentStore((s) => s.activeEnvironmentId)
  const projects = useProjectStore((s) => s.projects)
  const currentProjectId = useProjectStore((s) => s.currentProjectId)
  const folders = useProjectStore((s) => s.folders)
  const renameRequest = useProjectStore((s) => s.renameRequest)
  const [showSendMenu, setShowSendMenu] = useState(false)
  const [isEditingName, setIsEditingName] = useState(false)
  const [nameDraft, setNameDraft] = useState("")
  const [urlCaret, setUrlCaret] = useState(0)
  const [urlScrollLeft, setUrlScrollLeft] = useState(0)
  const [urlFocused, setUrlFocused] = useState(false)
  const [variableActiveIndex, setVariableActiveIndex] = useState(0)
  const [nameInputWidth, setNameInputWidth] = useState(96)
  const sendMenuRef = useRef<HTMLDivElement>(null)
  const nameInputRef = useRef<HTMLInputElement>(null)
  const nameContainerRef = useRef<HTMLDivElement>(null)
  const nameMeasureRef = useRef<HTMLSpanElement>(null)
  const urlInputRef = useRef<HTMLInputElement>(null)

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
  const isSending = activeTab.isSending
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
  const variableKeys = useMemo(() => {
    const seen = new Set<string>()
    const ordered: string[] = []
    const append = (rawKey: string) => {
      const key = rawKey.trim()
      if (!key || seen.has(key)) return
      seen.add(key)
      ordered.push(key)
    }

    const activeEnv = environments.find((env) => env.id === activeEnvironmentId)
    ;(activeEnv?.variables ?? []).forEach((variable) => {
      if (variable.enabled) append(variable.key)
    })

    environments.forEach((env) => {
      ;(env.variables ?? []).forEach((variable) => {
        if (variable.enabled) append(variable.key)
      })
    })

    return ordered
  }, [activeEnvironmentId, environments])

  const urlTokens = useMemo(() => tokenizeUrl(request.url), [request.url])
  const completionContext = useMemo(
    () => (urlFocused ? getVariableCompletionContext(request.url, urlCaret) : null),
    [request.url, urlCaret, urlFocused]
  )
  const variableSuggestions = useMemo(() => {
    if (!completionContext) return []
    const query = completionContext.query.trim().toLowerCase()
    if (!query) return variableKeys.slice(0, 8)

    const startsWithMatches = variableKeys.filter((key) => key.toLowerCase().startsWith(query))
    const containsMatches = variableKeys.filter((key) => !key.toLowerCase().startsWith(query) && key.toLowerCase().includes(query))
    return [...startsWithMatches, ...containsMatches].slice(0, 8)
  }, [completionContext, variableKeys])
  const showVariableSuggestion = variableSuggestions.length > 0 && Boolean(completionContext)

  useEffect(() => {
    setIsEditingName(false)
    setNameDraft(requestName)
    setUrlCaret(request.url.length)
    setUrlScrollLeft(0)
    setVariableActiveIndex(0)
  }, [activeTab.id, request.url.length, requestName])

  useEffect(() => {
    if (!isEditingName) return
    nameInputRef.current?.focus()
    nameInputRef.current?.select()
  }, [isEditingName])

  const recalcNameInputWidth = useCallback(() => {
    const container = nameContainerRef.current
    const measure = nameMeasureRef.current
    if (!container || !measure) return

    const availableWidth = Math.floor(container.clientWidth)
    if (availableWidth <= 0) return

    const sampleText = (isEditingName ? nameDraft : requestName).trim() || "Untitled"
    measure.textContent = sampleText
    const textWidth = Math.ceil(measure.getBoundingClientRect().width)

    const minWidth = isEditingName ? 96 : 28
    const paddingAndCursor = 18
    const desiredWidth = textWidth + paddingAndCursor
    const nextWidth = Math.min(availableWidth, Math.max(minWidth, desiredWidth))
    setNameInputWidth(nextWidth)
  }, [isEditingName, nameDraft, requestName])

  useEffect(() => {
    recalcNameInputWidth()

    const container = nameContainerRef.current
    if (!container || typeof ResizeObserver === "undefined") return

    const observer = new ResizeObserver(() => {
      recalcNameInputWidth()
    })
    observer.observe(container)
    return () => observer.disconnect()
  }, [recalcNameInputWidth])

  useEffect(() => {
    if (!showVariableSuggestion) {
      setVariableActiveIndex(0)
      return
    }
    setVariableActiveIndex((prev) => Math.min(prev, variableSuggestions.length - 1))
  }, [showVariableSuggestion, variableSuggestions.length])

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

  const applyVariableSuggestion = useCallback((variableKey: string) => {
    if (!completionContext) return

    const current = request.url
    const before = current.slice(0, completionContext.replaceStart)
    const afterRaw = current.slice(completionContext.caret)
    const skipClosing = afterRaw.startsWith("}}") ? 2 : afterRaw.startsWith("}") ? 1 : 0
    const after = afterRaw.slice(skipClosing)
    const inserted = `{{${variableKey}}}`
    const nextUrl = `${before}${inserted}${after}`
    const nextCaret = before.length + inserted.length

    updateTabRequest(activeTab.id, { url: nextUrl })
    setUrlCaret(nextCaret)

    requestAnimationFrame(() => {
      const input = urlInputRef.current
      if (!input) return
      input.focus()
      input.setSelectionRange(nextCaret, nextCaret)
    })
  }, [activeTab.id, completionContext, request.url, updateTabRequest])

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

          <div ref={nameContainerRef} className="relative min-w-0 flex-1">
            <span
              ref={nameMeasureRef}
              aria-hidden="true"
              className="pointer-events-none absolute -left-[9999px] top-0 whitespace-pre px-1.5 text-[12px] font-semibold text-transparent"
            />
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
                  "block h-6 min-w-0 max-w-full rounded-[6px] bg-transparent px-1.5 text-[12px] font-semibold text-[var(--fg)]",
                  "outline-none ring-1 ring-[var(--accent)]/40"
                )}
                style={{ width: `${nameInputWidth}px` }}
              />
            ) : (
              <Tooltip>
                <TooltipTrigger asChild>
                  <button
                    type="button"
                    onClick={() => setIsEditingName(true)}
                    className={cn(
                      "inline-flex h-6 min-w-0 max-w-full items-center rounded-[6px] px-1.5",
                      "text-left text-[12px] font-semibold text-[var(--fg)]",
                      "hover:bg-[var(--selected-bg)] transition-colors"
                    )}
                    style={{ width: `${nameInputWidth}px` }}
                  >
                    <span className="min-w-0 flex-1 truncate">{requestName}</span>
                  </button>
                </TooltipTrigger>
                <TooltipContent>{t("单击编辑请求名称", "Click to edit request name")}</TooltipContent>
              </Tooltip>
            )}
          </div>
        </div>
      </div>

      <div
        className={cn(
          "flex min-w-0 items-center gap-[var(--size-gap)] px-[var(--size-padding)] py-1",
          "bg-[var(--surface)]"
        )}
      >
        <div className="flex min-w-0 flex-1 items-center rounded-[10px] border border-[var(--button-border)] bg-[var(--surface)] focus-within:border-[var(--accent)] focus-within:ring-1 focus-within:ring-[var(--accent)]/20">
          <Select value={request.method} onValueChange={(value) => updateTabRequest(activeTab.id, { method: value as HttpMethod })}>
            <SelectTrigger
              className={cn(
                "h-[30px] w-[96px] rounded-none border-0 border-r border-[var(--button-border)] bg-transparent px-3",
                "text-[11px] font-mono font-semibold shadow-none focus:ring-0",
                "justify-between",
                METHOD_COLORS[request.method as HttpMethod]
              )}
            >
              <SelectValue placeholder={t("方法", "Method")} />
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

          <div className="relative h-[30px] min-w-0 flex-1">
            <div
              className={cn(
                "pointer-events-none absolute inset-0 z-[1] flex items-center overflow-hidden px-3",
                "font-mono text-[length:var(--size-font-xs)] text-[var(--fg)]"
              )}
            >
              {request.url ? (
                <div className="whitespace-pre" style={{ transform: `translateX(${-urlScrollLeft}px)` }}>
                  {urlTokens.map((token, index) => {
                    if (token.type === "variable") {
                      return (
                        <span
                          key={`${token.type}-${index}-${token.value}`}
                          className={cn(
                            "mx-[1px] inline-flex h-4 items-center rounded-[999px] px-1.5 align-middle",
                            "border border-[var(--accent)]/30 bg-[var(--accent)]/12",
                            "text-[10px] font-semibold text-[var(--accent)]"
                          )}
                        >
                          {`{{${token.value}}}`}
                        </span>
                      )
                    }
                    return <span key={`${token.type}-${index}-${token.value}`}>{token.value}</span>
                  })}
                </div>
              ) : null}
            </div>

            <input
              ref={urlInputRef}
              value={request.url}
              onChange={(e) => {
                updateTabRequest(activeTab.id, { url: e.target.value })
                setUrlCaret(e.target.selectionStart ?? e.target.value.length)
              }}
              onFocus={(e) => {
                setUrlFocused(true)
                setUrlCaret(e.target.selectionStart ?? e.target.value.length)
              }}
              onBlur={() => {
                setUrlFocused(false)
              }}
              onClick={(e) => setUrlCaret(e.currentTarget.selectionStart ?? e.currentTarget.value.length)}
              onKeyUp={(e) => {
                const target = e.currentTarget
                setUrlCaret(target.selectionStart ?? target.value.length)
              }}
              onSelect={(e) => {
                const target = e.currentTarget
                setUrlCaret(target.selectionStart ?? target.value.length)
              }}
              onScroll={(e) => setUrlScrollLeft(e.currentTarget.scrollLeft)}
              placeholder={t("输入请求 URL...", "Enter request URL...")}
              className={cn(
                "relative z-[2] h-[30px] w-full min-w-0 bg-transparent px-3",
                "border-0 font-mono text-[length:var(--size-font-xs)] caret-[var(--fg)]",
                request.url ? "text-transparent" : "text-[var(--fg)]",
                "placeholder:text-[var(--fg-muted)] focus:outline-none focus:ring-0"
              )}
              onKeyDown={(e) => {
                if (showVariableSuggestion && e.key === "ArrowDown") {
                  e.preventDefault()
                  setVariableActiveIndex((prev) => (prev + 1) % variableSuggestions.length)
                  return
                }
                if (showVariableSuggestion && e.key === "ArrowUp") {
                  e.preventDefault()
                  setVariableActiveIndex((prev) => (prev - 1 + variableSuggestions.length) % variableSuggestions.length)
                  return
                }
                if (showVariableSuggestion && (e.key === "Enter" || e.key === "Tab") && !e.metaKey && !e.ctrlKey) {
                  e.preventDefault()
                  applyVariableSuggestion(variableSuggestions[variableActiveIndex] ?? variableSuggestions[0])
                  return
                }
                if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) onSend()
              }}
            />

            {showVariableSuggestion && (
              <div
                className={cn(
                  "absolute left-2 right-2 top-[calc(100%+4px)] z-[30] overflow-hidden rounded-[8px] border",
                  "border-[var(--border-color)] bg-[var(--surface-elevated)] shadow-[var(--shadow-lg)]"
                )}
              >
                {variableSuggestions.map((key, index) => (
                  <button
                    key={key}
                    type="button"
                    className={cn(
                      "flex h-7 w-full items-center px-2 text-left text-[11px] font-mono",
                      index === variableActiveIndex
                        ? "bg-[var(--selected-bg)] text-[var(--fg)]"
                        : "text-[var(--fg-secondary)] hover:bg-[var(--sidebar-hover)]"
                    )}
                    onMouseDown={(event) => {
                      event.preventDefault()
                      applyVariableSuggestion(key)
                    }}
                  >
                    {`{{${key}}}`}
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>

        <Tooltip>
          <TooltipTrigger asChild>
            <button
              className={cn(
                "h-[var(--size-btn)] flex-shrink-0 px-3 flex items-center justify-center gap-1 rounded-[var(--radius-btn)]",
                "text-[length:var(--size-font-2xs)] font-medium transition-colors",
                "hover:bg-[var(--sidebar-hover)]",
                activeTab.dirty ? "text-[var(--fg-secondary)]" : "text-[var(--fg-muted)]"
              )}
              onClick={onSave}
              disabled={!activeTab.dirty}
            >
              <AppIcon name="save" size={12} strokeWidth={1.9} />
              Save
            </button>
          </TooltipTrigger>
          <TooltipContent>{t("保存", "Save")} (⌘S)</TooltipContent>
        </Tooltip>

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
                  "bg-[var(--accent)] text-[var(--accent-fg)] text-[13px] font-medium",
                  "hover:bg-[var(--accent-hover)] active:brightness-[0.96] transition-[background-color,color,filter]",
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
                  "bg-[var(--accent)] text-[var(--accent-fg)]",
                  "hover:bg-[var(--accent-hover)] active:brightness-[0.96] transition-[background-color,color,filter]",
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
