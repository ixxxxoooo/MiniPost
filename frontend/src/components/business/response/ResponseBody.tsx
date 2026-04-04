import { useState, useMemo } from "react"
import { Copy, Check, WrapText } from "lucide-react"
import { cn } from "@/lib/utils"

interface ResponseBodyProps {
  body: string
  contentType: string
  isDark: boolean
}

export function ResponseBody({ body, contentType, isDark }: ResponseBodyProps) {
  const [copied, setCopied] = useState(false)
  const [wordWrap, setWordWrap] = useState(true)

  const formattedBody = useMemo(() => {
    if (!body) return ""
    const isJson = contentType?.includes("json")
    if (isJson) {
      try {
        return JSON.stringify(JSON.parse(body), null, 2)
      } catch {
        return body
      }
    }
    return body
  }, [body, contentType])

  const handleCopy = async () => {
    await navigator.clipboard.writeText(formattedBody)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  return (
    <div className="h-full flex flex-col">
      {/* 工具栏 */}
      <div className="flex items-center justify-end gap-1 px-2 py-1 flex-shrink-0 border-b border-[var(--border-subtle)]">
        <button
          className={cn(
            "h-5 w-5 flex items-center justify-center rounded-[var(--radius-sm)] transition-colors",
            wordWrap ? "bg-[var(--sidebar-active)] text-[var(--accent)]" : "text-[var(--fg-muted)] hover:bg-[var(--sidebar-hover)]"
          )}
          onClick={() => setWordWrap(!wordWrap)}
          title="自动换行"
        >
          <WrapText className="h-3 w-3" />
        </button>
        <button
          className="h-5 w-5 flex items-center justify-center rounded-[var(--radius-sm)] text-[var(--fg-muted)] hover:bg-[var(--sidebar-hover)] transition-colors"
          onClick={handleCopy}
          title="复制响应"
        >
          {copied ? <Check className="h-3 w-3 text-[var(--success)]" /> : <Copy className="h-3 w-3" />}
        </button>
      </div>

      {/* 内容 */}
      <pre
        className={cn(
          "flex-1 overflow-auto p-3 text-[length:var(--size-font-2xs)] font-mono text-[var(--fg)] leading-relaxed",
          wordWrap ? "whitespace-pre-wrap break-all" : "whitespace-pre"
        )}
      >
        {formattedBody || "(空响应)"}
      </pre>
    </div>
  )
}
