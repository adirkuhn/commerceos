// @vitest-environment jsdom
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { describe, it, expect, vi } from 'vitest'
import type { BolPricingLevel, BolPricingPreview } from '../../../integrations/types'
import { BolPricingSection } from '../BolPricingSection'
import { emptyBolPricingForm, type BolPricingForm } from '../bolPricingForm'
import type { BolPricingPreviewState } from '../useBolPricingPreview'

function level(quantity: number, price: number, extra: Partial<BolPricingLevel> = {}): BolPricingLevel {
  return {
    quantity, unit_price_minor_units: price, auto: true, fee_fixed_minor_units: 30, fee_percent_basis_points: 1500,
    product_vat_minor_units: 318, bol_fee_incl_vat_minor_units: 735, shipping_share_minor_units: 495,
    net_minor_units: 2301, regular_net_minor_units: 2294, below_regular: false, profit_minor_units: 1101, ...extra,
  }
}

const preview: BolPricingPreview = {
  ean: '0630509892495', regular_price_minor_units: 2500, cost_price_minor_units: 1200, product_vat_basis_points: 900,
  shipping_minor_units: 495, levels: [level(1, 3849), level(2, 3549, { shipping_share_minor_units: 248 }), level(4, 3149, { below_regular: true, net_minor_units: 2201 })],
}

function Harness({ initial, state, onChange }: { initial: BolPricingForm, state: BolPricingPreviewState, onChange?: (f: BolPricingForm) => void }) {
  const [form, setForm] = useState(initial)
  return <BolPricingSection form={form} onChange={f => { setForm(f); onChange?.(f) }} preview={state} />
}

const ready: BolPricingPreviewState = { preview, loading: false, error: null }

describe('BolPricingSection', () => {
  it('shows the automatic price as the placeholder and a row per price level', () => {
    render(<Harness initial={emptyBolPricingForm()} state={ready} />)

    expect(screen.getByLabelText('Bol price')).toHaveAttribute('placeholder', '38.49')
    const rows = screen.getAllByRole('row').slice(1)
    expect(rows).toHaveLength(3)
    expect(within(rows[0]).getByText('38.49')).toBeInTheDocument()
    expect(within(rows[0]).getByText('7.35')).toBeInTheDocument()
    expect(within(rows[0]).getByText('23.01')).toBeInTheDocument()
    expect(within(rows[0]).getByText('11.01')).toBeInTheDocument()
  })

  it('flags a level that leaves less than the shop', () => {
    render(<Harness initial={emptyBolPricingForm()} state={ready} />)
    const rows = screen.getAllByRole('row').slice(1)
    expect(within(rows[2]).getByText('Less than the shop')).toBeInTheDocument()
    expect(within(rows[0]).queryByText('Less than the shop')).not.toBeInTheDocument()
  })

  it('shows the shop price, VAT and shipping the prices are based on', () => {
    render(<Harness initial={emptyBolPricingForm()} state={ready} />)
    expect(screen.getByText(/Shop price 25\.00 · VAT 9% · Shipping per parcel 4\.95/)).toBeInTheDocument()
  })

  it('typing a custom price reports the change', async () => {
    const onChange = vi.fn()
    render(<Harness initial={emptyBolPricingForm()} state={ready} onChange={onChange} />)
    await userEvent.type(screen.getByLabelText('Bol price'), '30')
    expect(onChange).toHaveBeenLastCalledWith(expect.objectContaining({ customPrice: '30' }))
  })

  it('switching bulk discount on shows the 2/4/6/8 levels to customise', async () => {
    render(<Harness initial={emptyBolPricingForm()} state={ready} />)
    await userEvent.click(screen.getByLabelText('Bulk discount'))

    const quantities = screen.getAllByLabelText(/^From units/)
    expect(quantities.map(i => (i as HTMLInputElement).value)).toEqual(['2', '4', '6', '8'])
    expect(screen.getAllByLabelText(/^Price each/)[0]).toHaveAttribute('placeholder', '35.49')
  })

  it('levels can be edited, added and removed', async () => {
    const onChange = vi.fn()
    const initial = { ...emptyBolPricingForm(), bundleEnabled: true, tiers: [{ quantity: '2', price: '' }, { quantity: '4', price: '' }] }
    render(<Harness initial={initial} state={ready} onChange={onChange} />)

    await userEvent.type(screen.getAllByLabelText(/^Price each/)[0], '25')
    expect(onChange).toHaveBeenLastCalledWith(expect.objectContaining({ tiers: [{ quantity: '2', price: '25' }, { quantity: '4', price: '' }] }))

    await userEvent.click(screen.getByRole('button', { name: 'Add level' }))
    expect(screen.getAllByLabelText(/^From units/)).toHaveLength(3)

    await userEvent.click(screen.getAllByRole('button', { name: 'Remove level' })[0])
    expect(screen.getAllByLabelText(/^From units/).map(i => (i as HTMLInputElement).value)).toEqual(['4', '6'])
  })

  it('shows the server error instead of prices', () => {
    render(<Harness initial={emptyBolPricingForm()} state={{ preview: null, loading: false, error: 'Each Bol bulk price must be at least 5% below the price before it.' }} />)
    expect(screen.getByText('Each Bol bulk price must be at least 5% below the price before it.')).toBeInTheDocument()
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
  })
})
