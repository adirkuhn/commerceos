import type { ReactNode } from 'react'
import { useEffect } from 'react'
import { BrowserRouter, Routes, Route, Link, useNavigate, useParams } from 'react-router-dom'
import { setStorefrontKey } from './api/client'
import { CartProvider, useCart } from './cart/CartContext'
import { WishlistProvider, useWishlist } from './wishlist/WishlistContext'
import { AuthProvider, useAuth } from './auth/AuthContext'
import { HomePage } from './pages/HomePage'
import { ProductsPage } from './pages/ProductsPage'
import { ProductPage } from './pages/ProductPage'
import { CartPage } from './pages/CartPage'
import { CheckoutPage } from './pages/CheckoutPage'
import { OrderConfirmationPage } from './pages/OrderConfirmationPage'
import { OrderLookupPage } from './pages/OrderLookupPage'
import { WishlistPage } from './pages/WishlistPage'
import { LoginPage } from './pages/LoginPage'
import { AccountPage } from './pages/AccountPage'

function CartLink() {
  const { cart } = useCart()
  const count = cart?.items.reduce((sum, i) => sum + i.quantity, 0) ?? 0
  return (
    <Link to="/cart" className="text-brand-500 hover:text-brand-700">
      Cart{count > 0 ? ` (${count})` : ''}
    </Link>
  )
}

function WishlistLink() {
  const { wishlist } = useWishlist()
  const count = wishlist?.product_ids.length ?? 0
  return (
    <Link to="/wishlist" className="ml-auto text-brand-500 hover:text-brand-700">
      Wishlist{count > 0 ? ` (${count})` : ''}
    </Link>
  )
}

function AccountLink() {
  const { account, loading } = useAuth()
  if (loading) return null
  return (
    <Link to={account ? '/account' : '/login'} className="text-brand-500 hover:text-brand-700">
      {account ? (account.first_name || 'Account') : 'Sign in'}
    </Link>
  )
}

// StoreEntry is this deployment's one entry point for selecting which
// tenant/channel to show -- a shopper never lands here directly (it's what
// a shared link/QR code from the admin app points at), it just remembers
// the key (see setStorefrontKey) and redirects into the app's normal
// routes, which stay entirely unaware a multi-tenant deployment exists.
function StoreEntry() {
  const { storeKey } = useParams<{ storeKey: string }>()
  const navigate = useNavigate()
  useEffect(() => {
    // Set the key before navigating away, in that order, within the same
    // effect -- CartProvider/WishlistProvider/AuthProvider's own mount
    // effects (higher up the tree, so React fires them after this one)
    // must never fire their first fetch against the old/empty key.
    if (storeKey) setStorefrontKey(storeKey)
    navigate('/', { replace: true })
  }, [storeKey, navigate])
  return null
}

function Layout({ children }: { children: ReactNode }) {
  return (
    <div className="min-h-screen bg-white text-brand-700">
      <header className="border-b border-brand-100">
        <nav className="mx-auto flex max-w-5xl items-center gap-6 px-6 py-4 text-sm">
          <Link to="/" className="font-semibold">Store</Link>
          <Link to="/products" className="text-brand-500 hover:text-brand-700">Products</Link>
          <CartLink />
          <WishlistLink />
          <AccountLink />
          <Link to="/order-lookup" className="text-brand-500 hover:text-brand-700">Track order</Link>
        </nav>
      </header>
      {children}
    </div>
  )
}

export function App() {
  return (
    <BrowserRouter>
      <CartProvider>
        <WishlistProvider>
          <AuthProvider>
            <Layout>
              <Routes>
                <Route path="/store/:storeKey" element={<StoreEntry />} />
                <Route path="/" element={<HomePage />} />
                <Route path="/products" element={<ProductsPage />} />
                <Route path="/products/:id" element={<ProductPage />} />
                <Route path="/cart" element={<CartPage />} />
                <Route path="/checkout" element={<CheckoutPage />} />
                <Route path="/order-confirmation/:orderNumber" element={<OrderConfirmationPage />} />
                <Route path="/order-lookup" element={<OrderLookupPage />} />
                <Route path="/wishlist" element={<WishlistPage />} />
                <Route path="/login" element={<LoginPage />} />
                <Route path="/account" element={<AccountPage />} />
              </Routes>
            </Layout>
          </AuthProvider>
        </WishlistProvider>
      </CartProvider>
    </BrowserRouter>
  )
}
