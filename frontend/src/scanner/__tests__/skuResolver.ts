import type { ResolveBarcodeResult, SKU } from '../../pms/types'

// Mimics the backend's resolveBarcode for tests: an exact SKU code resolves,
// anything else (a partial/misread scan) is "sku not found". SKU screens are
// reached by URL and re-resolve their SKU from its code, so tests that pick a
// SKU via the partial-match fallback need that second, exact lookup to work.
export function resolvesSkuCodes(entries: [SKU, string][]) {
  return async (_token: string, code: string): Promise<ResolveBarcodeResult> => {
    const hit = entries.find(([s]) => s.sku_code === code)
    if (!hit) throw new Error('sku not found')
    return { sku: hit[0], units_per_scan: 1, product_name: hit[1] } as ResolveBarcodeResult
  }
}
