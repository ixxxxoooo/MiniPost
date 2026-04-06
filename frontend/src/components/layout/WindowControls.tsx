import { useState } from "react"
import { cn } from "@/lib/utils"
import { useI18n } from "@/hooks/useI18n"

type ControlType = "close" | "minimise" | "maximise"

const CONTROL_STYLES: Record<ControlType, { bg: string; border: string; icon: string }> = {
  close: {
    bg: "#ff5f57",
    border: "#e0443e",
    icon: "#6b0f12",
  },
  minimise: {
    bg: "#ffbd2e",
    border: "#dea123",
    icon: "#6a4708",
  },
  maximise: {
    bg: "#28c840",
    border: "#1ea133",
    icon: "#0f5f1d",
  },
}

function ControlIcon({ type }: { type: ControlType }) {
  if (type === "close") {
    return (
      <svg width="6" height="6" viewBox="0 0 6 6" fill="none" aria-hidden="true">
        <path
          d="M1 1L5 5M5 1L1 5"
          stroke={CONTROL_STYLES.close.icon}
          strokeWidth="1"
          strokeLinecap="round"
        />
      </svg>
    )
  }

  if (type === "minimise") {
    return (
      <svg width="6" height="6" viewBox="0 0 6 6" fill="none" aria-hidden="true">
        <path
          d="M1 3H5"
          stroke={CONTROL_STYLES.minimise.icon}
          strokeWidth="1"
          strokeLinecap="round"
        />
      </svg>
    )
  }

  return (
    <svg width="6" height="6" viewBox="0 0 6 6" fill="none" aria-hidden="true">
      <path
        d="M1.25 4.75L4.75 1.25"
        stroke={CONTROL_STYLES.maximise.icon}
        strokeWidth="1"
        strokeLinecap="round"
      />
      <path
        d="M2.5 1.25H4.75V3.5"
        stroke={CONTROL_STYLES.maximise.icon}
        strokeWidth="1"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}

interface TrafficLightButtonProps {
  hovered: boolean
  onClick: () => void
  title: string
  type: ControlType
}

function TrafficLightButton({ hovered, onClick, title, type }: TrafficLightButtonProps) {
  const style = CONTROL_STYLES[type]

  return (
    <button
      onClick={onClick}
      className={cn(
        "flex h-[12px] w-[12px] items-center justify-center rounded-full",
        "border transition-all duration-150 ease-out focus:outline-none"
      )}
      style={{
        backgroundColor: style.bg,
        borderColor: style.border,
        boxShadow: "inset 0 0.5px 0 rgba(255,255,255,0.35)",
      }}
      title={title}
      aria-label={title}
      type="button"
    >
      <span className={cn("transition-opacity duration-150", hovered ? "opacity-100" : "opacity-0")}>
        <ControlIcon type={type} />
      </span>
    </button>
  )
}

export function WindowControls() {
  const [hovered, setHovered] = useState(false)
  const { t } = useI18n()

  const handleClose = () => {
    import("../../../wailsjs/runtime/runtime").then((r) => r.Quit())
  }

  const handleMinimise = () => {
    import("../../../wailsjs/runtime/runtime").then((r) => r.WindowMinimise())
  }

  const handleMaximise = () => {
    import("../../../wailsjs/runtime/runtime").then((r) => r.WindowToggleMaximise())
  }

  return (
    <div
      className="titlebar-no-drag flex flex-shrink-0 items-center gap-[6px] px-2.5"
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      <TrafficLightButton hovered={hovered} onClick={handleClose} title={t("关闭", "Close")} type="close" />
      <TrafficLightButton hovered={hovered} onClick={handleMinimise} title={t("最小化", "Minimize")} type="minimise" />
      <TrafficLightButton hovered={hovered} onClick={handleMaximise} title={t("最大化/还原", "Maximize/Restore")} type="maximise" />
    </div>
  )
}
