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

export interface NetworkDetails {
  httpVersion?: string
  localAddress?: string
  remoteAddress?: string
  tlsProtocol?: string
  cipherName?: string
  certificateCN?: string
  issuerCN?: string
  validUntil?: string
}

export interface HttpResponse {
  statusCode: number
  statusText: string
  headers: Record<string, string[]>
  body: string
  bodyBase64?: string
  bodyIsBinary?: boolean
  duration: number
  size: number
  contentType: string
  protocol?: string
  warnings?: string[]
  network?: NetworkDetails
  timings?: TimingBreakdown
  sizeDetails?: SizeBreakdown
}

export type StreamEntryKind = "response_start" | "data" | "event" | "chunk" | "connection_closed" | "error"

export interface HttpStreamEntry {
  id: string
  kind: StreamEntryKind
  data: string
  raw?: string
  timestamp: string
  sequence: number
  bytesTotal?: number
}

export interface HttpStreamEventPayload {
  streamId: string
  kind: StreamEntryKind
  data: string
  raw?: string
  timestamp: string
  sequence: number
  bytesTotal?: number
}
