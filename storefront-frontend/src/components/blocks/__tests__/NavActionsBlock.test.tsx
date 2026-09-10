// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import { NavActionsBlock } from '../NavActionsBlock'
import { CartProvider } from '../../../cart/CartContext'
import { WishlistProvider } from '../../../wishlist/WishlistContext'
import { AuthProvider } from '../../../auth/AuthContext'
import type { Account, Cart, Wishlist } from '../../../types'

vi.mock('../../../api/cart', () => ({
  createCart: vi.fn(), getCart: vi.fn(), updateCartItems: vi.fn(), checkoutCart: vi.fn(), applyCoupon: vi.fn(),
}))
vi.mock('../../../api/wishlist', () => ({ createWishlist: vi.fn(), getWishlist: vi.fn(), toggleWishlistItem: vi.fn() }))
vi.mock('../../../api/auth', () => ({
  requestMagicLink: vi.fn(), verifySession: vi.fn(), getAccount: vi.fn(), listAccountOrders: vi.fn(), logout: vi.fn(),
}))

import * as cartApi from '../../../api/cart'
import * as wishlistApi from '../../../api/wishlist'
import * as authApi from '../../../api/auth'

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
})

function renderBlock() {
  return render(
    <MemoryRouter>
      <CartProvider>
        <WishlistProvider>
          <AuthProvider>
            <NavActionsBlock />
          </AuthProvider>
        </WishlistProvider>
      </CartProvider>
    </MemoryRouter>,
  )
}

describe('NavActionsBlock', () => {
  it('shows Cart/Wishlist with no count and Sign in when empty/signed out', async () => {
    renderBlock()

    expect(await screen.findByRole('link', { name: 'Cart' })).toHaveAttribute('href', '/cart')
    expect(screen.getByRole('link', { name: 'Wishlist' })).toHaveAttribute('href', '/wishlist')
    expect(screen.getByRole('link', { name: 'Sign in' })).toHaveAttribute('href', '/login')
  })

  it('shows real item counts once the cart/wishlist load', async () => {
    const cart: Cart = { token: 'cart-1', items: [{ sku_code: 'SKU1', product_name: 'Widget', quantity: 2, price_minor_units: 500, currency_code: 'EUR' }], subtotal_minor_units: 1000, discount_minor_units: 0 }
    const wishlist: Wishlist = { token: 'wl-1', product_ids: ['p1'] }
    localStorage.setItem('commerceos_cart_token', 'cart-1')
    localStorage.setItem('commerceos_wishlist_token', 'wl-1')
    vi.mocked(cartApi.getCart).mockResolvedValue(cart)
    vi.mocked(wishlistApi.getWishlist).mockResolvedValue(wishlist)

    renderBlock()

    expect(await screen.findByRole('link', { name: 'Cart (2)' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Wishlist (1)' })).toBeInTheDocument()
  })

  it('shows the account name and links to /account once signed in', async () => {
    localStorage.setItem('commerceos_session_token', 'session-1')
    const account: Account = { email: 'shopper@example.com', first_name: 'Ada', last_name: 'Lovelace' }
    vi.mocked(authApi.getAccount).mockResolvedValue(account)
    vi.mocked(authApi.listAccountOrders).mockResolvedValue([])

    renderBlock()

    await waitFor(() => expect(screen.getByRole('link', { name: 'Ada' })).toHaveAttribute('href', '/account'))
  })
})
