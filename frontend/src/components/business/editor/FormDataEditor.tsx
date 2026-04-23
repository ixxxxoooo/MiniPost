import { useState, useEffect, useLayoutEffect, useRef, useMemo } from "react"
import { createPortal } from "react-dom"
import { AppIcon } from "@/components/ui/icon"
import { cn } from "@/lib/utils"
import { useTabStore, getProjectActiveTabFromState } from "@/stores/tabStore"
import type { FormDataItem } from "@/types/request"
import { createFormDataItem } from "@/types/request"
import { useI18n } from "@/hooks/useI18n"

interface FileHistoryItem {
  path: string
  name: string
  lastUsedAt: string
}

const FILE_UPLOAD_HISTORY_STORAGE_KEY = "minipost:file-upload-history"
const FILE_UPLOAD_HISTORY_LIMIT = 12
const DROPDOWN_CHAR_WIDTH = 7

function readFileUploadHistory(): FileHistoryItem[] {
  if (typeof window === "undefined") return []
  try {
    const raw = window.localStorage.getItem(FILE_UPLOAD_HISTORY_STORAGE_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw) as unknown
    if (!Array.isArray(parsed)) return []
    return parsed
      .filter((entry): entry is FileHistoryItem => (
        !!entry
        && typeof entry === "object"
        && typeof (entry as { path?: unknown }).path === "string"
        && typeof (entry as { name?: unknown }).name === "string"
        && typeof (entry as { lastUsedAt?: unknown }).lastUsedAt === "string"
      ))
      .slice(0, FILE_UPLOAD_HISTORY_LIMIT)
  } catch {
    return []
  }
}

function persistFileUploadHistory(history: FileHistoryItem[]) {
  if (typeof window === "undefined") return
  try {
    window.localStorage.setItem(FILE_UPLOAD_HISTORY_STORAGE_KEY, JSON.stringify(history.slice(0, FILE_UPLOAD_HISTORY_LIMIT)))
  } catch {
    // ignore persistence errors
  }
}

function upsertFileHistory(history: FileHistoryItem[], path: string, name: string): FileHistoryItem[] {
  const nextItem: FileHistoryItem = {
    path,
    name,
    lastUsedAt: new Date().toISOString(),
  }
  return [
    nextItem,
    ...history.filter((entry) => entry.path !== path),
  ].slice(0, FILE_UPLOAD_HISTORY_LIMIT)
}

export function FormDataEditor() {
  const { t, isZh } = useI18n()
  const activeTab = useTabStore(getProjectActiveTabFromState)
  const updateTabRequest = useTabStore((s) => s.updateTabRequest)
  const [fileHistory, setFileHistory] = useState<FileHistoryItem[]>(() => readFileUploadHistory())
  const tabId = activeTab?.id ?? ""
  const body = activeTab?.request.body ?? { type: "none" as const }
  const items: FormDataItem[] = body.formData ?? []

  const setItems = (formData: FormDataItem[]) => {
    if (!activeTab) return
    updateTabRequest(tabId, { body: { ...body, formData } })
  }

  useEffect(() => {
    if (!activeTab) return
    if (items.length === 0) {
      setItems([createFormDataItem()])
    }
  }, [activeTab, items.length])

  const handleUpdate = (id: string, field: keyof FormDataItem, value: string | boolean) => {
    setItems(items.map((item) => (item.id === id ? { ...item, [field]: value } : item)))
  }

  const handleDelete = (id: string) => {
    setItems(items.filter((item) => item.id !== id))
  }

  const handleNewRowKeyChange = (id: string, value: string) => {
    const updated = items.map((item) => (item.id === id ? { ...item, key: value } : item))
    if (value) updated.push(createFormDataItem())
    setItems(updated)
  }

  const handleTypeChange = (id: string, type: "text" | "file") => {
    setItems(items.map((item) =>
      item.id === id ? { ...item, type, value: "", filePath: undefined, fileName: undefined } : item
    ))
  }

  const rememberFileHistory = (path: string, name: string) => {
    setFileHistory((prev) => {
      const next = upsertFileHistory(prev, path, name)
      persistFileUploadHistory(next)
      return next
    })
  }

  const handleSelectFile = async (id: string): Promise<boolean> => {
    try {
      const { OpenFileDialogAny } = await import("../../../../wailsjs/go/main/App")
      const filePath = await OpenFileDialogAny()
      if (!filePath) return false
      const fileName = filePath.split("/").pop() || filePath.split("\\").pop() || filePath
      setItems(items.map((item) =>
        item.id === id ? { ...item, filePath, fileName, value: filePath } : item
      ))
      rememberFileHistory(filePath, fileName)
      return true
    } catch (err) {
      console.error(t("文件选择失败:", "Failed to choose file:"), err)
      return false
    }
  }

  const handleSelectHistoryFile = (id: string, file: FileHistoryItem) => {
    setItems(items.map((item) =>
      item.id === id ? { ...item, filePath: file.path, fileName: file.name, value: file.path } : item
    ))
    rememberFileHistory(file.path, file.name)
  }

  if (!activeTab) return null

  return (
    <div className="px-[var(--size-padding-sm)]">
      <table className="w-full border-collapse border border-[var(--border-color)]" style={{ fontSize: "11px" }}>
        <thead>
          <tr>
            <th className="w-7 border border-[var(--border-color)] px-1 py-1" />
            <th className="text-left border border-[var(--border-color)] px-2 py-1 font-semibold text-[var(--fg-secondary)]">{t("键", "Key")}</th>
            <th className="w-[72px] border border-[var(--border-color)] px-1 py-1 font-semibold text-[var(--fg-secondary)] text-center">{t("类型", "Type")}</th>
            <th className="text-left border border-[var(--border-color)] px-2 py-1 font-semibold text-[var(--fg-secondary)]">{t("值", "Value")}</th>
            <th className="text-left border border-[var(--border-color)] px-2 py-1 font-semibold text-[var(--fg-secondary)]">{t("说明", "Description")}</th>
            <th className="w-8 border border-[var(--border-color)] px-1 py-1" />
          </tr>
        </thead>
        <tbody>
          {items.map((item, idx) => {
            const isLastEmpty = idx === items.length - 1 && !item.key && !item.value
            return (
              <tr key={item.id} className="group hover:bg-[var(--surface-secondary)]/50 transition-colors">
                <td className="border border-[var(--border-color)] px-1 py-0 text-center">
                  <input
                    type="checkbox"
                    checked={item.enabled}
                    onChange={(e) => handleUpdate(item.id, "enabled", e.target.checked)}
                    className="w-3 h-3 rounded accent-[var(--accent)] cursor-pointer"
                    disabled={isLastEmpty}
                    style={isLastEmpty ? { opacity: 0.3 } : {}}
                  />
                </td>
                <td className="border border-[var(--border-color)] px-0 py-0">
                  <input
                    value={item.key}
                    onChange={(e) => {
                      if (isLastEmpty) handleNewRowKeyChange(item.id, e.target.value)
                      else handleUpdate(item.id, "key", e.target.value)
                    }}
                    placeholder={t("键", "Key")}
                    className={cn(
                      "w-full h-[24px] px-2 bg-transparent text-[var(--fg)] font-mono",
                      "text-[11px] placeholder:text-[var(--fg-muted)] placeholder:italic",
                      "focus:outline-none",
                      !item.enabled && !isLastEmpty && "opacity-40"
                    )}
                  />
                </td>
                <td className="border border-[var(--border-color)] px-1 py-0 text-center">
                  {!isLastEmpty ? (
                    <TypeDropdown value={item.type} onChange={(nextType) => handleTypeChange(item.id, nextType)} isZh={isZh} />
                  ) : (
                    <span className="text-[10px] text-[var(--fg-muted)]">{t("文本", "Text")}</span>
                  )}
                </td>
                <td className="border border-[var(--border-color)] px-0 py-0">
                  {item.type === "file" ? (
                    <FileValueDropdown
                      value={item.fileName || ""}
                      valuePath={item.filePath || ""}
                      history={fileHistory}
                      isZh={isZh}
                      onUploadLocal={() => handleSelectFile(item.id)}
                      onSelectHistory={(file) => handleSelectHistoryFile(item.id, file)}
                    />
                  ) : (
                    <input
                      value={item.value}
                      onChange={(e) => handleUpdate(item.id, "value", e.target.value)}
                      placeholder={t("值", "Value")}
                      className={cn(
                        "w-full h-[24px] px-2 bg-transparent text-[var(--fg)] font-mono",
                        "text-[11px] placeholder:text-[var(--fg-muted)] placeholder:italic",
                        "focus:outline-none",
                        !item.enabled && !isLastEmpty && "opacity-40"
                      )}
                      disabled={isLastEmpty}
                    />
                  )}
                </td>
                <td className="border border-[var(--border-color)] px-0 py-0">
                  <input
                    value={item.description ?? ""}
                    onChange={(e) => handleUpdate(item.id, "description", e.target.value)}
                    placeholder={t("说明", "Description")}
                    className={cn(
                      "w-full h-[24px] px-2 bg-transparent text-[var(--fg-secondary)]",
                      "text-[11px] placeholder:text-[var(--fg-muted)] placeholder:italic",
                      "focus:outline-none",
                      !item.enabled && !isLastEmpty && "opacity-40"
                    )}
                    disabled={isLastEmpty}
                  />
                </td>
                <td className="border border-[var(--border-color)] px-1 py-0 text-center">
                  {!isLastEmpty && (
                    <button
                      className="h-4 w-4 inline-flex items-center justify-center rounded-[2px] opacity-0 group-hover:opacity-100 hover:bg-[var(--sidebar-hover)] text-[var(--fg-muted)] hover:text-[var(--danger)] transition-all"
                      onClick={() => handleDelete(item.id)}
                    >
                      <AppIcon name="delete" size={10} />
                    </button>
                  )}
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

function TypeDropdown({
  value,
  onChange,
  isZh,
}: {
  value: "text" | "file"
  onChange: (v: "text" | "file") => void
  isZh: boolean
}) {
  const t = (zh: string, en: string) => (isZh ? zh : en)
  const [open, setOpen] = useState(false)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const [menuRect, setMenuRect] = useState<{ top: number; left: number; width: number }>({
    top: 0,
    left: 0,
    width: 72,
  })

  const updateMenuRect = () => {
    if (!triggerRef.current) return
    const rect = triggerRef.current.getBoundingClientRect()
    const labels = [t("文本", "Text"), t("文件", "File")]
    const contentWidth = labels.reduce((max, label) => Math.max(max, label.length), 0) * DROPDOWN_CHAR_WIDTH + 40
    setMenuRect({
      top: rect.bottom + 4,
      left: rect.left,
      width: Math.max(rect.width, contentWidth),
    })
  }

  useLayoutEffect(() => {
    if (!open) return
    updateMenuRect()

    const handleResize = () => updateMenuRect()
    const handleScroll = () => updateMenuRect()
    window.addEventListener("resize", handleResize)
    window.addEventListener("scroll", handleScroll, true)
    return () => {
      window.removeEventListener("resize", handleResize)
      window.removeEventListener("scroll", handleScroll, true)
    }
  }, [open])

  return (
    <div className="relative inline-block">
      <button
        ref={triggerRef}
        type="button"
        className={cn(
          "h-[24px] px-2 rounded-[4px] text-[10px] font-medium flex items-center gap-1 transition-colors",
          "border border-[var(--border-color)] bg-[var(--surface)]",
          "text-[var(--fg-secondary)] hover:border-[var(--accent)] hover:text-[var(--fg)]"
        )}
        onClick={() => setOpen(!open)}
      >
        {value === "text" ? t("文本", "Text") : t("文件", "File")}
        <AppIcon name="arrowDown" size={8} strokeWidth={2} className="text-[var(--fg-muted)]" />
      </button>
      {open && createPortal(
        <>
          <div className="fixed inset-0 z-[260]" onClick={() => setOpen(false)} />
          <div
            className={cn(
              "fixed z-[261] py-0.5 rounded-[6px] shadow-lg border",
              "bg-[var(--surface-elevated)] border-[var(--border-color)]"
            )}
            style={{ top: menuRect.top, left: menuRect.left, width: `${menuRect.width}px` }}
          >
            {(["text", "file"] as const).map((t) => (
              <button
                key={t}
                type="button"
                className={cn(
                  "w-full px-2.5 py-1 text-[10px] text-left transition-colors",
                  value === t
                    ? "bg-[var(--selected-bg)] text-[var(--fg)] font-medium"
                    : "text-[var(--fg)] hover:bg-[var(--sidebar-hover)]"
                )}
                onClick={() => { onChange(t); setOpen(false) }}
              >
                {t === "text" ? (isZh ? "文本" : "Text") : (isZh ? "文件" : "File")}
              </button>
            ))}
          </div>
        </>,
        document.body
      )}
    </div>
  )
}

function FileValueDropdown({
  value,
  valuePath,
  history,
  isZh,
  onUploadLocal,
  onSelectHistory,
}: {
  value: string
  valuePath: string
  history: FileHistoryItem[]
  isZh: boolean
  onUploadLocal: () => Promise<boolean>
  onSelectHistory: (file: FileHistoryItem) => void
}) {
  const t = (zh: string, en: string) => (isZh ? zh : en)
  const [open, setOpen] = useState(false)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const [menuRect, setMenuRect] = useState<{ top: number; left: number; width: number }>({
    top: 0,
    left: 0,
    width: 260,
  })
  const contentWidth = useMemo(() => {
    const baseLabels = [
      t("上传本地文件", "Upload local file"),
      t("历史上传文件", "Recent uploads"),
      t("暂无历史文件", "No upload history"),
    ]
    const historyLongest = history.reduce((max, item) => Math.max(max, item.name.length), 0)
    const baseLongest = baseLabels.reduce((max, label) => Math.max(max, label.length), 0)
    return Math.max(historyLongest, baseLongest) * DROPDOWN_CHAR_WIDTH + 84
  }, [history, isZh])

  const updateMenuRect = () => {
    if (!triggerRef.current) return
    const rect = triggerRef.current.getBoundingClientRect()
    setMenuRect({
      top: rect.bottom + 6,
      left: rect.left,
      width: Math.max(rect.width, contentWidth),
    })
  }

  useLayoutEffect(() => {
    if (!open) return
    updateMenuRect()
    const handleResize = () => updateMenuRect()
    const handleScroll = () => updateMenuRect()
    window.addEventListener("resize", handleResize)
    window.addEventListener("scroll", handleScroll, true)
    return () => {
      window.removeEventListener("resize", handleResize)
      window.removeEventListener("scroll", handleScroll, true)
    }
  }, [open, contentWidth])

  return (
    <div className="relative">
      <button
        ref={triggerRef}
        type="button"
        className={cn(
          "h-[24px] w-full px-2 rounded-[4px] border border-[var(--border-color)] bg-[var(--surface)]",
          "text-[11px] flex items-center gap-1.5 transition-colors",
          "text-[var(--fg)] hover:border-[var(--accent)]"
        )}
        onClick={() => setOpen((prev) => !prev)}
        title={value || t("选择文件", "Select files")}
      >
        <span className={cn("truncate text-left flex-1", value ? "text-[var(--fg)]" : "text-[var(--fg-muted)]")}>
          {value || t("选择文件", "Select files")}
        </span>
        <AppIcon name="arrowDown" size={9} strokeWidth={2} className="text-[var(--fg-muted)]" />
      </button>

      {open && createPortal(
        <>
          <div className="fixed inset-0 z-[260]" onClick={() => setOpen(false)} />
          <div
            className={cn(
              "fixed z-[261] rounded-[10px] border shadow-[var(--shadow-lg)] overflow-hidden",
              "border-[var(--border-color)] bg-[var(--surface-elevated)]"
            )}
            style={{ top: menuRect.top, left: menuRect.left, width: `${menuRect.width}px` }}
          >
            <div className="p-2">
              <button
                type="button"
                className={cn(
                  "w-full h-8 px-2.5 rounded-[7px] text-[12px] text-left flex items-center gap-2 transition-colors",
                  "text-[var(--fg)] hover:bg-[var(--sidebar-hover)]"
                )}
                onClick={() => {
                  void onUploadLocal().then((ok) => {
                    if (ok) setOpen(false)
                  })
                }}
              >
                <AppIcon name="add" size={12} className="text-[var(--fg-secondary)]" />
                {t("上传本地文件", "Upload local file")}
              </button>
            </div>

            <div className="h-px bg-[var(--border-subtle)] mx-2" />

            <div className="px-2 pt-2 pb-1 text-[10px] font-medium text-[var(--fg-muted)] uppercase tracking-wide">
              {t("历史上传文件", "Recent uploads")}
            </div>

            <div className="px-2 pb-2 max-h-[180px] overflow-y-auto">
              {history.length === 0 ? (
                <div className="px-2.5 py-2 text-[11px] text-[var(--fg-muted)]">{t("暂无历史文件", "No upload history")}</div>
              ) : (
                history.map((file) => {
                  const isSelected = !!valuePath && file.path === valuePath
                  return (
                    <button
                      key={file.path}
                      type="button"
                      className={cn(
                        "w-full px-2.5 py-1.5 rounded-[7px] text-left text-[12px] transition-colors",
                        isSelected
                          ? "bg-[var(--selected-bg)] text-[var(--fg)]"
                          : "text-[var(--fg)] hover:bg-[var(--sidebar-hover)]"
                      )}
                      onClick={() => { onSelectHistory(file); setOpen(false) }}
                      title={file.path}
                    >
                      <div className="truncate">{file.name}</div>
                    </button>
                  )
                })
              )}
            </div>
          </div>
        </>,
        document.body
      )}
    </div>
  )
}
