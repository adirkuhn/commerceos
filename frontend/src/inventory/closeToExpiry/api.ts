import { CodedApiError } from '../../api/codedApiError'
import type { APIResponse } from '../types'
import type { CloseToExpiryList, CloseToExpiryQuery } from './types'
import { appEnv } from '../../config/appEnv'

function base(): string {
  return `${appEnv().VITE_API_BASE_URL}/inventory/batches`
}

async function send(url: string, init: RequestInit, token: string): Promise<Response> {
  return fetch(url, {
    ...init,
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}`, ...init.headers },
  })
}

// Throws the API's coded error for a failed response.
async function failure(res: Response): Promise<never> {
  const body: APIResponse<null> | null = await res.json().catch(() => null)
  throw CodedApiError.fromEnvelope(body?.error ?? null, res.status)
}

export async function listCloseToExpiry(token: string, q: CloseToExpiryQuery): Promise<CloseToExpiryList> {
  const params = new URLSearchParams()
  if (q.window) params.set('window', String(q.window))
  if (q.acknowledged) params.set('acknowledged', 'true')
  if (q.page && q.page > 1) params.set('page', String(q.page))
  if (q.perPage) params.set('per_page', String(q.perPage))
  const qs = params.toString()
  const res = await send(`${base()}/close-to-expiry${qs ? `?${qs}` : ''}`, { method: 'GET' }, token)
  if (!res.ok) return failure(res)
  const body: APIResponse<CloseToExpiryList> = await res.json()
  if (body.error) throw CodedApiError.fromEnvelope(body.error, res.status)
  return body.data as CloseToExpiryList
}

/** Mark a batch as seen for the window it is in now ("aware"). */
export async function acknowledgeBatchExpiry(token: string, batchId: string, windowDays: number): Promise<void> {
  const res = await send(`${base()}/${encodeURIComponent(batchId)}/expiry-acknowledgements`,
    { method: 'POST', body: JSON.stringify({ window_days: windowDays }) }, token)
  if (!res.ok) return failure(res)
}

/** Undo marking a batch as seen for a window. */
export async function unacknowledgeBatchExpiry(token: string, batchId: string, windowDays: number): Promise<void> {
  const res = await send(`${base()}/${encodeURIComponent(batchId)}/expiry-acknowledgements/${windowDays}`, { method: 'DELETE' }, token)
  if (!res.ok) return failure(res)
}
