import type { Subscriber, SubscriberQuery } from './types'

function base(): string {
  return `${import.meta.env.VITE_API_BASE_URL}/newsletter/subscribers`
}

interface Envelope<T> {
  data: T | null
  total?: number
  error: { code?: string; message?: string } | null
}

async function send(path: string, init: RequestInit, token: string): Promise<Response> {
  const res = await fetch(`${base()}${path}`, { ...init, headers: { Authorization: `Bearer ${token}`, ...init.headers } })
  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as Envelope<unknown> | null
    throw new Error(body?.error?.message ?? `HTTP ${res.status}`)
  }
  return res
}

export async function listSubscribers(token: string, q: SubscriberQuery): Promise<{ items: Subscriber[]; total: number }> {
  const params = new URLSearchParams({ page: String(q.page), per_page: String(q.perPage) })
  if (q.status) params.set('status', q.status)
  if (q.search) params.set('search', q.search)
  const body = (await (await send(`?${params}`, { method: 'GET' }, token)).json()) as Envelope<Subscriber[]>
  return { items: body.data ?? [], total: body.total ?? 0 }
}

/** Confirmed subscribers as CSV (the server neutralises spreadsheet formulas). */
export async function exportSubscribersCSV(token: string): Promise<Blob> {
  return (await send('/export', { method: 'GET' }, token)).blob()
}

/** Removes the subscriber and their data for good (GDPR erasure). */
export async function eraseSubscriber(token: string, id: string): Promise<void> {
  await send(`/${encodeURIComponent(id)}`, { method: 'DELETE' }, token)
}
