export interface TimingBreakdown {
  prepare: number
  socketInitialization: number
  dnsLookup: number
  tcpHandshake: number
  sslHandshake: number
  waitingTTFB: number
  download: number
  process: number
  total: number
}

export interface SizeBreakdown {
  responseHeaders: number
  responseBody: number
  responseTotal: number
  requestHeaders: number
  requestBody: number
  requestTotal: number
}

export interface HttpResponse {
  statusCode: number
  statusText: string
  headers: Record<string, string[]>
  body: string
  duration: number
  size: number
  contentType: string
  protocol?: string
  warnings?: string[]
  timings?: TimingBreakdown
  sizeDetails?: SizeBreakdown
}
