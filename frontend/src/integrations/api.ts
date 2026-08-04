import type { Connection, CatalogEntry, StartWooConnectResponse, SyncSchedule, ImportJob, DeleteAllImportedProductsResult, ProductChannelListing, APIResponse } from './types'

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

export function updateConnectionSettings(token: string, connectionId: string, defaultCurrency: string): Promise<Connection> {
  return request<Connection>(
    `${base()}/connections/${connectionId}/settings`,
    { method: 'PATCH', body: JSON.stringify({ default_currency: defaultCurrency }) },
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
