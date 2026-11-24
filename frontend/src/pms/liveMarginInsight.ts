import type { SKUMarginInsight } from './types'
import { profitAt } from './profit'

// The price and sale price as typed on the product form, next to the
// saved ones they edit.
export interface MarginInsightDraft {
  price: string
  salePrice: string
  savedPriceMinorUnits: number
  savedSalePriceMinorUnits: number | null
}

// A typed amount in minor units; undefined while it isn't a usable price
// yet ('', '.', '-3').
function typedMinorUnits(value: string): number | undefined {
  const trimmed = value.trim()
  if (!/^\d*\.?\d*$/.test(trimmed) || !/\d/.test(trimmed)) return undefined
  return Math.round(parseFloat(trimmed) * 100)
}

// liveMarginInsight is the saved insight with profit, margin, markup and
// profit potential worked out at the typed prices instead (same formula as
// the backend's GetSKUMarginInsightUseCase, see profitAt), so staff see a
// price's margin before saving it. A price not typed in full keeps the
// saved figures; an empty sale price means no sale.
export function liveMarginInsight(
  insight: SKUMarginInsight, draft: MarginInsightDraft | undefined,
): { insight: SKUMarginInsight; unsaved: boolean } {
  const unchanged = { insight, unsaved: false }
  if (!draft || !insight.cost_price_available) return unchanged
  const price = typedMinorUnits(draft.price)
  if (price === undefined) return unchanged
  const sale = draft.salePrice.trim() === '' ? null : typedMinorUnits(draft.salePrice) ?? draft.savedSalePriceMinorUnits
  if (price === draft.savedPriceMinorUnits && sale === draft.savedSalePriceMinorUnits) return unchanged

  const cost = insight.cost_price_minor_units
  const atPrice = profitAt(price, cost)
  const atSale = sale == null ? null : profitAt(sale, cost)
  return {
    unsaved: true,
    insight: {
      ...insight,
      profit_at_price_minor_units: atPrice.profit,
      margin_at_price_basis_points: atPrice.marginBasisPoints,
      markup_at_price_basis_points: atPrice.markupBasisPoints,
      profit_at_sale_price_minor_units: atSale?.profit ?? null,
      margin_at_sale_price_basis_points: atSale?.marginBasisPoints ?? null,
      markup_at_sale_price_basis_points: atSale?.markupBasisPoints ?? null,
      total_profit_potential_minor_units: (atSale ?? atPrice).profit * insight.total_on_hand_quantity,
    },
  }
}
