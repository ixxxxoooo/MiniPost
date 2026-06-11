type Translate = (zh: string, en: string) => string

export function getEnvironmentDeleteConfirmTitle(t: Translate): string {
  return t("确认删除环境", "Confirm environment deletion")
}

export function getEnvironmentDeleteConfirmMessage(t: Translate, environmentName: string): string {
  return `${t("确定要删除环境", "Are you sure you want to delete environment")}「${environmentName}」${t("吗？该操作不可撤销，环境变量配置将被永久删除。", "? This action cannot be undone. Environment variables will be permanently deleted.")}`
}
