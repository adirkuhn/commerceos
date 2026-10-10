import type {
  APIResponse, Shipment, ShippingSettingsResult, ShippingConnectionSettings, ShippingCarrierOverride, CreateShipmentInput,
  MyParcelFulfilmentOrder, RecipientAddress, ShipmentTrackingEvent, DeliveryPriceOption,
} from './types'
import { CodedApiError } from '../api/codedApiError'

function base(): string {
  return `${import.meta.env.VITE_API_BASE_URL}/shipping`
}

async function request<T>(path: string, options: RequestInit, token: string): Promise<T> {
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
    throw CodedApiError.fromEnvelope(body.error, res.status)
  }
  return body.data as T
}

// lookupAddress returns null (not a thrown error) on a 404 -- no address
// matches that postal code + house number, a normal outcome
// (typo/new-build), not a fault. See backend's own AddressLookupClient
// doc comment for why this can be trusted: MyParcel has no equivalent
// endpoint, this hits PDOK's public Dutch address registry instead.
export async function lookupAddress(token: string, postalCode: string, houseNumber: string): Promise<{ street: string; city: string } | null> {
  const params = new URLSearchParams({ postal_code: postalCode, number: houseNumber })
  const res = await fetch(`${base()}/address-lookup?${params.toString()}`, {
    method: 'GET',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
  })
  if (res.status === 404) {
    return null
  }
  const body: APIResponse<{ street: string; city: string }> = await res.json()
  if (!res.ok || body.error) {
    throw CodedApiError.fromEnvelope(body.error, res.status)
  }
  return body.data
}

// getDeliveryPriceEstimate is AddressConfirmModal's pre-confirm cost
// preview -- straight from MyParcel's own GET /delivery_options, only ever
// populated for NL/BE (see backend's ErrDeliveryEstimateUnsupportedCountry,
// surfaced here as a 422 the caller should treat as "no estimate available"
// rather than an error to show).
export function getDeliveryPriceEstimate(
  token: string, params: { connection_id: string; cc: string; postal_code: string; number: string; carrier: number },
): Promise<DeliveryPriceOption[]> {
  const query = new URLSearchParams({
    connection_id: params.connection_id, cc: params.cc, postal_code: params.postal_code,
    number: params.number, carrier: String(params.carrier),
  })
  return request(`${base()}/delivery-price-estimate?${query.toString()}`, { method: 'GET' }, token)
}

export function connectMyParcel(token: string, apiKey: string): Promise<{ connection_id: string; provider: string; status: string }> {
  return request(`${base()}/myparcel/connect`, { method: 'POST', body: JSON.stringify({ api_key: apiKey }) }, token)
}

export function registerMyParcelWebhook(token: string, connectionId: string): Promise<{ ok: boolean }> {
  return request(`${base()}/connections/${connectionId}/register-webhook`, { method: 'POST' }, token)
}

export function getShippingSettings(token: string, connectionId: string): Promise<ShippingSettingsResult> {
  return request(`${base()}/connections/${connectionId}/settings`, { method: 'GET' }, token)
}

export function updateShippingSettings(
  token: string, connectionId: string, input: Omit<ShippingConnectionSettings, 'connection_id'>,
): Promise<ShippingConnectionSettings> {
  return request(`${base()}/connections/${connectionId}/settings`, { method: 'PUT', body: JSON.stringify(input) }, token)
}

export function upsertCarrierOverride(
  token: string, connectionId: string, carrier: number,
  input: { package_type?: number | null; signature_required?: boolean | null; only_recipient?: boolean | null; insure_by_default?: boolean | null; tracking_url_template?: string | null },
): Promise<ShippingCarrierOverride> {
  return request(`${base()}/connections/${connectionId}/carrier-overrides/${carrier}`, { method: 'PUT', body: JSON.stringify(input) }, token)
}

export function exportMyParcelOrder(
  token: string, connectionId: string, salesOrderId: string, input: { carrier: number; recipient: RecipientAddress },
): Promise<MyParcelFulfilmentOrder> {
  return request(`${base()}/connections/${connectionId}/sales-orders/${salesOrderId}/export-order`, { method: 'POST', body: JSON.stringify(input) }, token)
}

export function createShipment(token: string, input: CreateShipmentInput): Promise<Shipment> {
  return request(`${base()}/shipments`, { method: 'POST', body: JSON.stringify(input) }, token)
}

export function listShipmentsForOrder(token: string, salesOrderId: string): Promise<Shipment[]> {
  return request(`${base()}/sales-orders/${salesOrderId}/shipments`, { method: 'GET' }, token)
}

export interface CancelShipmentResult {
  deleted_on_myparcel: boolean
  // MyParcel's own wording when it refused the delete.
  carrier_error?: string
  // English; show cancelOutcomeText instead.
  message: string
}

// cancelShipment attempts MyParcel's real delete, then always marks the
// shipment cancelled locally regardless of that outcome -- see backend's
// CancelShipmentUseCase doc comment. deleted_on_myparcel: false is an
// expected, non-error outcome (MyParcel only allows deletion pre-label),
// not a failure -- the caller can proceed to create a new shipment either way.
export function cancelShipment(token: string, shipmentId: string): Promise<CancelShipmentResult> {
  return request(`${base()}/shipments/${shipmentId}/cancel`, { method: 'POST' }, token)
}

export function listShipmentTrackingEvents(token: string, shipmentId: string): Promise<ShipmentTrackingEvent[]> {
  return request(`${base()}/shipments/${shipmentId}/tracking-events`, { method: 'GET' }, token)
}

// refreshShipmentTracking is the timeline's "fetch more info from
// provider" button -- fetches the shipment's current status right now and
// saves it as a new tracking event, returned here so the caller can
// prepend it without a full reload.
export function refreshShipmentTracking(token: string, shipmentId: string): Promise<ShipmentTrackingEvent> {
  return request(`${base()}/shipments/${shipmentId}/tracking-events/refresh`, { method: 'POST' }, token)
}

// getShipmentLabel returns the raw label bytes as a Blob (PDF or ZPL,
// whatever the shipment was created with) -- not JSON-wrapped, so this
// bypasses the request() helper above entirely. size ('A4'/'A6') is a
// retrieval-time choice, re-askable on every print/reprint of the same
// label -- omit to use the connection's own default_label_size.
export async function getShipmentLabel(token: string, shipmentId: string, size?: 'A4' | 'A6'): Promise<Blob> {
  const query = size ? `?size=${size}` : ''
  const res = await fetch(`${base()}/shipments/${shipmentId}/label${query}`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) {
    let message = `HTTP ${res.status}`
    try {
      const body: APIResponse<unknown> = await res.json()
      message = body.error?.message ?? message
    } catch {
      // response wasn't JSON (the success path never is) -- keep the HTTP status message
    }
    throw new Error(message)
  }
  return res.blob()
}
