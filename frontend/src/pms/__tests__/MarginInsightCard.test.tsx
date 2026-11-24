// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { MarginInsightCard, type MarginInsightDraft } from '../ProductPricingCards'
import type { SKUMarginInsight } from '../types'

vi.mock('../../auth/AuthContext', () => ({ useAuth: () => ({ token: 'tok' }) }))
vi.mock('../api', () => ({ getSKUMarginInsight: vi.fn(), setManualCostPrice: vi.fn(), clearCostPriceOverride: vi.fn() }))
import * as api from '../api'

const insight = (price: number): SKUMarginInsight => ({
  cost_price_available: true, cost_price_minor_units: 700, cost_price_source: 'computed',
  effective_target_markup_basis_points: 3000, recommended_price_minor_units: 910,
  profit_at_price_minor_units: price - 700, margin_at_price_basis_points: Math.round((price - 700) * 10000 / price),
  markup_at_price_basis_points: Math.round((price - 700) * 10000 / 700),
  profit_at_sale_price_minor_units: null, margin_at_sale_price_basis_points: null, markup_at_sale_price_basis_points: null,
  total_on_hand_quantity: 5, total_profit_potential_minor_units: (price - 700) * 5, shrinkage_minor_units: 0,
})

const card = (draft: MarginInsightDraft) => (
  <MarginInsightCard skuId="sku-1" currency="EUR" taxRateBasisPoints={null} pricesIncludeTax={false} draft={draft} />
)

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(api.getSKUMarginInsight).mockResolvedValue(insight(1000))
})

describe('MarginInsightCard', () => {
  it('follows the price being typed, marked as unsaved', async () => {
    const { rerender } = render(card({ price: '10.00', salePrice: '', savedPriceMinorUnits: 1000, savedSalePriceMinorUnits: null }))
    expect(await screen.findByText(/30\.0% margin/)).toBeInTheDocument()
    expect(screen.queryByText('Unsaved price')).not.toBeInTheDocument()

    rerender(card({ price: '14', salePrice: '', savedPriceMinorUnits: 1000, savedSalePriceMinorUnits: null }))
    expect(screen.getByText(/50\.0% margin/)).toBeInTheDocument()
    expect(screen.getByText('Unsaved price')).toBeInTheDocument()

    rerender(card({ price: '14', salePrice: '7.70', savedPriceMinorUnits: 1000, savedSalePriceMinorUnits: null }))
    expect(screen.getByText('Profit at sale price')).toBeInTheDocument()
    expect(screen.getByText(/9\.1% margin/)).toBeInTheDocument()
    expect(api.getSKUMarginInsight).toHaveBeenCalledTimes(1) // typing never hits the server
  })

  it('reloads once the typed price is saved', async () => {
    const { rerender } = render(card({ price: '14', salePrice: '', savedPriceMinorUnits: 1000, savedSalePriceMinorUnits: null }))
    expect(await screen.findByText('Unsaved price')).toBeInTheDocument()

    vi.mocked(api.getSKUMarginInsight).mockResolvedValue(insight(1400))
    rerender(card({ price: '14', salePrice: '', savedPriceMinorUnits: 1400, savedSalePriceMinorUnits: null }))

    await vi.waitFor(() => expect(api.getSKUMarginInsight).toHaveBeenCalledTimes(2))
    expect(await screen.findByText(/50\.0% margin/)).toBeInTheDocument()
    expect(screen.queryByText('Unsaved price')).not.toBeInTheDocument()
  })
})
