import type { TenantSettings, TenantSettingsInput, StorefrontKeyResult, APIResponse } from './types'

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

// (Re)generates the tenant's storefront key -- the bearer token a
// connected storefront (e.g. the WooCommerce plugin) presents to call
// CommerceOS' authenticated public-storefront API. The response carries
// the PLAINTEXT value exactly once, it can't be retrieved again after this
// call (only the hash persists server-side).
export function setStorefrontKey(token: string): Promise<StorefrontKeyResult> {
  return request<StorefrontKeyResult>(`${base()}/storefront-key`, { method: 'POST' }, token)
}
