export function shouldAutoFormatJsonOnEntry(jsonBody: string, alreadyFormattedForTab: boolean): boolean {
  return !alreadyFormattedForTab && jsonBody.trim().length > 0
}
