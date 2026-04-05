import { createPortal } from "react-dom"
import { AppIcon } from "@/components/ui/icon"
import { useUIStore } from "@/stores/uiStore"
import { cn } from "@/lib/utils"

const SHORTCUTS = [
  { keys: "⌘ + Enter", desc: "发送请求" },
  { keys: "⌘ + S", desc: "保存请求" },
  { keys: "⌘ + N", desc: "新建请求" },
  { keys: "⌘ + I", desc: "导入 cURL" },
  { keys: "⌘ + W", desc: "关闭当前标签" },
  { keys: "⌘ + F", desc: "搜索（响应体/编辑器内）" },
  { keys: "⌘ + ,", desc: "打开设置" },
  { keys: "⌘ + B", desc: "切换侧边栏" },
  { keys: "Double Click Tab", desc: "重命名标签" },
  { keys: "Middle Click Tab", desc: "关闭标签" },
]

const FONT_SIZES = [10, 11, 12, 13, 14, 16]

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
  } = useUIStore()

  if (!settingsOpen) return null

  return createPortal(
    <div
      className="fixed inset-0 z-[400] flex items-center justify-center"
      onClick={() => setSettingsOpen(false)}
    >
      <div className="absolute inset-0 bg-black/40" />
      <div
        className={cn(
          "relative z-[401] w-[520px] max-h-[80vh] overflow-y-auto rounded-[var(--radius-panel)] border shadow-[var(--shadow-lg)]",
          "bg-[var(--surface)] border-[var(--border-color)]"
        )}
        onClick={(e) => e.stopPropagation()}
      >
        {/* 标题 */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-[var(--border-color)]">
          <div className="flex items-center gap-2">
            <AppIcon name="settings" size={18} className="text-[var(--fg-secondary)]" />
            <h2 className="text-[15px] font-semibold text-[var(--fg)]">设置</h2>
          </div>
          <button
            className="h-6 w-6 flex items-center justify-center rounded-[6px] hover:bg-[var(--sidebar-hover)] transition-colors"
            onClick={() => setSettingsOpen(false)}
          >
            <AppIcon name="clear" size={14} className="text-[var(--fg-muted)]" />
          </button>
        </div>

        <div className="px-5 py-4 space-y-6">
          {/* 主题 */}
          <section>
            <div className="flex items-center gap-2 mb-3">
              <AppIcon name="paintBoard" size={14} className="text-[var(--fg-secondary)]" />
              <h3 className="text-[13px] font-medium text-[var(--fg)]">外观主题</h3>
            </div>
            <div className="flex gap-2">
              {(["light", "dark", "system"] as const).map((t) => (
                <button
                  key={t}
                  className={cn(
                    "flex-1 h-[36px] flex items-center justify-center gap-2 rounded-[8px] border text-[12px] font-medium transition-all",
                    theme === t
                      ? "border-[var(--accent)] bg-[var(--accent)]/10 text-[var(--accent)]"
                      : "border-[var(--border-color)] text-[var(--fg-secondary)] hover:border-[var(--fg-muted)] hover:bg-[var(--surface-secondary)]"
                  )}
                  onClick={() => setTheme(t)}
                >
                  <AppIcon
                    name={t === "light" ? "sun" : t === "dark" ? "moon" : "paintBoard"}
                    size={14}
                  />
                  {t === "light" ? "浅色" : t === "dark" ? "深色" : "跟随系统"}
                </button>
              ))}
            </div>
          </section>

          {/* 字体大小 */}
          <section>
            <div className="flex items-center gap-2 mb-3">
              <span className="text-[14px] text-[var(--fg-secondary)] font-mono font-bold shrink-0 w-[14px] text-center">A</span>
              <h3 className="text-[13px] font-medium text-[var(--fg)]">字体大小</h3>
            </div>
            <div className="flex gap-2">
              {FONT_SIZES.map((s) => (
                <button
                  key={s}
                  className={cn(
                    "flex-1 h-[32px] flex items-center justify-center rounded-[6px] border text-[12px] font-mono transition-all",
                    fontSize === s
                      ? "border-[var(--accent)] bg-[var(--accent)]/10 text-[var(--accent)] font-semibold"
                      : "border-[var(--border-color)] text-[var(--fg-secondary)] hover:border-[var(--fg-muted)] hover:bg-[var(--surface-secondary)]"
                  )}
                  onClick={() => setFontSize(s)}
                >
                  {s}px
                </button>
              ))}
            </div>
          </section>

          {/* 滚动条 */}
          <section>
            <div className="flex items-center gap-2 mb-3">
              <AppIcon name="move" size={14} className="text-[var(--fg-secondary)]" />
              <h3 className="text-[13px] font-medium text-[var(--fg)]">滚动条</h3>
            </div>
            <div className="rounded-[8px] border border-[var(--border-color)] p-3 flex items-center justify-between">
              <div>
                <p className="text-[12px] text-[var(--fg)] font-medium">静止后自动隐藏</p>
                <p className="text-[11px] text-[var(--fg-muted)] mt-0.5">默认开启。关闭后滚动条将始终显示。</p>
              </div>
              <button
                className={cn(
                  "h-6 px-2.5 rounded-[6px] border text-[11px] font-medium transition-colors",
                  scrollbarAutoHide
                    ? "border-[var(--accent)] text-[var(--accent)] bg-[var(--accent)]/10"
                    : "border-[var(--border-color)] text-[var(--fg-secondary)] hover:bg-[var(--surface-secondary)]"
                )}
                onClick={() => setScrollbarAutoHide(!scrollbarAutoHide)}
              >
                {scrollbarAutoHide ? "已开启" : "已关闭"}
              </button>
            </div>
          </section>

          {/* 快捷键 */}
          <section>
            <div className="flex items-center gap-2 mb-3">
              <AppIcon name="keyboard" size={14} className="text-[var(--fg-secondary)]" />
              <h3 className="text-[13px] font-medium text-[var(--fg)]">快捷键</h3>
            </div>
            <div className="rounded-[8px] border border-[var(--border-color)] overflow-hidden">
              {SHORTCUTS.map((shortcut, idx) => (
                <div
                  key={idx}
                  className={cn(
                    "flex items-center justify-between px-3 py-2",
                    idx !== 0 && "border-t border-[var(--border-color)]"
                  )}
                >
                  <span className="text-[12px] text-[var(--fg-secondary)]">{shortcut.desc}</span>
                  <kbd className="px-2 py-0.5 rounded-[4px] bg-[var(--surface-secondary)] border border-[var(--border-color)] text-[11px] font-mono text-[var(--fg-muted)]">
                    {shortcut.keys}
                  </kbd>
                </div>
              ))}
            </div>
          </section>

          {/* 关于 */}
          <section>
            <div className="flex items-center gap-2 mb-3">
              <AppIcon name="info" size={14} className="text-[var(--fg-secondary)]" />
              <h3 className="text-[13px] font-medium text-[var(--fg)]">关于</h3>
            </div>
            <div className="rounded-[8px] border border-[var(--border-color)] p-4">
              <div className="flex items-center gap-3 mb-3">
                <div className="h-10 w-10 rounded-[10px] bg-[var(--accent)] flex items-center justify-center">
                  <span className="text-white font-bold text-[16px]">M</span>
                </div>
                <div>
                  <h4 className="text-[14px] font-semibold text-[var(--fg)]">MiniPost</h4>
                  <p className="text-[11px] text-[var(--fg-muted)]">Version 1.0.0</p>
                </div>
              </div>
              <p className="text-[12px] text-[var(--fg-secondary)] leading-relaxed">
                轻量级 HTTP 请求调试工具，基于 Wails + React + TypeScript 构建。
                支持多项目管理、环境变量、请求历史、导入导出等功能。
              </p>
            </div>
          </section>
        </div>
      </div>
    </div>,
    document.body
  )
}
