import type { APIResponse, WebhookFallbackEvent, WebhookFallbackStatus, WebhookForwardRule, WebhookForwardDelivery, WebhookEvent, ProviderWebhook } from '../types'
import { base, request } from './client'

export interface ListWebhookFallbackEventsResult {
  events: WebhookFallbackEvent[]
  total: number
}

// Defaults to pending_republish server-side when status is omitted --
// what an operator triages after a NATS outage.
export async function listWebhookFallbackEvents(
  token: string, status?: WebhookFallbackStatus, page?: number, perPage?: number,
): Promise<ListWebhookFallbackEventsResult> {
  const qs = new URLSearchParams()
  if (status) qs.set('status', status)
  if (page) qs.set('page', String(page))
  if (perPage) qs.set('per_page', String(perPage))
  const query = qs.toString()

  const res = await fetch(`${base()}/webhook-fallback-events${query ? `?${query}` : ''}`, {
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
  })
  const body: APIResponse<WebhookFallbackEvent[]> & { total?: number } = await res.json()
  if (!res.ok || body.error) {
    throw new Error(body.error?.message ?? `HTTP ${res.status}`)
  }
  return { events: body.data ?? [], total: body.total ?? 0 }
}

export function retryWebhookFallbackEvent(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/webhook-fallback-events/${id}/retry`, { method: 'POST' }, token)
}

export function abandonWebhookFallbackEvent(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/webhook-fallback-events/${id}`, { method: 'DELETE' }, token)
}

// Bulk form of abandonWebhookFallbackEvent -- marks every row at `status`
// (defaults to pending_republish server-side) abandoned in one call.
export function abandonAllWebhookFallbackEvents(token: string, status?: WebhookFallbackStatus): Promise<{ abandoned: number }> {
  const query = status ? `?status=${status}` : ''
  return request<{ abandoned: number }>(`${base()}/webhook-fallback-events/abandon-all${query}`, { method: 'POST' }, token)
}

export function listWebhookForwardRules(token: string): Promise<WebhookForwardRule[]> {
  return request<WebhookForwardRule[]>(`${base()}/webhook-forward-rules`, { method: 'GET' }, token)
}

export function createWebhookForwardRule(
  token: string, input: { provider: string; topic?: string; target_url: string },
): Promise<WebhookForwardRule> {
  return request<WebhookForwardRule>(`${base()}/webhook-forward-rules`, { method: 'POST', body: JSON.stringify(input) }, token)
}

export function deleteWebhookForwardRule(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/webhook-forward-rules/${id}`, { method: 'DELETE' }, token)
}

export function listWebhookForwardDeliveries(token: string, limit?: number): Promise<WebhookForwardDelivery[]> {
  const query = limit ? `?limit=${limit}` : ''
  return request<WebhookForwardDelivery[]>(`${base()}/webhook-forward-deliveries${query}`, { method: 'GET' }, token)
}

export function listWebhookEvents(token: string, opts?: { provider?: string; limit?: number }): Promise<WebhookEvent[]> {
  const params = new URLSearchParams()
  if (opts?.provider) params.set('provider', opts.provider)
  if (opts?.limit) params.set('limit', String(opts.limit))
  const query = params.toString() ? `?${params.toString()}` : ''
  return request<WebhookEvent[]>(`${base()}/webhook-events${query}`, { method: 'GET' }, token)
}

// retryWebhookEventDispatch re-runs dispatch for one message right now
// (myparcel shipment_status_change only) and returns its updated row --
// the Messages tab's per-row/bulk Retry action.
export function retryWebhookEventDispatch(token: string, eventId: string): Promise<WebhookEvent> {
  return request<WebhookEvent>(`${base()}/webhook-events/${eventId}/retry`, { method: 'POST' }, token)
}

export function listProviderWebhooks(token: string, connectionId: string): Promise<ProviderWebhook[]> {
  return request<ProviderWebhook[]>(`${base()}/connections/${connectionId}/provider-webhooks`, { method: 'GET' }, token)
}

export function listAllProviderWebhooks(token: string): Promise<ProviderWebhook[]> {
  return request<ProviderWebhook[]>(`${base()}/provider-webhooks`, { method: 'GET' }, token)
}

export interface AvailableWebhookTopics {
  // Every kind of event that could ever happen for a provider -- the
  // baseline catalog unioned with everything this tenant has actually
  // received. A reference list; picking one doesn't guarantee it's
  // actually being delivered.
  all_events: Record<string, string[]>
  // Topics with a currently active subscription registered at the
  // provider right now -- what's actually about to arrive.
  active_webhooks: Record<string, string[]>
}

// Drives the fan-out rule form's Topic dropdown -- see
// AvailableWebhookTopics' field docs for how the two groups differ.
export function listAvailableWebhookTopics(token: string): Promise<AvailableWebhookTopics> {
  return request<AvailableWebhookTopics>(`${base()}/webhook-topics`, { method: 'GET' }, token)
}

export function createProviderWebhook(token: string, connectionId: string, topic: string): Promise<ProviderWebhook> {
  return request<ProviderWebhook>(`${base()}/connections/${connectionId}/provider-webhooks`, { method: 'POST', body: JSON.stringify({ topic }) }, token)
}

export function deleteProviderWebhook(token: string, connectionId: string, webhookId: string): Promise<null> {
  return request<null>(`${base()}/connections/${connectionId}/provider-webhooks/${webhookId}`, { method: 'DELETE' }, token)
}
