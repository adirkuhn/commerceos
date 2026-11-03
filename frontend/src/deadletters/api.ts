import type { APIResponse, DeadLetter, DeadLetterFilter, DeadLetterPage, DeadLetterStream } from './types'
import { CodedApiError } from '../api/codedApiError'
import { appEnv } from '../config/appEnv'

function base(): string {
  return `${appEnv().VITE_API_BASE_URL}/admin/dead-letters`
}

async function request<T>(url: string, token: string, options: RequestInit = {}): Promise<T> {
  const res = await fetch(url, {
    ...options,
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}`, ...options.headers },
  })
  const body: APIResponse<T> = await res.json()
  if (!res.ok || body.error) {
    throw CodedApiError.fromEnvelope(body.error, res.status)
  }
  return body.data
}

export function listDeadLetters(token: string, filter: DeadLetterFilter): Promise<DeadLetterPage> {
  const q = new URLSearchParams({ stream: filter.stream })
  if (filter.subject) q.set('subject', filter.subject)
  if (filter.consumer) q.set('consumer', filter.consumer)
  if (filter.tenantId) q.set('tenant_id', filter.tenantId)
  if (filter.cursor) q.set('cursor', String(filter.cursor))
  if (filter.limit) q.set('limit', String(filter.limit))
  return request<DeadLetterPage>(`${base()}?${q}`, token)
}

function itemURL(stream: DeadLetterStream, seq: number): string {
  return `${base()}/${stream}/${seq}`
}

export function getDeadLetter(token: string, stream: DeadLetterStream, seq: number): Promise<DeadLetter> {
  return request<DeadLetter>(itemURL(stream, seq), token)
}

export function retryDeadLetter(token: string, stream: DeadLetterStream, seq: number): Promise<null> {
  return request<null>(`${itemURL(stream, seq)}/retry`, token, { method: 'POST' })
}

export function discardDeadLetter(token: string, stream: DeadLetterStream, seq: number): Promise<null> {
  return request<null>(itemURL(stream, seq), token, { method: 'DELETE' })
}
