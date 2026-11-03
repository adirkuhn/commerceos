// @vitest-environment jsdom
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { AppProvider } from '../../context/AppContext'
import { VariantRow } from '../VariantRow'
import type { useProductDetail } from '../hooks/useProductDetail'
import type { Product, SKU, VariantWithSKUs } from '../types'
import { acceptConfirm, dismissConfirm } from '../../test/confirm'

// Covers the VariantRow paths Products.test.tsx doesn't: removing a
// variant/SKU, the margin toggle, add-SKU error/reset/weight-pool linking,
// linking an existing SKU to a pool, and per-field save failures.

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../api', () => ({
  listProducts: vi.fn(() => Promise.resolve({ products: [], total: 0 })),
  getProduct: vi.fn(),
  listSKUSupplierCosts: vi.fn(() => Promise.resolve({
    currency_code: 'EUR', last: null, average_cost_minor_units: null, by_supplier: [],
    excluded_other_currency: 0, items: [], next_cursor: null, total: 0,
  })),
  getSKUMarginInsight: vi.fn(() => Promise.resolve({
    cost_price_available: false, cost_price_minor_units: 0, cost_price_source: 'computed', effective_target_markup_basis_points: 0,
    recommended_price_minor_units: 0, profit_at_price_minor_units: 0, margin_at_price_basis_points: 0, markup_at_price_basis_points: 0,
    total_on_hand_quantity: 0, total_profit_potential_minor_units: 0, shrinkage_minor_units: 0,
  })),
  listSKUCostPriceHistory: vi.fn(() => Promise.resolve([])),
  listBarcodeAliases: vi.fn(() => Promise.resolve([])),
  listScaleCodeAliases: vi.fn(() => Promise.resolve([])),
  listSKUCurrencyPrices: vi.fn(() => Promise.resolve([])),
}))

vi.mock('../../inventory/api', () => ({
  getSKUStockLevel: vi.fn(() => Promise.resolve({ sku_id: 'sku-1', total: 7, syncable: 7 })),
}))

vi.mock('../../settings/api', () => ({ getTenantSettings: vi.fn() }))

vi.mock('../../tax/api', () => ({
  listTaxClasses: vi.fn(() => Promise.resolve([])),
  listTaxRates: vi.fn(() => Promise.resolve([])),
}))

import * as pmsApi from '../api'
import * as settingsApi from '../../settings/api'

type Detail = ReturnType<typeof useProductDetail>

const sku: SKU = {
  id: 'sku-1', tenant_id: 't1', variant_id: 'var-1', sku_code: 'TSHIRT-1',
  barcode: '012345', price_minor_units: 1999, currency_code: 'USD', is_virtual: false,
  status: 'active', created_at: '', updated_at: '',
}

function variant(skus: SKU[] = [sku]): VariantWithSKUs {
  return { id: 'var-1', tenant_id: 't1', product_id: 'prod-1', name: 'Large', position: 0, created_at: '', updated_at: '', skus }
}

function fakeDetail(): Detail {
  return {
    detail: null, loading: false, error: null,
    refresh: vi.fn(),
    addVariant: vi.fn(() => Promise.resolve()),
    updateVariant: vi.fn(() => Promise.resolve()),
    removeVariant: vi.fn(() => Promise.resolve()),
    addSKU: vi.fn(() => Promise.resolve()),
    updateSKU: vi.fn(() => Promise.resolve()),
    removeSKU: vi.fn(() => Promise.resolve()),
  }
}

function renderRow(v: VariantWithSKUs, detail: Detail = fakeDetail()) {
  render(<MemoryRouter><AppProvider><VariantRow variant={v} detail={detail} productMedia={[]} /></AppProvider></MemoryRouter>)
  return detail
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(settingsApi.getTenantSettings).mockResolvedValue({
    tenant_id: 't1', default_currency: 'EUR', timezone: '', date_format: '', language: '',
    prices_include_tax: true, default_tax_country: '', inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_markup_basis_points: 0, updated_at: '',
  })
})

describe('VariantRow summary', () => {
  it('shows name, code, price, status and stock', async () => {
    renderRow(variant())
    expect(screen.getByText('Large')).toBeInTheDocument()
    expect(screen.getByText('TSHIRT-1')).toBeInTheDocument()
    expect(screen.getByText('$19.99')).toBeInTheDocument()
    expect(screen.getByText('active')).toBeInTheDocument()
    expect(await screen.findByText('7 in stock')).toBeInTheDocument()
  })

  it('shows the sale price struck-through pricing with its end date', () => {
    renderRow(variant([{ ...sku, sale_price_minor_units: 1500, sale_end_date: '2026-12-31' }]))
    expect(screen.getByText('$19.99').closest('.line-through')).toHaveClass('line-through')
    expect(screen.getByText('$15.00')).toBeInTheDocument()
    expect(screen.getByText('until 2026-12-31')).toBeInTheDocument()
  })

  it('shows "No SKU yet" and an Add SKU button for a variant without a SKU', () => {
    renderRow(variant([]))
    expect(screen.getByText('No SKU yet')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '+ Add SKU' })).toBeInTheDocument()
  })
})

describe('removing', () => {
  it('removes the variant only after confirming', async () => {
    const detail = renderRow(variant())
    await userEvent.click(screen.getByRole('button', { name: 'Remove variant' }))
    await dismissConfirm()
    expect(detail.removeVariant).not.toHaveBeenCalled()

    await userEvent.click(screen.getByRole('button', { name: 'Remove variant' }))
    await acceptConfirm()
    await waitFor(() => expect(detail.removeVariant).toHaveBeenCalledWith('var-1'))
  })

  it('removes the SKU after confirming', async () => {
    const detail = renderRow(variant())
    await userEvent.click(screen.getByRole('button', { name: 'Remove SKU' }))
    await acceptConfirm()
    await waitFor(() => expect(detail.removeSKU).toHaveBeenCalledWith('sku-1'))
  })
})

describe('margin insight toggle', () => {
  it('is collapsed until toggled open', async () => {
    renderRow(variant())
    expect(pmsApi.getSKUMarginInsight).not.toHaveBeenCalled()
    await userEvent.click(screen.getByRole('button', { name: /Cost & margin insight/ }))
    await waitFor(() => expect(pmsApi.getSKUMarginInsight).toHaveBeenCalledWith('test-token', 'sku-1'))
  })
})

describe('per-field save failures', () => {
  it('shows a barcode save error', async () => {
    const detail = fakeDetail()
    vi.mocked(detail.updateSKU).mockRejectedValue(new Error('EAN taken'))
    renderRow(variant(), detail)
    const ean = screen.getByLabelText('EAN')
    await userEvent.clear(ean)
    await userEvent.type(ean, '999')
    await userEvent.click(within(ean.closest('div')!).getByRole('button', { name: 'Save' }))
    expect(await screen.findByText('EAN taken')).toBeInTheDocument()
  })

  it('shows a weight-pool save error', async () => {
    const detail = fakeDetail()
    vi.mocked(detail.updateSKU).mockRejectedValue(new Error('pool nope'))
    renderRow(variant(), detail)
    await userEvent.click(screen.getByRole('button', { name: 'Make weight pool' }))
    expect(await screen.findByText('pool nope')).toBeInTheDocument()
  })
})

describe('linking an existing SKU to a weight pool', () => {
  it('opens the pool picker, and picking a kilo SKU saves the link', async () => {
    const pool: Product = { id: 'prod-pool', tenant_id: 't1', name: 'Beef bulk', slug: 'b', description: '', status: 'active', has_variants: false, tags: [], created_at: '', updated_at: '' }
    vi.mocked(pmsApi.listProducts).mockResolvedValue({ products: [pool], total: 1 })
    vi.mocked(pmsApi.getProduct).mockResolvedValue({
      product: pool,
      variants: [{ ...variant(), id: 'var-pool', skus: [{ ...sku, id: 'sku-pool', sku_code: 'POOL-1', unit_type: 'kilo' }, { ...sku, id: 'sku-other', sku_code: 'OTHER' }] }],
    })
    const detail = renderRow(variant())

    await userEvent.click(screen.getByRole('button', { name: 'Link to weight pool' }))
    expect(screen.queryByRole('button', { name: 'Make weight pool' })).not.toBeInTheDocument()
    await userEvent.click(await screen.findByRole('button', { name: 'Beef bulk' }))
    expect(await screen.findByRole('button', { name: 'POOL-1' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'OTHER' })).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'POOL-1' }))

    await waitFor(() => expect(detail.updateSKU).toHaveBeenCalledWith('sku-1', expect.objectContaining({
      unit_type: 'unit', portioned_from_sku_id: 'sku-pool', sku_code: 'TSHIRT-1', barcode: '012345',
    })))
    await waitFor(() => expect(screen.queryByPlaceholderText('Search products by name…')).not.toBeInTheDocument())
  })
})

describe('adding a SKU', () => {
  async function openAddForm() {
    await userEvent.click(screen.getByRole('button', { name: '+ Add SKU' }))
    return screen.getByPlaceholderText('SKU code').closest('div')!.parentElement!.parentElement!
  }

  it('sends the full add payload, closes, and resets the form to the tenant currency', async () => {
    const detail = renderRow(variant([]))
    const form = await openAddForm()
    await waitFor(() => expect(within(form).getByDisplayValue('EUR')).toBeInTheDocument())
    await userEvent.type(within(form).getByPlaceholderText('SKU code'), 'L-1')
    await userEvent.type(within(form).getByPlaceholderText('0.00'), '12.5')
    await userEvent.type(within(form).getAllByPlaceholderText('0')[0], '300')
    await userEvent.click(within(form).getByRole('button', { name: 'Add' }))

    await waitFor(() => expect(detail.addSKU).toHaveBeenCalledWith('var-1', {
      sku_code: 'L-1', barcode: undefined, price_minor_units: 1250, sale_price_minor_units: null, sale_end_date: null,
      currency_code: 'EUR', is_virtual: false, weight_grams: 300, length_mm: undefined, width_mm: undefined, height_mm: undefined,
      unit_type: 'unit', portioned_from_sku_id: null, scale_code: null, pricing_mode: 'fixed', price_per_kilo_minor_units: null,
      status: 'active',
    }))
    expect(vi.mocked(detail.addSKU).mock.calls[0][1]).not.toHaveProperty('tax_class_id')
    await waitFor(() => expect(screen.queryByPlaceholderText('SKU code')).not.toBeInTheDocument())

    const reopened = await openAddForm()
    expect(within(reopened).getByPlaceholderText('SKU code')).toHaveValue('')
  })

  it('keeps typed values when cancelled and reopened', async () => {
    renderRow(variant([]))
    const form = await openAddForm()
    await userEvent.type(within(form).getByPlaceholderText('SKU code'), 'KEEP-1')
    await userEvent.click(within(form).getByRole('button', { name: 'Cancel' }))
    expect(screen.queryByPlaceholderText('SKU code')).not.toBeInTheDocument()

    await openAddForm()
    expect(screen.getByPlaceholderText('SKU code')).toHaveValue('KEEP-1')
  })

  it('shows an add failure and stays open', async () => {
    const detail = fakeDetail()
    vi.mocked(detail.addSKU).mockRejectedValue(new Error('code taken'))
    renderRow(variant([]), detail)
    const form = await openAddForm()
    await userEvent.click(within(form).getByRole('button', { name: 'Add' }))
    expect(await screen.findByText('code taken')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('SKU code')).toBeInTheDocument()
  })

  it('hides dimensions for a virtual SKU and sends none', async () => {
    const detail = renderRow(variant([]))
    const form = await openAddForm()
    await userEvent.click(within(form).getByRole('checkbox'))
    expect(within(form).queryAllByPlaceholderText('0')).toHaveLength(0)
    await userEvent.click(within(form).getByRole('button', { name: 'Add' }))
    await waitFor(() => expect(detail.addSKU).toHaveBeenCalledWith('var-1', expect.objectContaining({ is_virtual: true, weight_grams: undefined })))
  })

  it('per-kilo pricing shows the price/kilo field and a scale-code warning, sending the kilo price', async () => {
    const detail = renderRow(variant([]))
    const form = await openAddForm()
    await userEvent.selectOptions(within(form).getByLabelText('Pricing'), 'per_weight')
    expect(within(form).getByText(/Set a scale code above too/)).toBeInTheDocument()
    await userEvent.type(within(form).getByPlaceholderText('Not sold via a scale barcode'), '29a00633')
    expect(within(form).queryByText(/Set a scale code above too/)).not.toBeInTheDocument()
    await userEvent.type(within(form).getByText(/Price\/kilo/).parentElement!.querySelector('input')!, '29.99')
    await userEvent.click(within(form).getByRole('button', { name: 'Add' }))
    await waitFor(() => expect(detail.addSKU).toHaveBeenCalledWith('var-1', expect.objectContaining({
      scale_code: '2900633', pricing_mode: 'per_weight', price_per_kilo_minor_units: 2999,
    })))
  })

  it('can link the new SKU to a weight pool, and unlink it again', async () => {
    const pool: Product = { id: 'prod-pool', tenant_id: 't1', name: 'Beef bulk', slug: 'b', description: '', status: 'active', has_variants: false, tags: [], created_at: '', updated_at: '' }
    vi.mocked(pmsApi.listProducts).mockResolvedValue({ products: [pool], total: 1 })
    vi.mocked(pmsApi.getProduct).mockResolvedValue({
      product: pool, variants: [{ ...variant(), id: 'var-pool', skus: [{ ...sku, id: 'sku-pool', sku_code: 'POOL-1', unit_type: 'kilo' }] }],
    })
    const detail = renderRow(variant([]))
    const form = await openAddForm()

    await userEvent.click(within(form).getByRole('button', { name: '+ Link to a weight pool' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Beef bulk' }))
    await userEvent.click(await screen.findByRole('button', { name: 'POOL-1' }))
    expect(within(form).getByText('Beef bulk · EAN 012345')).toBeInTheDocument()

    await userEvent.click(within(form).getByRole('button', { name: 'Unlink' }))
    expect(within(form).getByRole('button', { name: '+ Link to a weight pool' })).toBeInTheDocument()

    await userEvent.click(within(form).getByRole('button', { name: '+ Link to a weight pool' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Beef bulk' }))
    await userEvent.click(await screen.findByRole('button', { name: 'POOL-1' }))
    await userEvent.click(within(form).getByRole('button', { name: 'Add' }))
    await waitFor(() => expect(detail.addSKU).toHaveBeenCalledWith('var-1', expect.objectContaining({ portioned_from_sku_id: 'sku-pool', unit_type: 'unit' })))
  })
})

describe('editing the variant', () => {
  it('re-clicking Edit while open discards unsaved edits', async () => {
    renderRow(variant())
    await userEvent.click(screen.getByRole('button', { name: 'Edit variant' }))
    const name = screen.getByPlaceholderText('Variant name')
    await userEvent.clear(name)
    await userEvent.type(name, 'Changed')
    await userEvent.click(screen.getByRole('button', { name: 'Edit variant' }))
    expect(screen.getByPlaceholderText('Variant name')).toHaveValue('Large')
  })

  it('a virtual toggle in the edit panel hides dimensions and sends none', async () => {
    const detail = renderRow(variant([{ ...sku, weight_grams: 100 }]))
    await userEvent.click(screen.getByRole('button', { name: 'Edit variant' }))
    const panel = screen.getByPlaceholderText('Variant name').closest('div')!.parentElement!
    await userEvent.click(within(panel).getByRole('checkbox'))
    await userEvent.click(within(panel).getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(detail.updateSKU).toHaveBeenCalledWith('sku-1', expect.objectContaining({ is_virtual: true, weight_grams: undefined })))
    expect(vi.mocked(detail.updateSKU).mock.calls[0][1]).not.toHaveProperty('tax_class_id')
  })

  it('Save is disabled with an empty name', async () => {
    renderRow(variant())
    await userEvent.click(screen.getByRole('button', { name: 'Edit variant' }))
    const panel = screen.getByPlaceholderText('Variant name').closest('div')!.parentElement!
    await userEvent.clear(screen.getByPlaceholderText('Variant name'))
    expect(within(panel).getByRole('button', { name: 'Save' })).toBeDisabled()
  })
})
