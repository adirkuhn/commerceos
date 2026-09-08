import type { APIResponse } from '../types'

// The storefront key is embedded at build time (VITE_* vars are inlined
// into the bundle), same trust model as a Stripe publishable key: it's
// scoped to one internal sales channel at public/anonymous-shopper
// privilege, so it can't do anything a shopper standing in front of this
// site shouldn't already be able to do. See ResolvedAPIKey.SalesChannelID
// on the backend.
function base(): string {
  return `${import.meta.env.VITE_API_BASE_URL}/storefront/v1`
}

export async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const res = await fetch(`${base()}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${import.meta.env.VITE_STOREFRONT_API_KEY}`,
      ...options.headers,
    },
  })
  const body: APIResponse<T> = await res.json()
  if (!res.ok || body.error) {
    throw new Error(body.error?.message ?? `HTTP ${res.status}`)
  }
  return body.data as T
}
