import * as api from '../api'
import type { SeasonEvent, SeasonPlan, SeasonPlanQuery } from '../types'
import { useLatestRequest, type RequestState } from './useLatestRequest'

// Waits for a complete season: an event, or both ends of a custom range.
export function useSeasonPlan(token: string, query: SeasonPlanQuery): RequestState<SeasonPlan> {
  const complete = Boolean(query.event || (query.from && query.to))
  return useLatestRequest(token, JSON.stringify(query), t => api.getSeasonPlan(t, query), complete)
}

export function useSeasonEvents(token: string): RequestState<SeasonEvent[]> {
  return useLatestRequest(token, 'next', t => api.listSeasonEvents(t))
}
