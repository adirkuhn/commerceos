import type { Connection, CatalogEntry, StartWooConnectResponse, SyncSchedule, ImportJob, DeleteAllImportedProductsResult, ProductChannelListing, SyncLogEntry, AIConnection, AIProvider, APIResponse, ChannelType, EnrollmentMode, SalesChannel, SalesChannelProductEnrollment, WooTaxClass, WooCategory, ExternalCategoryLink, FetchWooOrderResult, WebhookFallbackEvent, WebhookFallbackStatus, WebhookSecretResult } from './types'

function base(): string {
  return `${import.meta.env.VITE_API_BASE_URL}/integrations`
}

async function request<T>(
  path: string,
  options: RequestInit,
  token: string,
): Promise<T> {
  const res = await fetch(path, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
      ...options.headers,
    },
  })
  const body: APIResponse<T> = await res.json()
  if (!res.ok || body.error) {
    throw new Error(body.error?.message ?? `HTTP ${res.status}`)
  }
  return body.data as T
}

export function listCatalog(token: string): Promise<CatalogEntry[]> {
  return request<CatalogEntry[]>(`${base()}/catalog`, { method: 'GET' }, token)
}

export function listConnections(token: string): Promise<Connection[]> {
  return request<Connection[]>(`${base()}/connections`, { method: 'GET' }, token)
}

export function startWooConnect(token: string, storeUrl: string): Promise<StartWooConnectResponse> {
  return request<StartWooConnectResponse>(
    `${base()}/woocommerce/connect/start`,
    { method: 'POST', body: JSON.stringify({ store_url: storeUrl }) },
    token,
  )
}

export function disconnectConnection(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/connections/${id}`, { method: 'DELETE' }, token)
}

export function testConnection(token: string, id: string): Promise<Connection> {
  return request<Connection>(`${base()}/connections/${id}/test`, { method: 'POST' }, token)
}

export function listWooTaxes(token: string, connectionId: string): Promise<WooTaxClass[]> {
  return request<WooTaxClass[]>(`${base()}/connections/${connectionId}/woo-taxes`, { method: 'GET' }, token)
}

export function listWooCategories(token: string, connectionId: string): Promise<WooCategory[]> {
  return request<WooCategory[]>(`${base()}/connections/${connectionId}/woo-categories`, { method: 'GET' }, token)
}

export function listExternalCategoryLinks(token: string, connectionId: string): Promise<ExternalCategoryLink[]> {
  return request<ExternalCategoryLink[]>(`${base()}/connections/${connectionId}/category-links`, { method: 'GET' }, token)
}

export function upsertExternalCategoryLink(
  token: string,
  connectionId: string,
  payload: { external_id: string; category_id: string },
): Promise<ExternalCategoryLink> {
  return request<ExternalCategoryLink>(
    `${base()}/connections/${connectionId}/category-links`,
    { method: 'POST', body: JSON.stringify(payload) },
    token,
  )
}

export function deleteExternalCategoryLink(token: string, connectionId: string, linkId: string): Promise<null> {
  return request<null>(`${base()}/connections/${connectionId}/category-links/${linkId}`, { method: 'DELETE' }, token)
}

export interface ManualConnectWooCommercePayload {
  storeUrl: string
  consumerKey: string
  consumerSecret: string
}

export function manualConnectWooCommerce(token: string, payload: ManualConnectWooCommercePayload): Promise<Connection> {
  return request<Connection>(
    `${base()}/woocommerce/connect/manual`,
    {
      method: 'POST',
      body: JSON.stringify({
        store_url: payload.storeUrl,
        consumer_key: payload.consumerKey,
        consumer_secret: payload.consumerSecret,
      }),
    },
    token,
  )
}

export function updateConnectionSettings(token: string, connectionId: string, defaultCurrency: string, defaultWarehouseId: string | null = null): Promise<Connection> {
  return request<Connection>(
    `${base()}/connections/${connectionId}/settings`,
    { method: 'PATCH', body: JSON.stringify({ default_currency: defaultCurrency, default_warehouse_id: defaultWarehouseId }) },
    token,
  )
}

export function fetchWooOrder(token: string, connectionId: string, externalOrderId: string): Promise<FetchWooOrderResult> {
  return request<FetchWooOrderResult>(
    `${base()}/connections/${connectionId}/fetch-order`,
    { method: 'POST', body: JSON.stringify({ external_order_id: externalOrderId }) },
    token,
  )
}

export function importWooProducts(token: string, connectionId: string, defaultWarehouseId: string, defaultCurrency: string): Promise<ImportJob> {
  return request<ImportJob>(
    `${base()}/connections/${connectionId}/import`,
    { method: 'POST', body: JSON.stringify({ default_warehouse_id: defaultWarehouseId, default_currency: defaultCurrency }) },
    token,
  )
}

export function deleteAllImportedProducts(token: string, connectionId: string): Promise<DeleteAllImportedProductsResult> {
  return request<DeleteAllImportedProductsResult>(
    `${base()}/connections/${connectionId}/imported-products`,
    { method: 'DELETE' },
    token,
  )
}

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
    throw new Error(body.error?.message ?? `HTTP ${res.status}`)
  }
  return body.data
}

export function listSyncLogs(token: string, connectionId: string, productId: string): Promise<SyncLogEntry[]> {
  return request<SyncLogEntry[]>(`${base()}/connections/${connectionId}/listings/${productId}/sync-logs`, { method: 'GET' }, token)
}

// Returns null (not an error) when the tenant hasn't configured an AI
// provider yet -- that's the common/default state, not a failure.
export function getAIConnection(token: string): Promise<AIConnection | null> {
  return request<AIConnection | null>(`${base()}/ai`, { method: 'GET' }, token)
}

export interface ConnectAIProviderPayload {
  provider: AIProvider
  apiKey?: string
  baseUrl?: string
  model?: string
}

export function connectAIProvider(token: string, payload: ConnectAIProviderPayload): Promise<AIConnection> {
  return request<AIConnection>(
    `${base()}/ai`,
    {
      method: 'POST',
      body: JSON.stringify({
        provider: payload.provider,
        api_key: payload.apiKey || undefined,
        base_url: payload.baseUrl || undefined,
        model: payload.model || undefined,
      }),
    },
    token,
  )
}

export function disconnectAIProvider(token: string): Promise<null> {
  return request<null>(`${base()}/ai`, { method: 'DELETE' }, token)
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
    throw new Error(body.error?.message ?? `HTTP ${res.status}`)
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
    throw new Error(body.error?.message ?? `HTTP ${res.status}`)
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

export function listSalesChannels(token: string): Promise<SalesChannel[]> {
  return request<SalesChannel[]>(`${base()}/sales-channels`, { method: 'GET' }, token)
}

export interface SalesChannelPayload {
  name: string
  type: ChannelType
  connectionId?: string | null
  enrollmentMode: EnrollmentMode
  enrollmentCategoryId?: string | null
  enrollmentTag?: string | null
}

export function createSalesChannel(token: string, payload: SalesChannelPayload): Promise<SalesChannel> {
  return request<SalesChannel>(
    `${base()}/sales-channels`,
    {
      method: 'POST',
      body: JSON.stringify({
        name: payload.name,
        type: payload.type,
        connection_id: payload.connectionId || undefined,
        enrollment_mode: payload.enrollmentMode,
        enrollment_category_id: payload.enrollmentCategoryId || undefined,
        enrollment_tag: payload.enrollmentTag || undefined,
      }),
    },
    token,
  )
}

export interface UpdateSalesChannelPayload {
  name: string
  connectionId?: string | null
  enrollmentMode: EnrollmentMode
  enrollmentCategoryId?: string | null
  enrollmentTag?: string | null
  active: boolean
}

export function updateSalesChannel(token: string, id: string, payload: UpdateSalesChannelPayload): Promise<SalesChannel> {
  return request<SalesChannel>(
    `${base()}/sales-channels/${id}`,
    {
      method: 'PATCH',
      body: JSON.stringify({
        name: payload.name,
        connection_id: payload.connectionId || undefined,
        enrollment_mode: payload.enrollmentMode,
        enrollment_category_id: payload.enrollmentCategoryId || undefined,
        enrollment_tag: payload.enrollmentTag || undefined,
        active: payload.active,
      }),
    },
    token,
  )
}

export function deleteSalesChannel(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/sales-channels/${id}`, { method: 'DELETE' }, token)
}

export interface ListSalesChannelProductsParams {
  search?: string
  page?: number
  perPage?: number
}

export interface ListSalesChannelProductsResult {
  items: SalesChannelProductEnrollment[]
  total: number
}

// Enrolled is computed live server-side on every call (never stored per
// product) — a channel's enrollment rule changing is reflected here
// immediately, no backfill needed.
export async function listSalesChannelProducts(token: string, channelId: string, params: ListSalesChannelProductsParams = {}): Promise<ListSalesChannelProductsResult> {
  const qs = new URLSearchParams()
  if (params.search) qs.set('search', params.search)
  if (params.page) qs.set('page', String(params.page))
  if (params.perPage) qs.set('per_page', String(params.perPage))
  const query = qs.toString()

  const res = await fetch(`${base()}/sales-channels/${channelId}/products${query ? `?${query}` : ''}`, {
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
  })
  const body: APIResponse<SalesChannelProductEnrollment[]> & { total?: number } = await res.json()
  if (!res.ok || body.error) {
    throw new Error(body.error?.message ?? `HTTP ${res.status}`)
  }
  return { items: body.data ?? [], total: body.total ?? 0 }
}

// Force-include ("add") or force-exclude ("remove") one product for this
// channel, regardless of its default enrollment rule.
export function setSalesChannelProductOverride(token: string, channelId: string, productId: string, included: boolean): Promise<null> {
  return request<null>(
    `${base()}/sales-channels/${channelId}/products/${productId}/override`,
    { method: 'PUT', body: JSON.stringify({ included }) },
    token,
  )
}

// (Re)generates this connection's webhook-signature secret -- the response
// carries the PLAINTEXT value exactly once, it can't be retrieved again
// after this call (only the encrypted form persists server-side).
export function setWebhookSecret(token: string, connectionId: string): Promise<WebhookSecretResult> {
  return request<WebhookSecretResult>(`${base()}/connections/${connectionId}/webhook-secret`, { method: 'POST' }, token)
}

// Defaults to pending_republish server-side when status is omitted --
// what an operator triages after a NATS outage.
export function listWebhookFallbackEvents(token: string, status?: WebhookFallbackStatus): Promise<WebhookFallbackEvent[]> {
  const query = status ? `?status=${status}` : ''
  return request<WebhookFallbackEvent[]>(`${base()}/webhook-fallback-events${query}`, { method: 'GET' }, token)
}

export function retryWebhookFallbackEvent(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/webhook-fallback-events/${id}/retry`, { method: 'POST' }, token)
}

export function abandonWebhookFallbackEvent(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/webhook-fallback-events/${id}`, { method: 'DELETE' }, token)
}
