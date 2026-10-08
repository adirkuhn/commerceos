import type { APIResponse } from './types'

// Mirrors api/orderdifferences/openapi.yaml: where an imported order differs
// from the channel's own order. Only reported -- the order is never changed.
export type OrderDifferenceKind = 'missing' | 'quantity' | 'total'

export interface OrderDifference {
  id: string
  sales_order_id: string
  order_number: string
  customer_name: string
  order_occurred_at: string
  kind: OrderDifferenceKind
  sku_code: string
  name: string
  channel_quantity: number
  local_quantity: number
  channel_total_minor_units: number
  local_total_minor_units: number
  currency_code: string
  detected_at: string
}

export interface OrderDifferencePage {
  differences: OrderDifference[]
  total: number
}

export interface AuditWooOrderDifferencesResult {
  checked: number
  errors: string[]
  next_cursor: string
}

async function request<T>(path: string, token: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(`${import.meta.env.VITE_API_BASE_URL}${path}`, {
    ...init,
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
  })
  const body: APIResponse<T> = await res.json()
  if (!res.ok || body.error) {
    throw new Error(body.error?.message ?? `HTTP ${res.status}`)
  }
  return body.data as T
}

export function listOrderDifferences(token: string, page: number, perPage: number): Promise<OrderDifferencePage> {
  return request(`/order-differences?page=${page}&per_page=${perPage}`, token)
}

export function dismissOrderDifference(token: string, id: string): Promise<null> {
  return request(`/order-differences/${id}/dismiss`, token, { method: 'POST' })
}

export function auditWooOrderDifferences(token: string, connectionId: string, cursor: string): Promise<AuditWooOrderDifferencesResult> {
  return request(`/integrations/connections/${connectionId}/audit-woo-order-differences`, token, {
    method: 'POST', body: JSON.stringify(cursor ? { cursor } : {}),
  })
}

// describeDifference picks the sentence (i18n key under
// stockIssues:differences.kind) and its values for one difference.
export function describeDifference(d: OrderDifference): { key: OrderDifferenceKind; values: Record<string, string | number> } {
  if (d.kind === 'total') {
    return { key: 'total', values: { channelTotal: d.channel_total_minor_units, localTotal: d.local_total_minor_units } }
  }
  return { key: d.kind, values: { name: d.name, sku: d.sku_code, channel: d.channel_quantity, local: d.local_quantity } }
}
