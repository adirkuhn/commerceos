import type { ProductChannelListing, BolProductOverride, BolProductContent, BolCommission } from '../types'
import { base, request } from './client'

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
