import { useEffect, useState, useCallback, useRef } from "react"
import { AppIcon } from "@/components/ui/icon"
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select"
import { Input } from "@/components/ui/input"
import { useUIStore } from "@/stores/uiStore"
import { useEnvironmentStore } from "@/stores/environmentStore"
import { useProjectStore } from "@/stores/projectStore"
import { useTabStore } from "@/stores/tabStore"
import { ImportCurl } from "../../../wailsjs/go/main/App"
import { WindowControls } from "./WindowControls"
import { WorkspaceHeader } from "./WorkspaceHeader"
import { cn } from "@/lib/utils"

function useTitlebarDoubleClick() {
  const lastClickRef = useRef<{ time: number; x: number; y: number }>({
    time: 0, x: 0, y: 0,
  })
  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    if (e.button !== 0) return
    const now = Date.now()
    const last = lastClickRef.current
    if (now - last.time < 400 && Math.abs(e.clientX - last.x) < 5 && Math.abs(e.clientY - last.y) < 5) {
      import("../../../wailsjs/runtime/runtime").then((r) => r.WindowToggleMaximise())
      lastClickRef.current = { time: 0, x: 0, y: 0 }
    } else {
      lastClickRef.current = { time: now, x: e.clientX, y: e.clientY }
    }
  }, [])
  return { handleMouseDown }
}

export function Toolbar() {
  const { resolved, setTheme } = useUIStore()
  const { currentProjectId } = useProjectStore()
  const activeTab = useTabStore((s) => s.getActiveTab())
  const updateTabRequest = useTabStore((s) => s.updateTabRequest)
  const [showCurlInput, setShowCurlInput] = useState(false)
  const [curlCommand, setCurlCommand] = useState("")
  const { environments, activeEnvironmentId, setActiveEnvironment, loadEnvironments } = useEnvironmentStore()
  const { handleMouseDown: handleTitlebarMouseDown } = useTitlebarDoubleClick()

  useEffect(() => {
    if (currentProjectId) {
      loadEnvironments(currentProjectId)
    }
  }, [currentProjectId, loadEnvironments])

  return (
    <div
      className={cn(
        "relative z-40 flex h-[var(--size-toolbar)] flex-shrink-0 items-center overflow-visible border-b titlebar-drag",
        "bg-[var(--toolbar-bg)] border-[var(--toolbar-border)] px-2.5"
      )}
      onMouseDown={handleTitlebarMouseDown}
    >
      <div onMouseDown={(e) => e.stopPropagation()}>
        <WindowControls />
      </div>

      <div className="ml-1.5">
        <WorkspaceHeader />
      </div>

      <div className="flex-1" />

      <div className="titlebar-no-drag flex items-center gap-[var(--size-gap-sm)]" onMouseDown={(e) => e.stopPropagation()}>
        {showCurlInput && (
          <Input
            value={curlCommand}
            onChange={(e) => setCurlCommand(e.target.value)}
            placeholder="粘贴 cURL 命令..."
            className="h-[28px] w-56 rounded-[10px] border-[var(--button-border)] bg-[var(--surface)] font-mono text-[11px]"
            autoFocus
            onKeyDown={async (e) => {
              if (e.key === "Enter" && curlCommand.trim() && activeTab) {
                try {
                  const result = await ImportCurl(curlCommand.trim())
                  updateTabRequest(activeTab.id, {
                    method: result.method as "GET" | "POST" | "PUT" | "PATCH" | "DELETE" | "HEAD" | "OPTIONS",
                    url: result.url,
                    headers: (result.headers ?? []).map((h: { key: string; value: string }) => ({
                      id: crypto.randomUUID(), key: h.key, value: h.value, enabled: true,
                    })),
                    body: result.body
                      ? { type: result.body.type as "none" | "raw" | "json" | "form-urlencoded", raw: result.body.raw, json: result.body.json }
                      : { type: "none" as const },
                  })
                  setCurlCommand("")
                  setShowCurlInput(false)
                } catch (err) {
                  console.error("cURL 解析失败:", err)
                }
              }
              if (e.key === "Escape") {
                setShowCurlInput(false)
                setCurlCommand("")
              }
            }}
          />
        )}

        {currentProjectId && environments.length > 0 && (
          <>
            <AppIcon name="globe" size={12} className="text-[var(--fg-muted)]" />
            <Select
              value={activeEnvironmentId ?? "none"}
              onValueChange={(v) => setActiveEnvironment(v === "none" ? null : v)}
            >
              <SelectTrigger className="h-[28px] w-[124px] rounded-[10px] border-[var(--button-border)] bg-[var(--surface)] text-[12px]">
                <SelectValue placeholder="No Env" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="none">No Environment</SelectItem>
                {environments.map((env) => (
                  <SelectItem key={env.id} value={env.id}>{env.name}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </>
        )}

        <button
          className="flex h-[28px] w-[28px] items-center justify-center rounded-[10px] border border-[var(--button-border)] bg-[var(--surface)] transition-colors hover:bg-[var(--surface-secondary)]"
          onClick={() => setShowCurlInput(!showCurlInput)}
          title="导入 cURL"
          type="button"
        >
          <AppIcon name="upload" size={12} className="text-[var(--fg-secondary)]" />
        </button>

        <button
          className="flex h-[28px] w-[28px] items-center justify-center rounded-[10px] border border-[var(--button-border)] bg-[var(--surface)] transition-colors hover:bg-[var(--surface-secondary)]"
          onClick={() => setTheme(resolved === "dark" ? "light" : "dark")}
          title={resolved === "dark" ? "切换为浅色主题" : "切换为深色主题"}
          type="button"
        >
          {resolved === "dark"
            ? <AppIcon name="sun" size={12} className="text-[var(--fg-secondary)]" />
            : <AppIcon name="moon" size={12} className="text-[var(--fg-secondary)]" />
          }
        </button>
      </div>
    </div>
  )
}
