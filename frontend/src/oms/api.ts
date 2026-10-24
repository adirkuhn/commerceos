import type { SalesOrder, SalesOrderDetail, SalesOrderStatus, SalesOrderSource, CreateSalesOrderInput, UpdateSalesOrderInput, APIResponse, SalesOrderSortField, SortDirection, LinkLineItemSKUInput, RetryStockDecrementResult, ForceAcceptStockResult, SalesOrderActivity, PickList, OrderQueue } from './types'
import { CodedApiError } from '../api/codedApiError'

function base(): string {
  return `${import.meta.env.VITE_API_BASE_URL}/sales-orders`
}

async function request<T>(path: string, token: string, options: RequestInit = {}): Promise<T> {
  const res = await fetch(path, {
    ...options,
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}`, ...options.headers },
  })
  if (res.status === 204) return null as T
  const body: APIResponse<T> = await res.json()
  if (!res.ok || body.error) {
    throw CodedApiError.fromEnvelope(body.error, res.status)
  }
  return body.data as T
}

export interface ListSalesOrdersParams {
  page?: number
  perPage?: number
  sortBy?: SalesOrderSortField
  sortDir?: SortDirection
  // needsStockReview/needsSkuReview: pass true to narrow to flagged orders
  // -- the "Stock Issues"/"Unmatched SKUs" pages. Omit for the normal
  // unfiltered sales orders list.
  needsStockReview?: boolean
  needsSkuReview?: boolean
  // fulfilled: false powers the picking scanner app's "orders to pick"
  // search. Omit for the normal unfiltered sales orders list.
  fulfilled?: boolean
  // awaitingPicking: true is the picking queue -- not yet picked, still
  // pending/processing/on-hold, not a POS sale. The rule lives in the
  // backend (ListSalesOrdersFilter.AwaitingPicking).
  awaitingPicking?: boolean
  // packed: filter on the order's packing status. Omit for the normal
  // unfiltered sales orders list.
  packed?: boolean
  // status: narrows to the given statuses -- combined with fulfilled:false,
  // the picking screens use this to exclude completed/cancelled/refunded/
  // failed orders. Omit for the normal unfiltered sales orders list.
  status?: SalesOrderStatus[]
  // source: narrows to the given channels (Sales Orders page's own filter).
  // Omit (or empty) for every channel.
  source?: SalesOrderSource[]
  // customerId: narrows to one customer's orders -- the Customer detail
  // page's order history and lifetime-spend figures. Omit for the normal
  // unfiltered sales orders list.
  customerId?: string
  // shipDue: true narrows to orders due to ship today or already late
  // (ship_by_date on/before today, not shipped yet) -- the Today page's
  // "orders to ship today" button. The rule lives in the backend
  // (ListSalesOrdersFilter.ShipDueOnOrBefore).
  shipDue?: boolean
  // Orders created from this quote.
  quoteId?: string
}

export interface ListSalesOrdersResult {
  salesOrders: SalesOrder[]
  total: number
}

export async function listSalesOrders(token: string, params: ListSalesOrdersParams = {}): Promise<ListSalesOrdersResult> {
  const qs = new URLSearchParams()
  if (params.page) qs.set('page', String(params.page))
  if (params.perPage) qs.set('per_page', String(params.perPage))
  if (params.sortBy) qs.set('sort_by', params.sortBy)
  if (params.sortDir) qs.set('sort_dir', params.sortDir)
  if (params.needsStockReview !== undefined) qs.set('needs_stock_review', String(params.needsStockReview))
  if (params.needsSkuReview !== undefined) qs.set('needs_sku_review', String(params.needsSkuReview))
  if (params.fulfilled !== undefined) qs.set('fulfilled', String(params.fulfilled))
  if (params.packed !== undefined) qs.set('packed', String(params.packed))
  if (params.awaitingPicking) qs.set('awaiting_picking', 'true')
  for (const status of params.status ?? []) qs.append('status', status)
  for (const source of params.source ?? []) qs.append('source', source)
  if (params.customerId) qs.set('customer_id', params.customerId)
  if (params.quoteId) qs.set('quote_id', params.quoteId)
  if (params.shipDue) qs.set('ship_due', 'true')
  const query = qs.toString()

  const result = await request<{ sales_orders: SalesOrder[]; total: number }>(
    `${base()}${query ? `?${query}` : ''}`, token,
  )
  return { salesOrders: result.sales_orders, total: result.total }
}

// The packing screens' queue: every unpacked order with who is working it.
// Lives outside /sales-orders on the backend, hence the separate base.
export function listPackingQueue(token: string, params: { page?: number; perPage?: number } = {}): Promise<OrderQueue> {
  const qs = new URLSearchParams()
  if (params.page) qs.set('page', String(params.page))
  if (params.perPage) qs.set('per_page', String(params.perPage))
  const query = qs.toString()
  return request<OrderQueue>(`${import.meta.env.VITE_API_BASE_URL}/packing-queue${query ? `?${query}` : ''}`, token)
}

// Picking list mode: marks the whole order picked in one action.
export function markOrderPicked(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/${id}/mark-picked`, token, { method: 'POST' })
}

// The picking screens' queue: every order still to be picked, with who is
// working it and whether packing has it (in_packing).
export function listPickingQueue(token: string, params: { page?: number; perPage?: number } = {}): Promise<OrderQueue> {
  const qs = new URLSearchParams()
  if (params.page) qs.set('page', String(params.page))
  if (params.perPage) qs.set('per_page', String(params.perPage))
  const query = qs.toString()
  return request<OrderQueue>(`${import.meta.env.VITE_API_BASE_URL}/picking-queue${query ? `?${query}` : ''}`, token)
}

export function getPickList(token: string, salesOrderIds: string[]): Promise<PickList> {
  const qs = new URLSearchParams()
  for (const id of salesOrderIds) qs.append('sales_order_id', id)
  return request<PickList>(`${base()}/pick-list?${qs.toString()}`, token)
}

export function getSalesOrder(token: string, id: string): Promise<SalesOrderDetail> {
  return request<SalesOrderDetail>(`${base()}/${id}`, token)
}

export function listSalesOrderActivities(token: string, id: string): Promise<{ activities: SalesOrderActivity[] }> {
  return request<{ activities: SalesOrderActivity[] }>(`${base()}/${id}/activities`, token)
}

export function createSalesOrder(token: string, input: CreateSalesOrderInput): Promise<SalesOrderDetail> {
  return request<SalesOrderDetail>(base(), token, { method: 'POST', body: JSON.stringify(input) })
}

export function updateSalesOrder(token: string, id: string, input: UpdateSalesOrderInput): Promise<SalesOrder> {
  return request<SalesOrder>(`${base()}/${id}`, token, { method: 'PATCH', body: JSON.stringify(input) })
}

export function deleteSalesOrder(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/${id}`, token, { method: 'DELETE' })
}

export interface BulkUpdateSalesOrdersInput {
  orderIds: string[]
  status?: SalesOrderStatus
  fulfilled?: boolean
}

export interface BulkUpdateSalesOrdersResult {
  updated: string[]
  failed: { orderId: string; error: string }[]
}

// bulkUpdateSalesOrders: every order_id is attempted independently -- one
// failing doesn't stop the rest, see the result's updated/failed split.
export async function bulkUpdateSalesOrders(token: string, input: BulkUpdateSalesOrdersInput): Promise<BulkUpdateSalesOrdersResult> {
  const result = await request<{ updated?: string[]; failed?: { order_id?: string; error?: string }[] }>(
    `${base()}/bulk-status`, token,
    { method: 'POST', body: JSON.stringify({ order_ids: input.orderIds, status: input.status, fulfilled: input.fulfilled }) },
  )
  return {
    updated: result.updated ?? [],
    failed: (result.failed ?? []).map(f => ({ orderId: f.order_id ?? '', error: f.error ?? '' })),
  }
}

// retryStockDecrement re-attempts the stock decrement for every line item
// on this order whose decrement is pending (sku_id linked, but an earlier
// attempt hit insufficient stock) -- the "Stock Issues" page's resolve
// action, run after staff fix the underlying stock. A line item still
// insufficient after retrying is not an error -- see still_pending.
export function retryStockDecrement(token: string, id: string): Promise<RetryStockDecrementResult> {
  return request<RetryStockDecrementResult>(`${base()}/${id}/retry-stock`, token, { method: 'POST' })
}

// forceAcceptStock bypasses the insufficiency check for every SKU-linked
// pending line item on this order -- the "Stock Issues" page's override
// action, for when staff know the item physically exists and the tracked
// ledger is just wrong. Unlike retryStockDecrement, always resolves every
// eligible line item.
export function forceAcceptStock(token: string, id: string): Promise<ForceAcceptStockResult> {
  return request<ForceAcceptStockResult>(`${base()}/${id}/force-accept-stock`, token, { method: 'POST' })
}

// linkLineItemSku links a line item that was imported without a matching
// catalog SKU to a real one -- the "Unmatched SKUs" page's resolve action.
// Does not itself retry the stock decrement; call retryStockDecrement
// separately afterward.
export function linkLineItemSku(token: string, salesOrderId: string, lineItemId: string, input: LinkLineItemSKUInput): Promise<SalesOrder> {
  return request<SalesOrder>(`${base()}/${salesOrderId}/line-items/${lineItemId}/link-sku`, token, {
    method: 'POST', body: JSON.stringify(input),
  })
}
