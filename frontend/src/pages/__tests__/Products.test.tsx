// @vitest-environment jsdom
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import Products from '../Products'
import { AppProvider, useApp } from '../../context/AppContext'
import type { Product, ProductDetail, Category } from '../../pms/types'
import type { Connection } from '../../integrations/types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../pms/api', () => ({
  listProducts: vi.fn(),
  createProduct: vi.fn(),
  updateProduct: vi.fn(),
  deleteProduct: vi.fn(),
  listCategories: vi.fn(),
  createCategory: vi.fn(),
  updateCategory: vi.fn(),
  getProduct: vi.fn(),
  createVariant: vi.fn(),
  updateVariant: vi.fn(),
  deleteVariant: vi.fn(),
  createSKU: vi.fn(),
  updateSKU: vi.fn(),
  deleteSKU: vi.fn(),
  getSKUMarginInsight: vi.fn(() => Promise.resolve({
    cost_price_available: false, cost_price_minor_units: 0, cost_price_source: 'computed', effective_target_margin_basis_points: 0,
    recommended_price_minor_units: 0, profit_at_price_minor_units: 0, margin_at_price_basis_points: 0,
    total_on_hand_quantity: 0, total_profit_potential_minor_units: 0, shrinkage_minor_units: 0,
  })),
  setManualCostPrice: vi.fn(),
  clearCostPriceOverride: vi.fn(),
  listSKUCostPriceHistory: vi.fn(),
  listMarginReport: vi.fn(),
  listProductMedia: vi.fn(),
  uploadProductMedia: vi.fn(),
  updateMedia: vi.fn(),
  deleteMedia: vi.fn(),
}))

vi.mock('../../integrations/api', () => ({
  listConnections: vi.fn(),
  getProductChannelListing: vi.fn(),
}))

vi.mock('../../settings/api', () => ({
  getTenantSettings: vi.fn(),
  updateTenantSettings: vi.fn(),
}))

vi.mock('../../tax/api', () => ({
  listTaxClasses: vi.fn(),
  createTaxClass: vi.fn(),
  updateTaxClass: vi.fn(),
  deleteTaxClass: vi.fn(),
  listTaxRates: vi.fn(),
  upsertTaxRate: vi.fn(),
  deleteTaxRate: vi.fn(),
}))

import * as pmsApi from '../../pms/api'
import * as integrationsApi from '../../integrations/api'
import * as tenantSettingsApi from '../../settings/api'
import * as taxApi from '../../tax/api'

function product(overrides: Partial<Product> = {}): Product {
  return {
    id: 'prod-1', tenant_id: 't1', name: 'T-Shirt', slug: 't-shirt',
    description: '', status: 'active', has_variants: false, tags: [], created_at: '', updated_at: '',
    ...overrides,
  }
}

function category(overrides: Partial<Category> = {}): Category {
  return { id: 'cat-1', tenant_id: 't1', name: 'Apparel', slug: 'apparel', position: 0, created_at: '', updated_at: '', ...overrides }
}

const baseVariant = {
  id: 'var-1', tenant_id: 't1', product_id: 'prod-1', name: 'T-Shirt', position: 0, created_at: '', updated_at: '',
}

const baseSku = {
  id: 'sku-1', tenant_id: 't1', variant_id: 'var-1', sku_code: 'TSHIRT-1',
  barcode: '012345', price_minor_units: 1999, currency_code: 'USD', is_virtual: false,
  status: 'active' as const, created_at: '', updated_at: '',
}

function wooConnection(overrides: Partial<Connection> = {}): Connection {
  return { id: 'conn-1', provider: 'woocommerce', store_url: 'https://mystore.com', status: 'connected', created_at: '', ...overrides }
}

// Simulates the server: paginates and searches `all` according to the
// params the hook actually sent, so tests exercise real request params
// instead of asserting against a client-side .filter()/.slice() that no
// longer exists in Products.tsx.
function mockPaginatedProducts(all: Product[]) {
  vi.mocked(pmsApi.listProducts).mockImplementation(async (_token, params = {}) => {
    const page = params.page ?? 1
    const perPage = params.perPage ?? 20
    const search = (params.search ?? '').toLowerCase()
    const filtered = search
      ? all.filter(p => p.name.toLowerCase().includes(search) || p.slug.toLowerCase().includes(search))
      : all
    return { products: filtered.slice((page - 1) * perPage, page * perPage), total: filtered.length }
  })
}

beforeEach(() => {
  vi.clearAllMocks()
  mockPaginatedProducts([product()])
  vi.mocked(pmsApi.listCategories).mockResolvedValue([category()])
  vi.mocked(integrationsApi.listConnections).mockResolvedValue([])
  vi.mocked(integrationsApi.getProductChannelListing).mockResolvedValue(null)
  vi.mocked(pmsApi.listProductMedia).mockResolvedValue([])
  vi.mocked(tenantSettingsApi.getTenantSettings).mockResolvedValue({
    tenant_id: 't1', default_currency: 'USD', timezone: '', date_format: '', language: '',
    prices_include_tax: true, default_tax_country: '', inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_margin_basis_points: 0, updated_at: '',
  })
  vi.mocked(taxApi.listTaxClasses).mockResolvedValue([])
  vi.mocked(taxApi.listTaxRates).mockResolvedValue([])
})

function NavReselectButton() {
  const { setActiveNav } = useApp()
  return <button onClick={() => setActiveNav('products')}>Reselect Products</button>
}

describe('re-clicking the active "Products" nav item', () => {
  // Regression: Sidebar sets activeNav to the same value it already has when
  // re-clicking the current page, so React bails out of re-rendering and
  // Products never remounts — its internal editingId (list vs. detail view)
  // must reset some other way, or the detail view stays stuck open.
  it('returns to the list view even though activeNav does not change', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [] })

    render(
      <MemoryRouter initialEntries={['/products']}>
        <AppProvider>
          <NavReselectButton />
          <Products />
        </AppProvider>
      </MemoryRouter>,
    )
    await userEvent.click(await screen.findByText('T-Shirt'))
    expect(await screen.findByDisplayValue('T-Shirt')).toBeInTheDocument()

    await userEvent.click(screen.getByText('Reselect Products'))

    expect(screen.queryByDisplayValue('T-Shirt')).not.toBeInTheDocument()
    expect(await screen.findByText('T-Shirt')).toBeInTheDocument()
  })
})

describe('product list', () => {
  it('opens the edit page when clicking anywhere on a row, not just the Edit button', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [] })

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByText('T-Shirt'))

    await waitFor(() => expect(pmsApi.getProduct).toHaveBeenCalledWith('test-token', 'prod-1'))
    expect(screen.getByDisplayValue('T-Shirt')).toBeInTheDocument()
  })

  it('deleting a row does not also navigate to the edit page', async () => {
    vi.mocked(pmsApi.deleteProduct).mockResolvedValue(null)
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Delete'))

    await waitFor(() => expect(pmsApi.deleteProduct).toHaveBeenCalledWith('test-token', 'prod-1'))
    expect(pmsApi.getProduct).not.toHaveBeenCalled()

    confirmSpy.mockRestore()
  })

  it('paginates when there are more than 20 products', async () => {
    const many = Array.from({ length: 25 }, (_, i) => product({ id: `prod-${i}`, name: `Product ${String(i).padStart(2, '0')}`, slug: `product-${i}` }))
    mockPaginatedProducts(many)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await screen.findByText('Product 00')

    expect(screen.queryByText('Product 20')).not.toBeInTheDocument()
    expect(screen.getByText('Page 1 of 2')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Next page' }))

    expect(await screen.findByText('Product 20')).toBeInTheDocument()
    expect(screen.queryByText('Product 00')).not.toBeInTheDocument()
  })

  it('resets to page 1 when the search filter changes', async () => {
    const many = Array.from({ length: 25 }, (_, i) => product({ id: `prod-${i}`, name: `Product ${String(i).padStart(2, '0')}`, slug: `product-${i}` }))
    mockPaginatedProducts(many)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await screen.findByText('Product 00')
    await userEvent.click(screen.getByRole('button', { name: 'Next page' }))
    await screen.findByText('Product 20')

    await userEvent.type(screen.getByPlaceholderText('Search by name or slug...'), 'Product 01')

    expect(await screen.findByText('Product 01')).toBeInTheDocument()
  })

  it('sends the search term to the server (debounced), not a client-side filter', async () => {
    const many = [product({ id: 'p1', name: 'T-Shirt', slug: 't-shirt' }), product({ id: 'p2', name: 'Mug', slug: 'mug' })]
    mockPaginatedProducts(many)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await screen.findByText('T-Shirt')
    await screen.findByText('Mug')

    await userEvent.type(screen.getByPlaceholderText('Search by name or slug...'), 'mug')

    await waitFor(() => expect(pmsApi.listProducts).toHaveBeenCalledWith('test-token', expect.objectContaining({ search: 'mug' })))
    expect(await screen.findByText('Mug')).toBeInTheDocument()
    expect(screen.queryByText('T-Shirt')).not.toBeInTheDocument()
  })
})

describe('editing an existing product infers variants mode from variant count', () => {
  it('shows the flat Pricing & SKU form for a product with 0 variants (not the variants manager)', async () => {
    const detail: ProductDetail = { product: product(), variants: [] }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))

    await waitFor(() => expect(screen.getByText('Pricing & SKU')).toBeInTheDocument())
    expect(screen.getByRole('button', { name: '+ Add variants like size or color' })).toBeInTheDocument()
  })

  it('shows the flat Pricing & SKU form, prefilled, for a product with exactly 1 variant', async () => {
    const detail: ProductDetail = {
      product: product(),
      variants: [{ ...baseVariant, skus: [baseSku] }],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))

    await waitFor(() => expect(screen.getByDisplayValue('TSHIRT-1')).toBeInTheDocument())
    expect(screen.getByDisplayValue('19.99')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '+ Add variants like size or color' })).toBeInTheDocument()
  })

  it('shows the full variants manager (not the flat form) for a product with 2+ variants', async () => {
    const detail: ProductDetail = {
      product: product({ has_variants: true }),
      variants: [
        { ...baseVariant, id: 'var-1', name: 'Small', skus: [] },
        { ...baseVariant, id: 'var-2', name: 'Large', skus: [] },
      ],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))

    await waitFor(() => expect(screen.getByText('Small')).toBeInTheDocument())
    expect(screen.getByText('Large')).toBeInTheDocument()
    expect(screen.queryByText('Pricing & SKU')).not.toBeInTheDocument()

    // Regression: the Variants card title and the variant-list's own
    // "+ Add Variant" trigger must each render exactly once — they were
    // briefly duplicated when VariantsSection wrapped itself in a second
    // bordered card with its own "Variants" heading, nested inside the
    // outer FormCard that already provides one.
    expect(screen.getAllByText('Variants')).toHaveLength(1)
    expect(screen.getAllByRole('button', { name: '+ Add Variant' })).toHaveLength(1)
  })
})

describe('Add Variant / Add SKU are not nested <form> elements', () => {
  // Regression: these inline "forms" render inside the page's outer
  // <form id="product-general-form">. Nested <form> is invalid HTML — a
  // real browser normalizes it away, so the inner submit button ends up
  // submitting the OUTER form as a genuine (non-JS-intercepted) navigation,
  // reloading the whole app back to its initial route. jsdom doesn't
  // perform real navigation on submit, so this bug was invisible to every
  // other test here — assert directly on DOM shape instead.
  it('never renders more than one <form> element even with both panels open', async () => {
    const detail: ProductDetail = {
      product: product({ has_variants: true }),
      variants: [
        { ...baseVariant, id: 'var-1', name: 'Small', skus: [] },
        { ...baseVariant, id: 'var-2', name: 'Large', skus: [] },
      ],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)

    const { container } = render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await screen.findByText('Small')

    await userEvent.click(screen.getByRole('button', { name: '+ Add Variant' }))
    await userEvent.click(screen.getAllByRole('button', { name: '+ Add SKU' })[0])

    expect(container.querySelectorAll('form')).toHaveLength(1)
  })

  it('clicking "Add Variant" calls the API via a plain button click, not form submit', async () => {
    const detail: ProductDetail = {
      product: product({ has_variants: true }),
      variants: [{ ...baseVariant, id: 'var-1', name: 'Small', skus: [] }],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
    vi.mocked(pmsApi.createVariant).mockResolvedValue({ ...baseVariant, id: 'var-2', name: 'Large' })

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await screen.findByText('Small')

    await userEvent.click(screen.getByRole('button', { name: '+ Add Variant' }))
    await userEvent.type(screen.getByPlaceholderText('Variant name (e.g. Large)'), 'Large')
    await userEvent.click(screen.getByRole('button', { name: 'Add Variant' }))

    await waitFor(() => expect(pmsApi.createVariant).toHaveBeenCalledWith('test-token', 'prod-1', { name: 'Large' }))
  })
})

describe('enabling variants mode on an already-saved product', () => {
  it('persists has_variants:true before letting the user add a variant', async () => {
    const detail: ProductDetail = { product: product(), variants: [{ ...baseVariant, skus: [baseSku] }] }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
    vi.mocked(pmsApi.updateProduct).mockResolvedValue(product({ has_variants: true }))

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await screen.findByText('Pricing & SKU')

    await userEvent.click(screen.getByRole('button', { name: '+ Add variants like size or color' }))

    await waitFor(() => expect(pmsApi.updateProduct).toHaveBeenCalledWith('test-token', 'prod-1', expect.objectContaining({
      has_variants: true,
    })))
    expect(await screen.findByRole('button', { name: '+ Add Variant' })).toBeInTheDocument()
  })

  it('shows an error and stays in flat mode if persisting has_variants fails', async () => {
    const detail: ProductDetail = { product: product(), variants: [{ ...baseVariant, skus: [baseSku] }] }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
    vi.mocked(pmsApi.updateProduct).mockRejectedValue(new Error('server exploded'))

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await screen.findByText('Pricing & SKU')

    await userEvent.click(screen.getByRole('button', { name: '+ Add variants like size or color' }))

    expect(await screen.findByText('server exploded')).toBeInTheDocument()
    expect(screen.getByText('Pricing & SKU')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '+ Add Variant' })).not.toBeInTheDocument()
  })
})

describe('creating a new product', () => {
  it('with variants off, creates product + a default variant + its SKU in one submit', async () => {
    const created = product({ id: 'new-prod' })
    vi.mocked(pmsApi.createProduct).mockResolvedValue(created)
    vi.mocked(pmsApi.createVariant).mockResolvedValue({ ...baseVariant, id: 'new-var', product_id: 'new-prod' })
    vi.mocked(pmsApi.createSKU).mockResolvedValue({ ...baseSku, id: 'new-sku', variant_id: 'new-var' })
    vi.mocked(pmsApi.getProduct).mockResolvedValue({
      product: created,
      variants: [{ ...baseVariant, id: 'new-var', product_id: 'new-prod', skus: [{ ...baseSku, id: 'new-sku', variant_id: 'new-var' }] }],
    })

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByText('New Product'))

    await userEvent.type(screen.getByLabelText('Name'), 'New Shirt')
    await userEvent.type(screen.getByPlaceholderText('e.g. SHIRT-BLK-M'), 'NEWSHIRT-1')
    await userEvent.type(screen.getByPlaceholderText('0.00'), '29.99')

    await userEvent.click(screen.getByRole('button', { name: 'Create Product' }))

    await waitFor(() => expect(pmsApi.createProduct).toHaveBeenCalled())
    expect(pmsApi.createProduct).toHaveBeenCalledWith('test-token', expect.objectContaining({ has_variants: false }))
    expect(pmsApi.createVariant).toHaveBeenCalledWith('test-token', 'new-prod', { name: created.name })
    expect(pmsApi.createSKU).toHaveBeenCalledWith('test-token', 'new-var', expect.objectContaining({
      sku_code: 'NEWSHIRT-1',
      price_minor_units: 2999,
    }))
  })

  it('with variants toggled on, creates only the product (no default variant/SKU)', async () => {
    const created = product({ id: 'new-prod-2', has_variants: true })
    vi.mocked(pmsApi.createProduct).mockResolvedValue(created)
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: created, variants: [] })

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByText('New Product'))

    await userEvent.type(screen.getByLabelText('Name'), 'Configurable Shirt')

    await userEvent.click(screen.getByRole('button', { name: '+ Add variants like size or color' }))
    await userEvent.click(screen.getByRole('button', { name: 'Create Product' }))

    await waitFor(() => expect(pmsApi.createProduct).toHaveBeenCalled())
    expect(pmsApi.createProduct).toHaveBeenCalledWith('test-token', expect.objectContaining({ has_variants: true }))
    expect(pmsApi.createVariant).not.toHaveBeenCalled()
    expect(pmsApi.createSKU).not.toHaveBeenCalled()

    // Regression: right after creating with variants on, the product has 0
    // variants (none added yet) — the reload this triggers must not flip
    // variantsMode back off and silently show the flat form instead.
    await waitFor(() => expect(screen.getByText('+ Add Variant')).toBeInTheDocument())
    expect(screen.queryByText('Pricing & SKU')).not.toBeInTheDocument()
  })
})

describe('Tags', () => {
  it('adds a tag on Enter and removes it via its remove button', async () => {
    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByText('New Product'))

    const tagsInput = screen.getByLabelText('Tags')
    await userEvent.type(tagsInput, 'summer{Enter}')
    expect(screen.getByText('summer')).toBeInTheDocument()
    expect(tagsInput).toHaveValue('')

    await userEvent.click(screen.getByLabelText('Remove tag summer'))
    expect(screen.queryByText('summer')).not.toBeInTheDocument()
  })

  it('adds a tag on comma and ignores a duplicate', async () => {
    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByText('New Product'))

    const tagsInput = screen.getByLabelText('Tags')
    await userEvent.type(tagsInput, 'sale,sale,')
    expect(screen.getAllByText('sale')).toHaveLength(1)
  })

  it('sends the entered tags on create', async () => {
    const created = product({ id: 'new-prod-3' })
    vi.mocked(pmsApi.createProduct).mockResolvedValue(created)
    vi.mocked(pmsApi.createVariant).mockResolvedValue({ ...baseVariant, id: 'new-var-3', product_id: 'new-prod-3' })
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: created, variants: [] })

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByText('New Product'))

    await userEvent.type(screen.getByLabelText('Name'), 'Tagged Shirt')
    await userEvent.type(screen.getByPlaceholderText('e.g. SHIRT-BLK-M'), 'TAGGEDSHIRT-1')
    await userEvent.type(screen.getByPlaceholderText('0.00'), '29.99')
    await userEvent.type(screen.getByLabelText('Tags'), 'summer{Enter}sale{Enter}')
    await userEvent.click(screen.getByRole('button', { name: 'Create Product' }))

    await waitFor(() => expect(pmsApi.createProduct).toHaveBeenCalled())
    expect(pmsApi.createProduct).toHaveBeenCalledWith('test-token', expect.objectContaining({ tags: ['summer', 'sale'] }))
  })

  it('prefills existing tags when editing a product', async () => {
    const detail: ProductDetail = { product: product({ tags: ['summer', 'sale'] }), variants: [] }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))

    expect(await screen.findByText('summer')).toBeInTheDocument()
    expect(screen.getByText('sale')).toBeInTheDocument()
  })
})

describe('sync variants as WooCommerce variations toggle', () => {
  const toggleLabel = /Sync variants as WooCommerce variations/

  it('is not shown when no WooCommerce store is connected, even with variants on', async () => {
    const detail: ProductDetail = {
      product: product({ has_variants: true }),
      variants: [
        { ...baseVariant, id: 'var-1', name: 'Small', skus: [] },
        { ...baseVariant, id: 'var-2', name: 'Large', skus: [] },
      ],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([])

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))

    await waitFor(() => expect(screen.getByText('Small')).toBeInTheDocument())
    expect(screen.queryByText(toggleLabel)).not.toBeInTheDocument()
  })

  it('is not shown while variants are off, even with WooCommerce connected', async () => {
    const detail: ProductDetail = { product: product(), variants: [] }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))

    await waitFor(() => expect(screen.getByText('Pricing & SKU')).toBeInTheDocument())
    expect(screen.queryByText(toggleLabel)).not.toBeInTheDocument()
  })

  it('is prefilled from the product and sends it back unchanged on save', async () => {
    const detail: ProductDetail = {
      product: product({ has_variants: true, sync_variants_as_woo_variations: true }),
      variants: [
        { ...baseVariant, id: 'var-1', name: 'Small', skus: [] },
        { ...baseVariant, id: 'var-2', name: 'Large', skus: [] },
      ],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    vi.mocked(pmsApi.updateProduct).mockResolvedValue(detail.product)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await waitFor(() => expect(screen.getByText('Small')).toBeInTheDocument())

    const checkbox = screen.getByRole('checkbox', { name: toggleLabel })
    expect(checkbox).toBeChecked()

    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(pmsApi.updateProduct).toHaveBeenCalledWith('test-token', 'prod-1', expect.objectContaining({
      sync_variants_as_woo_variations: true,
    })))
  })

  it('toggling it on and saving sends true', async () => {
    const detail: ProductDetail = {
      product: product({ has_variants: true, sync_variants_as_woo_variations: false }),
      variants: [
        { ...baseVariant, id: 'var-1', name: 'Small', skus: [] },
        { ...baseVariant, id: 'var-2', name: 'Large', skus: [] },
      ],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    vi.mocked(pmsApi.updateProduct).mockResolvedValue(detail.product)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await waitFor(() => expect(screen.getByText('Small')).toBeInTheDocument())

    const checkbox = screen.getByRole('checkbox', { name: toggleLabel })
    expect(checkbox).not.toBeChecked()
    await userEvent.click(checkbox)
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(pmsApi.updateProduct).toHaveBeenCalledWith('test-token', 'prod-1', expect.objectContaining({
      sync_variants_as_woo_variations: true,
    })))
  })
})

describe('weight pool for a variant-mode product (VariantRow)', () => {
  it('picking "Kilo" item type while adding a variant SKU sends unit_type:kilo', async () => {
    const detail: ProductDetail = {
      product: product({ has_variants: true }),
      variants: [{ ...baseVariant, id: 'var-1', name: 'Bulk', skus: [] }],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
    vi.mocked(pmsApi.createSKU).mockResolvedValue({ ...baseSku, id: 'new-sku' })

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await screen.findByText('Bulk')

    await userEvent.click(screen.getByRole('button', { name: '+ Add SKU' }))
    const skuForm = screen.getByPlaceholderText('SKU code').closest('div')!.parentElement!
    await userEvent.type(within(skuForm).getByPlaceholderText('SKU code'), 'BULK-1')
    await userEvent.type(within(skuForm).getByPlaceholderText('Price'), '0')
    await userEvent.selectOptions(within(skuForm).getByLabelText(/item type/i), 'kilo')
    await userEvent.click(within(skuForm).getByRole('button', { name: 'Add' }))

    await waitFor(() => expect(pmsApi.createSKU).toHaveBeenCalledWith('test-token', 'var-1', expect.objectContaining({
      unit_type: 'kilo', portioned_from_sku_id: null,
    })))
  })

  it('an existing variant SKU with no weight-pool link shows "Make weight pool" and "Link to weight pool", and clicking Make weight pool saves immediately', async () => {
    const detail: ProductDetail = {
      product: product({ has_variants: true }),
      variants: [{ ...baseVariant, id: 'var-1', name: '378g', skus: [baseSku] }],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
    vi.mocked(pmsApi.updateSKU).mockResolvedValue(baseSku)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await screen.findByText('378g')

    await userEvent.click(screen.getByRole('button', { name: 'Make weight pool' }))

    await waitFor(() => expect(pmsApi.updateSKU).toHaveBeenCalledWith('test-token', 'sku-1', expect.objectContaining({
      unit_type: 'kilo', portioned_from_sku_id: null,
    })))
  })

  it('an existing variant SKU already unit_type kilo shows a Turn off control', async () => {
    const bulkSku = { ...baseSku, unit_type: 'kilo' as const }
    const detail: ProductDetail = {
      product: product({ has_variants: true }),
      variants: [{ ...baseVariant, id: 'var-1', name: 'Bulk', skus: [bulkSku] }],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
    vi.mocked(pmsApi.updateSKU).mockResolvedValue(bulkSku)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await screen.findByText('Bulk')

    expect(screen.getByText('Weight pool')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Turn off' }))

    await waitFor(() => expect(pmsApi.updateSKU).toHaveBeenCalledWith('test-token', 'sku-1', expect.objectContaining({
      unit_type: 'unit', portioned_from_sku_id: null,
    })))
  })

  it('an existing variant SKU already portioned from a weight pool shows its link and an Unlink control', async () => {
    const portionedSku = { ...baseSku, portioned_from_sku_id: 'sku-bulk-1' }
    const detail: ProductDetail = {
      product: product({ has_variants: true }),
      variants: [{ ...baseVariant, id: 'var-1', name: '378g', skus: [portionedSku] }],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
    vi.mocked(pmsApi.updateSKU).mockResolvedValue(portionedSku)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await screen.findByText('378g')

    expect(screen.getByText('sku-bulk-1')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Unlink' }))

    await waitFor(() => expect(pmsApi.updateSKU).toHaveBeenCalledWith('test-token', 'sku-1', expect.objectContaining({
      unit_type: 'unit', portioned_from_sku_id: null,
    })))
  })
})

describe('weight pool (unit_type / portioned_from_sku_id)', () => {
  it('is not shown for a brand-new, not-yet-saved product', async () => {
    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByText('New Product'))

    expect(screen.queryByText('Weight pool')).not.toBeInTheDocument()
  })

  it('picking "Kilo" item type and saving sends unit_type:kilo', async () => {
    const detail: ProductDetail = { product: product(), variants: [{ ...baseVariant, skus: [baseSku] }] }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
    vi.mocked(pmsApi.updateSKU).mockResolvedValue(baseSku)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await screen.findByText('Weight pool')

    await userEvent.selectOptions(screen.getByLabelText(/item type/i), 'kilo')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(pmsApi.updateSKU).toHaveBeenCalledWith('test-token', 'sku-1', expect.objectContaining({
      unit_type: 'kilo', portioned_from_sku_id: null,
    })))
  })

  it('can clear an existing portioned-from relationship back to nil', async () => {
    const portionedSku = { ...baseSku, portioned_from_sku_id: 'sku-bulk-1' }
    const detail: ProductDetail = { product: product(), variants: [{ ...baseVariant, skus: [portionedSku] }] }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
    vi.mocked(pmsApi.updateSKU).mockResolvedValue(portionedSku)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await screen.findByText(/Portioned from/)

    await userEvent.click(screen.getByRole('button', { name: 'Remove' }))
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(pmsApi.updateSKU).toHaveBeenCalledWith('test-token', 'sku-1', expect.objectContaining({
      unit_type: 'unit', portioned_from_sku_id: null,
    })))
  })
})

describe('WooCommerce tab', () => {
  it('is not shown for a brand-new, not-yet-saved product', async () => {
    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByText('New Product'))

    expect(screen.queryByRole('button', { name: /WooCommerce/ })).not.toBeInTheDocument()
  })

  it('is not shown when no WooCommerce store is connected', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [] })
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([])

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))

    await waitFor(() => expect(pmsApi.getProduct).toHaveBeenCalled())
    expect(screen.queryByRole('button', { name: /WooCommerce/ })).not.toBeInTheDocument()
  })

  it('is not shown when a WooCommerce connection exists but is in an error state', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [] })
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection({ status: 'error' })])

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))

    await waitFor(() => expect(pmsApi.getProduct).toHaveBeenCalled())
    expect(screen.queryByRole('button', { name: /WooCommerce/ })).not.toBeInTheDocument()
  })

  it('shows "not linked yet" when the product has no listing for this connection', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [] })
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    vi.mocked(integrationsApi.getProductChannelListing).mockResolvedValue(null)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await userEvent.click(await screen.findByRole('button', { name: /WooCommerce/ }))

    await waitFor(() => expect(screen.getByText(/mystore\.com/)).toBeInTheDocument())
    expect(await screen.findByText(/not linked to this WooCommerce store yet/i)).toBeInTheDocument()
    expect(integrationsApi.getProductChannelListing).toHaveBeenCalledWith('test-token', 'conn-1', 'prod-1')
  })

  it('shows real sync state when the product is linked to this connection', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [] })
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    vi.mocked(integrationsApi.getProductChannelListing).mockResolvedValue({
      id: 'listing-1', product_id: 'prod-1', connection_id: 'conn-1', provider: 'woocommerce',
      external_id: '23609', sync_status: 'synced', last_synced_at: '2026-07-26T12:00:00Z',
      created_at: '2026-07-01T00:00:00Z', updated_at: '2026-07-26T12:00:00Z',
    })

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await userEvent.click(await screen.findByRole('button', { name: /WooCommerce/ }))

    expect(await screen.findByDisplayValue('23609')).toBeInTheDocument()
    expect(screen.getByDisplayValue('Synced')).toBeInTheDocument()
    expect(screen.queryByText(/not linked to this WooCommerce store yet/i)).not.toBeInTheDocument()
  })

  it('surfaces an error instead of silently showing "not linked" when the fetch fails', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [] })
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    vi.mocked(integrationsApi.getProductChannelListing).mockRejectedValue(new Error('internal server error'))

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await userEvent.click(await screen.findByRole('button', { name: /WooCommerce/ }))

    expect(await screen.findByText('internal server error')).toBeInTheDocument()
    expect(screen.queryByText(/not linked to this WooCommerce store yet/i)).not.toBeInTheDocument()
  })

  it("shows the SKU's Woo tax class/status when it was imported from Woo", async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({
      product: product(),
      variants: [{ ...baseVariant, skus: [{ ...baseSku, woo_tax_class_slug: 'reduced-rate', woo_tax_status: 'taxable' }] }],
    })
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    vi.mocked(integrationsApi.getProductChannelListing).mockResolvedValue(null)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await userEvent.click(await screen.findByRole('button', { name: /WooCommerce/ }))

    expect(await screen.findByDisplayValue('reduced-rate')).toBeInTheDocument()
    expect(screen.getByDisplayValue('taxable')).toBeInTheDocument()
  })

  it('shows "Standard" for an empty Woo tax class slug instead of a blank field', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({
      product: product(),
      variants: [{ ...baseVariant, skus: [{ ...baseSku, woo_tax_class_slug: '', woo_tax_status: 'taxable' }] }],
    })
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    vi.mocked(integrationsApi.getProductChannelListing).mockResolvedValue(null)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await userEvent.click(await screen.findByRole('button', { name: /WooCommerce/ }))

    expect(await screen.findByDisplayValue('Standard')).toBeInTheDocument()
  })

  it('does not show Woo tax fields for a SKU never imported from Woo', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({
      product: product(),
      variants: [{ ...baseVariant, skus: [baseSku] }],
    })
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    vi.mocked(integrationsApi.getProductChannelListing).mockResolvedValue(null)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await userEvent.click(await screen.findByRole('button', { name: /WooCommerce/ }))

    await screen.findByText(/not linked to this WooCommerce store yet/i)
    expect(screen.queryByLabelText('Tax class (Woo)')).not.toBeInTheDocument()
  })
})

describe('SKU weight/dimensions and virtual flag', () => {
  it('shows weight/dimensions fields and a virtual-product toggle for a simple product', async () => {
    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByText('New Product'))

    expect(screen.getByLabelText('Weight (g)')).toBeInTheDocument()
    expect(screen.getByLabelText('Length (mm)')).toBeInTheDocument()
    expect(screen.getByLabelText('Width (mm)')).toBeInTheDocument()
    expect(screen.getByLabelText('Height (mm)')).toBeInTheDocument()
    expect(screen.getByRole('switch', { name: 'Virtual product (no shipping)' })).toBeInTheDocument()
  })

  it('hides weight/dimensions fields once the product is marked virtual', async () => {
    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByText('New Product'))

    await userEvent.click(screen.getByRole('switch', { name: 'Virtual product (no shipping)' }))

    expect(screen.queryByLabelText('Weight (g)')).not.toBeInTheDocument()
  })

  it('sends weight/dimensions and is_virtual:false for a physical simple product', async () => {
    const created = product({ id: 'new-prod' })
    vi.mocked(pmsApi.createProduct).mockResolvedValue(created)
    vi.mocked(pmsApi.createVariant).mockResolvedValue({ ...baseVariant, id: 'new-var', product_id: 'new-prod' })
    vi.mocked(pmsApi.createSKU).mockResolvedValue({ ...baseSku, id: 'new-sku', variant_id: 'new-var' })
    vi.mocked(pmsApi.getProduct).mockResolvedValue({
      product: created,
      variants: [{ ...baseVariant, id: 'new-var', product_id: 'new-prod', skus: [{ ...baseSku, id: 'new-sku', variant_id: 'new-var' }] }],
    })

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByText('New Product'))

    await userEvent.type(screen.getByLabelText('Name'), 'New Shirt')
    await userEvent.type(screen.getByPlaceholderText('e.g. SHIRT-BLK-M'), 'NEWSHIRT-1')
    await userEvent.type(screen.getByPlaceholderText('0.00'), '29.99')
    await userEvent.type(screen.getByLabelText('Weight (g)'), '500')
    await userEvent.type(screen.getByLabelText('Length (mm)'), '100')
    await userEvent.type(screen.getByLabelText('Width (mm)'), '80')
    await userEvent.type(screen.getByLabelText('Height (mm)'), '20')

    await userEvent.click(screen.getByRole('button', { name: 'Create Product' }))

    await waitFor(() => expect(pmsApi.createSKU).toHaveBeenCalled())
    expect(pmsApi.createSKU).toHaveBeenCalledWith('test-token', 'new-var', expect.objectContaining({
      is_virtual: false, weight_grams: 500, length_mm: 100, width_mm: 80, height_mm: 20,
    }))
  })

  it('sends is_virtual:true and omits weight/dimensions for a virtual simple product', async () => {
    const created = product({ id: 'new-prod' })
    vi.mocked(pmsApi.createProduct).mockResolvedValue(created)
    vi.mocked(pmsApi.createVariant).mockResolvedValue({ ...baseVariant, id: 'new-var', product_id: 'new-prod' })
    vi.mocked(pmsApi.createSKU).mockResolvedValue({ ...baseSku, id: 'new-sku', variant_id: 'new-var' })
    vi.mocked(pmsApi.getProduct).mockResolvedValue({
      product: created,
      variants: [{ ...baseVariant, id: 'new-var', product_id: 'new-prod', skus: [{ ...baseSku, id: 'new-sku', variant_id: 'new-var' }] }],
    })

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByText('New Product'))

    await userEvent.type(screen.getByLabelText('Name'), 'E-Book')
    await userEvent.type(screen.getByPlaceholderText('e.g. SHIRT-BLK-M'), 'EBOOK-1')
    await userEvent.type(screen.getByPlaceholderText('0.00'), '9.99')
    await userEvent.click(screen.getByRole('switch', { name: 'Virtual product (no shipping)' }))

    await userEvent.click(screen.getByRole('button', { name: 'Create Product' }))

    await waitFor(() => expect(pmsApi.createSKU).toHaveBeenCalled())
    expect(pmsApi.createSKU).toHaveBeenCalledWith('test-token', 'new-var', expect.objectContaining({
      is_virtual: true, weight_grams: undefined, length_mm: undefined, width_mm: undefined, height_mm: undefined,
    }))
  })
})

describe('Tax class picker + price preview', () => {
  it('shows a Tax class select populated with the tenant tax classes', async () => {
    vi.mocked(taxApi.listTaxClasses).mockResolvedValue([
      { id: 'class-1', name: 'Standard', is_default: true, created_at: '', updated_at: '' },
      { id: 'class-2', name: 'Reduced rate', is_default: false, created_at: '', updated_at: '' },
    ])

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByText('New Product'))

    const select = await screen.findByLabelText('Tax class')
    expect(within(select as HTMLElement).getByText('Standard')).toBeInTheDocument()
    expect(within(select as HTMLElement).getByText('Reduced rate')).toBeInTheDocument()
  })

  it('creating a SKU with a tax class selected sends tax_class_id', async () => {
    vi.mocked(taxApi.listTaxClasses).mockResolvedValue([
      { id: 'class-1', name: 'Standard', is_default: true, created_at: '', updated_at: '' },
      { id: 'class-2', name: 'Reduced rate', is_default: false, created_at: '', updated_at: '' },
    ])
    const created = product({ id: 'new-prod' })
    vi.mocked(pmsApi.createProduct).mockResolvedValue(created)
    vi.mocked(pmsApi.createVariant).mockResolvedValue({ ...baseVariant, id: 'new-var', product_id: 'new-prod' })
    vi.mocked(pmsApi.createSKU).mockResolvedValue({ ...baseSku, id: 'new-sku', variant_id: 'new-var', tax_class_id: 'class-2' })
    vi.mocked(pmsApi.getProduct).mockResolvedValue({
      product: created,
      variants: [{ ...baseVariant, id: 'new-var', product_id: 'new-prod', skus: [{ ...baseSku, id: 'new-sku', variant_id: 'new-var', tax_class_id: 'class-2' }] }],
    })

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByText('New Product'))

    await userEvent.type(screen.getByLabelText('Name'), 'New Shirt')
    await userEvent.type(screen.getByPlaceholderText('e.g. SHIRT-BLK-M'), 'NEWSHIRT-1')
    await userEvent.type(screen.getByPlaceholderText('0.00'), '29.99')
    await userEvent.selectOptions(await screen.findByLabelText('Tax class'), 'class-2')

    await userEvent.click(screen.getByRole('button', { name: 'Create Product' }))

    await waitFor(() => expect(pmsApi.createSKU).toHaveBeenCalled())
    expect(pmsApi.createSKU).toHaveBeenCalledWith('test-token', 'new-var', expect.objectContaining({
      tax_class_id: 'class-2',
    }))
  })

  it('editing an existing SKU prefills its tax class', async () => {
    vi.mocked(taxApi.listTaxClasses).mockResolvedValue([
      { id: 'class-1', name: 'Standard', is_default: true, created_at: '', updated_at: '' },
      { id: 'class-2', name: 'Reduced rate', is_default: false, created_at: '', updated_at: '' },
    ])
    vi.mocked(pmsApi.getProduct).mockResolvedValue({
      product: product(),
      variants: [{ ...baseVariant, skus: [{ ...baseSku, tax_class_id: 'class-2' }] }],
    })

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByText('T-Shirt'))

    await waitFor(() => expect(screen.getByLabelText('Tax class')).toHaveValue('class-2'))
  })

  it('shows an incl./excl. tax price preview when a rate exists for the default tax country', async () => {
    vi.mocked(tenantSettingsApi.getTenantSettings).mockResolvedValue({
      tenant_id: 't1', default_currency: 'USD', timezone: '', date_format: '', language: '',
      prices_include_tax: true, default_tax_country: 'NL', inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_margin_basis_points: 0, updated_at: '',
    })
    vi.mocked(taxApi.listTaxClasses).mockResolvedValue([
      { id: 'class-1', name: 'Standard', is_default: true, created_at: '', updated_at: '' },
    ])
    vi.mocked(taxApi.listTaxRates).mockResolvedValue([
      { id: 'rate-1', tax_class_id: 'class-1', country: 'NL', rate_basis_points: 2100, created_at: '', updated_at: '' },
    ])

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByText('New Product'))
    await screen.findByLabelText('Tax class')

    await userEvent.type(screen.getByPlaceholderText('0.00'), '121.00')

    expect(await screen.findByText(/Excl\. tax: USD 100\.00/)).toBeInTheDocument()
    expect(screen.getByText(/Incl\. tax: USD 121\.00/)).toBeInTheDocument()
  })
})

describe('product edit page chrome', () => {
  it('Discard closes the editor without saving', async () => {
    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByText('New Product'))

    await userEvent.type(screen.getByLabelText('Name'), 'Abandoned Product')
    await userEvent.click(screen.getByRole('button', { name: 'Discard' }))

    expect(screen.queryByLabelText('Name')).not.toBeInTheDocument()
    expect(pmsApi.createProduct).not.toHaveBeenCalled()
  })

  it('shows the not-yet-supported fields as disabled placeholders', async () => {
    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByText('New Product'))

    for (const label of ['Vendor', 'Collections', 'Page title', 'Meta description']) {
      const field = screen.getByLabelText(label)
      expect(field).toBeInTheDocument()
      expect(field).toBeDisabled()
    }
  })

  it('shows categories in hierarchical order, children under their parent, not flat alphabetical', async () => {
    vi.mocked(pmsApi.listCategories).mockResolvedValue([
      category({ id: 'bebidas', name: 'Bebidas', parent_id: null }),
      category({ id: 'alimentos', name: 'Alimentos', parent_id: null }),
      category({ id: 'salgadinhos', name: 'Salgadinhos e snacks', parent_id: 'alimentos' }),
    ])

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByText('New Product'))

    const options = screen.getByLabelText('Category').querySelectorAll('option')
    const names = Array.from(options).map(o => o.textContent?.trim())

    // "Alimentos" then its child "Salgadinhos e snacks" immediately after,
    // both before the unrelated root "Bebidas" — plain alphabetical order
    // would put "Bebidas" between them instead.
    expect(names).toEqual(['No category', 'Alimentos', expect.stringContaining('Salgadinhos e snacks'), 'Bebidas'])
  })
})

describe('Variant image picker', () => {
  const media1 = {
    id: 'media-1', tenant_id: 't1', product_id: 'prod-1',
    url: 'https://api.example.com/uploads/t1/a.jpg', type: 'image' as const,
    position: 0, alt_text: 'Front', created_at: '',
  }
  const media2 = {
    id: 'media-2', tenant_id: 't1', product_id: 'prod-1',
    url: 'https://api.example.com/uploads/t1/b.jpg', type: 'image' as const,
    position: 1, alt_text: '', created_at: '',
  }

  it('does not render an image picker when the product has no media', async () => {
    const detail: ProductDetail = {
      product: product({ has_variants: true }),
      variants: [{ ...baseVariant, id: 'var-1', name: 'Small', skus: [] }, { ...baseVariant, id: 'var-2', name: 'Large', skus: [] }],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))

    await screen.findByText('Small')
    expect(screen.queryByLabelText('Small image')).not.toBeInTheDocument()
  })

  it('opens a visual thumbnail grid of the product images and selecting one updates the variant', async () => {
    const detail: ProductDetail = {
      product: product({ has_variants: true }),
      variants: [{ ...baseVariant, id: 'var-1', name: 'Small', skus: [] }],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
    vi.mocked(pmsApi.listProductMedia).mockResolvedValue([media1, media2])
    vi.mocked(pmsApi.updateVariant).mockResolvedValue({ ...baseVariant, id: 'var-1', name: 'Small', featured_media_id: 'media-1' })

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))

    const thumbnailButton = await screen.findByLabelText('Small image')
    expect(screen.queryByLabelText('Front')).not.toBeInTheDocument()

    await userEvent.click(thumbnailButton)

    expect(screen.getByLabelText('Front')).toBeInTheDocument()
    expect(screen.getByLabelText('Product image')).toBeInTheDocument()
    expect(screen.getByLabelText('No image')).toBeInTheDocument()

    await userEvent.click(screen.getByLabelText('Front'))

    await waitFor(() => expect(pmsApi.updateVariant).toHaveBeenCalledWith('test-token', 'var-1', expect.objectContaining({
      featured_media_id: 'media-1',
    })))
  })

  it('clearing the image via "No image" sends featured_media_id: null', async () => {
    const featuredId = 'media-1'
    const detail: ProductDetail = {
      product: product({ has_variants: true }),
      variants: [{ ...baseVariant, id: 'var-1', name: 'Small', skus: [], featured_media_id: featuredId }],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
    vi.mocked(pmsApi.listProductMedia).mockResolvedValue([media1])
    vi.mocked(pmsApi.updateVariant).mockResolvedValue({ ...baseVariant, id: 'var-1', name: 'Small' })

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))

    await userEvent.click(await screen.findByLabelText('Small image'))
    await userEvent.click(screen.getByLabelText('No image'))

    await waitFor(() => expect(pmsApi.updateVariant).toHaveBeenCalledWith('test-token', 'var-1', expect.objectContaining({
      featured_media_id: null,
    })))
  })

  it('shows an error if updating the variant image fails', async () => {
    const detail: ProductDetail = {
      product: product({ has_variants: true }),
      variants: [{ ...baseVariant, id: 'var-1', name: 'Small', skus: [] }],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
    vi.mocked(pmsApi.listProductMedia).mockResolvedValue([media1])
    vi.mocked(pmsApi.updateVariant).mockRejectedValue(new Error('boom'))

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))

    await userEvent.click(await screen.findByLabelText('Small image'))
    await userEvent.click(screen.getByLabelText('Front'))

    expect(await screen.findByText('boom')).toBeInTheDocument()
  })
})

describe('Media section', () => {
  const mediaItem = {
    id: 'media-1', tenant_id: 't1', product_id: 'prod-1',
    url: 'https://api.example.com/uploads/t1/x.jpg', type: 'image' as const,
    position: 0, alt_text: '', created_at: '',
  }

  it('shows a save-first message for a new (unsaved) product instead of the upload UI', async () => {
    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByText('New Product'))

    expect(screen.getByText('Save the product first, then add images.')).toBeInTheDocument()
  })

  it('lists existing media and uploads a new file for an existing product', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [] })
    vi.mocked(pmsApi.listProductMedia).mockResolvedValue([mediaItem])
    const uploaded = { ...mediaItem, id: 'media-2', url: 'https://api.example.com/uploads/t1/y.jpg' }
    vi.mocked(pmsApi.uploadProductMedia).mockResolvedValue(uploaded)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByText('T-Shirt'))

    await waitFor(() => expect(pmsApi.listProductMedia).toHaveBeenCalledWith('test-token', 'prod-1'))
    expect(await screen.findByAltText('')).toBeInTheDocument()

    const file = new File(['bytes'], 'photo.jpg', { type: 'image/jpeg' })
    const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement
    await userEvent.upload(fileInput, file)

    await waitFor(() => expect(pmsApi.uploadProductMedia).toHaveBeenCalledWith('test-token', 'prod-1', file, undefined))
  })

  it('removes a media item when its remove button is clicked', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [] })
    vi.mocked(pmsApi.listProductMedia).mockResolvedValue([mediaItem])
    vi.mocked(pmsApi.deleteMedia).mockResolvedValue(null)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByText('T-Shirt'))
    await screen.findByAltText('')

    await userEvent.click(screen.getByRole('button', { name: 'Remove image' }))

    await waitFor(() => expect(pmsApi.deleteMedia).toHaveBeenCalledWith('test-token', 'media-1'))
  })

  it('clicking an image opens a title editor prefilled with its current alt text, and saving persists it', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [] })
    vi.mocked(pmsApi.listProductMedia).mockResolvedValue([{ ...mediaItem, alt_text: 'Old title' }])
    vi.mocked(pmsApi.updateMedia).mockResolvedValue({ ...mediaItem, alt_text: 'Front view' })

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByText('T-Shirt'))
    await screen.findByAltText('Old title')

    await userEvent.click(screen.getByRole('button', { name: 'Edit image title' }))

    const titleInput = screen.getByLabelText('Title')
    expect(titleInput).toHaveValue('Old title')

    await userEvent.clear(titleInput)
    await userEvent.type(titleInput, 'Front view')
    await userEvent.click(screen.getByRole('button', { name: 'Save title' }))

    await waitFor(() => expect(pmsApi.updateMedia).toHaveBeenCalledWith('test-token', 'media-1', 'Front view'))
    expect(screen.queryByLabelText('Title')).not.toBeInTheDocument()
  })

  it('Cancel closes the title editor without saving', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [] })
    vi.mocked(pmsApi.listProductMedia).mockResolvedValue([mediaItem])

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByText('T-Shirt'))
    await screen.findByAltText('')

    await userEvent.click(screen.getByRole('button', { name: 'Edit image title' }))
    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }))

    expect(screen.queryByLabelText('Title')).not.toBeInTheDocument()
    expect(pmsApi.updateMedia).not.toHaveBeenCalled()
  })

  it('shows an error if saving the title fails', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [] })
    vi.mocked(pmsApi.listProductMedia).mockResolvedValue([mediaItem])
    vi.mocked(pmsApi.updateMedia).mockRejectedValue(new Error('title too long'))

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByText('T-Shirt'))
    await screen.findByAltText('')

    await userEvent.click(screen.getByRole('button', { name: 'Edit image title' }))
    await userEvent.click(screen.getByRole('button', { name: 'Save title' }))

    expect(await screen.findByText('title too long')).toBeInTheDocument()
  })
})

describe('Manual cost price override', () => {
  it('lets staff set a cost price manually when none has been computed yet', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [{ ...baseVariant, skus: [baseSku] }] })
    vi.mocked(pmsApi.setManualCostPrice).mockResolvedValue({ ...baseSku, cost_price_minor_units: 1250, cost_price_source: 'manual' })

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))

    await screen.findByText('Set cost price manually')
    await userEvent.type(screen.getByPlaceholderText('USD'), '12.50')
    await userEvent.click(screen.getByRole('button', { name: 'Set' }))

    await waitFor(() => expect(pmsApi.setManualCostPrice).toHaveBeenCalledWith('test-token', 'sku-1', 1250))
  })

  it('shows a revert control and calls clearCostPriceOverride when the cost price is a manual override', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [{ ...baseVariant, skus: [baseSku] }] })
    vi.mocked(pmsApi.getSKUMarginInsight).mockResolvedValue({
      cost_price_available: true, cost_price_minor_units: 1250, cost_price_source: 'manual', effective_target_margin_basis_points: 3000,
      recommended_price_minor_units: 1786, profit_at_price_minor_units: 749, margin_at_price_basis_points: 3746,
      total_on_hand_quantity: 0, total_profit_potential_minor_units: 0, shrinkage_minor_units: 0,
    })
    vi.mocked(pmsApi.clearCostPriceOverride).mockResolvedValue({ ...baseSku, cost_price_minor_units: 1250, cost_price_source: 'computed' })

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))

    await screen.findByText(/Cost price set manually/)
    expect(screen.queryByText('Set cost price manually')).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Revert to automatic' }))

    await waitFor(() => expect(pmsApi.clearCostPriceOverride).toHaveBeenCalledWith('test-token', 'sku-1'))
  })
})
