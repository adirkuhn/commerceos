import type { APIResponse, ReorderRecommendations, ReorderRecommendationsQuery } from './types'

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
