import { lazy, Suspense } from 'react'
import { Routes, Route } from 'react-router-dom'
import { Layout } from '@/app/Layout'
import { AccountLayout } from '@/components/layout/AccountLayout'
import { LoadingState } from '@/components/ui/States'

// Every route is lazy-loaded -- each page ships as its own chunk, so the
// initial bundle only pays for Layout + whichever route matched first.
// See spec section 24 (code splitting / lazy-loaded routes).
const HomePage = lazy(() => import('@/pages/HomePage'))
const SearchPage = lazy(() => import('@/pages/SearchPage'))
const CategoryPage = lazy(() => import('@/pages/CategoryPage'))
const ProductDetailPage = lazy(() => import('@/pages/ProductDetailPage'))
const CartPage = lazy(() => import('@/pages/CartPage'))
const CheckoutPage = lazy(() => import('@/pages/CheckoutPage'))
const OrderConfirmationPage = lazy(() => import('@/pages/OrderConfirmationPage'))
const OrderDetailPage = lazy(() => import('@/pages/OrderDetailPage'))
const OrderLookupPage = lazy(() => import('@/pages/OrderLookupPage'))
const WishlistPage = lazy(() => import('@/pages/WishlistPage'))
const ComparePage = lazy(() => import('@/pages/ComparePage'))
const LoginPage = lazy(() => import('@/pages/LoginPage'))
const RegisterPage = lazy(() => import('@/pages/RegisterPage'))
const ForgotPasswordPage = lazy(() => import('@/pages/ForgotPasswordPage'))
const ResetPasswordPage = lazy(() => import('@/pages/ResetPasswordPage'))
const CmsPage = lazy(() => import('@/pages/CmsPage'))
const BlogListPage = lazy(() => import('@/pages/BlogListPage'))
const BlogPostPage = lazy(() => import('@/pages/BlogPostPage'))
const NotFoundPage = lazy(() => import('@/pages/NotFoundPage'))

const AccountDashboardPage = lazy(() => import('@/pages/account/AccountDashboardPage'))
const AccountOrdersPage = lazy(() => import('@/pages/account/AccountOrdersPage'))
const AccountAddressesPage = lazy(() => import('@/pages/account/AccountAddressesPage'))
const AccountProfilePage = lazy(() => import('@/pages/account/AccountProfilePage'))
const AccountReturnsPage = lazy(() => import('@/pages/account/AccountReturnsPage'))

export function AppRouter() {
  return (
    <Suspense fallback={<LoadingState />}>
      <Routes>
        <Route element={<Layout />}>
          <Route path="/" element={<HomePage />} />
          <Route path="/search" element={<SearchPage />} />
          <Route path="/category/:slug" element={<CategoryPage />} />
          <Route path="/product/:slug" element={<ProductDetailPage />} />

          <Route path="/cart" element={<CartPage />} />
          <Route path="/checkout" element={<CheckoutPage />} />
          <Route path="/order/:id" element={<OrderDetailPage />} />
          <Route path="/order-confirmation/:id" element={<OrderConfirmationPage />} />
          <Route path="/order-lookup" element={<OrderLookupPage />} />

          <Route path="/wishlist" element={<WishlistPage />} />
          <Route path="/compare" element={<ComparePage />} />

          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterPage />} />
          <Route path="/forgot-password" element={<ForgotPasswordPage />} />
          <Route path="/reset-password" element={<ResetPasswordPage />} />

          <Route path="/account" element={<AccountLayout />}>
            <Route index element={<AccountDashboardPage />} />
            <Route path="orders" element={<AccountOrdersPage />} />
            <Route path="orders/:id" element={<OrderDetailPage />} />
            <Route path="addresses" element={<AccountAddressesPage />} />
            <Route path="profile" element={<AccountProfilePage />} />
            <Route path="wishlist" element={<WishlistPage />} />
            <Route path="returns" element={<AccountReturnsPage />} />
          </Route>

          <Route path="/page/:slug" element={<CmsPage />} />
          <Route path="/blog" element={<BlogListPage />} />
          <Route path="/blog/:slug" element={<BlogPostPage />} />

          <Route path="*" element={<NotFoundPage />} />
        </Route>
      </Routes>
    </Suspense>
  )
}
