import { getSuppressedAutoHeaders } from "@/lib/autoHeaders"
import {
  buildCurlCommand,
  resolveCurlRequestUrl,
  type CurlRequest,
  type VariableLike,
} from "@/lib/curlCommand"
import { useCookieStore } from "@/stores/cookieStore"
import { useUIStore } from "@/stores/uiStore"

export type RuntimeCurlCommand = {
  command: string
  cookieIncluded: boolean
}

export function buildRuntimeCurlCommand(
  request: CurlRequest,
  variables: VariableLike[]
): RuntimeCurlCommand {
  const cookiesDisabled = useUIStore.getState().disableCookies
  const cookieSuppressed = getSuppressedAutoHeaders(request.headers ?? []).has("cookie")
  const cookieHeader = cookiesDisabled || cookieSuppressed
    ? ""
    : useCookieStore.getState().getCookieHeader(resolveCurlRequestUrl(request, variables))

  return {
    command: buildCurlCommand(request, variables, { cookieHeader }),
    cookieIncluded: Boolean(cookieHeader),
  }
}
