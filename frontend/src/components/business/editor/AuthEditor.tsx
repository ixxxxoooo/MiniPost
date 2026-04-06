import { AUTH_TYPES, type AuthType } from "@/lib/constants"
import { useI18n } from "@/hooks/useI18n"
import { useTabStore, getProjectActiveTabFromState } from "@/stores/tabStore"
import { cn } from "@/lib/utils"

export function AuthEditor() {
  const { t } = useI18n()
  const activeTab = useTabStore(getProjectActiveTabFromState)
  const updateTabRequest = useTabStore((s) => s.updateTabRequest)

  if (!activeTab) return null

  const { auth } = activeTab.request
  const tabId = activeTab.id

  const setAuthType = (type: AuthType) => {
    updateTabRequest(tabId, { auth: { ...auth, type } })
  }

  const inputClass = cn(
    "w-full h-[var(--size-input-sm)] px-3 rounded-[var(--radius-input)]",
    "border border-[var(--border-color)] bg-[var(--surface)] text-[var(--fg)]",
    "text-[length:var(--size-font-xs)] font-mono placeholder:text-[var(--fg-muted)]",
    "focus:outline-none focus:border-[var(--accent)]"
  )

  return (
    <div className="p-[var(--size-padding-sm)]">
      {/* Auth 类型选择 */}
      <div className="flex items-center gap-1 mb-3">
        {AUTH_TYPES.map((t) => (
          <button
            key={t}
            className={cn(
              "px-2 py-1 rounded-[var(--radius-sm)] text-[length:var(--size-font-2xs)] transition-colors",
              auth.type === t
                ? "bg-[var(--sidebar-active)] text-[var(--accent)] font-medium"
                : "text-[var(--fg-muted)] hover:text-[var(--fg-secondary)] hover:bg-[var(--sidebar-hover)]"
            )}
            onClick={() => setAuthType(t)}
          >
            {t === "none" ? "None" : t === "basic" ? "Basic" : t === "bearer" ? "Bearer" : "API Key"}
          </button>
        ))}
      </div>

      <div className="pl-2">
        {auth.type === "none" && (
          <div className="text-center py-8 text-2xs text-[var(--fg-muted)]">
            {t("此请求不需要认证", "This request does not require authentication")}
          </div>
        )}

        {auth.type === "basic" && (
          <div className="space-y-2 max-w-md">
            <div>
              <label className="text-2xs text-[var(--fg-secondary)] mb-1 block">{t("用户名", "Username")}</label>
              <input
                value={auth.basic?.username ?? ""}
                onChange={(e) => updateTabRequest(tabId, {
                  auth: { ...auth, basic: { ...auth.basic!, username: e.target.value } }
                })}
                placeholder="Username"
                className={inputClass}
              />
            </div>
            <div>
              <label className="text-2xs text-[var(--fg-secondary)] mb-1 block">{t("密码", "Password")}</label>
              <input
                type="password"
                value={auth.basic?.password ?? ""}
                onChange={(e) => updateTabRequest(tabId, {
                  auth: { ...auth, basic: { ...auth.basic!, password: e.target.value } }
                })}
                placeholder="Password"
                className={inputClass}
              />
            </div>
          </div>
        )}

        {auth.type === "bearer" && (
          <div className="max-w-md">
            <label className="text-2xs text-[var(--fg-secondary)] mb-1 block">Token</label>
            <input
              value={auth.bearer?.token ?? ""}
              onChange={(e) => updateTabRequest(tabId, {
                auth: { ...auth, bearer: { token: e.target.value } }
              })}
              placeholder="Bearer token..."
              className={inputClass}
            />
          </div>
        )}

        {auth.type === "api-key" && (
          <div className="space-y-2 max-w-md">
            <div>
              <label className="text-2xs text-[var(--fg-secondary)] mb-1 block">Key</label>
              <input
                value={auth.apiKey?.key ?? ""}
                onChange={(e) => updateTabRequest(tabId, {
                  auth: { ...auth, apiKey: { ...auth.apiKey!, key: e.target.value } }
                })}
                placeholder="API key name"
                className={inputClass}
              />
            </div>
            <div>
              <label className="text-2xs text-[var(--fg-secondary)] mb-1 block">Value</label>
              <input
                value={auth.apiKey?.value ?? ""}
                onChange={(e) => updateTabRequest(tabId, {
                  auth: { ...auth, apiKey: { ...auth.apiKey!, value: e.target.value } }
                })}
                placeholder="API key value"
                className={inputClass}
              />
            </div>
            <div>
              <label className="text-2xs text-[var(--fg-secondary)] mb-1 block">{t("添加位置", "Add to")}</label>
              <div className="flex items-center gap-2">
                {(["header", "query"] as const).map((pos) => (
                  <button
                    key={pos}
                    className={cn(
                      "px-2 py-1 rounded-[var(--radius-sm)] text-[length:var(--size-font-2xs)] transition-colors",
                      (auth.apiKey?.addTo ?? "header") === pos
                        ? "bg-[var(--sidebar-active)] text-[var(--accent)] font-medium"
                        : "text-[var(--fg-muted)] hover:bg-[var(--sidebar-hover)]"
                    )}
                    onClick={() => updateTabRequest(tabId, {
                      auth: { ...auth, apiKey: { ...auth.apiKey!, addTo: pos } }
                    })}
                  >
                    {pos === "header" ? "Header" : "Query Param"}
                  </button>
                ))}
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
