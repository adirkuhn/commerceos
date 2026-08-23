import type { SalesSummaryReport, ProductPerformanceReport, APIResponse } from './types'

function base(): string {
  return `${import.meta.env.VITE_API_BASE_URL}/reports`
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

function rangeParams(from: string, to: string): string {
  return `from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`
}

export function getSalesSummaryReport(token: string, from: string, to: string): Promise<SalesSummaryReport> {
  return request<SalesSummaryReport>(`${base()}/sales-summary?${rangeParams(from, to)}`, { method: 'GET' }, token)
}

export function getProductPerformanceReport(token: string, from: string, to: string): Promise<ProductPerformanceReport> {
  return request<ProductPerformanceReport>(`${base()}/product-performance?${rangeParams(from, to)}`, { method: 'GET' }, token)
}
