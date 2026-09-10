import type { ReactNode } from 'react'
import { useEffect } from 'react'
import { BrowserRouter, Routes, Route, Link, useNavigate, useParams, useSearchParams } from 'react-router-dom'
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

// A safe redirect target: exactly one leading slash, never "//host/..."
// (a protocol-relative URL, which some browsers/history APIs treat as
// off-site) -- next comes from a query param a link/QR code controls, so
// it's untrusted input even though react-router's own navigate() never
// actually leaves this origin for it.
function safeNextPath(next: string | null): string {
  return next && next.startsWith('/') && !next.startsWith('//') ? next : '/'
}

// StoreEntry is this deployment's one entry point for selecting which
// tenant/channel to show -- a shopper never lands here directly (it's what
// a shared link/QR code from the admin app points at), it just remembers
// the key (see setStorefrontKey) and redirects into the app's normal
// routes, which stay entirely unaware a multi-tenant deployment exists.
// next (defaults to "/") is which page to land on -- the theme editor's
// own "Preview" button uses this to jump straight to the page it's
// previewing instead of always landing on home.
function StoreEntry() {
  const { storeKey } = useParams<{ storeKey: string }>()
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  useEffect(() => {
    // Set the key before navigating away, in that order, within the same
    // effect -- CartProvider/WishlistProvider/AuthProvider's own mount
    // effects (higher up the tree, so React fires them after this one)
    // must never fire their first fetch against the old/empty key.
    if (storeKey) setStorefrontKey(storeKey)
    navigate(safeNextPath(searchParams.get('next')), { replace: true })
  }, [storeKey, searchParams, navigate])
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

// Matches vite.config.ts's build-time `base` -- "/" locally/in tests (no
// prefix to strip), "/store/" in production, where Traefik path-mounts
// this app on the same host/domain the backend API already has DNS/TLS
// for (see docker-compose.yml's storefront service). basename must not
// carry a trailing slash, and passing "/" explicitly (rather than
// omitting it) makes react-router treat every route as one level deeper
// than intended.
const basename = import.meta.env.BASE_URL === '/' ? undefined : import.meta.env.BASE_URL.replace(/\/$/, '')

export function App() {
  return (
    <BrowserRouter basename={basename}>
      <CartProvider>
        <WishlistProvider>
          <AuthProvider>
            <Layout>
              <Routes>
                <Route path="/enter/:storeKey" element={<StoreEntry />} />
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
