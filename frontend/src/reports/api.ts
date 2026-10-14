import type { SalesSummaryReport, ProductPerformanceReport, ProductPerformanceMatrix, CustomerPerformanceReport, CategoryPerformanceReport, SlowMovingProductsReport, ProfitabilityReport, ProfitabilityTrendReport, RevenueByChannelReport, InventorySalesReport, BusinessInsightsReport, SalesTrendsReport, SalesTrendGranularity, ShipmentsReport, ShipmentGranularity, ShipmentBasis, ProductLifecycleReport, CustomerCohortsReport, PromotionPerformanceReport, CustomerRFMReport, FrequentlyBoughtTogetherReport, StockoutRiskReport, ProductRecommendationsReport, APIResponse } from './types'
import { CodedApiError } from '../api/codedApiError'

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
    throw CodedApiError.fromEnvelope(body.error, res.status)
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

export function getProfitabilityTrendReport(token: string, from: string, to: string, granularity: SalesTrendGranularity): Promise<ProfitabilityTrendReport> {
  return request<ProfitabilityTrendReport>(`${base()}/profitability-trend?${rangeParams(from, to)}&granularity=${granularity}`, { method: 'GET' }, token)
}

export function getRevenueByChannelReport(token: string, from: string, to: string, granularity: SalesTrendGranularity): Promise<RevenueByChannelReport> {
  return request<RevenueByChannelReport>(`${base()}/revenue-by-channel?${rangeParams(from, to)}&granularity=${granularity}`, { method: 'GET' }, token)
}

export function getBusinessInsights(token: string, from: string, to: string): Promise<BusinessInsightsReport> {
  return request<BusinessInsightsReport>(`${base()}/business-insights?${rangeParams(from, to)}`, { method: 'GET' }, token)
}

export function getShipmentsReport(token: string, from: string, to: string, granularity: ShipmentGranularity, basis: ShipmentBasis): Promise<ShipmentsReport> {
  return request<ShipmentsReport>(`${base()}/shipments?${rangeParams(from, to)}&granularity=${granularity}&basis=${basis}`, { method: 'GET' }, token)
}

export function getSalesTrendsReport(token: string, from: string, to: string, granularity: SalesTrendGranularity): Promise<SalesTrendsReport> {
  return request<SalesTrendsReport>(`${base()}/sales-trends?${rangeParams(from, to)}&granularity=${granularity}`, { method: 'GET' }, token)
}

export function getProductLifecycleReport(token: string, from: string, to: string): Promise<ProductLifecycleReport> {
  return request<ProductLifecycleReport>(`${base()}/product-lifecycle?${rangeParams(from, to)}`, { method: 'GET' }, token)
}

export function getCustomerCohortsReport(token: string, from: string, to: string): Promise<CustomerCohortsReport> {
  return request<CustomerCohortsReport>(`${base()}/customer-cohorts?${rangeParams(from, to)}`, { method: 'GET' }, token)
}

export function getPromotionPerformanceReport(token: string, promotionId: string): Promise<PromotionPerformanceReport> {
  return request<PromotionPerformanceReport>(`${base()}/promotions/${promotionId}/performance`, { method: 'GET' }, token)
}

export function getCustomerRFMReport(token: string, from: string, to: string): Promise<CustomerRFMReport> {
  return request<CustomerRFMReport>(`${base()}/customer-rfm?${rangeParams(from, to)}`, { method: 'GET' }, token)
}

export function getFrequentlyBoughtTogetherReport(token: string, from: string, to: string): Promise<FrequentlyBoughtTogetherReport> {
  return request<FrequentlyBoughtTogetherReport>(`${base()}/frequently-bought-together?${rangeParams(from, to)}`, { method: 'GET' }, token)
}

export function getStockoutRiskReport(token: string, from: string, to: string): Promise<StockoutRiskReport> {
  return request<StockoutRiskReport>(`${base()}/stockout-risk?${rangeParams(from, to)}`, { method: 'GET' }, token)
}

export function getProductRecommendations(token: string, skuId: string, from: string, to: string): Promise<ProductRecommendationsReport> {
  return request<ProductRecommendationsReport>(`${base()}/product-recommendations?sku_id=${encodeURIComponent(skuId)}&${rangeParams(from, to)}`, { method: 'GET' }, token)
}
