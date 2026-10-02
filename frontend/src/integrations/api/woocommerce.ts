import type { Connection, ImportJob, DeleteAllImportedProductsResult, ProductChannelListing, WooTaxClass, WooCategory, ExternalCategoryLink, FetchWooOrderResult, BackfillWooOrderPaymentsResult } from '../types'
import { base, request } from './client'

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

export function backfillWooOrderPayments(token: string, connectionId: string, cursor: string | null): Promise<BackfillWooOrderPaymentsResult> {
  return request<BackfillWooOrderPaymentsResult>(
    `${base()}/connections/${connectionId}/backfill-order-payments`,
    { method: 'POST', body: JSON.stringify({ cursor }) },
    token,
  )
}
