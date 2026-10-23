import { useLatestRequest, type RequestState } from '../../purchaseplanning/hooks/useLatestRequest'
import { listPriceList } from '../api'
import type { PriceListPage, PriceListQuery } from '../types'

export function usePriceList(token: string, query: PriceListQuery, enabled = true): RequestState<PriceListPage> {
  const { page, perPage, search, productIds } = query
  return useLatestRequest(token, JSON.stringify([page, perPage, search, productIds]),
    t => listPriceList(t, { page, perPage, search, productIds }), enabled)
}
