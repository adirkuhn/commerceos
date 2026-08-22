import type {
  Booking, BookingDetail, ScanBookingItemInput, ScanBookingItemResult,
  ResolveBookingScanCodeResult, CompleteBookingResult, ScannerPairingCode, APIResponse,
  PickingSession, PickingSessionDetail, ScanPickItemInput, ScanPickItemResult,
  ResolvePickScanCodeResult, CompletePickingResult, AddSalesOrderLineItemInput,
} from './types'
import type { SalesOrderLineItem } from '../oms/types'

function base(): string {
  return import.meta.env.VITE_API_BASE_URL as string
}

async function request<T>(path: string, options: RequestInit, token?: string): Promise<T> {
  const res = await fetch(path, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options.headers,
    },
  })
  const body: APIResponse<T> = await res.json()
  if (!res.ok || body.error) {
    throw new Error(body.error?.message ?? `HTTP ${res.status}`)
  }
  return body.data as T
}

// ── Pairing ──────────────────────────────────────────────────────────────────

// claimPairingCode is the one call in this file made with no token -- the
// scanner has none yet, that's the whole point of pairing.
export function claimPairingCode(code: string): Promise<{ token: string }> {
  return request<{ token: string }>(`${base()}/iam/scanner-pairing/claim`, { method: 'POST', body: JSON.stringify({ code }) })
}

// claimPairingCodeByToken is the QR/barcode counterpart to claimPairingCode
// -- also unauthenticated, but the value it submits is unguessable so this
// path carries no server-side rate limit.
export function claimPairingCodeByToken(token: string): Promise<{ token: string }> {
  return request<{ token: string }>(`${base()}/iam/scanner-pairing/claim-token`, { method: 'POST', body: JSON.stringify({ token }) })
}

export function generatePairingCode(token: string): Promise<ScannerPairingCode> {
  return request<ScannerPairingCode>(`${base()}/iam/scanner-pairing`, { method: 'POST' }, token)
}

export function getPairingCodeStatus(token: string, id: string): Promise<ScannerPairingCode> {
  return request<ScannerPairingCode>(`${base()}/iam/scanner-pairing/${id}`, { method: 'GET' }, token)
}

// ── Booking ──────────────────────────────────────────────────────────────────

// warehouseId is only consulted the first time a PO is booked (when it has
// none assigned yet) -- ignored on every later start/resume for the same PO.
export function startOrResumeBooking(token: string, purchaseOrderId: string, warehouseId?: string): Promise<Booking> {
  return request<Booking>(
    `${base()}/procurement/purchase-orders/${purchaseOrderId}/booking`,
    { method: 'POST', body: warehouseId ? JSON.stringify({ warehouse_id: warehouseId }) : undefined },
    token,
  )
}

export function getBooking(token: string, id: string): Promise<BookingDetail> {
  return request<BookingDetail>(`${base()}/procurement/bookings/${id}`, { method: 'GET' }, token)
}

export function scanBookingItem(token: string, bookingId: string, input: ScanBookingItemInput): Promise<ScanBookingItemResult> {
  return request<ScanBookingItemResult>(`${base()}/procurement/bookings/${bookingId}/scan`, { method: 'POST', body: JSON.stringify(input) }, token)
}

export function resolveBookingScanCode(token: string, bookingId: string, code: string): Promise<ResolveBookingScanCodeResult> {
  return request<ResolveBookingScanCodeResult>(`${base()}/procurement/bookings/${bookingId}/resolve-scan`, { method: 'POST', body: JSON.stringify({ code }) }, token)
}

export function deleteBookingScan(token: string, bookingId: string, scanId: string): Promise<null> {
  return request<null>(`${base()}/procurement/bookings/${bookingId}/scans/${scanId}`, { method: 'DELETE' }, token)
}

export function pauseBooking(token: string, id: string): Promise<Booking> {
  return request<Booking>(`${base()}/procurement/bookings/${id}/pause`, { method: 'POST' }, token)
}

export function completeBooking(token: string, id: string): Promise<CompleteBookingResult> {
  return request<CompleteBookingResult>(`${base()}/procurement/bookings/${id}/complete`, { method: 'POST' }, token)
}

// ── Picking ──────────────────────────────────────────────────────────────────
// oms's own routes mount with no module-path prefix (see backend/cmd/api/
// main.go's omsGroup), unlike procurement's Booking endpoints above -- so
// these paths are /sales-orders/... and /picking-sessions/... directly.

export function startOrResumePicking(token: string, salesOrderId: string): Promise<PickingSession> {
  return request<PickingSession>(`${base()}/sales-orders/${salesOrderId}/picking`, { method: 'POST' }, token)
}

export function getPickingSession(token: string, id: string): Promise<PickingSessionDetail> {
  return request<PickingSessionDetail>(`${base()}/picking-sessions/${id}`, { method: 'GET' }, token)
}

export function scanPickItem(token: string, sessionId: string, input: ScanPickItemInput): Promise<ScanPickItemResult> {
  return request<ScanPickItemResult>(`${base()}/picking-sessions/${sessionId}/scan`, { method: 'POST', body: JSON.stringify(input) }, token)
}

export function resolvePickScanCode(token: string, sessionId: string, code: string): Promise<ResolvePickScanCodeResult> {
  return request<ResolvePickScanCodeResult>(`${base()}/picking-sessions/${sessionId}/resolve-scan`, { method: 'POST', body: JSON.stringify({ code }) }, token)
}

export function deletePickScan(token: string, sessionId: string, scanId: string): Promise<null> {
  return request<null>(`${base()}/picking-sessions/${sessionId}/scans/${scanId}`, { method: 'DELETE' }, token)
}

export function pausePicking(token: string, id: string): Promise<PickingSession> {
  return request<PickingSession>(`${base()}/picking-sessions/${id}/pause`, { method: 'POST' }, token)
}

export function completePicking(token: string, id: string): Promise<CompletePickingResult> {
  return request<CompletePickingResult>(`${base()}/picking-sessions/${id}/complete`, { method: 'POST' }, token)
}

// addSalesOrderLineItem appends a real line item to an already-existing
// order -- used by the freebie/new-item picking flow. Not itself a picking
// endpoint (see oms's own openapi.yaml), just co-located here since only
// the picking flow calls it from the scanner app.
export function addSalesOrderLineItem(token: string, salesOrderId: string, input: AddSalesOrderLineItemInput): Promise<SalesOrderLineItem> {
  return request<SalesOrderLineItem>(`${base()}/sales-orders/${salesOrderId}/line-items`, { method: 'POST', body: JSON.stringify(input) }, token)
}
