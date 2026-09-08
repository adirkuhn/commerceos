import type { ReactNode } from 'react'
import { BrowserRouter, Routes, Route, Link } from 'react-router-dom'
import { CartProvider, useCart } from './cart/CartContext'
import { HomePage } from './pages/HomePage'
import { ProductsPage } from './pages/ProductsPage'
import { ProductPage } from './pages/ProductPage'
import { CartPage } from './pages/CartPage'
import { CheckoutPage } from './pages/CheckoutPage'
import { OrderConfirmationPage } from './pages/OrderConfirmationPage'
import { OrderLookupPage } from './pages/OrderLookupPage'

function CartLink() {
  const { cart } = useCart()
  const count = cart?.items.reduce((sum, i) => sum + i.quantity, 0) ?? 0
  return (
    <Link to="/cart" className="ml-auto text-brand-500 hover:text-brand-700">
      Cart{count > 0 ? ` (${count})` : ''}
    </Link>
  )
}

function Layout({ children }: { children: ReactNode }) {
  return (
    <div className="min-h-screen bg-white text-brand-700">
      <header className="border-b border-brand-100">
        <nav className="mx-auto flex max-w-5xl items-center gap-6 px-6 py-4 text-sm">
          <Link to="/" className="font-semibold">Store</Link>
          <Link to="/products" className="text-brand-500 hover:text-brand-700">Products</Link>
          <CartLink />
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
        <Layout>
          <Routes>
            <Route path="/" element={<HomePage />} />
            <Route path="/products" element={<ProductsPage />} />
            <Route path="/products/:id" element={<ProductPage />} />
            <Route path="/cart" element={<CartPage />} />
            <Route path="/checkout" element={<CheckoutPage />} />
            <Route path="/order-confirmation/:orderNumber" element={<OrderConfirmationPage />} />
            <Route path="/order-lookup" element={<OrderLookupPage />} />
          </Routes>
        </Layout>
      </CartProvider>
    </BrowserRouter>
  )
}
