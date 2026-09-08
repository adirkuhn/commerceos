// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { CheckoutFormBlock } from '../CheckoutFormBlock'
import { CartProvider } from '../../../cart/CartContext'
import type { Cart, ShippingOption } from '../../../types'

vi.mock('../../../api/cart', () => ({
  createCart: vi.fn(), getCart: vi.fn(), updateCartItems: vi.fn(), checkoutCart: vi.fn(), getShippingOptions: vi.fn(), applyCoupon: vi.fn(), createPaymentIntent: vi.fn(),
}))

// Card-payment tests below stub the whole Stripe.js/react-stripe-js
// surface -- Elements just renders its children, CardElement is a plain
// marker, and useStripe/useElements return a fake Stripe client this
// test controls confirmCardPayment's outcome for.
const confirmCardPayment = vi.fn()
vi.mock('@stripe/stripe-js', () => ({ loadStripe: vi.fn(() => Promise.resolve({})) }))
vi.mock('@stripe/react-stripe-js', () => ({
  Elements: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  CardElement: () => <div data-testid="card-element" />,
  useStripe: () => ({ confirmCardPayment }),
  useElements: () => ({ getElement: () => ({}) }),
}))

import * as cartApi from '../../../api/cart'

const STORAGE_KEY = 'commerceos_cart_token'

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  vi.mocked(cartApi.getShippingOptions).mockResolvedValue([])
  vi.mocked(cartApi.createPaymentIntent).mockRejectedValue(new Error('embedded card payment is not available on this channel'))
})

function renderBlock() {
  return render(<MemoryRouter><CartProvider><CheckoutFormBlock /></CartProvider></MemoryRouter>)
}

const cart: Cart = { token: 'cart-1', items: [{ sku_code: 'SKU-1', product_name: 'Widget', quantity: 1, price_minor_units: 1000, currency_code: 'EUR' }], subtotal_minor_units: 1000, discount_minor_units: 0 }

async function fillAddress(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByLabelText('Email'), 'shopper@example.com')
  await user.type(screen.getByLabelText('First name'), 'Ada')
  await user.type(screen.getByLabelText('Last name'), 'Lovelace')
  await user.type(screen.getByLabelText('Address'), 'Main St 1')
  await user.type(screen.getByLabelText('City'), 'Amsterdam')
  await user.type(screen.getByLabelText('Postcode'), '1000AB')
  await user.type(screen.getByLabelText('Country'), 'NL')
}

describe('CheckoutFormBlock', () => {
  it('shows an empty-cart message with nothing to check out', async () => {
    renderBlock()
    expect(await screen.findByText('Your cart is empty.')).toBeInTheDocument()
  })

  it('submits the address form and places the order', async () => {
    const user = userEvent.setup()
    localStorage.setItem(STORAGE_KEY, 'cart-1')
    vi.mocked(cartApi.getCart).mockResolvedValue(cart)
    vi.mocked(cartApi.checkoutCart).mockResolvedValue({ order_number: 'WEB-1', total_minor_units: 1000, currency_code: 'EUR' })

    renderBlock()
    await screen.findByLabelText('Email')
    await fillAddress(user)
    await user.click(screen.getByRole('button', { name: 'Place order' }))

    await waitFor(() => expect(cartApi.checkoutCart).toHaveBeenCalledWith('cart-1', {
      email: 'shopper@example.com',
      billing: { first_name: 'Ada', last_name: 'Lovelace', address1: 'Main St 1', city: 'Amsterdam', postcode: '1000AB', country: 'NL' },
      shipping: { first_name: 'Ada', last_name: 'Lovelace', address1: 'Main St 1', city: 'Amsterdam', postcode: '1000AB', country: 'NL' },
      shipping_method_title: undefined,
    }))
  })

  it('shows a server error and keeps the form open on failure', async () => {
    const user = userEvent.setup()
    localStorage.setItem(STORAGE_KEY, 'cart-1')
    vi.mocked(cartApi.getCart).mockResolvedValue(cart)
    vi.mocked(cartApi.checkoutCart).mockRejectedValue(new Error('insufficient stock'))

    renderBlock()
    await screen.findByLabelText('Email')
    await fillAddress(user)
    await user.click(screen.getByRole('button', { name: 'Place order' }))

    expect(await screen.findByText('insufficient stock')).toBeInTheDocument()
    expect(screen.getByLabelText('Email')).toBeInTheDocument()
  })

  it('fetches and shows shipping options once the address is complete, preselecting the first', async () => {
    const user = userEvent.setup()
    localStorage.setItem(STORAGE_KEY, 'cart-1')
    vi.mocked(cartApi.getCart).mockResolvedValue(cart)
    const options: ShippingOption[] = [
      { title: 'Standard', amount_minor_units: 495, currency_code: 'EUR' },
      { title: 'Evening', amount_minor_units: 895, currency_code: 'EUR' },
    ]
    vi.mocked(cartApi.getShippingOptions).mockResolvedValue(options)
    vi.mocked(cartApi.checkoutCart).mockResolvedValue({ order_number: 'WEB-1', total_minor_units: 1495, currency_code: 'EUR' })

    renderBlock()
    await screen.findByLabelText('Email')
    await fillAddress(user)

    expect(await screen.findByText('Standard')).toBeInTheDocument()
    expect(screen.getByText('Evening')).toBeInTheDocument()
    expect(screen.getByText('€4.95')).toBeInTheDocument()
    expect(screen.getByRole('radio', { name: /Standard/ })).toBeChecked()

    await user.click(screen.getByRole('radio', { name: /Evening/ }))
    await user.click(screen.getByRole('button', { name: 'Place order' }))

    await waitFor(() => expect(cartApi.checkoutCart).toHaveBeenCalledWith('cart-1', expect.objectContaining({ shipping_method_title: 'Evening' })))
  })

  it('shows no shipping picker when no options are available', async () => {
    const user = userEvent.setup()
    localStorage.setItem(STORAGE_KEY, 'cart-1')
    vi.mocked(cartApi.getCart).mockResolvedValue(cart)
    vi.mocked(cartApi.getShippingOptions).mockResolvedValue([])

    renderBlock()
    await screen.findByLabelText('Email')
    await fillAddress(user)

    await waitFor(() => expect(cartApi.getShippingOptions).toHaveBeenCalled())
    expect(screen.queryByRole('radio')).not.toBeInTheDocument()
  })

  it('shows the embedded card field once the channel supports card payment, and pays with it', async () => {
    const user = userEvent.setup()
    localStorage.setItem(STORAGE_KEY, 'cart-1')
    vi.mocked(cartApi.getCart).mockResolvedValue(cart)
    vi.mocked(cartApi.createPaymentIntent).mockResolvedValue({
      client_secret: 'pi_secret_1', publishable_key: 'pk_test_1', amount_minor_units: 1000, currency_code: 'EUR',
    })
    confirmCardPayment.mockResolvedValue({ paymentIntent: { id: 'pi_1', status: 'succeeded' } })
    vi.mocked(cartApi.checkoutCart).mockResolvedValue({ order_number: 'WEB-1', total_minor_units: 1000, currency_code: 'EUR' })

    renderBlock()
    await screen.findByLabelText('Email')
    await fillAddress(user)

    await screen.findByTestId('card-element')
    expect(screen.queryByRole('button', { name: 'Place order' })).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /Pay .* & place order/ }))

    await waitFor(() => expect(confirmCardPayment).toHaveBeenCalledWith('pi_secret_1', expect.anything()))
    await waitFor(() => expect(cartApi.checkoutCart).toHaveBeenCalledWith('cart-1', expect.objectContaining({ payment_intent_id: 'pi_1' })))
  })

  it('shows a card error and does not place the order when confirmCardPayment fails', async () => {
    const user = userEvent.setup()
    localStorage.setItem(STORAGE_KEY, 'cart-1')
    vi.mocked(cartApi.getCart).mockResolvedValue(cart)
    vi.mocked(cartApi.createPaymentIntent).mockResolvedValue({
      client_secret: 'pi_secret_1', publishable_key: 'pk_test_1', amount_minor_units: 1000, currency_code: 'EUR',
    })
    confirmCardPayment.mockResolvedValue({ error: { message: 'Your card was declined.' } })

    renderBlock()
    await screen.findByLabelText('Email')
    await fillAddress(user)

    await screen.findByTestId('card-element')
    await user.click(screen.getByRole('button', { name: /Pay .* & place order/ }))

    expect(await screen.findByText('Your card was declined.')).toBeInTheDocument()
    expect(cartApi.checkoutCart).not.toHaveBeenCalled()
  })
})
