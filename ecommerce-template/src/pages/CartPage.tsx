import { Link } from 'react-router-dom'
import { ShoppingBag } from 'lucide-react'
import { Container } from '@/components/layout/Container'
import { PageHeader } from '@/components/layout/PageHeader'
import { Seo } from '@/components/seo/Seo'
import { Button } from '@/components/ui/Button'
import { Separator } from '@/components/ui/Separator'
import { LoadingState, EmptyState } from '@/components/ui/States'
import { CartLineItem } from '@/components/ecommerce/CartLineItem'
import { CouponForm } from '@/components/ecommerce/CouponForm'
import { FreeShippingProgress } from '@/components/ecommerce/FreeShippingProgress'
import { CheckoutSummary } from '@/components/ecommerce/CheckoutSummary'
import { ProductGrid } from '@/components/ecommerce/ProductGrid'
import { useCart } from '@/features/cart/hooks'
import { useProducts } from '@/features/products/hooks'

export default function CartPage() {
  const { data: cart, isLoading } = useCart()
  const { data: recommended } = useProducts({ filter: 'recommended', perPage: 4 })

  return (
    <>
      <Seo title="Your cart" />
      <Container className="pb-16">
        <PageHeader title="Your cart" />
        {isLoading && <LoadingState />}
        {cart && cart.items.length === 0 && cart.savedItems.length === 0 && (
          <EmptyState
            icon={<ShoppingBag className="h-8 w-8" />}
            title="Your cart is empty"
            description="Looks like you haven't added anything yet."
            action={<Button asChild className="mt-2"><Link to="/">Continue shopping</Link></Button>}
          />
        )}
        {cart && (cart.items.length > 0 || cart.savedItems.length > 0) && (
          <div className="grid gap-10 lg:grid-cols-[1fr_20rem]">
            <div>
              {cart.items.length > 0 && (
                <div className="divide-y divide-border">
                  {cart.items.map(item => <CartLineItem key={item.id} item={item} currency={cart.currency} />)}
                </div>
              )}
              {cart.savedItems.length > 0 && (
                <div className="mt-8">
                  <h2 className="mb-2 text-sm font-semibold">Saved for later ({cart.savedItems.length})</h2>
                  <div className="divide-y divide-border">
                    {cart.savedItems.map(item => <CartLineItem key={item.id} item={item} currency={cart.currency} />)}
                  </div>
                </div>
              )}
              {recommended && recommended.items.length > 0 && (
                <div className="mt-12">
                  <h2 className="mb-4 text-lg font-semibold">You might also like</h2>
                  <ProductGrid products={recommended.items} />
                </div>
              )}
            </div>

            {cart.items.length > 0 && (
              <div className="space-y-4 rounded-[var(--radius)] border border-border p-5">
                <FreeShippingProgress totals={cart.totals} currency={cart.currency} />
                <Separator />
                <CouponForm coupons={cart.coupons} currency={cart.currency} />
                <Separator />
                <CheckoutSummary totals={cart.totals} currency={cart.currency} />
                <Button asChild size="lg" className="w-full"><Link to="/checkout">Proceed to checkout</Link></Button>
              </div>
            )}
          </div>
        )}
      </Container>
    </>
  )
}
