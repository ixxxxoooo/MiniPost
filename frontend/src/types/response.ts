export interface HttpResponse {
  statusCode: number
  statusText: string
  headers: Record<string, string[]>
  body: string
  duration: number
  size: number
  contentType: string
}
