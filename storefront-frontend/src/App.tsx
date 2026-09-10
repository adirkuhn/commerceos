import type { ReactNode } from 'react'
import { Suspense, lazy, useEffect } from 'react'
import { BrowserRouter, Routes, Route, useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { setStorefrontKey } from './api/client'
import { CartProvider } from './cart/CartContext'
import { WishlistProvider } from './wishlist/WishlistContext'
import { AuthProvider } from './auth/AuthContext'
import { CanvasPage } from './components/CanvasPage'
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

// Lazy: pulls in @puckeditor/core (see CanvasPage.tsx's own lazy import for
// why) -- a shopper's browser must never fetch this, only a merchant's
// edit-session-token-bearing visit to /edit/:slug does.
const EditPage = lazy(() => import('./pages/EditPage'))

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

// Layout is the site-wide chrome around every real page -- both header
// and footer are themselves ThemePage "pseudo-pages" (see backend
// domain/theme.PageSlugHeader's own doc comment), rendered by the same
// CanvasPage/BlockRenderer machinery every real page uses, not hardcoded
// markup. header's one required block (nav_actions) guarantees the real
// Cart/Wishlist/Account links always render even before a merchant ever
// opens the editor; footer has no requirement and simply renders nothing
// until a merchant adds content (see CanvasPage's own "chrome" handling).
function Layout({ children }: { children: ReactNode }) {
  return (
    <div className="min-h-screen bg-white text-brand-700">
      <header className="border-b border-brand-100">
        <CanvasPage slug="header" />
      </header>
      {children}
      <footer className="border-t border-brand-100">
        <CanvasPage slug="footer" />
      </footer>
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

// ShopperRoutes is everything a real shopper's browser ever renders --
// split out so /edit/:slug (below) mounts none of CartProvider/
// WishlistProvider/AuthProvider or the shopper header/nav: it's a
// merchant tool (the embedded Puck editor, see pages/EditPage.tsx), not
// part of the storefront a shopper sees.
function ShopperRoutes() {
  return (
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
  )
}

export function App() {
  return (
    <BrowserRouter basename={basename}>
      <Routes>
        <Route
          path="/edit/:slug"
          element={
            <Suspense fallback={<p className="p-6 text-sm text-gray-500">Loading editor…</p>}>
              <EditPage />
            </Suspense>
          }
        />
        <Route path="/*" element={<ShopperRoutes />} />
      </Routes>
    </BrowserRouter>
  )
}
