import type { APIResponse } from '../types'

// The storefront key is a bearer token at public/anonymous-shopper
// privilege only, same trust model as a Stripe publishable key -- it
// can't do anything a shopper standing in front of this site shouldn't
// already be able to do (see ResolvedAPIKey.SalesChannelID on the
// backend). That's what makes it safe to resolve at runtime instead of
// baking one into the build: this same deployed app serves any number of
// tenants/channels, each identified by its own key in the URL (see
// StoreEntry, which calls setStorefrontKey once on load). VITE_STOREFRONT_API_KEY
// remains a build-time fallback for a single-tenant deployment that never
// visits a /store/:key URL at all.
const STORAGE_KEY = 'commerceos_storefront_key'

let storefrontKey = readStoredKey() || (import.meta.env.VITE_STOREFRONT_API_KEY ?? '')

function readStoredKey(): string {
  try {
    return localStorage.getItem(STORAGE_KEY) ?? ''
  } catch {
    return ''
  }
}

export function setStorefrontKey(key: string): void {
  storefrontKey = key
  try {
    localStorage.setItem(STORAGE_KEY, key)
  } catch {
    // Private-browsing/storage-disabled: the key still works for the rest
    // of this page load via the in-memory variable, just doesn't persist
    // across a fresh visit.
  }
}

function base(): string {
  return `${import.meta.env.VITE_API_BASE_URL}/storefront/v1`
}

export async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const res = await fetch(`${base()}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${storefrontKey}`,
      ...options.headers,
    },
  })
  const body: APIResponse<T> = await res.json()
  if (!res.ok || body.error) {
    throw new Error(body.error?.message ?? `HTTP ${res.status}`)
  }
  return body.data as T
}
