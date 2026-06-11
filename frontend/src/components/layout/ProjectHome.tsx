import { useMemo, useState } from "react"
import { AppIcon } from "@/components/ui/icon"
import { useI18n } from "@/hooks/useI18n"
import { cn } from "@/lib/utils"
import { PROJECT_THEME_COLORS } from "@/lib/projectTheme"
import { useProjectStore } from "@/stores/projectStore"
import { useUIStore } from "@/stores/uiStore"

function formatUpdatedAt(value: string, locale: string, prefix: string): string {
  const timestamp = Date.parse(value)
  if (!Number.isFinite(timestamp)) return `${prefix}: --`
  const formatter = new Intl.DateTimeFormat(locale, {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  })
  return `${prefix}: ${formatter.format(new Date(timestamp))}`
}

export function ProjectHome() {
  const { t, locale } = useI18n()
  const [newProjectName, setNewProjectName] = useState("")
  const [themePickerProjectId, setThemePickerProjectId] = useState<string | null>(null)
  const [descriptionDrafts, setDescriptionDrafts] = useState<Record<string, string>>({})
  const { projects, currentProjectId, createProject, selectProject, updateProjectTheme, updateProjectDescription } = useProjectStore()
  const setWorkspaceView = useUIStore((s) => s.setWorkspaceView)
  const sortedProjects = useMemo(() => (
    [...projects].sort((a, b) => b.updatedAt.localeCompare(a.updatedAt))
  ), [projects])

  const handleOpenProject = async (projectId: string) => {
    if (projectId === currentProjectId) {
      setWorkspaceView("project")
      return
    }
    await selectProject(projectId)
  }

  const handleCreateProject = async () => {
    const name = newProjectName.trim()
    if (!name) return
    const createdProject = await createProject(name)
    setNewProjectName("")
    if (createdProject?.id) {
      await selectProject(createdProject.id)
    }
  }

  const handleSaveDescription = async (projectId: string) => {
    const project = projects.find((item) => item.id === projectId)
    if (!project) return
    const current = (project.description ?? "").trim()
    const draft = (descriptionDrafts[projectId] ?? project.description ?? "").trim()
    if (draft === current) {
      setDescriptionDrafts((prev) => {
        if (!(projectId in prev)) return prev
        const next = { ...prev }
        delete next[projectId]
        return next
      })
      return
    }

    await updateProjectDescription(projectId, draft)
    setDescriptionDrafts((prev) => {
      const next = { ...prev }
      delete next[projectId]
      return next
    })
  }

  return (
    <div className="h-full overflow-auto bg-[var(--surface)] px-8 py-6">
      <div className="mx-auto w-full max-w-[1080px]">
        <div className="mb-5 flex items-end justify-between gap-4">
          <div>
            <h2 className="text-[20px] font-semibold text-[var(--fg)]">{t("项目主页", "Project home")}</h2>
            <p className="mt-1 text-[13px] text-[var(--fg-secondary)]">
              {t("选择项目继续工作，并在这里配置项目主题色。", "Select a project to continue and configure its theme color here.")}
            </p>
            <p className="mt-1 text-[12px] text-[var(--fg-muted)]">
              {t("项目主题色会同步应用到选中态、高亮态以及发送按钮。", "Project theme color will be applied to selected states, highlights, and the send button.")}
            </p>
          </div>
          <div className="flex w-[320px] items-center gap-2">
            <input
              className={cn(
                "h-[var(--size-input)] flex-1 rounded-[var(--radius-input)] border border-[var(--border-color)] bg-[var(--surface)] px-3",
                "text-[length:var(--size-font-sm)] text-[var(--fg)] outline-none placeholder:text-[var(--fg-muted)] focus:border-[var(--accent)]"
              )}
              value={newProjectName}
              onChange={(event) => setNewProjectName(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === "Enter") void handleCreateProject()
              }}
              placeholder={t("输入项目名后回车新建", "Enter project name and press Enter")}
            />
            <button
              type="button"
              onClick={() => void handleCreateProject()}
              className={cn(
                "h-[var(--size-btn)] rounded-[var(--radius-btn)] px-3 text-[length:var(--size-font-sm)] font-medium transition-colors",
                "bg-[var(--accent)] text-[var(--accent-fg)] hover:bg-[var(--accent-hover)]"
              )}
            >
              {t("新建项目", "New project")}
            </button>
          </div>
        </div>

        {sortedProjects.length === 0 ? (
          <div className="flex h-[300px] items-center justify-center rounded-[12px] border border-dashed border-[var(--border-color)]">
            <div className="text-center">
              <div className="mx-auto mb-2 flex h-10 w-10 items-center justify-center rounded-[10px] bg-[var(--surface-secondary)]">
                <AppIcon name="folderOpen" size={18} className="text-[var(--fg-muted)]" />
              </div>
              <p className="text-[14px] text-[var(--fg-secondary)]">{t("还没有项目，先创建一个开始吧", "No projects yet. Create one to get started.")}</p>
            </div>
          </div>
        ) : (
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
            {sortedProjects.map((project) => {
              const active = project.id === currentProjectId
              return (
                <div
                  key={project.id}
                  onClick={() => {
                    void handleOpenProject(project.id)
                  }}
                  onKeyDown={(event) => {
                    if (event.key === "Enter" || event.key === " ") {
                      event.preventDefault()
                      void handleOpenProject(project.id)
                    }
                  }}
                  role="button"
                  tabIndex={0}
                  className={cn(
                    "group rounded-[12px] border p-4 text-left transition-all",
                    "cursor-pointer focus:outline-none focus:ring-1 focus:ring-[var(--accent)]/50",
                    active
                      ? "border-[var(--accent)] bg-[var(--selected-bg)]"
                      : "border-[var(--border-color)] bg-[var(--surface)] hover:border-[var(--accent)]/60 hover:bg-[var(--surface-secondary)]"
                  )}
                >
                  <div className="mb-3 flex items-center gap-2">
                    <span
                      className="h-2.5 w-2.5 rounded-full"
                      style={{ backgroundColor: project.themeColor || "var(--accent)" }}
                    />
                    <div className="truncate text-[14px] font-semibold text-[var(--fg)]">{project.name}</div>
                    {active && (
                      <span className="ml-auto rounded-full bg-[var(--accent)]/14 px-2 py-0.5 text-[11px] text-[var(--accent)]">
                        {t("当前", "Current")}
                      </span>
                    )}
                  </div>
                  <div className="mb-3 text-[12px] text-[var(--fg-muted)]">{formatUpdatedAt(project.updatedAt, locale, t("最近更新", "Updated"))}</div>
                  <div className="mb-2">
                    <input
                      value={descriptionDrafts[project.id] ?? project.description ?? ""}
                      onClick={(event) => event.stopPropagation()}
                      onChange={(event) => {
                        const value = event.target.value
                        setDescriptionDrafts((prev) => ({ ...prev, [project.id]: value }))
                      }}
                      onBlur={() => { void handleSaveDescription(project.id) }}
                      onKeyDown={(event) => {
                        if (event.key !== "Enter") return
                        event.preventDefault()
                        void handleSaveDescription(project.id)
                      }}
                      className={cn(
                        "h-[var(--size-input-sm)] w-full rounded-[var(--radius-input)] border border-[var(--border-color)] bg-[var(--surface)] px-3",
                        "text-[length:var(--size-font-xs)] text-[var(--fg-secondary)] outline-none transition-colors",
                        "focus:border-[var(--accent)] placeholder:text-[var(--fg-muted)]"
                      )}
                      placeholder={t("输入项目描述（回车或失焦保存）", "Enter project description (save on Enter/blur)")}
                    />
                  </div>
                  <div className="relative mt-2">
                    <button
                      type="button"
                      onClick={(event) => {
                        event.stopPropagation()
                        setThemePickerProjectId((prev) => prev === project.id ? null : project.id)
                      }}
                      className={cn(
                        "inline-flex h-[var(--size-btn)] items-center gap-1.5 rounded-[var(--radius-btn)] border px-3 text-[length:var(--size-font-xs)] transition-colors",
                        "border-[var(--border-color)] bg-[var(--surface-secondary)] text-[var(--fg-secondary)] hover:text-[var(--fg)]"
                      )}
                    >
                      <span
                        className="h-2.5 w-2.5 rounded-full"
                        style={{ backgroundColor: project.themeColor || "var(--accent)" }}
                      />
                      <span>{t("主题色", "Theme color")}</span>
                      <AppIcon
                        name="arrowDown"
                        size={10}
                        className={cn("transition-transform", themePickerProjectId === project.id && "rotate-180")}
                      />
                    </button>
                    {themePickerProjectId === project.id && (
                      <div
                        className={cn(
                          "absolute left-0 top-[calc(100%+6px)] z-20 w-[228px] rounded-[10px] border p-2 shadow-[var(--shadow-md)]",
                          "border-[var(--border-color)] bg-[var(--surface-elevated)]"
                        )}
                        onClick={(event) => event.stopPropagation()}
                      >
                        <div className="mb-2 text-[10px] text-[var(--fg-muted)]">{t("选择项目主题色", "Pick a project theme color")}</div>
                        <div className="flex flex-wrap gap-1.5">
                          {PROJECT_THEME_COLORS.map((color) => {
                            const selected = (project.themeColor || "").toUpperCase() === color.toUpperCase()
                            return (
                              <button
                                key={`${project.id}-${color}`}
                                type="button"
                                onClick={() => {
                                  void updateProjectTheme(project.id, color)
                                  setThemePickerProjectId(null)
                                }}
                                className={cn(
                                  "h-5 w-5 rounded-full border transition-transform hover:scale-105",
                                  selected ? "border-[var(--fg)] shadow-sm" : "border-white/45"
                                )}
                                style={{ backgroundColor: color }}
                              />
                            )
                          })}
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}
