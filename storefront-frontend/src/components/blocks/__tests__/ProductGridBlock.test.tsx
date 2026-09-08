// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { ProductGridBlock } from '../ProductGridBlock'
import type { StorefrontProductListResult } from '../../../types'

vi.mock('../../../api/products', () => ({ listProducts: vi.fn(), getProduct: vi.fn() }))

import * as productsApi from '../../../api/products'

beforeEach(() => { vi.clearAllMocks() })

describe('ProductGridBlock', () => {
  it('renders each enrolled product with its price', async () => {
    const result: StorefrontProductListResult = {
      items: [{ id: 'p1', slug: 'p1', name: 'Widget', category_id: null, tags: [], image_url: null, price_minor_units: 1999, currency_code: 'EUR' }],
      total: 1,
    }
    vi.mocked(productsApi.listProducts).mockResolvedValue(result)

    render(<MemoryRouter><ProductGridBlock /></MemoryRouter>)

    expect(await screen.findByText('Widget')).toBeInTheDocument()
    expect(screen.getByText('€19.99')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Widget/ })).toHaveAttribute('href', '/products/p1')
  })

  it('shows a sale price struck through against the original', async () => {
    vi.mocked(productsApi.listProducts).mockResolvedValue({
      items: [{ id: 'p1', slug: 'p1', name: 'Widget', category_id: null, tags: [], image_url: null, price_minor_units: 2000, sale_price_minor_units: 1500, currency_code: 'EUR' }],
      total: 1,
    })

    render(<MemoryRouter><ProductGridBlock /></MemoryRouter>)

    expect(await screen.findByText('€20.00')).toBeInTheDocument()
    expect(screen.getByText('€15.00')).toBeInTheDocument()
  })

  it('shows an empty state when there are no products', async () => {
    vi.mocked(productsApi.listProducts).mockResolvedValue({ items: [], total: 0 })

    render(<MemoryRouter><ProductGridBlock /></MemoryRouter>)

    expect(await screen.findByText('No products yet.')).toBeInTheDocument()
  })
})
