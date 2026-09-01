import type { GlobalSearchResult, APIResponse } from './types'

function base(): string {
  return `${import.meta.env.VITE_API_BASE_URL}/search`
}

async function request<T>(path: string, options: RequestInit, token: string): Promise<T> {
  const res = await fetch(path, {
    ...options,
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}`, ...options.headers },
  })
  const body: APIResponse<T> = await res.json()
  if (!res.ok || body.error) {
    throw new Error(body.error?.message ?? `HTTP ${res.status}`)
  }
  return body.data as T
}

export function globalSearch(token: string, query: string): Promise<GlobalSearchResult> {
  return request<GlobalSearchResult>(`${base()}?q=${encodeURIComponent(query)}`, { method: 'GET' }, token)
}
