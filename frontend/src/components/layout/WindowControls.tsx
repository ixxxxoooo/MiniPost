import { useState } from "react"

export function WindowControls() {
  const [hovered, setHovered] = useState(false)

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
      className="flex items-center gap-[7px] px-3 titlebar-no-drag flex-shrink-0"
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      <button
        onClick={handleClose}
        className="w-[12px] h-[12px] rounded-full flex items-center justify-center transition-colors focus:outline-none"
        style={{ backgroundColor: hovered ? "#ff5f57" : "var(--fg-muted)" }}
        title="关闭"
      >
        {hovered && (
          <svg width="6" height="6" viewBox="0 0 6 6" fill="none">
            <path d="M0.5 0.5L5.5 5.5M5.5 0.5L0.5 5.5" stroke="#4a0002" strokeWidth="1.2" strokeLinecap="round" />
          </svg>
        )}
      </button>
      <button
        onClick={handleMinimise}
        className="w-[12px] h-[12px] rounded-full flex items-center justify-center transition-colors focus:outline-none"
        style={{ backgroundColor: hovered ? "#febc2e" : "var(--fg-muted)" }}
        title="最小化"
      >
        {hovered && (
          <svg width="6" height="2" viewBox="0 0 6 2" fill="none">
            <path d="M0.5 1H5.5" stroke="#995700" strokeWidth="1.2" strokeLinecap="round" />
          </svg>
        )}
      </button>
      <button
        onClick={handleMaximise}
        className="w-[12px] h-[12px] rounded-full flex items-center justify-center transition-colors focus:outline-none"
        style={{ backgroundColor: hovered ? "#28c840" : "var(--fg-muted)" }}
        title="最大化/还原"
      >
        {hovered && (
          <svg width="6" height="6" viewBox="0 0 6 6" fill="none">
            <path d="M1 4.5L3 1.5L5 4.5" stroke="#006500" strokeWidth="1.2" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
        )}
      </button>
    </div>
  )
}
