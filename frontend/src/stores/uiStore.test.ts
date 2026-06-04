import { afterEach, describe, expect, it } from "vitest"
import { useUIStore } from "./uiStore"

describe("uiStore sidebar", () => {
  afterEach(() => {
    useUIStore.setState({ sidebarCollapsed: false })
  })

  it("sets sidebar collapse state explicitly and still supports toggling", () => {
    useUIStore.getState().setSidebarCollapsed(true)

    expect(useUIStore.getState().sidebarCollapsed).toBe(true)

    useUIStore.getState().toggleSidebar()

    expect(useUIStore.getState().sidebarCollapsed).toBe(false)
  })
})
