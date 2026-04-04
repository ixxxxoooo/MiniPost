import { create } from "zustand"
import type { RequestData } from "@/types/request"
import type { HttpResponse } from "@/types/response"
import { createDefaultRequest } from "@/types/request"

interface RequestState {
  currentRequest: RequestData
  savedSnapshot: string | null
  response: HttpResponse | null
  responseError: string | null

  setCurrentRequest: (request: RequestData) => void
  updateCurrentRequest: (partial: Partial<RequestData>) => void
  setResponse: (response: HttpResponse | null) => void
  setResponseError: (error: string | null) => void
  resetRequest: () => void
  markSaved: () => void
  isDirty: () => boolean
}

function requestToSnapshot(request: RequestData): string {
  const { updatedAt, ...rest } = request
  return JSON.stringify(rest)
}

export const useRequestStore = create<RequestState>((set, get) => ({
  currentRequest: createDefaultRequest(),
  savedSnapshot: null,
  response: null,
  responseError: null,

  setCurrentRequest: (currentRequest) => {
    set({
      currentRequest,
      savedSnapshot: requestToSnapshot(currentRequest),
      response: null,
      responseError: null,
    })
  },

  updateCurrentRequest: (partial) =>
    set((s) => ({
      currentRequest: {
        ...s.currentRequest,
        ...partial,
        updatedAt: new Date().toISOString(),
      },
    })),

  setResponse: (response) => set({ response, responseError: null }),
  setResponseError: (responseError) => set({ responseError, response: null }),

  resetRequest: () =>
    set({
      currentRequest: createDefaultRequest(),
      savedSnapshot: null,
      response: null,
      responseError: null,
    }),

  markSaved: () =>
    set((s) => ({
      savedSnapshot: requestToSnapshot(s.currentRequest),
    })),

  isDirty: () => {
    const { currentRequest, savedSnapshot } = get()
    if (!savedSnapshot) return false
    return requestToSnapshot(currentRequest) !== savedSnapshot
  },
}))
