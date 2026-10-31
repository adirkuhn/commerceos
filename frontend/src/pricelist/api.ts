import { CodedApiError } from '../api/codedApiError'
import type { PriceListPage, PriceListQuery, PriceListRow } from './types'
import { appEnv } from '../config/appEnv'

interface PriceListResponse {
  data: PriceListRow[] | null
  meta?: { total: number; prices_include_tax: boolean }
  error: { code: string; message: string } | null
}

export async function listPriceList(token: string, query: PriceListQuery): Promise<PriceListPage> {
  const qs = new URLSearchParams()
  if (query.page) qs.set('page', String(query.page))
  if (query.perPage) qs.set('per_page', String(query.perPage))
  if (query.search) qs.set('search', query.search)
  if (query.productIds?.length) qs.set('product_ids', query.productIds.join(','))
  const params = qs.toString()

  const res = await fetch(`${appEnv().VITE_API_BASE_URL}/pms/price-list${params ? `?${params}` : ''}`, {
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
  })
  const body: PriceListResponse = await res.json()
  if (!res.ok || body.error) {
    throw CodedApiError.fromEnvelope(body.error, res.status)
  }
  return { rows: body.data ?? [], total: body.meta?.total ?? 0, pricesIncludeTax: body.meta?.prices_include_tax ?? true }
}

/** Sets the SKU's cost without pinning it: the next goods receipt still blends into it. */
export async function setAutomaticCostPrice(token: string, skuId: string, costMinorUnits: number): Promise<void> {
  const res = await fetch(`${appEnv().VITE_API_BASE_URL}/pms/skus/${skuId}/cost-price/automatic`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
    body: JSON.stringify({ cost_price_minor_units: costMinorUnits }),
  })
  const body: { error: { code: string; message: string } | null } = await res.json()
  if (!res.ok || body.error) throw CodedApiError.fromEnvelope(body.error, res.status)
}
