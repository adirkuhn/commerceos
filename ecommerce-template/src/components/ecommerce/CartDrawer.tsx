import { Link } from 'react-router-dom'
import { Drawer, DrawerContent, DrawerTitle } from '@/components/ui/Drawer'
import { Button } from '@/components/ui/Button'
import { Separator } from '@/components/ui/Separator'
import { LoadingState, EmptyState } from '@/components/ui/States'
import { CartLineItem } from './CartLineItem'
import { FreeShippingProgress } from './FreeShippingProgress'
import { CheckoutSummary } from './CheckoutSummary'
import { useCart } from '@/features/cart/hooks'
import { useCartStore } from '@/stores/cartStore'
import { ShoppingBag } from 'lucide-react'

export function CartDrawer() {
  const isOpen = useCartStore(s => s.isDrawerOpen)
  const closeDrawer = useCartStore(s => s.closeDrawer)
  const { data: cart, isLoading } = useCart()

  return (
    <Drawer open={isOpen} onOpenChange={(open) => !open && closeDrawer()}>
      <DrawerContent aria-describedby={undefined}>
        <div className="flex items-center justify-between border-b border-border p-4">
          <DrawerTitle className="text-base font-semibold">Your cart</DrawerTitle>
        </div>
        <div className="flex-1 overflow-y-auto px-4">
          {isLoading && <LoadingState />}
          {cart && cart.items.length === 0 && <EmptyState icon={<ShoppingBag className="h-8 w-8" />} title="Your cart is empty" description="Add something you'll love." />}
          {cart && cart.items.length > 0 && (
            <div className="divide-y divide-border">
              {cart.items.map(item => <CartLineItem key={item.id} item={item} currency={cart.currency} />)}
            </div>
          )}
        </div>
        {cart && cart.items.length > 0 && (
          <div className="space-y-4 border-t border-border p-4">
            <FreeShippingProgress totals={cart.totals} currency={cart.currency} />
            <Separator />
            <CheckoutSummary totals={cart.totals} currency={cart.currency} />
            <Button asChild className="w-full" size="lg">
              <Link to="/checkout" onClick={closeDrawer}>Checkout</Link>
            </Button>
            <Button asChild variant="outline" className="w-full">
              <Link to="/cart" onClick={closeDrawer}>View full cart</Link>
            </Button>
          </div>
        )}
      </DrawerContent>
    </Drawer>
  )
}
