import type { APIResponse, ReorderRecommendations, ReorderRecommendationsQuery, SeasonEvent, SeasonPlan, SeasonPlanQuery } from './types'

function base(): string {
  return `${import.meta.env.VITE_API_BASE_URL}/procurement`
}

async function request<T>(path: string, token: string): Promise<T> {
  const res = await fetch(path, {
    method: 'GET',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
  })
  const body: APIResponse<T> = await res.json()
  if (!res.ok || body.error) {
    throw new Error(body.error?.message ?? `HTTP ${res.status}`)
  }
  return body.data as T
}

export function getReorderRecommendations(token: string, query: ReorderRecommendationsQuery): Promise<ReorderRecommendations> {
  const params = new URLSearchParams({
    coverage_days: String(query.coverage_days),
    safety_days: String(query.safety_days),
  })
  if (query.supplier_id) params.set('supplier_id', query.supplier_id)
  return request<ReorderRecommendations>(`${base()}/recommendations?${params}`, token)
}

export function listSeasonEvents(token: string, year?: number | null): Promise<SeasonEvent[]> {
  const query = year ? `?year=${year}` : ''
  return request<SeasonEvent[]>(`${base()}/season-events${query}`, token)
}

export function getSeasonPlan(token: string, query: SeasonPlanQuery): Promise<SeasonPlan> {
  const params = new URLSearchParams()
  if (query.event) params.set('event', query.event)
  if (query.event && query.year) params.set('year', String(query.year))
  if (!query.event && query.from && query.to) {
    params.set('from', query.from)
    params.set('to', query.to)
  }
  params.set('safety_days', String(query.safety_days))
  if (query.supplier_id) params.set('supplier_id', query.supplier_id)
  if (query.all_products) params.set('all_products', 'true')
  return request<SeasonPlan>(`${base()}/season-plan?${params}`, token)
}
