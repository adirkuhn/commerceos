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
