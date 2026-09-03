import type { Connection, CatalogEntry, StartWooConnectResponse, SyncSchedule, ImportJob, DeleteAllImportedProductsResult, ProductChannelListing, SyncLogEntry, AIConnection, AIProvider, APIResponse, ChannelType, EnrollmentMode, SalesChannel, SalesChannelProductEnrollment, WooTaxClass, WooCategory, ExternalCategoryLink, FetchWooOrderResult, WebhookFallbackEvent, WebhookFallbackStatus, WebhookSecretResult, StorefrontKeyResult, WebhookForwardRule, WebhookForwardDelivery, WebhookEvent, ProviderWebhook, BolProductOverride, BolProductContent, BolCommission } from './types'

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

export function updateConnectionSettings(
  token: string, connectionId: string, defaultCurrency: string, defaultWarehouseId: string | null = null,
  bolEconomicOperatorId: string | null = null, bolDeliveryCode: string | null = null,
): Promise<Connection> {
  return request<Connection>(
    `${base()}/connections/${connectionId}/settings`,
    {
      method: 'PATCH',
      body: JSON.stringify({
        default_currency: defaultCurrency, default_warehouse_id: defaultWarehouseId,
        bol_economic_operator_id: bolEconomicOperatorId, bol_delivery_code: bolDeliveryCode,
      }),
    },
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

// Manual escape hatch for a product whose listing row was lost while the
// real product still exists on WooCommerce (e.g. after a connection
// disconnect/reconnect) -- looks the SKU up on Woo and links to whatever it
// finds, instead of requiring a full re-import.
export function linkExistingWooProduct(
  token: string,
  productId: string,
  connectionId: string,
  sku: string,
): Promise<ProductChannelListing> {
  return request<ProductChannelListing>(
    `${base()}/products/${productId}/link-existing-woo-product`,
    { method: 'POST', body: JSON.stringify({ connection_id: connectionId, sku }) },
    token,
  )
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

// (Re)generates this connection's storefront key -- the bearer token a
// connected storefront (e.g. the WooCommerce plugin) presents to call
// CommerceOS' authenticated public-storefront API. The response carries the
// PLAINTEXT value exactly once, same "shown once" contract as
// setWebhookSecret above.
export function setStorefrontKey(token: string, connectionId: string): Promise<StorefrontKeyResult> {
  return request<StorefrontKeyResult>(`${base()}/connections/${connectionId}/storefront-key`, { method: 'POST' }, token)
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

export interface ConnectBolPayload {
  clientId: string
  clientSecret: string
  economicOperatorId: string
}

// Bol uses client_credentials (client_id/client_secret pasted from Bol's
// Retailer API portal) -- unlike WooCommerce there's no OAuth redirect
// flow, so this is the only connect path.
export function connectBol(token: string, payload: ConnectBolPayload): Promise<Connection> {
  return request<Connection>(
    `${base()}/bol/connect`,
    {
      method: 'POST',
      body: JSON.stringify({
        client_id: payload.clientId,
        client_secret: payload.clientSecret,
        economic_operator_id: payload.economicOperatorId,
      }),
    },
    token,
  )
}

export function pushProductToBol(token: string, connectionId: string, productId: string): Promise<ProductChannelListing[]> {
  return request<ProductChannelListing[]>(
    `${base()}/connections/${connectionId}/push-product-bol`,
    { method: 'POST', body: JSON.stringify({ product_id: productId }) },
    token,
  )
}

// Returns null (not an error) when this product has no Bol-specific
// overrides saved yet -- the common/default state.
export function getBolProductOverride(token: string, connectionId: string, productId: string): Promise<BolProductOverride | null> {
  return request<BolProductOverride | null>(`${base()}/connections/${connectionId}/bol-overrides/${productId}`, { method: 'GET' }, token)
}

export interface UpsertBolProductOverridePayload {
  eanOverride?: string | null
  gpcCode?: string | null
  conditionCategory?: BolProductOverride['condition_category']
  fulfilmentMethod?: BolProductOverride['fulfilment_method']
  commissionOverrideBasisPoints?: number | null
  referenceOverride?: string | null
  customPriceMinorUnits?: number | null
  nameOverride?: string | null
}

export function upsertBolProductOverride(token: string, connectionId: string, productId: string, payload: UpsertBolProductOverridePayload): Promise<BolProductOverride> {
  return request<BolProductOverride>(
    `${base()}/connections/${connectionId}/bol-overrides/${productId}`,
    {
      method: 'PUT',
      body: JSON.stringify({
        ean_override: payload.eanOverride ?? undefined,
        gpc_code: payload.gpcCode ?? undefined,
        condition_category: payload.conditionCategory,
        fulfilment_method: payload.fulfilmentMethod,
        commission_override_basis_points: payload.commissionOverrideBasisPoints ?? undefined,
        reference_override: payload.referenceOverride ?? undefined,
        custom_price_minor_units: payload.customPriceMinorUnits ?? undefined,
        name_override: payload.nameOverride ?? undefined,
      }),
    },
    token,
  )
}

// Returns null (not an error) when this product has no Bol product-content
// fields saved yet -- the common/default state.
export function getBolProductContent(token: string, connectionId: string, productId: string): Promise<BolProductContent | null> {
  return request<BolProductContent | null>(`${base()}/connections/${connectionId}/bol-product-content/${productId}`, { method: 'GET' }, token)
}

export type UpsertBolProductContentPayload = Omit<BolProductContent, 'product_id' | 'connection_id'>

export function upsertBolProductContent(token: string, connectionId: string, productId: string, payload: UpsertBolProductContentPayload): Promise<BolProductContent> {
  return request<BolProductContent>(
    `${base()}/connections/${connectionId}/bol-product-content/${productId}`,
    { method: 'PUT', body: JSON.stringify(payload) },
    token,
  )
}

// Live per-EAN/price commission from Bol's Commissions API, or the
// merchant's own override when one is saved (see BolCommission.overridden).
export function getBolCommission(token: string, connectionId: string, productId: string, skuId: string): Promise<BolCommission> {
  return request<BolCommission>(`${base()}/connections/${connectionId}/bol-commission/${productId}/${skuId}`, { method: 'GET' }, token)
}

export interface BolEconomicOperator {
  id: string
  name: string
  status: 'PENDING' | 'VALID' | 'INVALID'
}

// Fetches every GPSR economic operator registered against this Bol
// account, straight from Bol -- powers a "fetch from Bol" button so the id
// doesn't have to be copied manually out of Bol's Partner Platform. Can
// reject with a 502 if the connection's app registration lacks the
// economic-operators.read/.manage scope -- callers should fall back to
// manual entry, not treat it as fatal.
export function listBolEconomicOperators(token: string, connectionId: string): Promise<BolEconomicOperator[]> {
  return request<BolEconomicOperator[]>(`${base()}/connections/${connectionId}/bol-economic-operators`, { method: 'GET' }, token)
}
