import type { APIResponse } from './types'

// Mirrors api/bolconfirmations/openapi.yaml's BolShipmentConfirmation: one
// attempt to tell Bol a Bol order shipped, and what Bol made of it.
export type BolConfirmationStatus = 'pending' | 'success' | 'failure' | 'timeout'

// create_shipment marks the order shipped on Bol; update_transport corrects
// the track & trace Bol has after the parcel was cancelled and replaced.
export type BolConfirmationAction = 'create_shipment' | 'update_transport'

export interface BolShipmentConfirmation {
  id: string
  shipment_id: string
  external_order_id: string
  action: BolConfirmationAction
  transport_id: string
  transporter_code: string
  track_and_trace: string
  process_status_id: string
  status: BolConfirmationStatus
  error_message: string
  // failure/timeout: Bol doesn't have the right details until fixed there by hand.
  needs_manual_action: boolean
  created_at: string
  resolved_at?: string | null
}

// listBolShipmentConfirmations returns the order's attempts, newest first.
export async function listBolShipmentConfirmations(token: string, salesOrderId: string): Promise<BolShipmentConfirmation[]> {
  const res = await fetch(`${import.meta.env.VITE_API_BASE_URL}/sales-orders/${salesOrderId}/bol-shipment-confirmations`, {
    method: 'GET',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
  })
  const body: APIResponse<BolShipmentConfirmation[]> = await res.json()
  if (!res.ok || body.error) {
    throw new Error(body.error?.message ?? `HTTP ${res.status}`)
  }
  return body.data ?? []
}

export function latestBolConfirmation(confirmations: BolShipmentConfirmation[]): BolShipmentConfirmation | null {
  return confirmations[0] ?? null
}

// isBolConfirmationPending: Bol hasn't answered the newest attempt yet --
// worth checking again shortly.
export function isBolConfirmationPending(confirmations: BolShipmentConfirmation[]): boolean {
  return latestBolConfirmation(confirmations)?.status === 'pending'
}
