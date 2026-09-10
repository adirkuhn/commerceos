// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { describe, expect, it, beforeEach } from 'vitest'
import CheckoutPage from '../CheckoutPage'
import { TooltipProvider } from '@/components/ui/Tooltip'
import { useCartStore } from '@/stores/cartStore'
import { useAuthStore } from '@/stores/authStore'
import { useCheckoutStore } from '@/stores/checkoutStore'
import { cartRepository } from '@/repositories'
import { PRODUCTS } from '@/config/demoData'

function OrderConfirmationProbe() {
  return <p>Order confirmed</p>
}

function renderCheckout() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return render(
    <QueryClientProvider client={queryClient}>
      <TooltipProvider>
        <MemoryRouter initialEntries={['/checkout']}>
          <Routes>
            <Route path="/checkout" element={<CheckoutPage />} />
            <Route path="/order-confirmation/:id" element={<OrderConfirmationProbe />} />
          </Routes>
        </MemoryRouter>
      </TooltipProvider>
    </QueryClientProvider>,
  )
}

beforeEach(async () => {
  useCartStore.setState({ cartId: `checkout-test-${Math.random()}`, isDrawerOpen: false })
  useAuthStore.setState({ session: null, isAuthenticated: false })
  useCheckoutStore.getState().reset()
  await cartRepository.addItem(useCartStore.getState().cartId, {
    productId: PRODUCTS[0].id, name: PRODUCTS[0].name, sku: PRODUCTS[0].sku, price: PRODUCTS[0].price, quantity: 1,
  })
})

describe('CheckoutPage', () => {
  it('walks through customer -> shipping -> payment -> review and places the order', async () => {
    const user = userEvent.setup()
    renderCheckout()

    // Customer step
    await user.type(await screen.findByLabelText('Email'), 'guest@example.com')
    await user.click(screen.getByRole('button', { name: 'Continue to shipping' }))

    // Shipping step
    await user.type(await screen.findByLabelText('First name'), 'Morgan')
    await user.type(screen.getByLabelText('Last name'), 'Lee')
    await user.type(screen.getByLabelText('Street address'), '482 Ridgeview Ave')
    await user.type(screen.getByLabelText('City'), 'Boulder')
    await user.type(screen.getByLabelText('Postal code'), '80302')
    await user.click(screen.getByRole('button', { name: 'Continue' }))
    await user.click(await screen.findByRole('button', { name: 'Continue to payment' }))

    // Payment step
    await user.click(await screen.findByRole('button', { name: 'Review order' }))

    // Review step
    await user.click(await screen.findByLabelText(/I agree to the/))
    await user.click(screen.getByRole('button', { name: 'Place order' }))

    expect(await screen.findByText('Order confirmed')).toBeInTheDocument()
    const cartAfter = await cartRepository.get(useCartStore.getState().cartId)
    expect(cartAfter.items).toHaveLength(0)
  })

  it('requires accepting terms before placing the order', async () => {
    useCheckoutStore.setState({
      step: 'review', email: 'guest@example.com',
      shippingAddress: { id: 's', firstName: 'Morgan', lastName: 'Lee', line1: '482 Ridgeview Ave', city: 'Boulder', postalCode: '80302', country: 'US' },
      billingAddress: { id: 'b', firstName: 'Morgan', lastName: 'Lee', line1: '482 Ridgeview Ave', city: 'Boulder', postalCode: '80302', country: 'US' },
    })
    renderCheckout()

    expect(await screen.findByRole('button', { name: 'Place order' })).toBeDisabled()
  })
})
