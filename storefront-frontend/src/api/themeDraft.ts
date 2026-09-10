import type { APIResponse, PageSlug, ThemePage } from '../types'

// The theme-edit-session bearer token is a distinct, higher-privilege
// credential from the public storefront key api/client.ts's request()
// sends -- see backend application/theme.ResolveEditSessionUseCase's own
// doc comment. It's passed explicitly per call (from the /edit/:slug
// route's own ?token= query param, see pages/EditPage.tsx), never stored
// in localStorage alongside the public key: it authorizes writes, and its
// only holder is the merchant currently in the embedded Puck editor.
function base(): string {
  return `${import.meta.env.VITE_API_BASE_URL}/storefront/v1/theme/draft`
}

async function draftRequest<T>(path: string, token: string, options: RequestInit = {}): Promise<T> {
  const res = await fetch(`${base()}${path}`, {
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

export function getThemeDraft(token: string, slug: PageSlug): Promise<ThemePage> {
  return draftRequest<ThemePage>(`/pages/${slug}`, token, { method: 'GET' })
}

export interface SaveThemeDraftPayload {
  canvas_width: number
  blocks: ThemePage['blocks']
  custom_css: string
  editor_mode: ThemePage['editor_mode']
  puck_data?: Record<string, unknown> | null
}

export function saveThemeDraft(token: string, slug: PageSlug, payload: SaveThemeDraftPayload): Promise<ThemePage> {
  return draftRequest<ThemePage>(`/pages/${slug}`, token, { method: 'PUT', body: JSON.stringify(payload) })
}
