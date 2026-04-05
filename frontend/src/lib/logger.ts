type LogLevel = "DEBUG" | "INFO" | "WARN" | "ERROR"

const LOG_LEVEL_PRIORITY: Record<LogLevel, number> = {
  DEBUG: 0,
  INFO: 1,
  WARN: 2,
  ERROR: 3,
}

let currentLevel: LogLevel = import.meta.env.DEV ? "DEBUG" : "INFO"

function shouldLog(level: LogLevel): boolean {
  return LOG_LEVEL_PRIORITY[level] >= LOG_LEVEL_PRIORITY[currentLevel]
}

function formatLog(level: LogLevel, module: string, message: string, data?: Record<string, unknown>): string {
  const timestamp = new Date().toISOString()
  const dataStr = data ? " " + Object.entries(data).map(([k, v]) => `${k}=${JSON.stringify(v)}`).join(" ") : ""
  return `${timestamp} [${level}] [${module}] ${message}${dataStr}`
}

export function setLogLevel(level: LogLevel) {
  currentLevel = level
}

export function debug(module: string, message: string, data?: Record<string, unknown>) {
  if (!shouldLog("DEBUG")) return
  console.debug(formatLog("DEBUG", module, message, data))
}

export function info(module: string, message: string, data?: Record<string, unknown>) {
  if (!shouldLog("INFO")) return
  console.info(formatLog("INFO", module, message, data))
}

export function warn(module: string, message: string, data?: Record<string, unknown>) {
  if (!shouldLog("WARN")) return
  console.warn(formatLog("WARN", module, message, data))
}

export function error(module: string, message: string, data?: Record<string, unknown>) {
  if (!shouldLog("ERROR")) return
  console.error(formatLog("ERROR", module, message, data))
}

// 注册全局未捕获错误处理
export function initGlobalErrorHandlers() {
  window.addEventListener("error", (event) => {
    error("GlobalError", "未捕获的运行时错误", {
      message: event.message,
      filename: event.filename,
      lineno: event.lineno,
      colno: event.colno,
    })
  })

  window.addEventListener("unhandledrejection", (event) => {
    error("GlobalError", "未处理的 Promise 拒绝", {
      reason: String(event.reason),
    })
  })

  info("App", "全局错误处理器已注册")
}
