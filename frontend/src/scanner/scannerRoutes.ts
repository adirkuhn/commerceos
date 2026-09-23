// Single source of truth for scanner URLs. Every screen that shows a
// specific resource (a booking, a picking/packing session, a SKU) carries
// that resource's id in the path, so a refresh or the device's Back button
// lands on the same screen instead of dumping the operator on the launcher.
// Screens reload their data from the id -- nothing is passed through
// in-memory state that a reload would lose.

export const SCANNER_BASE = '/scanner'

const enc = encodeURIComponent

export const scannerPaths = {
  launcher: SCANNER_BASE,
  receive: `${SCANNER_BASE}/receive`,
  booking: (bookingId: string) => `${SCANNER_BASE}/receive/${enc(bookingId)}`,
  inventory: `${SCANNER_BASE}/inventory`,
  inventorySku: (code: string) => `${SCANNER_BASE}/inventory/${enc(code)}`,
  priceLookup: `${SCANNER_BASE}/price-lookup`,
  priceLookupSku: (code: string) => `${SCANNER_BASE}/price-lookup/${enc(code)}`,
  discard: `${SCANNER_BASE}/discard`,
  discardSku: (code: string) => `${SCANNER_BASE}/discard/${enc(code)}`,
  pick: `${SCANNER_BASE}/pick`,
  pickSession: (sessionId: string) => `${SCANNER_BASE}/pick/${enc(sessionId)}`,
  pickOrder: (orderId: string) => `${SCANNER_BASE}/pick/order/${enc(orderId)}`,
  pack: `${SCANNER_BASE}/pack`,
  packSession: (sessionId: string) => `${SCANNER_BASE}/pack/${enc(sessionId)}`,
} as const

export function isScannerPath(pathname: string): boolean {
  return pathname === SCANNER_BASE || pathname.startsWith(`${SCANNER_BASE}/`)
}
