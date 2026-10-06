// @vitest-environment jsdom
import { describe, it, expect } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import { SupplierRecommendationCard } from '../SupplierRecommendationCard'
import type { RecommendationLine, SupplierRecommendation } from '../types'

function line(overrides: Partial<RecommendationLine> = {}): RecommendationLine {
  return {
    sku_id: 'sku-1', sku_code: 'PAN-1', product_name: 'Panettone', variant_name: 'Default', is_weight_pool: false,
    on_hand: 12, on_order: { sent: 50, draft: 8 },
    forecast: { units: 968, method: 'seasonal', growth_factor: 1.18, days: 51, recent_units: 118, last_year_units: 820 },
    daily_demand: 18.98, safety_stock: 133, needed_units: 1031, purchase_quantity: 172, units_per_purchased_item: 6,
    order_units: 1032, unit_cost_minor_units: 450, estimated_cost_minor_units: 77400, currency_code: 'EUR',
    stockout_date: '2026-10-30', order_by_date: '2026-10-09',
    ...overrides,
  }
}

function group(overrides: Partial<SupplierRecommendation> = {}): SupplierRecommendation {
  return {
    supplier_id: 'sup-1', supplier_name: 'Bauducco NL', lead_time_days: 21, lead_time_assumed: false,
    lines: [line()], estimated_cost_minor_units: 77400, currency_code: 'EUR', earliest_order_by_date: '2026-10-09',
    ...overrides,
  }
}

const AS_OF = '2026-10-06'

describe('SupplierRecommendationCard', () => {
  it('shows the supplier, lead time, order-by date and estimated total', () => {
    render(<SupplierRecommendationCard group={group()} asOf={AS_OF} />)

    expect(screen.getByRole('heading', { name: 'Bauducco NL' })).toBeInTheDocument()
    expect(screen.getByText('Lead time 21 days')).toBeInTheDocument()
    expect(screen.getByText('Order by 09/10/2026')).toBeInTheDocument()
    expect(screen.getByText(/Est\. €774\.00/)).toBeInTheDocument()
  })

  it('explains a seasonal forecast and what to order', () => {
    render(<SupplierRecommendationCard group={group()} asOf={AS_OF} />)

    const row = screen.getByTestId('recommendation-row')
    expect(within(row).getByText('Panettone')).toBeInTheDocument()
    expect(within(row).getByText('PAN-1')).toBeInTheDocument()
    expect(within(row).getByText('968')).toBeInTheDocument()
    expect(within(row).getByText('Last year 820 × 1.18')).toBeInTheDocument()
    expect(within(row).getByText('next 51 days')).toBeInTheDocument()
    expect(within(row).getByText('172 × 6')).toBeInTheDocument()
    expect(within(row).getByText('58')).toBeInTheDocument()
    expect(within(row).getByText('8 in draft')).toBeInTheDocument()
    expect(within(row).getByText('This week')).toBeInTheDocument()
  })

  it('shows a weight pool in kg and a recent-velocity forecast', () => {
    render(<SupplierRecommendationCard group={group({
      lines: [line({
        is_weight_pool: true, on_hand: 2000, on_order: { sent: 0, draft: 0 }, purchase_quantity: 42, units_per_purchased_item: 1000,
        forecast: { units: 44000, method: 'recent_velocity', growth_factor: 1, days: 44, recent_units: 56000, last_year_units: 0 },
        order_by_date: '2026-10-01',
      })],
    })} asOf={AS_OF} />)

    const row = screen.getByTestId('recommendation-row')
    expect(within(row).getByText('2 kg')).toBeInTheDocument()
    expect(within(row).getByText('44 kg')).toBeInTheDocument()
    expect(within(row).getByText('Recent sales: 56 kg in 56 days')).toBeInTheDocument()
    expect(within(row).getByText('42 kg')).toBeInTheDocument()
    expect(within(row).getByText('Late')).toBeInTheDocument()
  })

  it('flags SKUs without a supplier and an assumed lead time', () => {
    render(<SupplierRecommendationCard group={group({
      supplier_id: null, supplier_name: '', lead_time_days: 14, lead_time_assumed: true, estimated_cost_minor_units: null,
      lines: [line({ unit_cost_minor_units: null, estimated_cost_minor_units: null })],
    })} asOf={AS_OF} />)

    expect(screen.getByRole('heading', { name: 'No supplier yet' })).toBeInTheDocument()
    expect(screen.getByText(/Never bought on a purchase order/)).toBeInTheDocument()
    expect(screen.getByText('assumed')).toBeInTheDocument()
    expect(screen.queryByText(/^Est\. €/)).not.toBeInTheDocument()
  })

  it('shows a season line\'s pre-season demand and seasonal lift', () => {
    render(<SupplierRecommendationCard group={group({
      lines: [line({ pre_season: { units: 48, method: 'seasonal', growth_factor: 1.18, days: 38 }, seasonal_lift: 18.64 })],
    })} asOf={AS_OF} />)

    const row = screen.getByTestId('recommendation-row')
    expect(within(row).getByText('+48 until the season')).toBeInTheDocument()
    expect(within(row).getByText('18.6× usual sales')).toBeInTheDocument()
  })
})
