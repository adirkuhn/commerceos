import type { Coupon, CouponInput, CouponCurrencyAmount, UpsertCouponCurrencyAmountInput, APIResponse } from './types'

function base(): string {
  return `${import.meta.env.VITE_API_BASE_URL}/coupons`
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

export function listCoupons(token: string): Promise<Coupon[]> {
  return request<Coupon[]>(`${base()}`, { method: 'GET' }, token)
}

export function createCoupon(token: string, input: CouponInput): Promise<Coupon> {
  return request<Coupon>(`${base()}`, { method: 'POST', body: JSON.stringify(input) }, token)
}

export function updateCoupon(token: string, id: string, input: CouponInput): Promise<Coupon> {
  return request<Coupon>(`${base()}/${id}`, { method: 'PATCH', body: JSON.stringify(input) }, token)
}

export function deleteCoupon(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/${id}`, { method: 'DELETE' }, token)
}

export function listCouponCurrencyAmounts(token: string, id: string): Promise<CouponCurrencyAmount[]> {
  return request<CouponCurrencyAmount[]>(`${base()}/${id}/currency-amounts`, { method: 'GET' }, token)
}

export function upsertCouponCurrencyAmount(token: string, id: string, input: UpsertCouponCurrencyAmountInput): Promise<CouponCurrencyAmount> {
  return request<CouponCurrencyAmount>(`${base()}/${id}/currency-amounts`, { method: 'PUT', body: JSON.stringify(input) }, token)
}

export function deleteCouponCurrencyAmount(token: string, id: string, currencyCode: string): Promise<null> {
  return request<null>(`${base()}/${id}/currency-amounts/${currencyCode}`, { method: 'DELETE' }, token)
}
