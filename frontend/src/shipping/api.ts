import type {
  APIResponse, Shipment, ShippingSettingsResult, ShippingConnectionSettings, ShippingCarrierOverride, CreateShipmentInput,
  MyParcelFulfilmentOrder, RecipientAddress,
} from './types'

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
    throw new Error(body.error?.message ?? `HTTP ${res.status}`)
  }
  return body.data as T
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
  input: { package_type?: number | null; signature_required?: boolean | null; only_recipient?: boolean | null; insurance_amount_minor_units?: number | null },
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

// getShipmentLabel returns the raw label bytes as a Blob (PDF or ZPL,
// whatever the shipment was created with) -- not JSON-wrapped, so this
// bypasses the request() helper above entirely.
export async function getShipmentLabel(token: string, shipmentId: string): Promise<Blob> {
  const res = await fetch(`${base()}/shipments/${shipmentId}/label`, {
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
