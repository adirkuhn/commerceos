import type { SalesOrder, SalesOrderDetail, CreateSalesOrderInput, UpdateSalesOrderInput, APIResponse } from './types'

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
}

export interface ListSalesOrdersResult {
  salesOrders: SalesOrder[]
  total: number
}

export async function listSalesOrders(token: string, params: ListSalesOrdersParams = {}): Promise<ListSalesOrdersResult> {
  const qs = new URLSearchParams()
  if (params.page) qs.set('page', String(params.page))
  if (params.perPage) qs.set('per_page', String(params.perPage))
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
