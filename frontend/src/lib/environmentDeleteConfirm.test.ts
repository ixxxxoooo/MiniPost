import { describe, expect, it } from "vitest"
import { getEnvironmentDeleteConfirmMessage, getEnvironmentDeleteConfirmTitle } from "./environmentDeleteConfirm"

describe("environment delete confirmation copy", () => {
  it("builds Chinese confirmation copy with the environment name and irreversible warning", () => {
    const t = (zh: string) => zh

    expect(getEnvironmentDeleteConfirmTitle(t)).toBe("确认删除环境")
    expect(getEnvironmentDeleteConfirmMessage(t, "测试环境")).toBe(
      "确定要删除环境「测试环境」吗？该操作不可撤销，环境变量配置将被永久删除。"
    )
  })

  it("builds English confirmation copy with the environment name and irreversible warning", () => {
    const t = (_zh: string, en: string) => en

    expect(getEnvironmentDeleteConfirmTitle(t)).toBe("Confirm environment deletion")
    expect(getEnvironmentDeleteConfirmMessage(t, "Staging")).toBe(
      "Are you sure you want to delete environment「Staging」? This action cannot be undone. Environment variables will be permanently deleted."
    )
  })
})
