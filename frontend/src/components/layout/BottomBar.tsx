import { cn } from "@/lib/utils"
import { AppIcon } from "@/components/ui/icon"
import { useUIStore } from "@/stores/uiStore"

export function BottomBar() {
  const { layoutDirection, setLayoutDirection } = useUIStore()

  return (
    <div
      className={cn(
        "flex h-[25px] items-center justify-between border-t px-3",
        "border-[var(--border-subtle)] bg-[var(--surface-secondary)]"
      )}
    >
      <div className="flex items-center gap-3 text-[11px] text-[var(--fg-muted)]">
        <button className="transition-colors hover:text-[var(--fg-secondary)]" type="button">Connect Git</button>
        <button className="transition-colors hover:text-[var(--fg-secondary)]" type="button">Console</button>
        <button className="transition-colors hover:text-[var(--fg-secondary)]" type="button">Terminal</button>
      </div>

      <div className="flex items-center gap-1">
        <button
          className={cn(
            "flex h-5 w-5 items-center justify-center rounded-[6px] border transition-colors",
            "border-transparent text-[var(--fg-secondary)] hover:bg-[var(--button-bg)]"
          )}
          onClick={() => setLayoutDirection("vertical")}
          title="上下布局"
          type="button"
        >
          <AppIcon
            name="arrowUpDown"
            size={11}
            strokeWidth={1.9}
            className={cn(layoutDirection === "vertical" ? "text-[var(--fg)]" : "text-[var(--fg-secondary)]")}
          />
        </button>
        <button
          className={cn(
            "flex h-5 w-5 items-center justify-center rounded-[6px] border transition-colors",
            "border-transparent text-[var(--fg-secondary)] hover:bg-[var(--button-bg)]"
          )}
          onClick={() => setLayoutDirection("horizontal")}
          title="左右布局"
          type="button"
        >
          <AppIcon
            name="arrowLeftRight"
            size={11}
            strokeWidth={1.9}
            className={cn(layoutDirection === "horizontal" ? "text-[var(--fg)]" : "text-[var(--fg-secondary)]")}
          />
        </button>
      </div>
    </div>
  )
}
