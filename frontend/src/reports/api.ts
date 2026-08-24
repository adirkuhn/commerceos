import type { SalesSummaryReport, ProductPerformanceReport, ProductPerformanceMatrix, CustomerPerformanceReport, CategoryPerformanceReport, SlowMovingProductsReport, ProfitabilityReport, InventorySalesReport, APIResponse } from './types'

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

export function getProductPerformanceMatrix(token: string, from: string, to: string): Promise<ProductPerformanceMatrix> {
  return request<ProductPerformanceMatrix>(`${base()}/product-performance-matrix?${rangeParams(from, to)}`, { method: 'GET' }, token)
}

export function getCustomerPerformanceReport(token: string, from: string, to: string): Promise<CustomerPerformanceReport> {
  return request<CustomerPerformanceReport>(`${base()}/customer-performance?${rangeParams(from, to)}`, { method: 'GET' }, token)
}

export function getCategoryPerformanceReport(token: string, from: string, to: string): Promise<CategoryPerformanceReport> {
  return request<CategoryPerformanceReport>(`${base()}/category-performance?${rangeParams(from, to)}`, { method: 'GET' }, token)
}

export function getSlowMovingProductsReport(token: string, from: string, to: string): Promise<SlowMovingProductsReport> {
  return request<SlowMovingProductsReport>(`${base()}/slow-moving-products?${rangeParams(from, to)}`, { method: 'GET' }, token)
}

export function getProfitabilityReport(token: string, from: string, to: string): Promise<ProfitabilityReport> {
  return request<ProfitabilityReport>(`${base()}/profitability?${rangeParams(from, to)}`, { method: 'GET' }, token)
}

export function getInventorySalesReport(token: string, from: string, to: string): Promise<InventorySalesReport> {
  return request<InventorySalesReport>(`${base()}/inventory-sales?${rangeParams(from, to)}`, { method: 'GET' }, token)
}
