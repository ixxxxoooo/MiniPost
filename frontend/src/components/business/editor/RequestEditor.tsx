import { useEffect, useCallback } from "react"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import { UrlBar } from "./UrlBar"
import { ParamsEditor } from "./ParamsEditor"
import { HeadersEditor } from "./HeadersEditor"
import { BodyEditor } from "./BodyEditor"
import { AuthEditor } from "./AuthEditor"
import { useTabStore } from "@/stores/tabStore"
import { useUIStore } from "@/stores/uiStore"
import { useProjectStore } from "@/stores/projectStore"
import { useEnvironmentStore } from "@/stores/environmentStore"
import { sendHttpRequest } from "@/services/httpService"
import { cn } from "@/lib/utils"

export function RequestEditor() {
  const activeTab = useTabStore((s) => s.getActiveTab())
  const setTabResponse = useTabStore((s) => s.setTabResponse)
  const setTabResponseError = useTabStore((s) => s.setTabResponseError)
  const markTabDirty = useTabStore((s) => s.markTabDirty)
  const { setIsSending } = useUIStore()
  const { currentProjectId, saveRequestToBackend } = useProjectStore()
  const { activeEnvironmentId } = useEnvironmentStore()

  const handleSend = async () => {
    if (!activeTab || !activeTab.request.url.trim()) return

    setIsSending(true)
    setTabResponseError(activeTab.id, null)

    try {
      const result = await sendHttpRequest(
        activeTab.request,
        currentProjectId ?? undefined,
        activeEnvironmentId ?? undefined,
      )
      setTabResponse(activeTab.id, result)
    } catch (err) {
      setTabResponseError(activeTab.id, err instanceof Error ? err.message : String(err))
    } finally {
      setIsSending(false)
    }
  }

  const handleSave = useCallback(async () => {
    if (!activeTab || !currentProjectId) return

    const req = activeTab.request
    const requestItem = {
      id: req.id,
      name: req.name,
      method: req.method,
      url: req.url,
      params: req.params.filter((p) => p.key).map((p) => ({ key: p.key, value: p.value })),
      headers: req.headers.filter((h) => h.key).map((h) => ({ key: h.key, value: h.value })),
      body: {
        type: req.body.type,
        raw: req.body.raw ?? "",
        json: req.body.json ?? "",
        formUrlEncoded: (req.body.formUrlEncoded ?? []).filter((f) => f.key).map((f) => ({ key: f.key, value: f.value })),
      },
      auth: {
        type: req.auth.type,
        basic: req.auth.basic ?? { username: "", password: "" },
        bearer: req.auth.bearer ?? { token: "" },
        apiKey: req.auth.apiKey ?? { key: "", value: "", addTo: "header" },
      },
      folderId: req.folderId ?? "",
      projectId: currentProjectId,
      createdAt: req.createdAt,
      updatedAt: new Date().toISOString(),
    }

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    await saveRequestToBackend(requestItem as any)
    markTabDirty(activeTab.id, false)
  }, [activeTab, currentProjectId, saveRequestToBackend, markTabDirty])

  useEffect(() => {
    const listener = () => handleSave()
    window.addEventListener("minipost:save", listener)
    return () => window.removeEventListener("minipost:save", listener)
  }, [handleSave])

  if (!activeTab) return null

  const { request } = activeTab

  return (
    <div className="flex h-full flex-col bg-[var(--surface)]">
      <UrlBar onSend={handleSend} onSave={handleSave} />

      <Tabs defaultValue="params" className="flex-1 flex flex-col overflow-hidden">
        <TabsList className="w-full justify-start px-[var(--size-padding-sm)] py-1">
          <TabsTrigger value="params">
            Params
            {request.params.length > 0 && (
              <span className="ml-1 text-2xs text-[var(--fg-muted)]">({request.params.length})</span>
            )}
          </TabsTrigger>
          <TabsTrigger value="headers">
            Headers
            {request.headers.length > 0 && (
              <span className="ml-1 text-2xs text-[var(--fg-muted)]">({request.headers.length})</span>
            )}
          </TabsTrigger>
          <TabsTrigger value="body">Body</TabsTrigger>
          <TabsTrigger value="auth">Auth</TabsTrigger>
        </TabsList>

        <div className="flex-1 overflow-y-auto">
          <TabsContent value="params" className="m-0">
            <ParamsEditor />
          </TabsContent>
          <TabsContent value="headers" className="m-0">
            <HeadersEditor />
          </TabsContent>
          <TabsContent value="body" className="m-0">
            <BodyEditor />
          </TabsContent>
          <TabsContent value="auth" className="m-0">
            <AuthEditor />
          </TabsContent>
        </div>
      </Tabs>
    </div>
  )
}
