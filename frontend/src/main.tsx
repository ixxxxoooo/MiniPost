import React from 'react'
import { createRoot } from 'react-dom/client'
import '@/styles/globals.css'
import App from './App'
import { ErrorBoundary } from './components/ErrorBoundary'
import { initGlobalErrorHandlers, info } from '@/lib/logger'

initGlobalErrorHandlers()
info("App", "MiniPost 前端启动中...")

const container = document.getElementById('root')
const root = createRoot(container!)

root.render(
  <React.StrictMode>
    <ErrorBoundary>
      <App />
    </ErrorBoundary>
  </React.StrictMode>
)

info("App", "React 根组件已挂载")
