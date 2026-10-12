import type { APIResponse, DashboardStatsReport, SalesComparison, SalesPeriod, TodayQueue } from './types'
import { CodedApiError } from '../api/codedApiError'

function base(): string {
  return `${import.meta.env.VITE_API_BASE_URL}/dashboard`
}

export async function getDashboardStats(token: string): Promise<DashboardStatsReport> {
  const res = await fetch(`${base()}/stats`, {
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
  })
  const body: APIResponse<DashboardStatsReport> = await res.json()
  if (!res.ok || body.error) {
    throw CodedApiError.fromEnvelope(body.error, res.status)
  }
  return body.data as DashboardStatsReport
}

export async function getTodayQueue(token: string): Promise<TodayQueue> {
  const res = await fetch(`${base()}/today`, {
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
  })
  const body: APIResponse<TodayQueue> = await res.json()
  if (!res.ok || body.error) {
    throw CodedApiError.fromEnvelope(body.error, res.status)
  }
  return body.data as TodayQueue
}

export async function getSalesComparison(token: string, period: SalesPeriod): Promise<SalesComparison> {
  const res = await fetch(`${base()}/sales?period=${encodeURIComponent(period)}`, {
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
  })
  const body: APIResponse<SalesComparison> = await res.json()
  if (!res.ok || body.error) {
    throw CodedApiError.fromEnvelope(body.error, res.status)
  }
  return body.data as SalesComparison
}
