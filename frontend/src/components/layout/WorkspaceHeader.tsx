import { useEffect, useMemo, useRef, useState } from "react"
import { cn } from "@/lib/utils"
import { AppIcon } from "@/components/ui/icon"
import { useProjectStore } from "@/stores/projectStore"

const BUTTON_MIN_WIDTH = 92
const BUTTON_MAX_WIDTH = 180
const PANEL_MIN_WIDTH = 220
const PANEL_MAX_WIDTH = 320
const BUTTON_BASE_WIDTH = 54
const PANEL_BASE_WIDTH = 84
const CHAR_WIDTH = 11

export function WorkspaceHeader() {
  const { currentProjectId, projects, selectProject, createProject } = useProjectStore()
  const [open, setOpen] = useState(false)
  const [searchQuery, setSearchQuery] = useState("")
  const [newProjectName, setNewProjectName] = useState("")
  const ref = useRef<HTMLDivElement>(null)

  const current = projects.find((project) => project.id === currentProjectId)
  const currentLabel = current?.name || "选择项目"
  const longestProjectNameLength = useMemo(() => {
    return projects.reduce((max, project) => Math.max(max, project.name.length), currentLabel.length)
  }, [projects, currentLabel])

  const triggerWidth = useMemo(() => {
    return Math.min(BUTTON_MAX_WIDTH, Math.max(BUTTON_MIN_WIDTH, BUTTON_BASE_WIDTH + currentLabel.length * CHAR_WIDTH))
  }, [currentLabel])

  const panelWidth = useMemo(() => {
    return Math.min(PANEL_MAX_WIDTH, Math.max(PANEL_MIN_WIDTH, PANEL_BASE_WIDTH + longestProjectNameLength * CHAR_WIDTH))
  }, [longestProjectNameLength])

  useEffect(() => {
    if (!open) return

    const handleOutside = (event: MouseEvent) => {
      if (ref.current && !ref.current.contains(event.target as Node)) {
        setOpen(false)
      }
    }

    document.addEventListener("mousedown", handleOutside)
    return () => document.removeEventListener("mousedown", handleOutside)
  }, [open])

  const handleCreate = async () => {
    const name = searchQuery.trim() || newProjectName.trim()
    if (!name) return

    await createProject(name)
    const { projects: updatedProjects } = useProjectStore.getState()
    const project = updatedProjects[updatedProjects.length - 1]
    if (project) {
      await selectProject(project.id)
    }

    setSearchQuery("")
    setNewProjectName("")
    setOpen(false)
  }

  return (
    <div className="titlebar-no-drag flex items-center gap-1" ref={ref} onMouseDown={(event) => event.stopPropagation()}>
      <button
        className={cn(
          "flex h-[24px] w-[24px] items-center justify-center rounded-[7px] border transition-colors",
          "border-transparent bg-[var(--surface-secondary)] text-[var(--fg-secondary)] hover:bg-[var(--button-bg)]"
        )}
        title="主页"
        type="button"
      >
        <AppIcon name="home" size={15} strokeWidth={1.65} />
      </button>

      <div className="relative" style={{ width: `${triggerWidth}px`, maxWidth: `${BUTTON_MAX_WIDTH}px` }}>
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
            {current?.name || "选择项目"}
          </span>
          <AppIcon
            name="arrowDown"
            size={11}
            strokeWidth={1.9}
            className={cn("text-[var(--fg-muted)] transition-transform", open && "rotate-180")}
          />
        </button>

        {open && (
          <div
            className={cn(
              "absolute left-0 top-[calc(100%+5px)] z-[120] overflow-hidden rounded-[9px] border shadow-[var(--shadow-lg)]",
              "border-[var(--button-border)] bg-[var(--surface-elevated)]"
            )}
            style={{ width: `${Math.max(triggerWidth, panelWidth)}px`, maxWidth: `${PANEL_MAX_WIDTH}px` }}
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
                  placeholder="Search workspaces..."
                  value={searchQuery}
                  onChange={(event) => setSearchQuery(event.target.value)}
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
                Create
              </button>
            </div>

            <div className="max-h-[240px] overflow-y-auto p-1">
              {projects
                .filter((project) => project.name.toLowerCase().includes(searchQuery.toLowerCase()))
                .map((project) => {
                const selected = project.id === currentProjectId
                return (
                  <button
                    key={project.id}
                    className={cn(
                      "flex w-full items-center gap-2 rounded-[7px] px-2 py-1.5 text-left transition-colors",
                      selected
                        ? "bg-[var(--surface-secondary)] text-[var(--fg)]"
                        : "text-[var(--fg)] hover:bg-[var(--surface-secondary)]"
                    )}
                    onClick={() => {
                      void selectProject(project.id)
                      setOpen(false)
                    }}
                    type="button"
                  >
                    <div className="flex h-5 w-5 items-center justify-center rounded-[6px] bg-[var(--surface-secondary)] text-[var(--fg-secondary)]">
                      <AppIcon name="lock" size={11} strokeWidth={1.9} />
                    </div>
                    <div className="min-w-0 flex-1">
                      <div className="truncate text-[12px] font-medium">{project.name}</div>
                    </div>
                    {selected && (
                      <span className="text-[9px] font-medium text-[var(--fg-muted)]">当前</span>
                    )}
                  </button>
                )
              })}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
