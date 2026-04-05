import { useEffect, useMemo, useState, type ReactNode } from "react"
import { createPortal } from "react-dom"
import { AppIcon, type AppIconName } from "@/components/ui/icon"
import { useUIStore } from "@/stores/uiStore"
import { cn } from "@/lib/utils"

const SHORTCUTS = [
  { keys: "⌘ + Enter", desc: "发送请求" },
  { keys: "⌘ + S", desc: "保存请求" },
  { keys: "⌘ + N", desc: "新建请求" },
  { keys: "⌘ + I", desc: "导入 cURL" },
  { keys: "⌘ + E", desc: "重命名选中请求/文件夹" },
  { keys: "⌘ + C", desc: "复制选中请求/文件夹" },
  { keys: "⌘ + D", desc: "Duplicate 选中请求/文件夹" },
  { keys: "⌫", desc: "删除选中请求/文件夹" },
  { keys: "⌘ + W", desc: "关闭当前标签" },
  { keys: "⌘ + F", desc: "搜索（响应体/编辑器内）" },
  { keys: "⌘ + ,", desc: "打开设置" },
  { keys: "⌘ + B", desc: "切换侧边栏" },
]

const FONT_SIZES = [10, 11, 12, 13, 14, 16]

type SettingsSection = "general" | "theme" | "shortcuts" | "about"

const SETTINGS_MENU: Array<{ id: SettingsSection; title: string; icon: AppIconName }> = [
  { id: "general", title: "通用", icon: "settings" },
  { id: "theme", title: "主题", icon: "paintBoard" },
  { id: "shortcuts", title: "快捷键", icon: "keyboard" },
  { id: "about", title: "关于", icon: "info" },
]

function ToggleSwitch({ checked, onToggle, disabled = false }: { checked: boolean; onToggle: () => void; disabled?: boolean }) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onToggle}
      className={cn(
        "relative h-5 w-9 rounded-full border transition-colors",
        checked ? "border-[var(--accent)] bg-[var(--accent)]" : "border-[var(--button-border)] bg-[var(--button-bg)]",
        disabled && "opacity-50 cursor-not-allowed"
      )}
    >
      <span
        className={cn(
          "absolute top-[1px] block h-[15px] w-[15px] rounded-full border border-black/5 bg-white shadow-[var(--shadow-sm)] transition-transform",
          checked ? "translate-x-[18px]" : "translate-x-[1px]"
        )}
      />
    </button>
  )
}

function SettingsRow({
  title,
  description,
  control,
  last = false,
}: {
  title: string
  description: string
  control: ReactNode
  last?: boolean
}) {
  return (
    <div className={cn("grid grid-cols-[1fr_auto] gap-5 py-3", !last && "border-b border-[var(--border-subtle)]")}>
      <div>
        <div className="text-[13px] font-medium text-[var(--fg)]">{title}</div>
        <div className="mt-0.5 text-[11px] leading-5 text-[var(--fg-secondary)]">{description}</div>
      </div>
      <div className="flex items-center">{control}</div>
    </div>
  )
}

export function SettingsPanel() {
  const {
    settingsOpen,
    setSettingsOpen,
    theme,
    setTheme,
    fontSize,
    setFontSize,
    scrollbarAutoHide,
    setScrollbarAutoHide,
    followRedirects,
    setFollowRedirects,
    httpVersion,
    setHttpVersion,
    requestTimeoutMs,
    setRequestTimeoutMs,
    maxResponseSizeMB,
    setMaxResponseSizeMB,
    sslCertificateVerification,
    setSSLCertificateVerification,
    sslTlsKeyLog,
    setSSLTlsKeyLog,
    disableCookies,
    setDisableCookies,
    responseFormatDetection,
    setResponseFormatDetection,
  } = useUIStore()
  const [activeSection, setActiveSection] = useState<SettingsSection>("general")

  useEffect(() => {
    if (!settingsOpen) return
    const handler = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return
      event.preventDefault()
      setSettingsOpen(false)
    }
    window.addEventListener("keydown", handler)
    return () => window.removeEventListener("keydown", handler)
  }, [settingsOpen, setSettingsOpen])

  useEffect(() => {
    if (!settingsOpen) setActiveSection("general")
  }, [settingsOpen])

  const sectionTitle = useMemo(() => SETTINGS_MENU.find((item) => item.id === activeSection)?.title ?? "设置", [activeSection])

  if (!settingsOpen) return null

  return createPortal(
    <div className="fixed inset-0 z-[400] flex items-center justify-center" onClick={() => setSettingsOpen(false)}>
      <div className="absolute inset-0 bg-black/40" />
      <div
        className={cn(
          "relative z-[401] h-[600px] w-[880px] overflow-hidden rounded-[14px] border shadow-[var(--shadow-lg)]",
          "bg-[var(--surface)] border-[var(--border-color)]"
        )}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex h-full">
          <aside className="w-[210px] border-r border-[var(--border-color)] bg-[var(--surface-secondary)] px-3 py-3">
            <div className="mb-2 px-2 text-[11px] font-semibold text-[var(--fg-muted)]">设置</div>
            <div className="space-y-1">
              {SETTINGS_MENU.map((item) => (
                <button
                  key={item.id}
                  type="button"
                  onClick={() => setActiveSection(item.id)}
                  className={cn(
                    "h-9 w-full rounded-[9px] px-2.5 text-left transition-colors",
                    "flex items-center gap-2 text-[13px]",
                    activeSection === item.id
                      ? "bg-[var(--selected-bg)] text-[var(--fg)]"
                      : "text-[var(--fg-secondary)] hover:bg-[var(--button-bg)]"
                  )}
                >
                  <AppIcon name={item.icon} size={14} />
                  {item.title}
                </button>
              ))}
            </div>
          </aside>

          <main className="flex-1 min-w-0 bg-[var(--surface)]">
            <div className="flex h-11 items-center justify-between border-b border-[var(--border-color)] px-4">
              <h2 className="text-[14px] font-semibold text-[var(--fg)]">{sectionTitle}</h2>
              <button
                type="button"
                className="h-6 w-6 rounded-[7px] text-[var(--fg-muted)] hover:bg-[var(--button-bg)] hover:text-[var(--fg)]"
                onClick={() => setSettingsOpen(false)}
              >
                <div className="flex items-center justify-center">
                  <AppIcon name="clear" size={12} />
                </div>
              </button>
            </div>

            <div className="h-[calc(100%-44px)] overflow-y-auto px-4 py-2.5">
              {activeSection === "general" && (
                <section className="rounded-[11px] border border-[var(--border-color)] px-3.5">
                  <SettingsRow
                    title="HTTP 版本"
                    description="设置发送请求时优先使用的 HTTP 版本。"
                    control={
                      <div className="flex h-7 items-center overflow-hidden rounded-[8px] border border-[var(--button-border)] bg-[var(--surface)]">
                        {[
                          { id: "auto", label: "Auto" },
                          { id: "http1", label: "HTTP/1.1" },
                          { id: "http2", label: "HTTP/2" },
                        ].map((item) => (
                          <button
                            key={item.id}
                            type="button"
                            onClick={() => setHttpVersion(item.id as "auto" | "http1" | "http2")}
                            className={cn(
                              "h-full px-2.5 text-[11px] transition-colors",
                              httpVersion === item.id
                                ? "bg-[var(--selected-bg)] text-[var(--fg)]"
                                : "text-[var(--fg-secondary)] hover:bg-[var(--button-bg)]"
                            )}
                          >
                            {item.label}
                          </button>
                        ))}
                      </div>
                    }
                  />
                  <SettingsRow
                    title="请求超时"
                    description="单位毫秒，设置为 0 表示不主动超时。"
                    control={
                      <div className="flex h-7 items-center overflow-hidden rounded-[8px] border border-[var(--button-border)] bg-[var(--surface)] pr-2">
                        <input
                          value={requestTimeoutMs}
                          type="number"
                          min={0}
                          onChange={(event) => setRequestTimeoutMs(Number(event.target.value || 0))}
                          className="h-full w-[92px] border-0 bg-transparent px-2.5 text-right text-[11px] text-[var(--fg)] outline-none"
                        />
                        <span className="text-[11px] text-[var(--fg-muted)]">ms</span>
                      </div>
                    }
                  />
                  <SettingsRow
                    title="最大响应大小"
                    description="单位 MB，设置为 0 表示不限制。"
                    control={
                      <div className="flex h-7 items-center overflow-hidden rounded-[8px] border border-[var(--button-border)] bg-[var(--surface)] pr-2">
                        <input
                          value={maxResponseSizeMB}
                          type="number"
                          min={0}
                          onChange={(event) => setMaxResponseSizeMB(Number(event.target.value || 0))}
                          className="h-full w-[92px] border-0 bg-transparent px-2.5 text-right text-[11px] text-[var(--fg)] outline-none"
                        />
                        <span className="text-[11px] text-[var(--fg-muted)]">MB</span>
                      </div>
                    }
                  />
                  <SettingsRow
                    title="SSL 证书校验"
                    description="关闭后将跳过证书校验（仅建议在调试环境使用）。"
                    control={<ToggleSwitch checked={sslCertificateVerification} onToggle={() => setSSLCertificateVerification(!sslCertificateVerification)} />}
                  />
                  <SettingsRow
                    title="SSL/TLS Key Log"
                    description="记录 TLS 会话密钥用于抓包调试。"
                    control={<ToggleSwitch checked={sslTlsKeyLog} onToggle={() => setSSLTlsKeyLog(!sslTlsKeyLog)} />}
                  />
                  <SettingsRow
                    title="禁用 Cookies"
                    description="关闭 Cookie Jar 读写，不自动注入或保存 Cookie。"
                    control={<ToggleSwitch checked={disableCookies} onToggle={() => setDisableCookies(!disableCookies)} />}
                  />
                  <SettingsRow
                    title="自动跟随重定向（301/302/307/308）"
                    description="对应 Postman：Settings > General > Request > Automatically follow redirects。"
                    control={<ToggleSwitch checked={followRedirects} onToggle={() => setFollowRedirects(!followRedirects)} />}
                  />
                  <SettingsRow
                    title="响应格式检测"
                    description="Auto 按 Content-Type 自动识别；JSON 会优先用 JSON 视图。"
                    control={
                      <div className="flex h-7 items-center overflow-hidden rounded-[8px] border border-[var(--button-border)] bg-[var(--surface)]">
                        <button
                          type="button"
                          onClick={() => setResponseFormatDetection("auto")}
                          className={cn(
                            "h-full px-2.5 text-[11px] transition-colors",
                            responseFormatDetection === "auto"
                              ? "bg-[var(--selected-bg)] text-[var(--fg)]"
                              : "text-[var(--fg-secondary)] hover:bg-[var(--button-bg)]"
                          )}
                        >
                          Auto
                        </button>
                        <button
                          type="button"
                          onClick={() => setResponseFormatDetection("json")}
                          className={cn(
                            "h-full px-2.5 text-[11px] transition-colors",
                            responseFormatDetection === "json"
                              ? "bg-[var(--selected-bg)] text-[var(--fg)]"
                              : "text-[var(--fg-secondary)] hover:bg-[var(--button-bg)]"
                          )}
                        >
                          JSON
                        </button>
                      </div>
                    }
                  />
                  <SettingsRow
                    title="滚动条自动隐藏"
                    description="静止一段时间后自动隐藏滚动条。"
                    control={<ToggleSwitch checked={scrollbarAutoHide} onToggle={() => setScrollbarAutoHide(!scrollbarAutoHide)} />}
                    last
                  />
                </section>
              )}

              {activeSection === "theme" && (
                <section className="space-y-3">
                  <div className="rounded-[11px] border border-[var(--border-color)] p-3.5">
                    <div className="mb-2 text-[13px] font-medium text-[var(--fg)]">外观主题</div>
                    <div className="grid grid-cols-3 gap-2">
                      {(["light", "dark", "system"] as const).map((value) => (
                        <button
                          key={value}
                          type="button"
                          className={cn(
                            "h-[34px] rounded-[8px] border text-[11px] font-medium transition-colors",
                            theme === value
                              ? "border-[var(--accent)] bg-[var(--accent)]/12 text-[var(--accent)]"
                              : "border-[var(--button-border)] text-[var(--fg-secondary)] hover:bg-[var(--button-bg)]"
                          )}
                          onClick={() => setTheme(value)}
                        >
                          {value === "light" ? "浅色" : value === "dark" ? "深色" : "跟随系统"}
                        </button>
                      ))}
                    </div>
                  </div>
                  <div className="rounded-[11px] border border-[var(--border-color)] p-3.5">
                    <div className="mb-2 text-[13px] font-medium text-[var(--fg)]">编辑器字号</div>
                    <div className="grid grid-cols-6 gap-2">
                      {FONT_SIZES.map((size) => (
                        <button
                          key={size}
                          type="button"
                          className={cn(
                            "h-7 rounded-[7px] border text-[11px] font-mono transition-colors",
                            fontSize === size
                              ? "border-[var(--accent)] bg-[var(--accent)]/12 text-[var(--accent)]"
                              : "border-[var(--button-border)] text-[var(--fg-secondary)] hover:bg-[var(--button-bg)]"
                          )}
                          onClick={() => setFontSize(size)}
                        >
                          {size}px
                        </button>
                      ))}
                    </div>
                  </div>
                </section>
              )}

              {activeSection === "shortcuts" && (
                <section className="rounded-[11px] border border-[var(--border-color)] overflow-hidden">
                  {SHORTCUTS.map((shortcut, idx) => (
                    <div
                      key={shortcut.keys}
                      className={cn(
                        "flex items-center justify-between px-3.5 py-2.5",
                        idx > 0 && "border-t border-[var(--border-subtle)]"
                      )}
                    >
                      <span className="text-[12px] text-[var(--fg-secondary)]">{shortcut.desc}</span>
                      <kbd className="rounded-[6px] border border-[var(--button-border)] bg-[var(--surface-secondary)] px-2 py-0.5 text-[10px] font-mono text-[var(--fg-muted)]">
                        {shortcut.keys}
                      </kbd>
                    </div>
                  ))}
                </section>
              )}

              {activeSection === "about" && (
                <section className="rounded-[11px] border border-[var(--border-color)] p-4">
                  <div className="flex items-center gap-2.5">
                    <div className="flex h-9 w-9 items-center justify-center rounded-[9px] bg-[var(--accent)] text-white font-semibold text-[13px]">
                      M
                    </div>
                    <div>
                      <div className="text-[14px] font-semibold text-[var(--fg)]">MiniPost</div>
                      <div className="text-[11px] text-[var(--fg-muted)]">Version 1.0.0</div>
                    </div>
                  </div>
                  <div className="mt-3 text-[12px] leading-6 text-[var(--fg-secondary)]">
                    MiniPost 是一个轻量 HTTP 请求调试工具，支持多项目、环境变量、历史记录、响应预览与 Cookie 管理。
                  </div>
                </section>
              )}
            </div>
          </main>
        </div>
      </div>
    </div>,
    document.body
  )
}
