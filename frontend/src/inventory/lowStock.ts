import { DEFAULT_LOW_STOCK_THRESHOLD } from '../settings/types'

// Same rule the backend's ?stock=low|out product filter applies (see
// backend/api/pms/openapi.yaml listProducts) -- this only colours a total
// the screen already shows.

// effectiveLowStockThreshold: the product's own line, else the shop's
// setting, else the default a never-saved shop reads.
export function effectiveLowStockThreshold(productOwn: number | null | undefined, shop: number | undefined): number {
  return productOwn ?? shop ?? DEFAULT_LOW_STOCK_THRESHOLD
}

export type StockTone = 'out' | 'low' | 'ok'

export function stockTone(total: number, threshold: number): StockTone {
  if (total <= 0) return 'out'
  return total <= threshold ? 'low' : 'ok'
}
