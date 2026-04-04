import { GetHistory, ClearHistory } from "../../wailsjs/go/main/App"

export const historyService = {
  getHistory: (projectId: string) => GetHistory(projectId),
  clearHistory: (projectId: string) => ClearHistory(projectId),
}
