import type { TenantSettings, TenantSettingsInput, ApiKey, ApiKeyCreated, APIResponse } from './types'
import { CodedApiError } from '../api/codedApiError'

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
    throw CodedApiError.fromEnvelope(body.error, res.status)
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

export async function listApiKeys(token: string): Promise<ApiKey[]> {
  const result = await request<{ items: ApiKey[] }>(`${base()}/api-keys`, { method: 'GET' }, token)
  return result.items
}

// Creates a new storefront API key -- the bearer token a connected
// storefront (e.g. the WooCommerce plugin) presents to call CommerceOS'
// authenticated public-storefront API. The response carries the
// PLAINTEXT value exactly once, it can't be retrieved again after this
// call (only the hash persists server-side).
export function createApiKey(token: string, name: string, description: string, salesChannelId: string | null = null): Promise<ApiKeyCreated> {
  return request<ApiKeyCreated>(
    `${base()}/api-keys`,
    { method: 'POST', body: JSON.stringify({ name, description, sales_channel_id: salesChannelId }) },
    token,
  )
}

export function revokeApiKey(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/api-keys/${id}`, { method: 'DELETE' }, token)
}
