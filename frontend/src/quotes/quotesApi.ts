import { CodedApiError } from '../api/codedApiError'
import type { QuoteInput, SavedQuote, SavedQuotePage, SavedQuoteSummary } from './types'

interface Envelope<T> {
  data: T | null
  meta?: { total: number }
  error: { code: string; message: string } | null
}

const base = () => import.meta.env.VITE_API_BASE_URL

async function call<T>(token: string, path: string, init: RequestInit = {}): Promise<Envelope<T>> {
  const res = await fetch(`${base()}${path}`, {
    ...init,
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
  })
  if (res.status === 204) return { data: null, error: null }
  const body: Envelope<T> = await res.json()
  if (!res.ok || body.error) throw CodedApiError.fromEnvelope(body.error, res.status)
  return body
}

export async function listQuotes(token: string, query: { page?: number; perPage?: number; search?: string }): Promise<SavedQuotePage> {
  const qs = new URLSearchParams()
  if (query.page) qs.set('page', String(query.page))
  if (query.perPage) qs.set('per_page', String(query.perPage))
  if (query.search) qs.set('search', query.search)
  const params = qs.toString()
  const body = await call<SavedQuoteSummary[]>(token, `/quotes${params ? `?${params}` : ''}`)
  return { quotes: body.data ?? [], total: body.meta?.total ?? 0 }
}

export async function getQuote(token: string, id: string): Promise<SavedQuote> {
  return (await call<SavedQuote>(token, `/quotes/${id}`)).data as SavedQuote
}

export async function createQuote(token: string, input: QuoteInput): Promise<SavedQuote> {
  return (await call<SavedQuote>(token, '/quotes', { method: 'POST', body: JSON.stringify(input) })).data as SavedQuote
}

export async function updateQuote(token: string, id: string, input: QuoteInput): Promise<SavedQuote> {
  return (await call<SavedQuote>(token, `/quotes/${id}`, { method: 'PUT', body: JSON.stringify(input) })).data as SavedQuote
}

export async function deleteQuote(token: string, id: string): Promise<void> {
  await call<null>(token, `/quotes/${id}`, { method: 'DELETE' })
}
