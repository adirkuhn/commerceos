import type { APIResponse, ChannelType, EnrollmentMode, SalesChannel, SalesChannelProductEnrollment, QuantityBreak, StorefrontLocale, StorefrontMode } from '../types'
import { base, request } from './client'
import { CodedApiError } from '../../api/codedApiError'

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
  priceAdjustmentBasisPoints?: number | null
  quantityBreaks?: QuantityBreak[]
  // currencyCode is settable only at creation -- immutable after (see
  // backend domain/channels.SalesChannel.CurrencyCode's own doc comment).
  // "" (or omitted) means "use the tenant's default currency".
  currencyCode?: string
  // defaultWarehouseId is required before an 'internal'/'whatsapp' channel
  // can process an order (see SalesChannel.default_warehouse_id's doc
  // comment) -- unlike currencyCode, changeable after creation too.
  defaultWarehouseId?: string | null
  // Omitted means headless. Only 'internal' channels may use 'theme'.
  storefrontMode?: StorefrontMode
  // Headless 'internal' channels only; omitted or "" means none.
  signInUrl?: string
  // 'internal' channels only; omitted means English.
  storefrontLocale?: StorefrontLocale
  // Other languages shoppers can switch to; the default is always offered.
  storefrontLocales?: StorefrontLocale[]
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
        price_adjustment_basis_points: payload.priceAdjustmentBasisPoints ?? undefined,
        quantity_breaks: payload.quantityBreaks ?? [],
        currency_code: payload.currencyCode || undefined,
        default_warehouse_id: payload.defaultWarehouseId || undefined,
        storefront_mode: payload.storefrontMode,
        sign_in_url: payload.signInUrl || undefined,
        storefront_locale: payload.storefrontLocale,
        storefront_locales: payload.storefrontLocales,
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
  priceAdjustmentBasisPoints?: number | null
  quantityBreaks?: QuantityBreak[]
  active: boolean
  defaultWarehouseId?: string | null
  // Omitted keeps the channel's current mode.
  storefrontMode?: StorefrontMode
  // Omitted keeps the current page; "" clears it.
  signInUrl?: string
  // Omitted keeps the channel's current language.
  storefrontLocale?: StorefrontLocale
  // Omitted keeps the offered languages.
  storefrontLocales?: StorefrontLocale[]
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
        price_adjustment_basis_points: payload.priceAdjustmentBasisPoints ?? undefined,
        quantity_breaks: payload.quantityBreaks ?? [],
        enrollment_tag: payload.enrollmentTag || undefined,
        active: payload.active,
        default_warehouse_id: payload.defaultWarehouseId || undefined,
        storefront_mode: payload.storefrontMode,
        sign_in_url: payload.signInUrl,
        storefront_locale: payload.storefrontLocale,
        storefront_locales: payload.storefrontLocales,
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
    throw CodedApiError.fromEnvelope(body.error, res.status)
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

// paymentConnectionId null disables the channel's payment method.
export function setSalesChannelPaymentMethod(token: string, channelId: string, paymentConnectionId: string | null): Promise<SalesChannel> {
  return request<SalesChannel>(
    `${base()}/sales-channels/${channelId}/payment-method`,
    { method: 'PATCH', body: JSON.stringify({ payment_connection_id: paymentConnectionId }) },
    token,
  )
}
