// @vitest-environment jsdom
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ClearancePromotionDialog } from '../ClearancePromotionDialog'
import * as promotionsApi from '../../../promotions/api'
import { CodedApiError } from '../../../api/codedApiError'
import type { CloseToExpiryBatch } from '../types'
import type { SkuVat } from '../clearance'
import { zonedMidnight } from '../../../format/calendarDay'

vi.mock('../../../promotions/api')

const batch: CloseToExpiryBatch = {
  batch_id: 'b-1', lot_number: 'L1', expiry_date: '2026-10-30', days_left: 18, quantity: 7,
  warehouse_id: 'wh-1', warehouse_name: 'Amsterdam', sku_id: 'sku-1', sku_code: 'PQ-1',
  product_id: 'p-1', product_name: 'Pão de queijo', image_url: '', price_minor_units: 650, currency_code: 'EUR',
}

function setup() {
  const onClose = vi.fn()
  const onCreated = vi.fn()
  render(<ClearancePromotionDialog token="tok" batch={batch} onClose={onClose} onCreated={onCreated} />)
  return { onClose, onCreated }
}

beforeEach(() => {
  vi.mocked(promotionsApi.createPromotion).mockReset().mockResolvedValue({
    promotion: { id: 'promo-1', name: 'x' } as never, warnings: [],
  })
})

describe('ClearancePromotionDialog', () => {
  it('starts a fixed-price sale for the batch, ending on its expiry date', async () => {
    const { onCreated } = setup()
    fireEvent.change(screen.getByLabelText(/sale price/i), { target: { value: '4,99' } })
    expect(screen.getByText(/€4\.99/)).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: /start sale/i }))

    await waitFor(() => expect(onCreated).toHaveBeenCalledWith({ promotionId: 'promo-1', name: 'Clearance Pão de queijo (lot L1)', warnings: [] }))
    const input = vi.mocked(promotionsApi.createPromotion).mock.calls[0][1]
    expect(input).toMatchObject({
      name: 'Clearance Pão de queijo (lot L1)', type: 'fixed_price', status: 'active',
      lines: [{ sku_id: 'sku-1', batch_id: 'b-1', discount_type: 'fixed_price', discount_value: 499 }],
    })
    // Runs through the whole expiry day (local zone without a tenant time zone).
    expect(input.ends_at).toBe(zonedMidnight('2026-10-31', undefined).toISOString())
  })

  it('takes a percentage off', async () => {
    setup()
    fireEvent.click(screen.getByRole('radio', { name: /% off/i }))
    fireEvent.change(screen.getByLabelText(/discount/i), { target: { value: '30' } })
    fireEvent.click(screen.getByRole('button', { name: /start sale/i }))
    await waitFor(() => expect(promotionsApi.createPromotion).toHaveBeenCalled())
    expect(vi.mocked(promotionsApi.createPromotion).mock.calls[0][1].lines[0]).toMatchObject({ discount_type: 'percentage', discount_value: 3000 })
  })

  it('runs until the batch is gone when the end date is cleared', async () => {
    setup()
    fireEvent.change(screen.getByLabelText(/sale price/i), { target: { value: '5' } })
    fireEvent.change(screen.getByLabelText(/ends/i), { target: { value: '' } })
    fireEvent.click(screen.getByRole('button', { name: /start sale/i }))
    await waitFor(() => expect(promotionsApi.createPromotion).toHaveBeenCalled())
    expect(vi.mocked(promotionsApi.createPromotion).mock.calls[0][1].ends_at).toBeNull()
  })

  it('does not submit an invalid price', () => {
    setup()
    fireEvent.change(screen.getByLabelText(/sale price/i), { target: { value: '0' } })
    fireEvent.click(screen.getByRole('button', { name: /start sale/i }))
    expect(screen.getByRole('alert')).toHaveTextContent(/above zero/i)
    expect(promotionsApi.createPromotion).not.toHaveBeenCalled()
  })

  it('shows the API error and stays open', async () => {
    vi.mocked(promotionsApi.createPromotion).mockRejectedValue(new CodedApiError('PROMOTION_NAME_REQUIRED', 'name required'))
    const { onCreated } = setup()
    fireEvent.change(screen.getByLabelText(/sale price/i), { target: { value: '5' } })
    fireEvent.click(screen.getByRole('button', { name: /start sale/i }))
    await waitFor(() => expect(screen.getByRole('alert')).toBeInTheDocument())
    expect(onCreated).not.toHaveBeenCalled()
  })

  it('cancel closes without creating anything', () => {
    const { onClose } = setup()
    fireEvent.click(screen.getByRole('button', { name: /^cancel$/i }))
    expect(onClose).toHaveBeenCalled()
    expect(promotionsApi.createPromotion).not.toHaveBeenCalled()
  })
})

describe('ClearancePromotionDialog -- what the sale still makes', () => {
  const costed: CloseToExpiryBatch = { ...batch, quantity: 12, price_minor_units: 450, unit_cost_minor_units: 210, unit_cost_source: 'batch' }
  const vat = { rateBasisPoints: 900, pricesIncludeTax: true }

  function renderWith(b: CloseToExpiryBatch, v: SkuVat = vat) {
    render(<ClearancePromotionDialog token="tok" batch={b} vat={v} onClose={vi.fn()} onCreated={vi.fn()} />)
  }

  it('shows the lot cost and the profit at the regular price, excl. and incl. VAT', () => {
    renderWith(costed)

    const panel = within(screen.getByRole('region', { name: 'Profit' }))
    expect(panel.getByText('€2.10')).toBeInTheDocument()
    expect(panel.getByText('this lot')).toBeInTheDocument()
    const regular = within(panel.getByRole('row', { name: /regular/i }))
    expect(regular.getByText(/€2\.03 · 49\.1% · 96\.7%/)).toBeInTheDocument() // excl. VAT: 4.13 - 2.10
    expect(regular.getByText(/€2\.40 · 53\.3% · 114\.3%/)).toBeInTheDocument() // incl. VAT
  })

  it('updates the sale row and the whole-lot profit as the price is typed', () => {
    renderWith(costed)
    fireEvent.change(screen.getByLabelText(/sale price/i), { target: { value: '2,99' } })

    const panel = within(screen.getByRole('region', { name: 'Profit' }))
    const sale = within(panel.getByRole('row', { name: /sale/i }))
    expect(sale.getByText(/€0\.64 · 23\.4% · 30\.5%/)).toBeInTheDocument()
    expect(sale.getByText(/€0\.89 · 29\.8% · 42\.4%/)).toBeInTheDocument()
    expect(panel.getByText('Whole lot (12 left): €7.68 profit excl. VAT')).toBeInTheDocument()
    expect(screen.queryByRole('alert')).toBeNull()
  })

  it('warns, without blocking, when the sale price is below cost', async () => {
    renderWith(costed)
    fireEvent.change(screen.getByLabelText(/sale price/i), { target: { value: '1,99' } })

    expect(screen.getByRole('alert')).toHaveTextContent('Below cost: loses €0.27 per unit (€3.24 on this lot).')
    fireEvent.click(screen.getByRole('button', { name: /start sale/i }))
    await waitFor(() => expect(promotionsApi.createPromotion).toHaveBeenCalled())
  })

  it('names the average cost when the lot has none', () => {
    renderWith({ ...costed, unit_cost_source: 'sku' })

    expect(within(screen.getByRole('region', { name: 'Profit' })).getByText('average cost')).toBeInTheDocument()
  })

  it('without a VAT rate shows the profit at the price as stored', () => {
    renderWith(costed, { rateBasisPoints: null, pricesIncludeTax: true })

    const regular = within(screen.getByRole('row', { name: /regular/i }))
    expect(regular.getByText(/€2\.40 · 53\.3% · 114\.3%/)).toBeInTheDocument()
    expect(regular.queryByText(/excl\. VAT/)).toBeNull()
  })

  it('says so when no cost is known', () => {
    renderWith({ ...costed, unit_cost_minor_units: null, unit_cost_source: null })

    expect(screen.getByText("No cost price for this product, so the profit can't be shown.")).toBeInTheDocument()
    expect(screen.queryByRole('region', { name: 'Profit' })).toBeNull()
  })

  // A kilo SKU is an internal weight pool: never sold itself, cost per gram.
  it('a weight pool is not put on sale: shows cost and stock per kg and points to the packages', () => {
    renderWith({ ...costed, weight_pool: true, quantity: 2500, unit_cost_minor_units: 1 })

    expect(screen.getByText('This is a bulk weight pool. It is sold as packages, so put the packages on sale instead.')).toBeInTheDocument()
    expect(screen.getByText('2.5 kg left · cost €10.00 per kg')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /start sale/i })).toBeNull()
  })
})
