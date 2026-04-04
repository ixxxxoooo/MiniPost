import { Loader2, AlertCircle } from "lucide-react"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import { useTabStore } from "@/stores/tabStore"
import { useUIStore } from "@/stores/uiStore"
import { StatusBar } from "./StatusBar"
import { ResponseBody } from "./ResponseBody"
import { ResponseHeaders } from "./ResponseHeaders"
import { cn } from "@/lib/utils"

export function ResponseViewer() {
  const activeTab = useTabStore((s) => s.getActiveTab())
  const { isSending, resolved } = useUIStore()

  const isDark = resolved === "dark"
  const response = activeTab?.response ?? null
  const responseError = activeTab?.responseError ?? null

  if (isSending) {
    return (
      <div className="flex h-full items-center justify-center gap-2 bg-[var(--surface)]">
        <Loader2 className="h-4 w-4 text-[var(--fg-muted)] animate-spin" />
        <span className="text-[length:var(--size-font-xs)] text-[var(--fg-muted)]">发送请求中...</span>
      </div>
    )
  }

  if (responseError) {
    return (
      <div className="flex h-full items-center justify-center gap-2 px-6 bg-[var(--surface)]">
        <AlertCircle className="h-4 w-4 text-[var(--danger)] flex-shrink-0" />
        <span className="text-[length:var(--size-font-xs)] text-[var(--danger)]">{responseError}</span>
      </div>
    )
  }

  if (!response) {
    return (
      <div className="flex h-full items-center justify-center bg-[var(--surface)]">
        <span className="text-[length:var(--size-font-xs)] text-[var(--fg-muted)]">
          发送请求后在此查看响应
        </span>
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col bg-[var(--surface)]">
      <StatusBar response={response} />

      <Tabs defaultValue="body" className="flex-1 flex flex-col overflow-hidden">
        <TabsList className="w-full justify-start border-b border-[var(--border-color)] px-[var(--size-padding-sm)]">
          <TabsTrigger value="body">Body</TabsTrigger>
          <TabsTrigger value="headers">
            Headers
            <span className="ml-1 text-2xs text-[var(--fg-muted)]">
              ({Object.keys(response.headers).length})
            </span>
          </TabsTrigger>
        </TabsList>

        <TabsContent value="body" className="flex-1 m-0 overflow-hidden">
          <ResponseBody body={response.body} contentType={response.contentType} isDark={isDark} />
        </TabsContent>

        <TabsContent value="headers" className="flex-1 m-0 overflow-hidden">
          <ResponseHeaders headers={response.headers} />
        </TabsContent>
      </Tabs>
    </div>
  )
}
