import { useLatestRequest, type RequestState } from '../../purchaseplanning/hooks/useLatestRequest'
import { listSalesOrders } from '../../oms/api'
import type { SalesOrder } from '../../oms/types'

/** The sales orders already created from a saved quote. */
export function useQuoteOrders(token: string, quoteId: string | null): RequestState<SalesOrder[]> {
  return useLatestRequest(token, quoteId ?? '',
    async t => (await listSalesOrders(t, { quoteId: quoteId as string, perPage: 50 })).salesOrders, quoteId !== null)
}
