import { useLatestRequest, type RequestState } from '../../purchaseplanning/hooks/useLatestRequest'
import { getQuote, listQuotes } from '../quotesApi'
import type { SavedQuote, SavedQuotePage } from '../types'

export function useSavedQuote(token: string, id: string | null): RequestState<SavedQuote> {
  return useLatestRequest(token, id ?? '', t => getQuote(t, id as string), id !== null)
}

export function useSavedQuotes(token: string, query: { page: number; perPage: number; search: string }): RequestState<SavedQuotePage> {
  const { page, perPage, search } = query
  return useLatestRequest(token, JSON.stringify([page, perPage, search]), t => listQuotes(t, { page, perPage, search }))
}
