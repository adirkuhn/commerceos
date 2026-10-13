// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ProductLowStockCard, isWeighedProduct } from '../ProductLowStockCard'
import type { Product, VariantWithSKUs } from '../types'
import { CodedApiError } from '../../api/codedApiError'

vi.mock('../../auth/AuthContext', () => ({ useAuth: () => ({ token: 'tok' }) }))
vi.mock('../api', () => ({ setProductLowStockThreshold: vi.fn() }))
vi.mock('../../settings/api', () => ({ getTenantSettings: vi.fn(async () => ({ low_stock_threshold: 12 })), updateTenantSettings: vi.fn() }))
import * as api from '../api'

const product = { id: 'p1', name: 'Farofa', low_stock_threshold: null } as unknown as Product

beforeEach(() => vi.clearAllMocks())

describe('ProductLowStockCard', () => {
  it("shows the shop's line as the default when the product has none", async () => {
    render(<ProductLowStockCard product={product} weighed={false} />)

    const input = await screen.findByLabelText('Low stock at or below (pieces)')
    expect(input).toHaveValue(null)
    expect(input).toHaveAttribute('placeholder', 'Shop default: 12')
  })

  it("saves the product's own line", async () => {
    vi.mocked(api.setProductLowStockThreshold).mockResolvedValue({ ...product, low_stock_threshold: 4 })
    const user = userEvent.setup()
    render(<ProductLowStockCard product={product} weighed={false} />)

    await user.type(await screen.findByLabelText('Low stock at or below (pieces)'), '4')
    await user.click(screen.getByRole('button', { name: 'Save low stock' }))

    expect(api.setProductLowStockThreshold).toHaveBeenCalledWith('tok', 'p1', 4)
    expect(await screen.findByText('Saved.')).toBeInTheDocument()
  })

  it('emptying the field goes back to the shop default', async () => {
    vi.mocked(api.setProductLowStockThreshold).mockResolvedValue({ ...product, low_stock_threshold: null })
    const user = userEvent.setup()
    render(<ProductLowStockCard product={{ ...product, low_stock_threshold: 7 }} weighed={false} />)

    const input = await screen.findByLabelText('Low stock at or below (pieces)')
    expect(input).toHaveValue(7)
    await user.clear(input)
    await user.click(screen.getByRole('button', { name: 'Save low stock' }))

    expect(api.setProductLowStockThreshold).toHaveBeenCalledWith('tok', 'p1', null)
  })

  // Weighed stock is stored in grams; people think in kg.
  it('a weighed product is entered in kg and saved in grams', async () => {
    vi.mocked(api.setProductLowStockThreshold).mockResolvedValue({ ...product, low_stock_threshold: 1500 })
    const user = userEvent.setup()
    render(<ProductLowStockCard product={{ ...product, low_stock_threshold: 2000 }} weighed />)

    const input = await screen.findByLabelText('Low stock at or below (kg)')
    expect(input).toHaveValue(2)
    expect(input).toHaveAttribute('placeholder', 'Shop default: 0.012')
    await user.clear(input)
    await user.type(input, '1.5')
    await user.click(screen.getByRole('button', { name: 'Save low stock' }))

    expect(api.setProductLowStockThreshold).toHaveBeenCalledWith('tok', 'p1', 1500)
  })

  it('shows a save error from the API', async () => {
    vi.mocked(api.setProductLowStockThreshold).mockRejectedValue(new CodedApiError('PRODUCT_INVALID_LOW_STOCK_THRESHOLD', 'bad'))
    const user = userEvent.setup()
    render(<ProductLowStockCard product={product} weighed={false} />)

    await user.type(await screen.findByLabelText('Low stock at or below (pieces)'), '3')
    await user.click(screen.getByRole('button', { name: 'Save low stock' }))

    expect(await screen.findByRole('alert')).toHaveTextContent("The product's low stock threshold must be 0 or more.")
  })
})

describe('isWeighedProduct', () => {
  const variant = (...unitTypes: ('kilo' | 'unit' | undefined)[]) =>
    ({ skus: unitTypes.map(unit_type => ({ unit_type })) }) as unknown as VariantWithSKUs

  it('is true only when every SKU is weighed', () => {
    expect(isWeighedProduct([variant('kilo'), variant('kilo')])).toBe(true)
    expect(isWeighedProduct([variant('kilo', 'unit')])).toBe(false)
    expect(isWeighedProduct([variant(undefined)])).toBe(false)
    expect(isWeighedProduct([])).toBe(false)
  })
})
