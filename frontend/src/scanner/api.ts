import type {
  Booking, BookingDetail, ScanBookingItemInput, ScanBookingItemResult,
  ResolveBookingScanCodeResult, CompleteBookingResult, ScannerPairingCode, APIResponse,
  PickingSession, PickingSessionDetail, ScanPickItemInput, ScanPickItemResult,
  SubstituteLineItemInput, SubstituteLineItemResult, RefundLineItemRequest, RefundLineItemResult,
  ResolvePickScanCodeResult, CompletePickingResult, StockShortfallResolution,
  PackingSession, PackingSessionDetail, ScanPackItemInput, ScanPackItemResult, CompletePackingResult,
} from './types'
import { CodedApiError } from '../api/codedApiError'

function base(): string {
  return import.meta.env.VITE_API_BASE_URL as string
}

// onTokenActivity lets useScannerAuth observe real API traffic without this
// module holding any session state itself -- every authenticated call funnels
// through request() below, so registering one callback here is enough for
// useScannerAuth to drive activity-based token refresh (see its doc comment).
// A single module-level slot is enough: only one scanner session is ever
// active per app instance.
type TokenActivityHandler = (token: string) => void
let onTokenActivity: TokenActivityHandler | null = null

export function setOnTokenActivity(handler: TokenActivityHandler | null): void {
  onTokenActivity = handler
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
  // 204 No Content (heartbeats, deletes, releases) has no body to parse --
  // res.json() would throw on it.
  if (res.status === 204) {
    if (token) onTokenActivity?.(token)
    return null as T
  }
  const body: APIResponse<T> = await res.json()
  if (!res.ok || body.error) {
    throw new CodedApiError(body.error?.code ?? 'HTTP_ERROR', body.error?.message ?? `HTTP ${res.status}`)
  }
  if (token) onTokenActivity?.(token)
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

// refreshScannerSession reissues a still-valid pairing-code token with a
// fresh expiry, same claims -- see useScannerAuth's activity-based refresh.
export function refreshScannerSession(token: string): Promise<{ token: string }> {
  return request<{ token: string }>(`${base()}/iam/scanner-pairing/refresh`, { method: 'POST' }, token)
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

// touchPickingSession is a heartbeat call while a session stays open --
// refreshes the caller's lock so it doesn't go stale under the backend's
// PickingLockTTL. Fire-and-poll from a setInterval; a 409 means someone
// else has since taken over the session.
export function touchPickingSession(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/picking-sessions/${id}/touch`, { method: 'POST' }, token)
}

// resetPickingSession discards every staged scan on the session and
// releases its lock, handing it back fresh -- the picker's "start over"
// button. Only legal while the session is in_progress/paused.
export function resetPickingSession(token: string, id: string): Promise<PickingSession> {
  return request<PickingSession>(`${base()}/picking-sessions/${id}/reset`, { method: 'POST' }, token)
}

// resolutions maps a PickScan id (one already returned in a prior call's
// stock_shortfalls) to how staff want it handled -- see
// StockShortfallResolution. Omit/empty on the first attempt.
export function completePicking(
  token: string, id: string, resolutions?: Record<string, StockShortfallResolution>,
): Promise<CompletePickingResult> {
  return request<CompletePickingResult>(
    `${base()}/picking-sessions/${id}/complete`,
    { method: 'POST', body: JSON.stringify({ resolutions }) },
    token,
  )
}

// ── Packing ──────────────────────────────────────────────────────────────────

export function startOrResumePacking(token: string, salesOrderId: string): Promise<PackingSession> {
  return request<PackingSession>(`${base()}/sales-orders/${salesOrderId}/packing`, { method: 'POST' }, token)
}

export function getPackingSession(token: string, id: string): Promise<PackingSessionDetail> {
  return request<PackingSessionDetail>(`${base()}/packing-sessions/${id}`, { method: 'GET' }, token)
}

export function scanPackItem(token: string, sessionId: string, input: ScanPackItemInput): Promise<ScanPackItemResult> {
  return request<ScanPackItemResult>(`${base()}/packing-sessions/${sessionId}/scan`, { method: 'POST', body: JSON.stringify(input) }, token)
}

export function deletePackScan(token: string, sessionId: string, scanId: string): Promise<null> {
  return request<null>(`${base()}/packing-sessions/${sessionId}/scans/${scanId}`, { method: 'DELETE' }, token)
}

// touchPackingSession mirrors touchPickingSession -- heartbeat call while a
// packing session stays open.
export function touchPackingSession(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/packing-sessions/${id}/touch`, { method: 'POST' }, token)
}

// resetPackingSession mirrors resetPickingSession -- the packer's "start
// over" button.
export function resetPackingSession(token: string, id: string): Promise<PackingSession> {
  return request<PackingSession>(`${base()}/packing-sessions/${id}/reset`, { method: 'POST' }, token)
}

// resolutions maps a pack scan id (from a prior call's stock_shortfalls) to
// how staff want it handled. Omit on the first attempt.
export function completePacking(
  token: string, id: string, resolutions?: Record<string, StockShortfallResolution>,
): Promise<CompletePackingResult> {
  return request<CompletePackingResult>(
    `${base()}/packing-sessions/${id}/complete`,
    { method: 'POST', body: JSON.stringify({ resolutions }) },
    token,
  )
}

// Order changes happen in packing: substitute (part of) a line, refund
// (remove) part of a line, or resolve a scanned code before staging one.
// Nothing is real until completePacking.
export function substitutePackLine(token: string, sessionId: string, input: SubstituteLineItemInput): Promise<SubstituteLineItemResult> {
  return request<SubstituteLineItemResult>(`${base()}/packing-sessions/${sessionId}/substitute-batch`, { method: 'POST', body: JSON.stringify(input) }, token)
}

export function refundPackLine(token: string, sessionId: string, input: RefundLineItemRequest): Promise<RefundLineItemResult> {
  return request<RefundLineItemResult>(`${base()}/packing-sessions/${sessionId}/refund`, { method: 'POST', body: JSON.stringify(input) }, token)
}

export function resolvePackScanCode(token: string, sessionId: string, code: string): Promise<ResolvePickScanCodeResult> {
  return request<ResolvePickScanCodeResult>(`${base()}/packing-sessions/${sessionId}/resolve-scan`, { method: 'POST', body: JSON.stringify({ code }) }, token)
}

// ── Session locks ────────────────────────────────────────────────────────────
// Opening a picking/packing session only views it; the first action takes
// the lock. Release it when leaving the order -- keepalive so it still goes
// out while the page is closing. Take-over is the explicit "Take over"
// action (admins, or a stale lock).

export function releasePackingSessionLock(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/packing-sessions/${id}/release`, { method: 'POST', keepalive: true }, token)
}

export function releasePickingSessionLock(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/picking-sessions/${id}/release`, { method: 'POST', keepalive: true }, token)
}

export function takeOverPackingSession(token: string, id: string): Promise<PackingSession> {
  return request<PackingSession>(`${base()}/packing-sessions/${id}/take-over`, { method: 'POST' }, token)
}

export function takeOverPickingSession(token: string, id: string): Promise<PickingSession> {
  return request<PickingSession>(`${base()}/picking-sessions/${id}/take-over`, { method: 'POST' }, token)
}
