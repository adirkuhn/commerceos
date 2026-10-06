import * as api from '../api'
import type { ReorderRecommendations, ReorderRecommendationsQuery } from '../types'
import { useLatestRequest, type RequestState } from './useLatestRequest'

export function useReorderRecommendations(token: string, query: ReorderRecommendationsQuery): RequestState<ReorderRecommendations> {
  const { coverage_days, safety_days, supplier_id } = query
  return useLatestRequest(token, JSON.stringify([coverage_days, safety_days, supplier_id]),
    t => api.getReorderRecommendations(t, { coverage_days, safety_days, supplier_id }))
}
