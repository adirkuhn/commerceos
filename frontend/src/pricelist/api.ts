import { CodedApiError } from '../api/codedApiError'
import type { PriceListPage, PriceListQuery, PriceListRow } from './types'

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

  const res = await fetch(`${import.meta.env.VITE_API_BASE_URL}/pms/price-list${params ? `?${params}` : ''}`, {
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
  })
  const body: PriceListResponse = await res.json()
  if (!res.ok || body.error) {
    throw CodedApiError.fromEnvelope(body.error, res.status)
  }
  return { rows: body.data ?? [], total: body.meta?.total ?? 0, pricesIncludeTax: body.meta?.prices_include_tax ?? true }
}
