// Where the storefront app (shops + theme editor) is served: /store on
// VITE_STOREFRONT_URL, or on the API's own origin when that isn't set
// (the storefront is path-mounted on API_DOMAIN -- see docker-compose.yml).
export function storefrontOrigin(env: { VITE_STOREFRONT_URL?: string; VITE_API_BASE_URL?: string } = import.meta.env): string {
  if (env.VITE_STOREFRONT_URL) return env.VITE_STOREFRONT_URL.replace(/\/+$/, '')
  try {
    return new URL(env.VITE_API_BASE_URL ?? '').origin
  } catch {
    return ''
  }
}

export const storeURL = (channelId: string, origin = storefrontOrigin()) => `${origin}/store/${encodeURIComponent(channelId)}/`

/** The token rides in the fragment: it never reaches a server or its logs. */
export const editorURL = (sessionToken: string, origin = storefrontOrigin()) =>
  `${origin}/store/editor#session=${encodeURIComponent(sessionToken)}`
