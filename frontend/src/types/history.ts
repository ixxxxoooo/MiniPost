import type { HttpMethod } from "@/lib/constants"

export interface HistoryEntry {
  id: string
  requestId?: string
  name: string
  method: HttpMethod
  url: string
  statusCode: number
  duration: number
  size: number
  timestamp: string
}
