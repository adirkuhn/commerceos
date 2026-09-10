import type { APIResponse, EditSession, PageSlug, SaveThemePagePayload, ThemePage } from './types'

function base(): string {
  return `${import.meta.env.VITE_API_BASE_URL}/theme`
}

async function request<T>(path: string, options: RequestInit, token: string): Promise<T> {
  const res = await fetch(path, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
      ...options.headers,
    },
  })
  const body: APIResponse<T> = await res.json()
  if (!res.ok || body.error) {
    throw new Error(body.error?.message ?? `HTTP ${res.status}`)
  }
  return body.data as T
}

export function getThemePage(token: string, channelId: string, slug: PageSlug): Promise<ThemePage> {
  return request<ThemePage>(`${base()}/sales-channels/${channelId}/pages/${slug}`, { method: 'GET' }, token)
}

export function saveThemePage(
  token: string, channelId: string, slug: PageSlug, payload: SaveThemePagePayload,
): Promise<ThemePage> {
  return request<ThemePage>(
    `${base()}/sales-channels/${channelId}/pages/${slug}`,
    { method: 'PUT', body: JSON.stringify(payload) },
    token,
  )
}

// createEditSession mints the short-lived token storefront-frontend's
// embedded Puck editor (/edit/:slug) authenticates with -- a distinct,
// higher-privilege credential from a storefront API key, never shown to a
// shopper (see backend application/theme.CreateEditSessionUseCase's own
// doc comment).
export function createEditSession(token: string, channelId: string): Promise<EditSession> {
  return request<EditSession>(`${base()}/sales-channels/${channelId}/edit-sessions`, { method: 'POST' }, token)
}
