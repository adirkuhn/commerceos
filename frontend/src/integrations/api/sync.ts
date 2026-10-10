import type { SyncSchedule, ProductChannelListing, SyncLogEntry, APIResponse } from '../types'
import { base, request } from './client'
import { CodedApiError } from '../../api/codedApiError'

export function listSyncSchedules(token: string, connectionId: string): Promise<SyncSchedule[]> {
  return request<SyncSchedule[]>(`${base()}/connections/${connectionId}/sync-schedules`, { method: 'GET' }, token)
}

export function upsertSyncSchedules(token: string, connectionId: string, schedules: SyncSchedule[]): Promise<SyncSchedule[]> {
  return request<SyncSchedule[]>(
    `${base()}/connections/${connectionId}/sync-schedules`,
    { method: 'PUT', body: JSON.stringify({ schedules }) },
    token,
  )
}

// Returns null (not an error) when the product has never been linked to
// this connection — that's an expected, common state (most products aren't
// imported from / pushed to every connected channel), not a failure.
export async function getProductChannelListing(token: string, connectionId: string, productId: string): Promise<ProductChannelListing | null> {
  const res = await fetch(`${base()}/connections/${connectionId}/listings/${productId}`, {
    method: 'GET',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
  })
  if (res.status === 404) {
    return null
  }
  const body: APIResponse<ProductChannelListing> = await res.json()
  if (!res.ok || body.error) {
    throw new CodedApiError(body.error?.code ?? 'HTTP_ERROR', body.error?.message ?? `HTTP ${res.status}`)
  }
  return body.data
}

// Every listing row for this product on this connection -- one entry per
// variant for a multi-variant product (plus the parent row too, in
// variations mode), unlike getProductChannelListing's single representative
// row. An empty array means the product has never been linked.
export function listProductChannelListings(token: string, connectionId: string, productId: string): Promise<ProductChannelListing[]> {
  return request<ProductChannelListing[]>(`${base()}/connections/${connectionId}/listings/${productId}/all`, { method: 'GET' }, token)
}

export function listSyncLogs(token: string, connectionId: string, productId: string): Promise<SyncLogEntry[]> {
  return request<SyncLogEntry[]>(`${base()}/connections/${connectionId}/listings/${productId}/sync-logs`, { method: 'GET' }, token)
}

export interface ListAllSyncLogsParams {
  page?: number
  perPage?: number
}

export interface ListAllSyncLogsResult {
  entries: SyncLogEntry[]
  total: number
}

// Tenant-wide sync history — every push/import/stock attempt across every
// product and connection, newest first.
export async function listAllSyncLogs(token: string, params: ListAllSyncLogsParams = {}): Promise<ListAllSyncLogsResult> {
  const qs = new URLSearchParams()
  if (params.page) qs.set('page', String(params.page))
  if (params.perPage) qs.set('per_page', String(params.perPage))
  const query = qs.toString()

  const res = await fetch(`${base()}/sync-logs${query ? `?${query}` : ''}`, {
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
  })
  const body: APIResponse<SyncLogEntry[]> & { total?: number } = await res.json()
  if (!res.ok || body.error) {
    throw new CodedApiError(body.error?.code ?? 'HTTP_ERROR', body.error?.message ?? `HTTP ${res.status}`)
  }
  return { entries: body.data ?? [], total: body.total ?? 0 }
}

// Every listing whose most recent sync attempt failed and hasn't been
// dismissed, newest first -- the Sync History page's "Needs attention"
// filter. A successful retry, or a dismiss, drops the listing out of this
// list on its own.
export async function listSyncLogsNeedingAttention(token: string, params: ListAllSyncLogsParams = {}): Promise<ListAllSyncLogsResult> {
  const qs = new URLSearchParams()
  if (params.page) qs.set('page', String(params.page))
  if (params.perPage) qs.set('per_page', String(params.perPage))
  const query = qs.toString()

  const res = await fetch(`${base()}/sync-logs/needing-attention${query ? `?${query}` : ''}`, {
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
  })
  const body: APIResponse<SyncLogEntry[]> & { total?: number } = await res.json()
  if (!res.ok || body.error) {
    throw new CodedApiError(body.error?.code ?? 'HTTP_ERROR', body.error?.message ?? `HTTP ${res.status}`)
  }
  return { entries: body.data ?? [], total: body.total ?? 0 }
}

// Re-push the failed entry's product to the same connection -- a full
// resync, not a narrow replay of just the field that failed.
export function retryFailedSync(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/sync-logs/${id}/retry`, { method: 'POST' }, token)
}

// Silence a failure that isn't worth retrying, without pretending it
// succeeded.
export function dismissFailedSync(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/sync-logs/${id}/dismiss`, { method: 'POST' }, token)
}

// Force-push this product's current fields and stock to every channel it's
// already linked to, regardless of whether anything changed since the last
// sync.
export function resyncProductToChannels(token: string, productId: string): Promise<null> {
  return request<null>(`${base()}/products/${productId}/resync`, { method: 'POST' }, token)
}
