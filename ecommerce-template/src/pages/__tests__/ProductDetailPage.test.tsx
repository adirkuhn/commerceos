// @vitest-environment jsdom
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { render } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { describe, expect, it, beforeEach } from 'vitest'
import ProductDetailPage from '../ProductDetailPage'
import { TooltipProvider } from '@/components/ui/Tooltip'
import { useCartStore } from '@/stores/cartStore'
import { PRODUCTS } from '@/config/demoData'

function renderProductPage(slug: string) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={queryClient}>
      <TooltipProvider>
        <MemoryRouter initialEntries={[`/product/${slug}`]}>
          <Routes><Route path="/product/:slug" element={<ProductDetailPage />} /></Routes>
        </MemoryRouter>
      </TooltipProvider>
    </QueryClientProvider>,
  )
}

const variableProduct = PRODUCTS.find(p => p.variantOptionNames.length > 0)!
const simpleProduct = PRODUCTS.find(p => p.variantOptionNames.length === 0)!

beforeEach(() => {
  useCartStore.setState({ cartId: `test-cart-${Math.random()}` })
})

describe('ProductDetailPage', () => {
  it('renders product info for a simple product and adds it to the cart', async () => {
    const user = userEvent.setup()
    renderProductPage(simpleProduct.slug)

    expect(await screen.findByRole('heading', { name: simpleProduct.name })).toBeInTheDocument()
    expect(screen.getAllByText(new RegExp(`SKU ${simpleProduct.sku}`)).length).toBeGreaterThan(0)

    await user.click(screen.getByRole('button', { name: 'Add to cart' }))
    await waitFor(() => expect(useCartStore.getState().isDrawerOpen).toBe(true))
  })

  it('disables add to cart for a variable product until size and color are picked', async () => {
    const user = userEvent.setup()
    renderProductPage(variableProduct.slug)

    await screen.findByRole('heading', { name: variableProduct.name })
    expect(screen.getByRole('button', { name: 'Add to cart' })).toBeDisabled()

    const sizeValue = variableProduct.variants[0].options.find(o => o.name === 'Size')!.value
    const colorValue = variableProduct.variants.find(v => v.stock > 0 && v.options.some(o => o.name === 'Size' && o.value === sizeValue))!.options.find(o => o.name === 'Color')!.value

    await user.click(screen.getByRole('radio', { name: sizeValue }))
    await user.click(screen.getByRole('radio', { name: colorValue }))

    await waitFor(() => expect(screen.getByRole('button', { name: 'Add to cart' })).toBeEnabled())
  })

  it('shows a not-found state for an unknown slug', async () => {
    renderProductPage('not-a-real-product-slug')
    expect(await screen.findByText(/Product not found/)).toBeInTheDocument()
  })
})
