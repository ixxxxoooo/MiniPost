import React from "react"
import { normalizeLocale } from "@/lib/locale"
import { error as logError } from "@/lib/logger"

interface ErrorBoundaryState {
  hasError: boolean
  errorMessage: string
  errorStack: string
}

export class ErrorBoundary extends React.Component<
  { children: React.ReactNode },
  ErrorBoundaryState
> {
  constructor(props: { children: React.ReactNode }) {
    super(props)
    this.state = { hasError: false, errorMessage: "", errorStack: "" }
  }

  static getDerivedStateFromError(err: Error): ErrorBoundaryState {
    return {
      hasError: true,
      errorMessage: err.message,
      errorStack: err.stack ?? "",
    }
  }

  componentDidCatch(err: Error, info: React.ErrorInfo) {
    logError("ErrorBoundary", "React 组件渲染崩溃", {
      message: err.message,
      stack: err.stack ?? "",
      componentStack: info.componentStack ?? "",
    })
  }

  handleReload = () => {
    window.location.reload()
  }

  render() {
    const locale = normalizeLocale(typeof navigator === "undefined" ? "zh-CN" : navigator.language)
    const t = (zh: string, en: string) => (locale === "zh-CN" ? zh : en)

    if (this.state.hasError) {
      return (
        <div
          style={{
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
            justifyContent: "center",
            height: "100vh",
            padding: "40px",
            fontFamily: "-apple-system, BlinkMacSystemFont, sans-serif",
            color: "#333",
            background: "#fafafa",
          }}
        >
          <div style={{ fontSize: "40px", marginBottom: "16px", opacity: 0.3 }}>⚠️</div>
          <h2 style={{ fontSize: "16px", fontWeight: 600, marginBottom: "8px" }}>
            {t("应用发生了意外错误", "The app encountered an unexpected error")}
          </h2>
          <p style={{ fontSize: "13px", color: "#666", marginBottom: "16px", textAlign: "center" }}>
            {this.state.errorMessage}
          </p>
          <button
            onClick={this.handleReload}
            style={{
              padding: "8px 20px",
              fontSize: "13px",
              borderRadius: "6px",
              border: "1px solid #ddd",
              background: "#fff",
              cursor: "pointer",
            }}
          >
            {t("重新加载", "Reload")}
          </button>
          {import.meta.env.DEV && this.state.errorStack && (
            <pre
              style={{
                marginTop: "24px",
                padding: "16px",
                fontSize: "11px",
                background: "#f0f0f0",
                borderRadius: "8px",
                maxWidth: "600px",
                maxHeight: "200px",
                overflow: "auto",
                whiteSpace: "pre-wrap",
                wordBreak: "break-all",
              }}
            >
              {this.state.errorStack}
            </pre>
          )}
        </div>
      )
    }

    return this.props.children
  }
}
