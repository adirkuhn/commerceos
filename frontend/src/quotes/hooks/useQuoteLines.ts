import { useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'
import { decodeQuoteLines, encodeQuoteLines } from '../quoteLines'
import type { QuoteLine } from '../types'

const KEY = 'lines'

export interface QuoteLines {
  lines: QuoteLine[]
  add: (productId: string, skuId: string) => void
  setQuantity: (skuId: string, quantity: number) => void
  // undefined goes back to the discounted price.
  setUnitPrice: (skuId: string, unitPrice: number | undefined) => void
  remove: (skuId: string) => void
  clear: () => void
}

// The quote's lines live in ?lines= so refresh and Back keep the quote.
// Every change reads the URL at write time, so quick successive edits
// compose instead of each starting from a stale render.
export function useQuoteLines(): QuoteLines {
  const [searchParams, setSearchParams] = useSearchParams()
  const lines = decodeQuoteLines(searchParams.get(KEY))

  const update = useCallback((change: (lines: QuoteLine[]) => QuoteLine[]) => {
    setSearchParams(prev => {
      const next = new URLSearchParams(prev)
      const encoded = encodeQuoteLines(change(decodeQuoteLines(prev.get(KEY))))
      if (encoded === null) next.delete(KEY)
      else next.set(KEY, encoded)
      return next
    }, { replace: true })
  }, [setSearchParams])

  const edit = useCallback((skuId: string, patch: (line: QuoteLine) => QuoteLine) =>
    update(ls => ls.map(l => (l.skuId === skuId ? patch(l) : l))), [update])

  return {
    lines,
    add: useCallback((productId: string, skuId: string) =>
      update(ls => (ls.some(l => l.skuId === skuId) ? ls : [...ls, { productId, skuId, quantity: 1 }])), [update]),
    setQuantity: useCallback((skuId: string, quantity: number) => edit(skuId, l => ({ ...l, quantity })), [edit]),
    setUnitPrice: useCallback((skuId: string, unitPrice: number | undefined) => edit(skuId, ({ unitPrice: _old, ...l }) =>
      (unitPrice === undefined ? l : { ...l, unitPrice })), [edit]),
    remove: useCallback((skuId: string) => update(ls => ls.filter(l => l.skuId !== skuId)), [update]),
    clear: useCallback(() => update(() => []), [update]),
  }
}
