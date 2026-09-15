// @vitest-environment jsdom
import { render, screen, waitFor, within, fireEvent } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, useNavigate } from 'react-router-dom'
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
  duplicateProduct: vi.fn(),
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
    cost_price_available: false, cost_price_minor_units: 0, cost_price_source: 'computed', effective_target_markup_basis_points: 0,
    recommended_price_minor_units: 0, profit_at_price_minor_units: 0, margin_at_price_basis_points: 0, markup_at_price_basis_points: 0,
    total_on_hand_quantity: 0, total_profit_potential_minor_units: 0, shrinkage_minor_units: 0,
  })),
  setManualCostPrice: vi.fn(),
  clearCostPriceOverride: vi.fn(),
  listSKUCostPriceHistory: vi.fn(),
  listSKUPriceHistory: vi.fn(() => Promise.resolve([])),
  listProductHistory: vi.fn(() => Promise.resolve([])),
  listVariantHistory: vi.fn(() => Promise.resolve([])),
  listMarginReport: vi.fn(),
  listProductMedia: vi.fn(),
  uploadProductMedia: vi.fn(),
  updateMedia: vi.fn(),
  deleteMedia: vi.fn(),
  listSKUCurrencyPrices: vi.fn(() => Promise.resolve([])),
  upsertSKUCurrencyPrice: vi.fn(),
  deleteSKUCurrencyPrice: vi.fn(),
  listBarcodeAliases: vi.fn(() => Promise.resolve([])),
  createBarcodeAlias: vi.fn(),
  deleteBarcodeAlias: vi.fn(),
  listScaleCodeAliases: vi.fn(() => Promise.resolve([])),
  createScaleCodeAlias: vi.fn(),
  deleteScaleCodeAlias: vi.fn(),
  bulkAddWeightPortions: vi.fn(),
}))

vi.mock('../../inventory/api', () => ({
  listWarehouses: vi.fn(() => Promise.resolve([])),
  listBatches: vi.fn(() => Promise.resolve([])),
  getSKUStockLevel: vi.fn(() => Promise.resolve({ sku_id: 'sku-1', total: 0, syncable: 0 })),
}))

vi.mock('../../integrations/api', () => ({
  listConnections: vi.fn(),
  getProductChannelListing: vi.fn(),
  listProductChannelListings: vi.fn(),
  resyncProductToChannels: vi.fn(),
  linkExistingWooProduct: vi.fn(),
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
import * as inventoryApi from '../../inventory/api'
import * as integrationsApi from '../../integrations/api'
import * as tenantSettingsApi from '../../settings/api'
import * as taxApi from '../../tax/api'

// inputAfterLabel finds the <input> sibling of a plain-text <label> that
// has no htmlFor/id association (VariantRow's edit panel labels don't wrap
// or reference their inputs) -- getByLabelText can't resolve these, and
// several inputs in the same panel share the same placeholder ("0" for
// weight/length/width/height), so placeholder lookup alone is ambiguous.
function inputAfterLabel(labelText: string): HTMLInputElement {
  const label = screen.getByText(labelText)
  const input = label.parentElement?.querySelector('input')
  if (!input) throw new Error(`No input found next to label "${labelText}"`)
  return input
}

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
  vi.mocked(integrationsApi.listProductChannelListings).mockResolvedValue([])
  vi.mocked(pmsApi.listProductMedia).mockResolvedValue([])
  vi.mocked(tenantSettingsApi.getTenantSettings).mockResolvedValue({
    tenant_id: 't1', default_currency: 'USD', timezone: '', date_format: '', language: '',
    prices_include_tax: true, default_tax_country: '', inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_markup_basis_points: 0, updated_at: '',
  })
  vi.mocked(taxApi.listTaxClasses).mockResolvedValue([])
  vi.mocked(taxApi.listTaxRates).mockResolvedValue([])
})

function NavReselectButton() {
  const { setActiveNav } = useApp()
  return <button onClick={() => setActiveNav('products')}>Reselect Products</button>
}

// Stands in for CommandPalette's openRecord: a `navigate()` call to a
// per-record URL fired while Products is already mounted, without Products
// itself remounting (it's a static sibling here, same as in the real app --
// only the matched <Route> element changes, not this component's position).
function NavigateToProductButton({ id }: { id: string }) {
  const navigate = useNavigate()
  return <button onClick={() => navigate(`/products/${id}`)}>simulate search result click</button>
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

describe('which product is open survives a page refresh', () => {
  // The open product lives in the URL (/products/<id>), not local component
  // state -- rendering straight at that URL (what a refresh does, since the
  // SPA remounts from scratch) must land directly on the edit page, with no
  // click required to get there.
  it('opens straight into the edit page when the URL already points at a product', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [] })

    render(
      <MemoryRouter initialEntries={['/products/prod-1']}>
        <AppProvider><Products /></AppProvider>
      </MemoryRouter>,
    )

    await waitFor(() => expect(pmsApi.getProduct).toHaveBeenCalledWith('test-token', 'prod-1'))
    expect(await screen.findByDisplayValue('T-Shirt')).toBeInTheDocument()
  })

  it('opens straight into the "new product" form when the URL is /products/new', async () => {
    render(
      <MemoryRouter initialEntries={['/products/new']}>
        <AppProvider><Products /></AppProvider>
      </MemoryRouter>,
    )

    expect(await screen.findByRole('button', { name: 'Create Product' })).toBeInTheDocument()
    expect(pmsApi.getProduct).not.toHaveBeenCalled()
  })

  // Regression: global search's CommandPalette navigates to "/products/<id>"
  // without unmounting whatever page is currently showing. The old ?id=
  // scheme read the query param into a useState initializer that only ran
  // on mount, so this navigation silently did nothing while already on
  // Products. Reading the id straight from the URL on every render fixes it.
  it('opens the product when navigated to its URL while already mounted on the list (search result click)', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [] })

    render(
      <MemoryRouter initialEntries={['/products']}>
        <AppProvider>
          <NavigateToProductButton id="prod-1" />
          <Products />
        </AppProvider>
      </MemoryRouter>,
    )
    await screen.findByText('T-Shirt')

    await userEvent.click(screen.getByText('simulate search result click'))

    expect(await screen.findByDisplayValue('T-Shirt')).toBeInTheDocument()
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

  it('opens the duplicated product for editing once duplication succeeds', async () => {
    const duplicated: ProductDetail = { product: product({ id: 'prod-2', name: 'T-Shirt (Copy)', slug: 't-shirt-copy', status: 'draft' }), variants: [] }
    vi.mocked(pmsApi.duplicateProduct).mockResolvedValue(duplicated)
    vi.mocked(pmsApi.getProduct).mockResolvedValue(duplicated)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Duplicate'))

    await waitFor(() => expect(pmsApi.getProduct).toHaveBeenCalledWith('test-token', 'prod-2'))
    expect(await screen.findByDisplayValue('T-Shirt (Copy)')).toBeInTheDocument()
  })

  it('surfaces an error instead of silently failing when duplication fails', async () => {
    vi.mocked(pmsApi.duplicateProduct).mockRejectedValue(new Error('duplicate failed'))

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Duplicate'))

    expect(await screen.findByText('duplicate failed')).toBeInTheDocument()
    expect(pmsApi.getProduct).not.toHaveBeenCalled()
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

// Regression coverage for a duplicated product being permanently stuck
// with its "-copy" slug: there used to be no way, anywhere in the UI, to
// change a product's slug once created.
describe('editing a product\'s slug', () => {
  it('shows no slug field when creating a new product', async () => {
    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'New Product' }))

    expect(screen.queryByLabelText('Slug')).not.toBeInTheDocument()
  })

  // has_variants:true + 2 variants (as in the sync-toggle tests above)
  // skips the flat Pricing & SKU form, which has its own required fields
  // unrelated to slug and would otherwise block submit in this harness.
  it('is prefilled from the product and omitted from the save payload when left unchanged', async () => {
    const detail: ProductDetail = {
      product: product({ has_variants: true, slug: 't-shirt' }),
      variants: [
        { ...baseVariant, id: 'var-1', name: 'Small', skus: [] },
        { ...baseVariant, id: 'var-2', name: 'Large', skus: [] },
      ],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
    vi.mocked(pmsApi.updateProduct).mockResolvedValue(detail.product)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await waitFor(() => expect(screen.getByLabelText('Slug')).toHaveValue('t-shirt'))

    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(pmsApi.updateProduct).toHaveBeenCalled())
    expect(pmsApi.updateProduct).toHaveBeenCalledWith('test-token', 'prod-1',
      expect.not.objectContaining({ slug: expect.anything() }))
  })

  it('sends the edited slug on save, fixing a duplicate stuck with its "-copy" slug', async () => {
    const detail: ProductDetail = {
      product: product({
        has_variants: true, name: 'Biscoito de Maizena - Renata - 360g (Copy)',
        slug: 'biscoito-de-maizena-renata-360g-copy',
      }),
      variants: [
        { ...baseVariant, id: 'var-1', name: 'Small', skus: [] },
        { ...baseVariant, id: 'var-2', name: 'Large', skus: [] },
      ],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
    vi.mocked(pmsApi.updateProduct).mockResolvedValue(detail.product)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await waitFor(() => expect(screen.getByLabelText('Slug')).toHaveValue('biscoito-de-maizena-renata-360g-copy'))

    const slugInput = screen.getByLabelText('Slug')
    await userEvent.clear(slugInput)
    await userEvent.type(slugInput, 'biscoito-de-maizena-renata-360g')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(pmsApi.updateProduct).toHaveBeenCalledWith('test-token', 'prod-1', expect.objectContaining({
      slug: 'biscoito-de-maizena-renata-360g',
    })))
  })

  // Regression: the server regenerates a cleared slug from the product name
  // (and can further disambiguate it on collision), so what gets persisted
  // can differ from the empty string the user submitted. The `hydrated`
  // sync-from-`existing` effect only ever runs once, so without patching
  // local state directly from the save response, the field kept showing
  // empty until a full page reload.
  it('shows the server-regenerated slug immediately after clearing it, without a page reload', async () => {
    const detail: ProductDetail = {
      product: product({ has_variants: true, name: 'T-Shirt', slug: 't-shirt' }),
      variants: [
        { ...baseVariant, id: 'var-1', name: 'Small', skus: [] },
        { ...baseVariant, id: 'var-2', name: 'Large', skus: [] },
      ],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
    vi.mocked(pmsApi.updateProduct).mockResolvedValue(product({ name: 'T-Shirt', slug: 't-shirt-2' }))

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await waitFor(() => expect(screen.getByLabelText('Slug')).toHaveValue('t-shirt'))

    await userEvent.clear(screen.getByLabelText('Slug'))
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(screen.getByLabelText('Slug')).toHaveValue('t-shirt-2'))
  })

  // Regression: a save error (e.g. the backend's 409 "slug already exists")
  // used to render as the very last element of the General tab's main
  // column, below five FormCards -- visually invisible without scrolling
  // past all of them. It must render outside the scrollable form body.
  it('shows a save error outside the scrollable form body, not buried at the bottom of the page', async () => {
    const detail: ProductDetail = {
      product: product({ has_variants: true, slug: 't-shirt' }),
      variants: [
        { ...baseVariant, id: 'var-1', name: 'Small', skus: [] },
        { ...baseVariant, id: 'var-2', name: 'Large', skus: [] },
      ],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
    vi.mocked(pmsApi.updateProduct).mockRejectedValue(new Error('updateProduct: slug already exists'))

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await waitFor(() => expect(screen.getByLabelText('Slug')).toHaveValue('t-shirt'))

    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    const error = await screen.findByText('updateProduct: slug already exists')
    expect(error.closest('.overflow-y-auto')).toBeNull()
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
    // 2 levels up from the SKU code field: its own wrapper div, then the
    // grid div -- the outer panel (Item type + Add/Cancel buttons live
    // there, as siblings of the grid, not inside it) is one level further.
    const skuForm = screen.getByPlaceholderText('SKU code').closest('div')!.parentElement!.parentElement!
    await userEvent.type(within(skuForm).getByPlaceholderText('SKU code'), 'BULK-1')
    await userEvent.type(within(skuForm).getByPlaceholderText('0.00'), '0')
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

  it('a weight pool SKU with per-weight pricing and a scale code shows "Bulk add by weight", opening the modal on click', async () => {
    const poolSku = {
      ...baseSku, unit_type: 'kilo' as const, scale_code: '2994721',
      pricing_mode: 'per_weight' as const, price_per_kilo_minor_units: 29990,
    }
    const detail: ProductDetail = {
      product: product({ has_variants: true }),
      variants: [{ ...baseVariant, id: 'var-1', name: 'Carne Seca', skus: [poolSku] }],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await screen.findByText('Carne Seca')

    await userEvent.click(screen.getByRole('button', { name: 'Bulk add by weight' }))

    expect(await screen.findByText('Bulk add by weight', { selector: 'h3' })).toBeInTheDocument()
    await waitFor(() => expect(inventoryApi.listBatches).toHaveBeenCalledWith('test-token', 'sku-1'))
  })

  it('a weight pool SKU with no scale code does not show "Bulk add by weight" -- nothing to barcode packages with yet', async () => {
    const poolSkuNoScaleCode = { ...baseSku, unit_type: 'kilo' as const, pricing_mode: 'per_weight' as const, price_per_kilo_minor_units: 29990 }
    const detail: ProductDetail = {
      product: product({ has_variants: true }),
      variants: [{ ...baseVariant, id: 'var-1', name: 'Carne Seca', skus: [poolSkuNoScaleCode] }],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await screen.findByText('Carne Seca')

    expect(screen.getByText('Weight pool')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Bulk add by weight' })).not.toBeInTheDocument()
  })
})

// Weight-embedded (butcher/deli scale) barcode support -- see
// pms.SKU.ScaleCode/PricingMode/PricePerKiloMinorUnits. Every existing
// per-field save (barcode, weight-pool link, variant/SKU edit) sends the
// SKU's other fields as a straight pass-through, so these three new ones
// must be preserved the same way or an unrelated save silently wipes them.
describe('weight-embedded barcode pricing (scale_code / pricing_mode / price_per_kilo_minor_units)', () => {
  it('flat SKU form: Pricing selector is always visible, price/kilo only once Per kilo is chosen', async () => {
    const detail: ProductDetail = { product: product(), variants: [{ ...baseVariant, skus: [baseSku] }] }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await waitFor(() => expect(screen.getByDisplayValue('TSHIRT-1')).toBeInTheDocument())

    // Visible even with no scale code yet -- "sold by unit or weight" must
    // be discoverable without first figuring out the scale-code field.
    expect(screen.getByLabelText('Pricing')).toBeInTheDocument()
    expect(screen.queryByLabelText(/Price per kilo/)).not.toBeInTheDocument()

    await userEvent.selectOptions(screen.getByLabelText('Pricing'), 'per_weight')
    expect(screen.getByLabelText(/Price per kilo/)).toBeInTheDocument()
    expect(screen.getByText(/Set a scale code above too/)).toBeInTheDocument()

    await userEvent.type(screen.getByLabelText('Scale code'), '00633')
    expect(screen.queryByText(/Set a scale code above too/)).not.toBeInTheDocument()
  })

  it('scale code field keeps the full 7-digit code (prefix included), just digits-only and capped', async () => {
    const detail: ProductDetail = { product: product(), variants: [{ ...baseVariant, skus: [baseSku] }] }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
    vi.mocked(pmsApi.updateProduct).mockResolvedValue(detail.product)
    vi.mocked(pmsApi.updateSKU).mockResolvedValue(baseSku)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await waitFor(() => expect(screen.getByDisplayValue('TSHIRT-1')).toBeInTheDocument())

    // Real calabresa label: prefix 29, scale code 00633 -- staff read/type
    // the full 7-digit code exactly as shown on the scale, and that's what
    // gets stored -- no stripping, matching DecodeWeightEmbeddedBarcode's
    // own [0:7] (prefix+code) extraction.
    await userEvent.type(screen.getByLabelText('Scale code'), '2900633')
    expect(screen.getByLabelText('Scale code')).toHaveValue('2900633')

    await userEvent.selectOptions(screen.getByLabelText('Pricing'), 'per_weight')
    await userEvent.type(screen.getByLabelText(/Price per kilo/), '49.99')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(pmsApi.updateSKU).toHaveBeenCalledWith('test-token', 'sku-1', expect.objectContaining({
      scale_code: '2900633',
    })))
  })

  it('flat SKU form: scale code + per-kilo pricing sends pricing_mode/price_per_kilo_minor_units on save', async () => {
    const detail: ProductDetail = { product: product(), variants: [{ ...baseVariant, skus: [baseSku] }] }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
    vi.mocked(pmsApi.updateProduct).mockResolvedValue(detail.product)
    vi.mocked(pmsApi.updateSKU).mockResolvedValue(baseSku)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await waitFor(() => expect(screen.getByDisplayValue('TSHIRT-1')).toBeInTheDocument())

    await userEvent.type(screen.getByLabelText('Scale code'), '94721')
    await userEvent.selectOptions(screen.getByLabelText('Pricing'), 'per_weight')
    await userEvent.type(screen.getByLabelText(/Price per kilo/), '49.99')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(pmsApi.updateSKU).toHaveBeenCalledWith('test-token', 'sku-1', expect.objectContaining({
      scale_code: '94721', pricing_mode: 'per_weight', price_per_kilo_minor_units: 4999,
    })))
  })

  it('the EAN field is disabled (not editable) for a SKU that already has a scale code', async () => {
    // A scale-priced SKU's real printed barcode changes per package (the
    // weight is baked in), so there's nothing meaningful to type here --
    // editing it would just be silently ignored/misleading. See the
    // Barcode/EAN inputs' scale_code-gated placeholder+disabled in
    // Products.tsx for the flat-form equivalent.
    const existingSku = { ...baseSku, scale_code: '00633', pricing_mode: 'per_weight' as const, price_per_kilo_minor_units: 4999 }
    const detail: ProductDetail = {
      product: product({ has_variants: true }),
      variants: [{ ...baseVariant, id: 'var-1', name: '378g', skus: [existingSku] }],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await screen.findByText('378g')

    const ean = await screen.findByLabelText('EAN')
    expect(ean).toBeDisabled()
    expect(ean).toHaveValue('')
  })

  it('making an existing SKU a weight pool preserves its scale_code/pricing_mode/price_per_kilo_minor_units unchanged', async () => {
    const existingSku = { ...baseSku, scale_code: '00633', pricing_mode: 'per_weight' as const, price_per_kilo_minor_units: 4999 }
    const detail: ProductDetail = {
      product: product({ has_variants: true }),
      variants: [{ ...baseVariant, id: 'var-1', name: '378g', skus: [existingSku] }],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
    vi.mocked(pmsApi.updateSKU).mockResolvedValue(existingSku)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await screen.findByText('378g')

    await userEvent.click(screen.getByRole('button', { name: 'Make weight pool' }))

    await waitFor(() => expect(pmsApi.updateSKU).toHaveBeenCalledWith('test-token', 'sku-1', expect.objectContaining({
      scale_code: '00633', pricing_mode: 'per_weight', price_per_kilo_minor_units: 4999,
    })))
  })

  it('adding a variant SKU with a scale code and per-kilo pricing sends the right fields', async () => {
    const detail: ProductDetail = {
      product: product({ has_variants: true }),
      variants: [{ ...baseVariant, id: 'var-1', name: 'Dry Meat', skus: [] }],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
    vi.mocked(pmsApi.createSKU).mockResolvedValue({ ...baseSku, id: 'new-sku' })

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await screen.findByText('Dry Meat')

    await userEvent.click(screen.getByRole('button', { name: '+ Add SKU' }))
    const skuForm = screen.getByPlaceholderText('SKU code').closest('div')!.parentElement!.parentElement!
    await userEvent.type(within(skuForm).getByPlaceholderText('SKU code'), 'DRYMEAT-1')
    await userEvent.type(within(skuForm).getByPlaceholderText('0.00'), '0')
    await userEvent.type(within(skuForm).getByPlaceholderText('Not sold via a scale barcode'), '94721')
    await userEvent.selectOptions(within(skuForm).getByLabelText('Pricing'), 'per_weight')
    const pricePerKiloInput = within(skuForm).getByText(/Price\/kilo/).parentElement!.querySelector('input')!
    await userEvent.type(pricePerKiloInput, '49.99')
    await userEvent.click(within(skuForm).getByRole('button', { name: 'Add' }))

    await waitFor(() => expect(pmsApi.createSKU).toHaveBeenCalledWith('test-token', 'var-1', expect.objectContaining({
      scale_code: '94721', pricing_mode: 'per_weight', price_per_kilo_minor_units: 4999,
    })))
  })

  it('saving edited variant/SKU fields preserves scale_code/pricing_mode/price_per_kilo_minor_units unchanged', async () => {
    const existingSku = { ...baseSku, scale_code: '00633', pricing_mode: 'per_weight' as const, price_per_kilo_minor_units: 4999 }
    const detail: ProductDetail = {
      product: product({ has_variants: true }),
      variants: [{ ...baseVariant, id: 'var-1', name: '378g', skus: [existingSku] }],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
    vi.mocked(pmsApi.updateVariant).mockResolvedValue({ ...baseVariant, id: 'var-1', name: '378g' })
    vi.mocked(pmsApi.updateSKU).mockResolvedValue(existingSku)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await screen.findByText('378g')

    await userEvent.click(screen.getByRole('button', { name: 'Edit variant' }))
    const editPanel = screen.getByPlaceholderText('Variant name').closest('div')!.parentElement!
    await userEvent.click(within(editPanel).getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(pmsApi.updateSKU).toHaveBeenCalledWith('test-token', 'sku-1', expect.objectContaining({
      scale_code: '00633', pricing_mode: 'per_weight', price_per_kilo_minor_units: 4999,
    })))
  })
})

describe('editing an existing variant SKU\'s barcode/EAN (VariantRow)', () => {
  it('prefills the existing barcode and hides Save until it changes', async () => {
    const detail: ProductDetail = {
      product: product({ has_variants: true }),
      variants: [{ ...baseVariant, id: 'var-1', name: '378g', skus: [{ ...baseSku, barcode: '0630509892495' }] }],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await screen.findByText('378g')

    const ean = await screen.findByLabelText('EAN')
    expect(ean).toHaveValue('0630509892495')
    expect(within(ean.closest('div')!).queryByRole('button', { name: 'Save' })).not.toBeInTheDocument()
  })

  it('editing an existing SKU\'s barcode shows Save and persists the new value, keeping every other field unchanged', async () => {
    const existingSku = { ...baseSku, barcode: '0630509892495' }
    const detail: ProductDetail = {
      product: product({ has_variants: true }),
      variants: [{ ...baseVariant, id: 'var-1', name: '378g', skus: [existingSku] }],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
    vi.mocked(pmsApi.updateSKU).mockResolvedValue(existingSku)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await screen.findByText('378g')

    const ean = await screen.findByLabelText('EAN')
    await userEvent.clear(ean)
    await userEvent.type(ean, '7891010249939')
    await userEvent.click(within(ean.closest('div')!).getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(pmsApi.updateSKU).toHaveBeenCalledWith('test-token', 'sku-1', expect.objectContaining({
      barcode: '7891010249939', sku_code: existingSku.sku_code, price_minor_units: existingSku.price_minor_units,
      status: existingSku.status,
    })))
  })
})

describe('editing an existing variant and its SKU\'s own fields (VariantRow)', () => {
  it('clicking Edit prefills name/price/status/weight from the current variant and SKU', async () => {
    const detail: ProductDetail = {
      product: product({ has_variants: true }),
      variants: [{ ...baseVariant, id: 'var-1', name: '378g', skus: [{ ...baseSku, weight_grams: 378, status: 'inactive' as const }] }],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await screen.findByText('378g')

    await userEvent.click(screen.getByRole('button', { name: 'Edit variant' }))

    expect(screen.getByPlaceholderText('Variant name')).toHaveValue('378g')
    expect(screen.getByPlaceholderText('SKU code')).toHaveValue(baseSku.sku_code)
    expect(screen.getByPlaceholderText('0.00')).toHaveValue(19.99)
    expect(inputAfterLabel('Weight (g)')).toHaveValue(378)
    expect(screen.getByDisplayValue('Inactive')).toBeInTheDocument()
  })

  it('saving updates both the variant name and the SKU\'s fields', async () => {
    const detail: ProductDetail = {
      product: product({ has_variants: true }),
      variants: [{ ...baseVariant, id: 'var-1', name: '378g', skus: [baseSku] }],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
    vi.mocked(pmsApi.updateVariant).mockResolvedValue({ ...baseVariant, id: 'var-1', name: '412g' })
    vi.mocked(pmsApi.updateSKU).mockResolvedValue(baseSku)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await screen.findByText('378g')

    await userEvent.click(screen.getByRole('button', { name: 'Edit variant' }))
    const nameInput = screen.getByPlaceholderText('Variant name')
    const editPanel = nameInput.closest('div')!.parentElement!
    await userEvent.clear(nameInput)
    await userEvent.type(nameInput, '412g')
    const priceInput = within(editPanel).getByPlaceholderText('0.00')
    await userEvent.clear(priceInput)
    await userEvent.type(priceInput, '24.99')
    await userEvent.click(within(editPanel).getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(pmsApi.updateVariant).toHaveBeenCalledWith('test-token', 'var-1', expect.objectContaining({
      name: '412g',
    })))
    expect(pmsApi.updateSKU).toHaveBeenCalledWith('test-token', 'sku-1', expect.objectContaining({
      sku_code: baseSku.sku_code, price_minor_units: 2499, barcode: baseSku.barcode, status: baseSku.status,
    }))
  })

  it('shows an error and keeps the form open when saving fails', async () => {
    const detail: ProductDetail = {
      product: product({ has_variants: true }),
      variants: [{ ...baseVariant, id: 'var-1', name: '378g', skus: [baseSku] }],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
    vi.mocked(pmsApi.updateVariant).mockRejectedValue(new Error('name already used'))

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await screen.findByText('378g')

    await userEvent.click(screen.getByRole('button', { name: 'Edit variant' }))
    const editPanel = screen.getByPlaceholderText('Variant name').closest('div')!.parentElement!
    await userEvent.click(within(editPanel).getByRole('button', { name: 'Save' }))

    expect(await screen.findByText('name already used')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('Variant name')).toBeInTheDocument()
  })

  it('Cancel closes the form without saving', async () => {
    const detail: ProductDetail = {
      product: product({ has_variants: true }),
      variants: [{ ...baseVariant, id: 'var-1', name: '378g', skus: [baseSku] }],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await screen.findByText('378g')

    await userEvent.click(screen.getByRole('button', { name: 'Edit variant' }))
    expect(screen.getByPlaceholderText('Variant name')).toBeInTheDocument()
    const editPanel = screen.getByPlaceholderText('Variant name').closest('div')!.parentElement!
    await userEvent.click(within(editPanel).getByRole('button', { name: 'Cancel' }))

    expect(screen.queryByPlaceholderText('Variant name')).not.toBeInTheDocument()
    expect(pmsApi.updateVariant).not.toHaveBeenCalled()
    expect(pmsApi.updateSKU).not.toHaveBeenCalled()
  })
})

describe('SKU currency overrides (VariantRow)', () => {
  it('is collapsed by default, loads and lists overrides once expanded, and can add one', async () => {
    const detail: ProductDetail = {
      product: product({ has_variants: true }),
      variants: [{ ...baseVariant, id: 'var-1', name: '378g', skus: [baseSku] }],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
    vi.mocked(pmsApi.listSKUCurrencyPrices).mockResolvedValue([
      { sku_id: 'sku-1', currency_code: 'USD', price_minor_units: 2199, updated_at: '' },
    ])
    vi.mocked(pmsApi.upsertSKUCurrencyPrice).mockResolvedValue({ sku_id: 'sku-1', currency_code: 'GBP', price_minor_units: 1799, updated_at: '' })

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await screen.findByText('378g')
    await userEvent.click(screen.getByRole('button', { name: 'Edit variant' }))

    expect(screen.queryByText(/currency overrides/i)).toBeInTheDocument()
    expect(pmsApi.listSKUCurrencyPrices).not.toHaveBeenCalled()

    await userEvent.click(screen.getByRole('button', { name: /currency overrides/i }))
    expect(await screen.findByText(/USD.*21\.99/)).toBeInTheDocument()

    const overridesPanel = screen.getByText(/USD.*21\.99/).closest('div')!.parentElement!
    await userEvent.selectOptions(within(overridesPanel).getByLabelText('Currency'), 'GBP')
    await userEvent.type(within(overridesPanel).getByPlaceholderText('0.00'), '17.99')
    await userEvent.click(within(overridesPanel).getByRole('button', { name: 'Add / update' }))

    await waitFor(() => expect(pmsApi.upsertSKUCurrencyPrice).toHaveBeenCalledWith('test-token', 'sku-1', {
      currency_code: 'GBP', price_minor_units: 1799, sale_price_minor_units: null,
    }))
  })

  it('can delete an existing override', async () => {
    const detail: ProductDetail = {
      product: product({ has_variants: true }),
      variants: [{ ...baseVariant, id: 'var-1', name: '378g', skus: [baseSku] }],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
    vi.mocked(pmsApi.listSKUCurrencyPrices).mockResolvedValue([
      { sku_id: 'sku-1', currency_code: 'USD', price_minor_units: 2199, updated_at: '' },
    ])
    vi.mocked(pmsApi.deleteSKUCurrencyPrice).mockResolvedValue(null)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await screen.findByText('378g')
    await userEvent.click(screen.getByRole('button', { name: 'Edit variant' }))
    await userEvent.click(screen.getByRole('button', { name: /currency overrides/i }))
    await screen.findByText(/USD.*21\.99/)

    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))

    await waitFor(() => expect(pmsApi.deleteSKUCurrencyPrice).toHaveBeenCalledWith('test-token', 'sku-1', 'USD'))
    expect(screen.queryByText(/USD.*21\.99/)).not.toBeInTheDocument()
  })
})

describe('alternate barcodes / scale codes (VariantRow)', () => {
  it('adds an alternate unit barcode for a SKU with no scale code', async () => {
    const detail: ProductDetail = {
      product: product({ has_variants: true }),
      variants: [{ ...baseVariant, id: 'var-1', name: '378g', skus: [baseSku] }],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
    vi.mocked(pmsApi.listBarcodeAliases).mockResolvedValue([])
    vi.mocked(pmsApi.createBarcodeAlias).mockResolvedValue({
      id: 'alias-1', sku_id: 'sku-1', code: '9876543210987', units_per_scan: 1, created_at: '', updated_at: '',
    })

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await screen.findByText('378g')

    await userEvent.click(screen.getByRole('button', { name: /alternate barcodes/i }))
    const panel = screen.getByText(/e.g. the same product repackaged/i).closest('div')!
    await userEvent.type(within(panel).getByLabelText('Barcode'), '9876543210987')
    await userEvent.click(within(panel).getByRole('button', { name: 'Add' }))

    await waitFor(() => expect(pmsApi.createBarcodeAlias).toHaveBeenCalledWith('test-token', 'sku-1', {
      code: '9876543210987', units_per_scan: 1,
    }))
    expect(await screen.findByText('9876543210987')).toBeInTheDocument()
  })

  it('deletes an existing alternate barcode', async () => {
    const detail: ProductDetail = {
      product: product({ has_variants: true }),
      variants: [{ ...baseVariant, id: 'var-1', name: '378g', skus: [baseSku] }],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
    vi.mocked(pmsApi.listBarcodeAliases).mockResolvedValue([
      { id: 'alias-1', sku_id: 'sku-1', code: '9876543210987', units_per_scan: 1, created_at: '', updated_at: '' },
    ])
    vi.mocked(pmsApi.deleteBarcodeAlias).mockResolvedValue(null)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await screen.findByText('378g')
    await userEvent.click(screen.getByRole('button', { name: /alternate barcodes/i }))
    await screen.findByText('9876543210987')

    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))

    await waitFor(() => expect(pmsApi.deleteBarcodeAlias).toHaveBeenCalledWith('test-token', 'alias-1'))
    expect(screen.queryByText('9876543210987')).not.toBeInTheDocument()
  })

  it('adds an additional scale code for a SKU that already has one', async () => {
    const scaleSku = { ...baseSku, scale_code: '94721', pricing_mode: 'per_weight' as const, price_per_kilo_minor_units: 4999 }
    const detail: ProductDetail = {
      product: product({ has_variants: true }),
      variants: [{ ...baseVariant, id: 'var-1', name: 'Dry Meat', skus: [scaleSku] }],
    }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
    vi.mocked(pmsApi.listScaleCodeAliases).mockResolvedValue([])
    vi.mocked(pmsApi.createScaleCodeAlias).mockResolvedValue({ id: 'sa-1', sku_id: 'sku-1', code: '94722', created_at: '', updated_at: '' })

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await screen.findByText('Dry Meat')

    await userEvent.click(screen.getByRole('button', { name: /additional scale codes/i }))
    const panel = screen.getByText(/second \(or third\) physical scale/i).closest('div')!
    await userEvent.type(within(panel).getByLabelText('Scale code'), '94722')
    await userEvent.click(within(panel).getByRole('button', { name: 'Add' }))

    await waitFor(() => expect(pmsApi.createScaleCodeAlias).toHaveBeenCalledWith('test-token', 'sku-1', { code: '94722' }))
  })
})

describe('weight pool (unit_type / portioned_from_sku_id)', () => {
  it('is shown for a brand-new, not-yet-saved product (no two-step create-then-edit needed)', async () => {
    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByText('New Product'))

    expect(await screen.findByText('Weight pool')).toBeInTheDocument()
    expect(screen.getByLabelText(/item type/i)).toBeInTheDocument()
  })

  it('weight-pool fields set on the create form are sent in the initial createSKU call', async () => {
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

    await userEvent.type(screen.getByLabelText('Name'), 'Dried Beef')
    await userEvent.type(screen.getByPlaceholderText('e.g. SHIRT-BLK-M'), 'DRIED-BEEF')
    await userEvent.type(screen.getByPlaceholderText('0.00'), '29.99')

    await userEvent.selectOptions(screen.getByLabelText(/item type/i), 'kilo')
    await userEvent.type(screen.getByLabelText(/scale code/i), '2900633')
    await userEvent.selectOptions(screen.getByLabelText(/pricing/i), 'per_weight')
    await userEvent.type(screen.getByLabelText(/price per kilo/i), '19.99')

    await userEvent.click(screen.getByRole('button', { name: 'Create Product' }))

    await waitFor(() => expect(pmsApi.createSKU).toHaveBeenCalled())
    expect(pmsApi.createSKU).toHaveBeenCalledWith('test-token', 'new-var', expect.objectContaining({
      unit_type: 'kilo', scale_code: '2900633', pricing_mode: 'per_weight', price_per_kilo_minor_units: 1999,
    }))
  })

  it('regression: does not fetch product detail until the just-created variant/SKU finish writing (race left Weight pool permanently hidden)', async () => {
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

    await waitFor(() => expect(pmsApi.getProduct).toHaveBeenCalled())

    const skuCallOrder = vi.mocked(pmsApi.createSKU).mock.invocationCallOrder[0]
    const getProductCallOrder = vi.mocked(pmsApi.getProduct).mock.invocationCallOrder[0]
    expect(getProductCallOrder).toBeGreaterThan(skuCallOrder)
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
    vi.mocked(integrationsApi.listProductChannelListings).mockResolvedValue([])

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await userEvent.click(await screen.findByRole('button', { name: /WooCommerce/ }))

    await waitFor(() => expect(screen.getByText(/mystore\.com/)).toBeInTheDocument())
    expect(await screen.findByText(/not linked to this WooCommerce store yet/i)).toBeInTheDocument()
    expect(integrationsApi.listProductChannelListings).toHaveBeenCalledWith('test-token', 'conn-1', 'prod-1')
  })

  it('links to an existing WooCommerce product by SKU and shows the resulting sync state', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [] })
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    vi.mocked(integrationsApi.listProductChannelListings).mockResolvedValue([])
    vi.mocked(integrationsApi.linkExistingWooProduct).mockResolvedValue({
      id: 'listing-1', product_id: 'prod-1', connection_id: 'conn-1', provider: 'woocommerce',
      external_id: '23609', sync_status: 'synced', last_synced_at: '2026-07-26T12:00:00Z',
      created_at: '2026-07-01T00:00:00Z', updated_at: '2026-07-26T12:00:00Z',
    })

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await userEvent.click(await screen.findByRole('button', { name: /WooCommerce/ }))
    await screen.findByText(/not linked to this WooCommerce store yet/i)

    await userEvent.type(screen.getByPlaceholderText('SKU on WooCommerce'), 'TSHIRT-1')
    await userEvent.click(screen.getByRole('button', { name: /Link existing product/i }))

    expect(integrationsApi.linkExistingWooProduct).toHaveBeenCalledWith('test-token', 'prod-1', 'conn-1', 'TSHIRT-1')
    expect(await screen.findByDisplayValue('23609')).toBeInTheDocument()
    expect(screen.queryByText(/not linked to this WooCommerce store yet/i)).not.toBeInTheDocument()
  })

  it('surfaces an error when linking to an existing WooCommerce product fails', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [] })
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    vi.mocked(integrationsApi.listProductChannelListings).mockResolvedValue([])
    vi.mocked(integrationsApi.linkExistingWooProduct).mockRejectedValue(new Error('this product already has a listing for this connection'))

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await userEvent.click(await screen.findByRole('button', { name: /WooCommerce/ }))
    await screen.findByText(/not linked to this WooCommerce store yet/i)

    await userEvent.type(screen.getByPlaceholderText('SKU on WooCommerce'), 'TSHIRT-1')
    await userEvent.click(screen.getByRole('button', { name: /Link existing product/i }))

    expect(await screen.findByText('this product already has a listing for this connection')).toBeInTheDocument()
  })

  it('shows real sync state when the product is linked to this connection', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [] })
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    vi.mocked(integrationsApi.listProductChannelListings).mockResolvedValue([{
      id: 'listing-1', product_id: 'prod-1', connection_id: 'conn-1', provider: 'woocommerce',
      external_id: '23609', sync_status: 'synced', last_synced_at: '2026-07-26T12:00:00Z',
      created_at: '2026-07-01T00:00:00Z', updated_at: '2026-07-26T12:00:00Z',
    }])

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
    vi.mocked(integrationsApi.listProductChannelListings).mockRejectedValue(new Error('internal server error'))

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
    vi.mocked(integrationsApi.listProductChannelListings).mockResolvedValue([])

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
    vi.mocked(integrationsApi.listProductChannelListings).mockResolvedValue([])

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
    vi.mocked(integrationsApi.listProductChannelListings).mockResolvedValue([])

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await userEvent.click(await screen.findByRole('button', { name: /WooCommerce/ }))

    await screen.findByText(/not linked to this WooCommerce store yet/i)
    expect(screen.queryByLabelText('Tax class (Woo)')).not.toBeInTheDocument()
  })

  it('shows one entry per variant for a multi-variant product instead of a single summary', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({
      product: product({ has_variants: true }),
      variants: [
        { ...baseVariant, id: 'var-1', name: '378g', skus: [{ ...baseSku, id: 'sku-1', variant_id: 'var-1', sku_code: 'CARNE-378' }] },
        { ...baseVariant, id: 'var-2', name: '412g', skus: [{ ...baseSku, id: 'sku-2', variant_id: 'var-2', sku_code: 'CARNE-412' }] },
      ],
    })
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    vi.mocked(integrationsApi.listProductChannelListings).mockResolvedValue([
      { id: 'listing-1', product_id: 'prod-1', connection_id: 'conn-1', provider: 'woocommerce', variant_id: 'var-1', external_id: '100', sync_status: 'synced', created_at: '', updated_at: '' },
    ])

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await userEvent.click(await screen.findByRole('button', { name: /WooCommerce/ }))

    const tab = within(await screen.findByTestId('woocommerce-tab'))
    expect(tab.getByText('378g')).toBeInTheDocument()
    expect(tab.getByText('412g')).toBeInTheDocument()
    expect(tab.getByText('CARNE-378')).toBeInTheDocument()
    expect(tab.getByText('CARNE-412')).toBeInTheDocument()
    expect(tab.getByText(/Woo ID 100/)).toBeInTheDocument()
    expect(tab.getByText('Not linked')).toBeInTheDocument()
    expect(tab.queryByText(/not linked to this WooCommerce store yet/i)).not.toBeInTheDocument()
    expect(tab.queryByPlaceholderText('SKU on WooCommerce')).not.toBeInTheDocument()
  })
})

describe('Resync to channels button', () => {
  it('is not shown for a brand-new, not-yet-saved product', async () => {
    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByText('New Product'))

    expect(screen.queryByRole('button', { name: /Resync to channels/ })).not.toBeInTheDocument()
  })

  it('is not shown when no WooCommerce store is connected', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [] })
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([])

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))

    await waitFor(() => expect(pmsApi.getProduct).toHaveBeenCalled())
    expect(screen.queryByRole('button', { name: /Resync to channels/ })).not.toBeInTheDocument()
  })

  it('force-pushes the product to every linked channel when clicked', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [] })
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    vi.mocked(integrationsApi.getProductChannelListing).mockResolvedValue(null)
    vi.mocked(integrationsApi.resyncProductToChannels).mockResolvedValue(null)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))

    await userEvent.click(await screen.findByRole('button', { name: /Resync to channels/ }))

    expect(integrationsApi.resyncProductToChannels).toHaveBeenCalledWith('test-token', 'prod-1')
    expect(await screen.findByText('Resync started.')).toBeInTheDocument()
  })

  it('surfaces an error instead of a false success message', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [] })
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    vi.mocked(integrationsApi.getProductChannelListing).mockResolvedValue(null)
    vi.mocked(integrationsApi.resyncProductToChannels).mockRejectedValue(new Error('woo unreachable'))

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))

    await userEvent.click(await screen.findByRole('button', { name: /Resync to channels/ }))

    expect(await screen.findByText('woo unreachable')).toBeInTheDocument()
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
      prices_include_tax: true, default_tax_country: 'NL', inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_markup_basis_points: 0, updated_at: '',
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

  it('uploads every selected file when multiple are chosen at once', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [] })
    vi.mocked(pmsApi.listProductMedia).mockResolvedValue([])
    vi.mocked(pmsApi.uploadProductMedia).mockImplementation(async (_token, _id, file) =>
      ({ ...mediaItem, id: file.name, url: `https://api.example.com/uploads/t1/${file.name}` }))

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByText('T-Shirt'))
    await waitFor(() => expect(pmsApi.listProductMedia).toHaveBeenCalled())

    const fileA = new File(['a'], 'a.jpg', { type: 'image/jpeg' })
    const fileB = new File(['b'], 'b.jpg', { type: 'image/jpeg' })
    const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement
    await userEvent.upload(fileInput, [fileA, fileB])

    await waitFor(() => expect(pmsApi.uploadProductMedia).toHaveBeenCalledTimes(2))
    expect(pmsApi.uploadProductMedia).toHaveBeenCalledWith('test-token', 'prod-1', fileA, undefined)
    expect(pmsApi.uploadProductMedia).toHaveBeenCalledWith('test-token', 'prod-1', fileB, undefined)
  })

  it('uploads a file dropped onto the media drop zone', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [] })
    vi.mocked(pmsApi.listProductMedia).mockResolvedValue([])
    const uploaded = { ...mediaItem, id: 'media-2' }
    vi.mocked(pmsApi.uploadProductMedia).mockResolvedValue(uploaded)

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByText('T-Shirt'))
    await waitFor(() => expect(pmsApi.listProductMedia).toHaveBeenCalled())

    const file = new File(['bytes'], 'dropped.jpg', { type: 'image/jpeg' })
    const dropZone = screen.getByTestId('media-dropzone')
    fireEvent.drop(dropZone, { dataTransfer: { files: [file] } })

    await waitFor(() => expect(pmsApi.uploadProductMedia).toHaveBeenCalledWith('test-token', 'prod-1', file, undefined))
  })

  it('uploads every file dragged in together, in one drop', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [] })
    vi.mocked(pmsApi.listProductMedia).mockResolvedValue([])
    vi.mocked(pmsApi.uploadProductMedia).mockImplementation(async (_token, _id, file) =>
      ({ ...mediaItem, id: file.name, url: `https://api.example.com/uploads/t1/${file.name}` }))

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByText('T-Shirt'))
    await waitFor(() => expect(pmsApi.listProductMedia).toHaveBeenCalled())

    const fileA = new File(['a'], 'a.jpg', { type: 'image/jpeg' })
    const fileB = new File(['b'], 'b.jpg', { type: 'image/jpeg' })
    const dropZone = screen.getByTestId('media-dropzone')
    fireEvent.drop(dropZone, { dataTransfer: { files: [fileA, fileB] } })

    await waitFor(() => expect(pmsApi.uploadProductMedia).toHaveBeenCalledTimes(2))
    expect(pmsApi.uploadProductMedia).toHaveBeenCalledWith('test-token', 'prod-1', fileA, undefined)
    expect(pmsApi.uploadProductMedia).toHaveBeenCalledWith('test-token', 'prod-1', fileB, undefined)
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

  it('prefills the override input with the computed cost price instead of starting blank', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [{ ...baseVariant, skus: [baseSku] }] })
    vi.mocked(pmsApi.getSKUMarginInsight).mockResolvedValue({
      cost_price_available: true, cost_price_minor_units: 700, cost_price_source: 'computed', effective_target_markup_basis_points: 3000,
      recommended_price_minor_units: 910, profit_at_price_minor_units: 300, margin_at_price_basis_points: 3000, markup_at_price_basis_points: 4285,
      total_on_hand_quantity: 0, total_profit_potential_minor_units: 0, shrinkage_minor_units: 0,
    })

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))

    await screen.findByText('Override cost price')
    expect(screen.getByPlaceholderText('USD')).toHaveValue(7)
    expect(screen.queryByText('Set cost price manually')).not.toBeInTheDocument()
  })

  it('shows a revert control and calls clearCostPriceOverride when the cost price is a manual override', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [{ ...baseVariant, skus: [baseSku] }] })
    vi.mocked(pmsApi.getSKUMarginInsight).mockResolvedValue({
      cost_price_available: true, cost_price_minor_units: 1250, cost_price_source: 'manual', effective_target_markup_basis_points: 3000,
      recommended_price_minor_units: 1786, profit_at_price_minor_units: 749, margin_at_price_basis_points: 3746, markup_at_price_basis_points: 5992,
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

describe('Margin insight shows both markup and profit, with and without BTW', () => {
  it('shows margin % and markup % side by side for profit at current price', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [{ ...baseVariant, skus: [baseSku] }] })
    vi.mocked(pmsApi.getSKUMarginInsight).mockResolvedValue({
      cost_price_available: true, cost_price_minor_units: 800, cost_price_source: 'computed', effective_target_markup_basis_points: 2500,
      recommended_price_minor_units: 1000, profit_at_price_minor_units: 450, margin_at_price_basis_points: 3600, markup_at_price_basis_points: 5625,
      total_on_hand_quantity: 0, total_profit_potential_minor_units: 0, shrinkage_minor_units: 0,
    })

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))

    expect(await screen.findByText(/36\.0% margin \/ 56\.3% markup/)).toBeInTheDocument()
    expect(screen.getByText(/25\.0% target markup/)).toBeInTheDocument()
  })

  it('shows excl./incl. BTW profit and margin/markup breakdown when a tax rate is resolvable', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [{ ...baseVariant, skus: [baseSku] }] })
    vi.mocked(pmsApi.getSKUMarginInsight).mockResolvedValue({
      cost_price_available: true, cost_price_minor_units: 800, cost_price_source: 'computed', effective_target_markup_basis_points: 2500,
      recommended_price_minor_units: 1000, profit_at_price_minor_units: 450, margin_at_price_basis_points: 3600, markup_at_price_basis_points: 5625,
      total_on_hand_quantity: 0, total_profit_potential_minor_units: 0, shrinkage_minor_units: 0,
    })
    vi.mocked(tenantSettingsApi.getTenantSettings).mockResolvedValue({
      tenant_id: 't1', default_currency: 'USD', timezone: '', date_format: '', language: '',
      prices_include_tax: true, default_tax_country: 'NL', inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_markup_basis_points: 0, updated_at: '',
    })
    vi.mocked(taxApi.listTaxClasses).mockResolvedValue([
      { id: 'class-1', name: 'Standard', is_default: true, created_at: '', updated_at: '' },
    ])
    vi.mocked(taxApi.listTaxRates).mockResolvedValue([
      { id: 'rate-1', tax_class_id: 'class-1', country: 'NL', rate_basis_points: 2500, created_at: '', updated_at: '' },
    ])

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))

    // stored price (cost 800 + profit 450 = 1250) is tax-inclusive per
    // prices_include_tax:true; at 25% rate: excl. = 1000 (profit 200, 20.0%
    // margin / 25.0% markup), incl. = 1250 (profit 450, 36.0% margin / 56.3%
    // markup, matching the backend-supplied top-line figures).
    expect(await screen.findByText(/Excl\. BTW: USD 2\.00 \(20\.0% margin \/ 25\.0% markup\)/)).toBeInTheDocument()
    expect(screen.getByText(/Incl\. BTW: USD 4\.50 \(36\.0% margin \/ 56\.3% markup\)/)).toBeInTheDocument()
  })
})

describe('Price history card', () => {
  it('shows an empty state when the SKU has no recorded price changes', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [{ ...baseVariant, skus: [baseSku] }] })
    vi.mocked(pmsApi.listSKUPriceHistory).mockResolvedValue([])

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))

    expect(await screen.findByText('No price changes recorded yet.')).toBeInTheDocument()
  })

  it('shows each interval with its reason, marking the open interval as Current', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [{ ...baseVariant, skus: [baseSku] }] })
    vi.mocked(pmsApi.listSKUPriceHistory).mockResolvedValue([
      {
        id: 'hist-2', sku_id: 'sku-1', price_minor_units: 1049, sale_price_minor_units: null,
        currency_code: 'USD', valid_from: '2026-03-01T00:00:00Z', valid_until: null, reason: 'manual_update',
      },
      {
        id: 'hist-1', sku_id: 'sku-1', price_minor_units: 999, sale_price_minor_units: 799,
        currency_code: 'USD', valid_from: '2026-01-01T00:00:00Z', valid_until: '2026-03-01T00:00:00Z', reason: 'created',
      },
    ])

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))

    await screen.findByText('Price history')
    expect(screen.getByText('Current')).toBeInTheDocument()
    expect(screen.getByText('Manual edit')).toBeInTheDocument()
    expect(screen.getByText('Created')).toBeInTheDocument()
    expect(screen.getByText('USD 7.99')).toBeInTheDocument()
  })

  it('shows an error if loading price history fails', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [{ ...baseVariant, skus: [baseSku] }] })
    vi.mocked(pmsApi.listSKUPriceHistory).mockRejectedValue(new Error('internal server error'))

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))

    expect(await screen.findByText('internal server error')).toBeInTheDocument()
  })
})

describe('Change history card', () => {
  it('shows an empty state when nothing has changed yet', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [{ ...baseVariant, skus: [baseSku] }] })
    vi.mocked(pmsApi.listProductHistory).mockResolvedValue([])

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))

    expect(await screen.findByText('No changes recorded yet.')).toBeInTheDocument()
  })

  it('shows each recorded name/status change', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [{ ...baseVariant, skus: [baseSku] }] })
    vi.mocked(pmsApi.listProductHistory).mockResolvedValue([
      { id: 'h1', entity_type: 'product', entity_id: 'p1', field: 'status', old_value: 'draft', new_value: 'active', changed_at: '2026-03-01T00:00:00Z' },
      { id: 'h2', entity_type: 'product', entity_id: 'p1', field: 'name', old_value: 'Old Name', new_value: 'New Name', changed_at: '2026-02-01T00:00:00Z' },
    ])

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))

    await screen.findByText('Change history')
    expect(screen.getByText('draft')).toBeInTheDocument()
    expect(screen.getByText('active')).toBeInTheDocument()
    expect(screen.getByText('Old Name')).toBeInTheDocument()
    expect(screen.getByText('New Name')).toBeInTheDocument()
  })

  it('shows an error if loading change history fails', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [{ ...baseVariant, skus: [baseSku] }] })
    vi.mocked(pmsApi.listProductHistory).mockRejectedValue(new Error('history fetch failed'))

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))

    expect(await screen.findByText('history fetch failed')).toBeInTheDocument()
  })
})

describe('editing SKU code on an already-saved SKU', () => {
  // Regression: UpdateSKUPayload had no sku_code field at all, so every
  // save site (this form included) silently dropped whatever the user
  // typed into the SKU code input -- most visible right after duplicating
  // a product, since the auto-generated "<code>-COPY" is exactly what a
  // user wants to replace immediately.
  it('sends the edited SKU code on save', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [{ ...baseVariant, skus: [baseSku] }] })
    vi.mocked(pmsApi.updateProduct).mockResolvedValue(product())
    vi.mocked(pmsApi.updateSKU).mockResolvedValue({ ...baseSku, sku_code: 'NEW-CODE' })

    render(<MemoryRouter><AppProvider><Products /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Edit'))

    const skuCodeInput = await screen.findByDisplayValue('TSHIRT-1')
    await userEvent.clear(skuCodeInput)
    await userEvent.type(skuCodeInput, 'NEW-CODE')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(pmsApi.updateSKU).toHaveBeenCalledWith('test-token', 'sku-1', expect.objectContaining({
      sku_code: 'NEW-CODE',
    })))
  })
})
