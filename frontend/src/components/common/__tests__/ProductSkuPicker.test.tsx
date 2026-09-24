// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ProductSkuPicker } from '../ProductSkuPicker'

vi.mock('../../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../../pms/api', () => ({
  listProducts: vi.fn(),
  getProduct: vi.fn(),
}))

import * as pmsApi from '../../../pms/api'

const product = {
  id: 'product-1', tenant_id: 'tenant-1', category_id: null, name: 'Acai 1L',
  slug: 'acai-1l', description: '', status: 'active' as const, has_variants: false, tags: [],
  created_at: '', updated_at: '',
}

const sku = {
  id: 'sku-1', tenant_id: 'tenant-1', variant_id: 'variant-1', sku_code: 'ACAI-1L',
  price_minor_units: 999, currency_code: 'EUR', is_virtual: false, status: 'active',
  created_at: '', updated_at: '',
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(pmsApi.listProducts).mockResolvedValue({ products: [product], total: 1 } as never)
})

describe('ProductSkuPicker', () => {
  it('searches products by name and selects a SKU with its product name', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({
      product,
      variants: [{ id: 'variant-1', tenant_id: 'tenant-1', product_id: 'product-1', name: 'Default', position: 0, created_at: '', updated_at: '', skus: [sku] }],
    } as never)
    const onSelect = vi.fn()

    render(<ProductSkuPicker onSelect={onSelect} onClose={vi.fn()} />)
    await userEvent.type(screen.getByPlaceholderText('Search products by name…'), 'Acai')
    await userEvent.click(await screen.findByRole('button', { name: 'Acai 1L' }))
    await userEvent.click(await screen.findByText('ACAI-1L'))

    expect(pmsApi.listProducts).toHaveBeenCalledWith('test-token', expect.objectContaining({ search: 'Acai' }))
    expect(onSelect).toHaveBeenCalledWith(expect.objectContaining({ id: 'sku-1' }), 'Acai 1L — ACAI-1L', 'Acai 1L')
  })

  it('shows an empty state when the product has no SKUs', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product, variants: [] } as never)

    render(<ProductSkuPicker onSelect={vi.fn()} onClose={vi.fn()} />)
    await userEvent.click(await screen.findByRole('button', { name: 'Acai 1L' }))

    expect(await screen.findByText('No SKUs on this product')).toBeInTheDocument()
  })

  it('calls onClose from the close button', async () => {
    const onClose = vi.fn()
    render(<ProductSkuPicker onSelect={vi.fn()} onClose={onClose} />)
    await userEvent.click(screen.getByRole('button', { name: 'Close' }))
    expect(onClose).toHaveBeenCalled()
  })
})
