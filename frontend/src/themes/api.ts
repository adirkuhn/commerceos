import type { SessionPurpose, SessionToken, Theme, ThemePreset, VersionSummary } from './types'
import { CodedApiError } from '../api/codedApiError'

function base(): string {
  return `${import.meta.env.VITE_API_BASE_URL}/theme`
}

interface Envelope<T> {
  data: T | null
  total?: number
  error: { code?: string; message?: string } | null
}

async function request<T>(path: string, options: RequestInit, token: string): Promise<Envelope<T>> {
  const res = await fetch(`${base()}${path}`, {
    ...options,
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}`, ...options.headers },
  })
  const body: Envelope<T> = await res.json()
  if (!res.ok || body.error) throw CodedApiError.fromEnvelope(body.error, res.status)
  return body
}

export async function listThemes(token: string, salesChannelId: string, page = 1, perPage = 50): Promise<{ items: Theme[]; total: number }> {
  const q = new URLSearchParams({ sales_channel_id: salesChannelId, page: String(page), per_page: String(perPage) })
  const body = await request<Theme[]>(`/themes?${q}`, { method: 'GET' }, token)
  return { items: body.data ?? [], total: body.total ?? 0 }
}

/** presetKey: what the theme starts from (see ThemePreset). */
export async function createTheme(token: string, salesChannelId: string, name: string, presetKey: ThemePreset = 'default'): Promise<Theme> {
  return (await request<Theme>('/themes', { method: 'POST', body: JSON.stringify({ sales_channel_id: salesChannelId, name, preset_key: presetKey }) }, token)).data!
}

export async function renameTheme(token: string, id: string, name: string): Promise<Theme> {
  return (await request<Theme>(`/themes/${id}`, { method: 'PATCH', body: JSON.stringify({ name }) }, token)).data!
}

export async function deleteTheme(token: string, id: string): Promise<void> {
  await request<null>(`/themes/${id}`, { method: 'DELETE' }, token)
}

export async function listVersions(token: string, id: string): Promise<VersionSummary[]> {
  return (await request<VersionSummary[]>(`/themes/${id}/versions?per_page=50`, { method: 'GET' }, token)).data ?? []
}

export async function rollbackTheme(token: string, id: string, versionId: string): Promise<Theme> {
  return (await request<Theme>(`/themes/${id}/rollback`, { method: 'POST', body: JSON.stringify({ version_id: versionId }) }, token)).data!
}

/** locale: the merchant's language -- the editor opened with this session speaks it. */
export async function createThemeSession(token: string, id: string, purpose: SessionPurpose, locale?: string): Promise<SessionToken> {
  return (await request<SessionToken>(`/themes/${id}/sessions`, { method: 'POST', body: JSON.stringify({ purpose, locale }) }, token)).data!
}
