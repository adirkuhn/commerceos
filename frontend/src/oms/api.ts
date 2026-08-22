import type { SalesOrder, SalesOrderDetail, SalesOrderStatus, CreateSalesOrderInput, UpdateSalesOrderInput, APIResponse, SalesOrderSortField, SortDirection, LinkLineItemSKUInput, RetryStockDecrementResult, ForceAcceptStockResult } from './types'

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
    throw new Error(body.error?.message ?? `HTTP ${res.status}`)
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
  // status: narrows to the given statuses -- combined with fulfilled:false,
  // the picking screens use this to exclude completed/cancelled/refunded/
  // failed orders. Omit for the normal unfiltered sales orders list.
  status?: SalesOrderStatus[]
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
  for (const status of params.status ?? []) qs.append('status', status)
  const query = qs.toString()

  const result = await request<{ sales_orders: SalesOrder[]; total: number }>(
    `${base()}${query ? `?${query}` : ''}`, token,
  )
  return { salesOrders: result.sales_orders, total: result.total }
}

export function getSalesOrder(token: string, id: string): Promise<SalesOrderDetail> {
  return request<SalesOrderDetail>(`${base()}/${id}`, token)
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
