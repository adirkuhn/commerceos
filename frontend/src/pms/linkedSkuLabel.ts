import type { SKULinkSummary } from './types'

// How a SKU of another product is named wherever it's linked (case pack,
// weight pool): its product's name plus its EAN -- what's printed on the
// shelf -- or its SKU code when it has no barcode.
export function linkedSkuLabel(s: Pick<SKULinkSummary, 'product_name' | 'sku_code'> & { barcode?: string | null }): string {
  return s.barcode ? `${s.product_name} · EAN ${s.barcode}` : `${s.product_name} · ${s.sku_code}`
}
