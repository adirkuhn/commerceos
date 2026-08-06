import type {
  Booking, BookingDetail, ScanBookingItemInput, ScanBookingItemResult,
  CompleteBookingResult, ScannerPairingCode, APIResponse,
} from './types'

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

export function pauseBooking(token: string, id: string): Promise<Booking> {
  return request<Booking>(`${base()}/procurement/bookings/${id}/pause`, { method: 'POST' }, token)
}

export function completeBooking(token: string, id: string): Promise<CompleteBookingResult> {
  return request<CompleteBookingResult>(`${base()}/procurement/bookings/${id}/complete`, { method: 'POST' }, token)
}
