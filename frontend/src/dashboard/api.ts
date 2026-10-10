import type { APIResponse, DashboardStatsReport } from './types'
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
    throw new CodedApiError(body.error?.code ?? 'HTTP_ERROR', body.error?.message ?? `HTTP ${res.status}`)
  }
  return body.data as DashboardStatsReport
}
