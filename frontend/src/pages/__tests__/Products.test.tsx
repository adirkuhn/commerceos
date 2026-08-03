// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import Products from '../Products'
import { AppProvider } from '../../context/AppContext'
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
  getProduct: vi.fn(),
  createVariant: vi.fn(),
  updateVariant: vi.fn(),
  deleteVariant: vi.fn(),
  createSKU: vi.fn(),
  updateSKU: vi.fn(),
  deleteSKU: vi.fn(),
  listProductMedia: vi.fn(),
  uploadProductMedia: vi.fn(),
  updateMedia: vi.fn(),
  deleteMedia: vi.fn(),
}))

vi.mock('../../integrations/api', () => ({
  listConnections: vi.fn(),
}))

import * as pmsApi from '../../pms/api'
import * as integrationsApi from '../../integrations/api'

function product(overrides: Partial<Product> = {}): Product {
  return {
    id: 'prod-1', tenant_id: 't1', name: 'T-Shirt', slug: 't-shirt',
    description: '', status: 'active', has_variants: false, created_at: '', updated_at: '',
    ...overrides,
  }
}

function category(): Category {
  return { id: 'cat-1', tenant_id: 't1', name: 'Apparel', slug: 'apparel', position: 0, created_at: '', updated_at: '' }
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

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(pmsApi.listProducts).mockResolvedValue([product()])
  vi.mocked(pmsApi.listCategories).mockResolvedValue([category()])
  vi.mocked(integrationsApi.listConnections).mockResolvedValue([])
  vi.mocked(pmsApi.listProductMedia).mockResolvedValue([])
})

describe('product list', () => {
  it('opens the edit page when clicking anywhere on a row, not just the Edit button', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [] })

    render(<Products />)
    await userEvent.click(await screen.findByText('T-Shirt'))

    await waitFor(() => expect(pmsApi.getProduct).toHaveBeenCalledWith('test-token', 'prod-1'))
    expect(screen.getByDisplayValue('T-Shirt')).toBeInTheDocument()
  })

  it('deleting a row does not also navigate to the edit page', async () => {
    vi.mocked(pmsApi.deleteProduct).mockResolvedValue(null)
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true)

    render(<Products />)
    await userEvent.click(await screen.findByTitle('Delete'))

    await waitFor(() => expect(pmsApi.deleteProduct).toHaveBeenCalledWith('test-token', 'prod-1'))
    expect(pmsApi.getProduct).not.toHaveBeenCalled()

    confirmSpy.mockRestore()
  })

  it('paginates when there are more than 20 products', async () => {
    const many = Array.from({ length: 25 }, (_, i) => product({ id: `prod-${i}`, name: `Product ${String(i).padStart(2, '0')}`, slug: `product-${i}` }))
    vi.mocked(pmsApi.listProducts).mockResolvedValue(many)

    render(<Products />)
    await screen.findByText('Product 00')

    expect(screen.queryByText('Product 20')).not.toBeInTheDocument()
    expect(screen.getByText('Page 1 of 2')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Next page' }))

    expect(await screen.findByText('Product 20')).toBeInTheDocument()
    expect(screen.queryByText('Product 00')).not.toBeInTheDocument()
  })

  it('resets to page 1 when the search filter changes', async () => {
    const many = Array.from({ length: 25 }, (_, i) => product({ id: `prod-${i}`, name: `Product ${String(i).padStart(2, '0')}`, slug: `product-${i}` }))
    vi.mocked(pmsApi.listProducts).mockResolvedValue(many)

    render(<Products />)
    await screen.findByText('Product 00')
    await userEvent.click(screen.getByRole('button', { name: 'Next page' }))
    await screen.findByText('Product 20')

    await userEvent.type(screen.getByPlaceholderText('Search by name or slug...'), 'Product 01')

    expect(await screen.findByText('Product 01')).toBeInTheDocument()
  })
})

describe('editing an existing product infers variants mode from variant count', () => {
  it('shows the flat Pricing & SKU form for a product with 0 variants (not the variants manager)', async () => {
    const detail: ProductDetail = { product: product(), variants: [] }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)

    render(<Products />)
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

    render(<Products />)
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

    render(<Products />)
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

    const { container } = render(<Products />)
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

    render(<Products />)
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

    render(<Products />)
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

    render(<Products />)
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

    render(<Products />)
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

    render(<Products />)
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

describe('WooCommerce tab', () => {
  it('is not shown for a brand-new, not-yet-saved product', async () => {
    render(<AppProvider><Products /></AppProvider>)
    await userEvent.click(await screen.findByText('New Product'))

    expect(screen.queryByRole('button', { name: /WooCommerce/ })).not.toBeInTheDocument()
  })

  it('is not shown when no WooCommerce store is connected', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [] })
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([])

    render(<AppProvider><Products /></AppProvider>)
    await userEvent.click(await screen.findByTitle('Edit'))

    await waitFor(() => expect(pmsApi.getProduct).toHaveBeenCalled())
    expect(screen.queryByRole('button', { name: /WooCommerce/ })).not.toBeInTheDocument()
  })

  it('is not shown when a WooCommerce connection exists but is in an error state', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [] })
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection({ status: 'error' })])

    render(<AppProvider><Products /></AppProvider>)
    await userEvent.click(await screen.findByTitle('Edit'))

    await waitFor(() => expect(pmsApi.getProduct).toHaveBeenCalled())
    expect(screen.queryByRole('button', { name: /WooCommerce/ })).not.toBeInTheDocument()
  })

  it('shows the channel field shell for a connected WooCommerce store', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [] })
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])

    render(<AppProvider><Products /></AppProvider>)
    await userEvent.click(await screen.findByTitle('Edit'))
    await userEvent.click(await screen.findByRole('button', { name: /WooCommerce/ }))

    await waitFor(() => expect(screen.getByText(/mystore\.com/)).toBeInTheDocument())
    expect(screen.getByPlaceholderText('Not linked yet')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('Not linked yet')).toBeDisabled()
  })
})

describe('SKU weight/dimensions and virtual flag', () => {
  it('shows weight/dimensions fields and a virtual-product toggle for a simple product', async () => {
    render(<Products />)
    await userEvent.click(await screen.findByText('New Product'))

    expect(screen.getByLabelText('Weight (g)')).toBeInTheDocument()
    expect(screen.getByLabelText('Length (mm)')).toBeInTheDocument()
    expect(screen.getByLabelText('Width (mm)')).toBeInTheDocument()
    expect(screen.getByLabelText('Height (mm)')).toBeInTheDocument()
    expect(screen.getByRole('switch', { name: 'Virtual product (no shipping)' })).toBeInTheDocument()
  })

  it('hides weight/dimensions fields once the product is marked virtual', async () => {
    render(<Products />)
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

    render(<Products />)
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

    render(<Products />)
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

describe('product edit page chrome', () => {
  it('Discard closes the editor without saving', async () => {
    render(<Products />)
    await userEvent.click(await screen.findByText('New Product'))

    await userEvent.type(screen.getByLabelText('Name'), 'Abandoned Product')
    await userEvent.click(screen.getByRole('button', { name: 'Discard' }))

    expect(screen.queryByLabelText('Name')).not.toBeInTheDocument()
    expect(pmsApi.createProduct).not.toHaveBeenCalled()
  })

  it('shows the not-yet-supported fields as disabled placeholders', async () => {
    render(<Products />)
    await userEvent.click(await screen.findByText('New Product'))

    for (const label of ['Vendor', 'Collections', 'Tags', 'Page title', 'Meta description']) {
      const field = screen.getByLabelText(label)
      expect(field).toBeInTheDocument()
      expect(field).toBeDisabled()
    }
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

    render(<Products />)
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

    render(<Products />)
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

    render(<Products />)
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

    render(<Products />)
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
    render(<Products />)
    await userEvent.click(await screen.findByText('New Product'))

    expect(screen.getByText('Save the product first, then add images.')).toBeInTheDocument()
  })

  it('lists existing media and uploads a new file for an existing product', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [] })
    vi.mocked(pmsApi.listProductMedia).mockResolvedValue([mediaItem])
    const uploaded = { ...mediaItem, id: 'media-2', url: 'https://api.example.com/uploads/t1/y.jpg' }
    vi.mocked(pmsApi.uploadProductMedia).mockResolvedValue(uploaded)

    render(<Products />)
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

    render(<Products />)
    await userEvent.click(await screen.findByText('T-Shirt'))
    await screen.findByAltText('')

    await userEvent.click(screen.getByRole('button', { name: 'Remove image' }))

    await waitFor(() => expect(pmsApi.deleteMedia).toHaveBeenCalledWith('test-token', 'media-1'))
  })

  it('clicking an image opens a title editor prefilled with its current alt text, and saving persists it', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: product(), variants: [] })
    vi.mocked(pmsApi.listProductMedia).mockResolvedValue([{ ...mediaItem, alt_text: 'Old title' }])
    vi.mocked(pmsApi.updateMedia).mockResolvedValue({ ...mediaItem, alt_text: 'Front view' })

    render(<Products />)
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

    render(<Products />)
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

    render(<Products />)
    await userEvent.click(await screen.findByText('T-Shirt'))
    await screen.findByAltText('')

    await userEvent.click(screen.getByRole('button', { name: 'Edit image title' }))
    await userEvent.click(screen.getByRole('button', { name: 'Save title' }))

    expect(await screen.findByText('title too long')).toBeInTheDocument()
  })
})
