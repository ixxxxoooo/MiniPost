type AppBackupAPI = {
  CreateBackup: () => Promise<string>
  RestoreBackup: (backupPath: string) => Promise<void>
}

type WailsAppBridge = {
  main?: {
    App?: AppBackupAPI
  }
}

function getBackupAPI(): AppBackupAPI {
  const bridge = (window as Window & { go?: WailsAppBridge }).go
  const app = bridge?.main?.App
  if (!app || typeof app.CreateBackup !== "function" || typeof app.RestoreBackup !== "function") {
    throw new Error("备份能力不可用，请重启应用后重试")
  }
  return app
}

export const backupService = {
  createBackup: async () => getBackupAPI().CreateBackup(),
  restoreBackup: async (backupPath: string) => getBackupAPI().RestoreBackup(backupPath),
}
