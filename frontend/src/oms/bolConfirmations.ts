import type { TFunction } from 'i18next'
import type { APIResponse } from './types'
import { CodedApiError } from '../api/codedApiError'
import { appEnv } from '../config/appEnv'

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
  // Why it failed, when CommerceOS knows (see bolFailureText); '' otherwise.
  failure_reason?: '' | 'no_transporter_code' | 'no_bol_shipment' | 'rejected'
  // English or Bol's own words.
  error_message: string
  // failure/timeout: Bol doesn't have the right details until fixed there by hand.
  needs_manual_action: boolean
  created_at: string
  resolved_at?: string | null
}

// listBolShipmentConfirmations returns the order's attempts, newest first.
export async function listBolShipmentConfirmations(token: string, salesOrderId: string): Promise<BolShipmentConfirmation[]> {
  const res = await fetch(`${appEnv().VITE_API_BASE_URL}/sales-orders/${salesOrderId}/bol-shipment-confirmations`, {
    method: 'GET',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
  })
  const body: APIResponse<BolShipmentConfirmation[]> = await res.json()
  if (!res.ok || body.error) {
    throw CodedApiError.fromEnvelope(body.error, res.status)
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

// bolFailureText is why a confirmation failed, in the reader's language
// when CommerceOS itself knows the reason; otherwise Bol's own message.
export function bolFailureText(t: TFunction<'orders'>, c: BolShipmentConfirmation): string {
  switch (c.failure_reason) {
    case 'no_transporter_code': return t('bolConfirmation.reasonNoTransporterCode')
    case 'no_bol_shipment': return t('bolConfirmation.reasonNoBolShipment')
    default: return c.error_message
  }
}
