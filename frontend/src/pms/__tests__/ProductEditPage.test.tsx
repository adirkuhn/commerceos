// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { AppProvider } from '../../context/AppContext'
import { ProductEditPage } from '../ProductEditPage'
import type { Category, Product, ProductDetail, SKU } from '../types'

type SaveBarcode = (skuId: string, barcode: string) => Promise<void>

// Covers the parts of the product editor that Products.test.tsx doesn't
// exercise: case pack linking, the sidebar category controls, and the Bol
// tab's barcode write-back.

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../api', () => ({
  listProducts: vi.fn(() => Promise.resolve({ products: [], total: 0 })),
  getProduct: vi.fn(),
  createVariant: vi.fn(),
  createSKU: vi.fn(),
  updateSKU: vi.fn(),
  getSKUMarginInsight: vi.fn(() => Promise.resolve({
    cost_price_available: false, cost_price_minor_units: 0, cost_price_source: 'computed', effective_target_markup_basis_points: 0,
    recommended_price_minor_units: 0, profit_at_price_minor_units: 0, margin_at_price_basis_points: 0, markup_at_price_basis_points: 0,
    total_on_hand_quantity: 0, total_profit_potential_minor_units: 0, shrinkage_minor_units: 0,
  })),
  listSKUCostPriceHistory: vi.fn(() => Promise.resolve([])),
  listSKUPriceHistory: vi.fn(() => Promise.resolve([])),
  listProductHistory: vi.fn(() => Promise.resolve([])),
  listProductMedia: vi.fn(() => Promise.resolve([])),
  listBarcodeAliases: vi.fn(() => Promise.resolve([])),
  listScaleCodeAliases: vi.fn(() => Promise.resolve([])),
}))

vi.mock('../../inventory/api', () => ({
  getSKUStockLevel: vi.fn((_token: string, skuId: string) => Promise.resolve({ sku_id: skuId, total: 0, syncable: 0 })),
}))

vi.mock('../../integrations/api', () => ({
  listConnections: vi.fn(),
  resyncProductToChannels: vi.fn(),
}))

vi.mock('../../settings/api', () => ({
  getTenantSettings: vi.fn(),
}))

vi.mock('../../tax/api', () => ({
  listTaxClasses: vi.fn(() => Promise.resolve([])),
  listTaxRates: vi.fn(() => Promise.resolve([])),
}))

// The Bol tab's own UI is covered by ProductBolTab.test.tsx -- here only
// the page's onSaveBarcode wiring matters.
vi.mock('../ProductBolTab', () => ({
  ProductBolTab: ({ onSaveBarcode }: { onSaveBarcode: SaveBarcode }) => (
    <button type="button" onClick={() => { void onSaveBarcode('sku-1', '8712345678906') }}>fake bol save barcode</button>
  ),
}))

// The split modal's own UI is covered by SplitIntoPackagesModal.test.tsx --
// here only the card's open/close wiring and the page's reaction matter.
vi.mock('../SplitIntoPackagesModal', () => ({
  SplitIntoPackagesModal: ({ bulkSkuId, onPacked, onClose }: { bulkSkuId: string; onPacked: () => void; onClose: () => void }) => (
    <div>
      <span>split modal for {bulkSkuId}</span>
      <button type="button" onClick={onPacked}>fake pack</button>
      <button type="button" onClick={onClose}>fake close</button>
    </div>
  ),
}))

import * as pmsApi from '../api'
import * as integrationsApi from '../../integrations/api'
import * as settingsApi from '../../settings/api'

const product: Product = {
  id: 'prod-1', tenant_id: 't1', name: 'T-Shirt', slug: 't-shirt',
  description: '', status: 'active', has_variants: false, tags: [], created_at: '', updated_at: '',
}

const sku: SKU = {
  id: 'sku-1', tenant_id: 't1', variant_id: 'var-1', sku_code: 'TSHIRT-1',
  barcode: '012345', price_minor_units: 1999, currency_code: 'USD', is_virtual: false,
  status: 'active', created_at: '', updated_at: '',
}

const category: Category = { id: 'cat-1', tenant_id: 't1', name: 'Apparel', slug: 'apparel', position: 0, created_at: '', updated_at: '' }

function detailWith(skuOverrides: Partial<SKU> = {}): ProductDetail {
  return {
    product,
    variants: [{ id: 'var-1', tenant_id: 't1', product_id: 'prod-1', name: 'T-Shirt', position: 0, created_at: '', updated_at: '', skus: [{ ...sku, ...skuOverrides }] }],
  }
}

function renderPage(overrides: Partial<Parameters<typeof ProductEditPage>[0]> = {}) {
  const props = {
    productId: 'prod-1' as string | null,
    categories: [category],
    onCreateCategory: vi.fn(),
    onUpdateCategory: vi.fn(() => Promise.resolve(category)),
    onCreate: vi.fn(() => Promise.resolve(product)),
    onUpdate: vi.fn(() => Promise.resolve(product)),
    onClose: vi.fn(),
    ...overrides,
  }
  render(<MemoryRouter><AppProvider><ProductEditPage {...props} /></AppProvider></MemoryRouter>)
  return props
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(integrationsApi.listConnections).mockResolvedValue([])
  vi.mocked(settingsApi.getTenantSettings).mockResolvedValue({
    tenant_id: 't1', default_currency: 'EUR', timezone: '', date_format: '', language: '',
    prices_include_tax: true, default_tax_country: '', inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_markup_basis_points: 0, updated_at: '',
  })
  vi.mocked(pmsApi.updateSKU).mockResolvedValue(sku)
})

describe('Case pack card (flat SKU)', () => {
  it('shows an existing case link, and saves an edited units-per-case', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detailWith({ case_of_sku_id: 'sku-unit-1', units_per_case: 12 }))
    renderPage()

    expect(await screen.findByText('sku-unit-1')).toBeInTheDocument()
    const units = screen.getByLabelText('Units/case')
    expect(units).toHaveValue(12)
    await userEvent.clear(units)
    await userEvent.type(units, '24')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(pmsApi.updateSKU).toHaveBeenCalledWith('test-token', 'sku-1', expect.objectContaining({
      case_of_sku_id: 'sku-unit-1', units_per_case: 24,
    })))
  })

  it('names the linked product and its EAN after a reload, never the raw id', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detailWith({
      case_of_sku_id: 'sku-unit-1', units_per_case: 30,
      case_of_sku: { id: 'sku-unit-1', product_name: 'Refresco em pó Uva - MID - Ajinomoto', sku_code: 'REF-UVA', barcode: '7891132082346' },
    }))
    renderPage()

    expect(await screen.findByText('Refresco em pó Uva - MID - Ajinomoto · EAN 7891132082346')).toBeInTheDocument()
    expect(screen.queryByText('sku-unit-1')).not.toBeInTheDocument()
  })

  it('Remove clears the link, sending nulls on save', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detailWith({ case_of_sku_id: 'sku-unit-1', units_per_case: 12 }))
    renderPage()

    await screen.findByText('sku-unit-1')
    await userEvent.click(screen.getByRole('button', { name: 'Remove' }))
    expect(screen.getByRole('button', { name: /This is a case of another product/ })).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(pmsApi.updateSKU).toHaveBeenCalledWith('test-token', 'sku-1', expect.objectContaining({
      case_of_sku_id: null, units_per_case: null,
    })))
  })

  it('opens the picker, and closing it returns to the link button', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detailWith())
    renderPage()

    await userEvent.click(await screen.findByRole('button', { name: /This is a case of another product/ }))
    expect(screen.getByPlaceholderText('Search products by name…')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Close' }))
    expect(screen.getByRole('button', { name: /This is a case of another product/ })).toBeInTheDocument()
  })

  it('picking a SKU shows its product/sku label and sends it on save', async () => {
    vi.mocked(pmsApi.listProducts).mockResolvedValue({ products: [{ ...product, id: 'prod-unit', name: 'Single can' }], total: 1 })
    vi.mocked(pmsApi.getProduct).mockImplementation(async (_t, id) => id === 'prod-unit'
      ? { product: { ...product, id: 'prod-unit', name: 'Single can' }, variants: [{ id: 'var-u', tenant_id: 't1', product_id: 'prod-unit', name: 'x', position: 0, created_at: '', updated_at: '', skus: [{ ...sku, id: 'sku-can', sku_code: 'CAN-1' }] }] }
      : detailWith())
    renderPage()

    await userEvent.click(await screen.findByRole('button', { name: /This is a case of another product/ }))
    await userEvent.click(await screen.findByRole('button', { name: 'Single can' }))
    await userEvent.click(await screen.findByRole('button', { name: 'CAN-1' }))

    expect(screen.getByText('Single can · EAN 012345')).toBeInTheDocument()
    await userEvent.type(screen.getByLabelText('Units/case'), '6')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(pmsApi.updateSKU).toHaveBeenCalledWith('test-token', 'sku-1', expect.objectContaining({
      case_of_sku_id: 'sku-can', units_per_case: 6,
    })))
  })
})

describe('sidebar category controls', () => {
  it('creating a new category selects it for the product', async () => {
    const created = { id: 'cat-2', name: 'Shoes' }
    const props = renderPage({
      productId: null,
      categories: [category, { ...category, id: 'cat-2', name: 'Shoes', slug: 'shoes' }],
      onCreateCategory: vi.fn(() => Promise.resolve(created)),
    })

    await userEvent.type(screen.getByPlaceholderText('+ new category name'), '  Shoes ')
    await userEvent.click(screen.getByRole('button', { name: 'Add' }))

    await waitFor(() => expect(props.onCreateCategory).toHaveBeenCalledWith({ name: 'Shoes' }))
    await waitFor(() => expect(screen.getByLabelText('Category')).toHaveValue('cat-2'))
    expect(screen.getByPlaceholderText('+ new category name')).toHaveValue('')
  })

  it('a blank new category name does nothing', async () => {
    const props = renderPage({ productId: null })
    await userEvent.click(screen.getByRole('button', { name: 'Add' }))
    expect(props.onCreateCategory).not.toHaveBeenCalled()
  })

  it('shows a create-category failure as the form error', async () => {
    renderPage({ productId: null, onCreateCategory: vi.fn(() => Promise.reject(new Error('name taken'))) })
    await userEvent.type(screen.getByPlaceholderText('+ new category name'), 'Shoes')
    await userEvent.click(screen.getByRole('button', { name: 'Add' }))
    expect(await screen.findByText('name taken')).toBeInTheDocument()
  })

  it('saving a category margin updates that category, keeping its name', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ ...detailWith(), product: { ...product, category_id: 'cat-1' } })
    const props = renderPage()

    const margin = await screen.findByPlaceholderText('Tenant default')
    await userEvent.type(margin, '30')
    await userEvent.click(screen.getByText('% target markup for this category').parentElement!.querySelector('button')!)

    await waitFor(() => expect(props.onUpdateCategory).toHaveBeenCalledWith('cat-1', { name: 'Apparel', target_markup_basis_points: 3000 }))
  })
})

describe('Bol tab barcode write-back', () => {
  it('re-sends every current SKU field with only the barcode changed', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([
      { id: 'conn-bol', provider: 'bol', store_url: '', status: 'connected', created_at: '' },
    ])
    const fullSku: Partial<SKU> = {
      sale_price_minor_units: 1500, sale_end_date: '2026-12-31', weight_grams: 200, length_mm: 1, width_mm: 2, height_mm: 3,
      tracks_expiry: true, case_of_sku_id: null, units_per_case: null, unit_type: 'unit', portioned_from_sku_id: null,
      scale_code: null, pricing_mode: 'fixed', price_per_kilo_minor_units: null, tax_class_id: 'tc-1',
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detailWith(fullSku))
    renderPage()

    await userEvent.click(await screen.findByRole('button', { name: /Bol\.com/ }))
    await userEvent.click(screen.getByRole('button', { name: 'fake bol save barcode' }))

    await waitFor(() => expect(pmsApi.updateSKU).toHaveBeenCalledWith('test-token', 'sku-1', {
      sku_code: 'TSHIRT-1', barcode: '8712345678906', price_minor_units: 1999,
      sale_price_minor_units: 1500, sale_end_date: '2026-12-31', currency_code: 'USD',
      weight_grams: 200, length_mm: 1, width_mm: 2, height_mm: 3, is_virtual: false, tracks_expiry: true,
      case_of_sku_id: null, units_per_case: null, unit_type: 'unit', portioned_from_sku_id: null,
      scale_code: null, pricing_mode: 'fixed', price_per_kilo_minor_units: null, tax_class_id: 'tc-1', status: 'active',
    }))
  })
})

describe('new product currency', () => {
  it('seeds the flat SKU currency from the tenant default', async () => {
    renderPage({ productId: null })
    await waitFor(() => expect(screen.getByLabelText('Currency')).toHaveValue('EUR'))
  })
})

describe('Weight pool card: split into packages (flat SKU)', () => {
  const pool: Partial<SKU> = {
    unit_type: 'kilo', pricing_mode: 'per_weight', scale_code: '2994721', price_per_kilo_minor_units: 29990, currency_code: 'EUR',
  }

  it('offers the split for a saved scale-sold weight pool', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detailWith(pool))
    renderPage()

    await userEvent.click(await screen.findByRole('button', { name: 'Split into packages' }))

    expect(screen.getByText('split modal for sku-1')).toBeInTheDocument()
  })

  it('does not offer it for a weight pool with no scale code yet', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detailWith({ ...pool, scale_code: null }))
    renderPage()

    await screen.findByText('Weight pool')
    expect(screen.queryByRole('button', { name: 'Split into packages' })).not.toBeInTheDocument()
  })

  it('after packing, closing the modal reloads the product and switches to its variants', async () => {
    const before = detailWith(pool)
    const after: ProductDetail = {
      product: { ...product, has_variants: true },
      variants: [
        before.variants[0],
        { id: 'var-380', tenant_id: 't1', product_id: 'prod-1', name: '~380g', position: 1, created_at: '', updated_at: '',
          skus: [{ ...sku, id: 'sku-380', variant_id: 'var-380', sku_code: 'TSHIRT-1-380G', portioned_from_sku_id: 'sku-1' }] },
      ],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValueOnce(before).mockResolvedValue(after)
    renderPage()

    await userEvent.click(await screen.findByRole('button', { name: 'Split into packages' }))
    await userEvent.click(screen.getByRole('button', { name: 'fake pack' }))
    expect(screen.getByText('split modal for sku-1')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'fake close' }))

    expect(await screen.findByText('~380g')).toBeInTheDocument()
    await waitFor(() => expect(pmsApi.getProduct).toHaveBeenCalledTimes(2))
  })

  it('closing without packing changes nothing', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detailWith(pool))
    renderPage()

    await userEvent.click(await screen.findByRole('button', { name: 'Split into packages' }))
    await userEvent.click(screen.getByRole('button', { name: 'fake close' }))

    expect(screen.getByRole('button', { name: 'Split into packages' })).toBeInTheDocument()
    expect(pmsApi.getProduct).toHaveBeenCalledTimes(1)
  })
})
