import type { TenantSettings, TenantSettingsInput, APIResponse } from './types'

function base(): string {
  return `${import.meta.env.VITE_API_BASE_URL}/settings`
}

async function request<T>(
  path: string,
  options: RequestInit,
  token: string,
): Promise<T> {
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

export function getTenantSettings(token: string): Promise<TenantSettings> {
  return request<TenantSettings>(`${base()}/tenant`, { method: 'GET' }, token)
}

export function updateTenantSettings(token: string, input: TenantSettingsInput): Promise<TenantSettings> {
  return request<TenantSettings>(
    `${base()}/tenant`,
    { method: 'PATCH', body: JSON.stringify(input) },
    token,
  )
}
