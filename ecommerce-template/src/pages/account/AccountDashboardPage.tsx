import { Link } from 'react-router-dom'
import { Seo } from '@/components/seo/Seo'
import { PageHeader } from '@/components/layout/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { LoadingState } from '@/components/ui/States'
import { ProductGrid } from '@/components/ecommerce/ProductGrid'
import { useAuthStore } from '@/stores/authStore'
import { useCustomerOrders } from '@/features/orders/hooks'
import { useWishlistProducts } from '@/features/wishlist/hooks'
import { useProducts } from '@/features/products/hooks'
import { formatDate, formatMoney } from '@/lib/utils'

export default function AccountDashboardPage() {
  const customer = useAuthStore(s => s.session?.customer)
  const { data: orders = [], isLoading } = useCustomerOrders(customer?.id)
  const { data: wishlistProducts = [] } = useWishlistProducts()
  const { data: recommended } = useProducts({ filter: 'recommended', perPage: 4 })

  return (
    <>
      <Seo title="My account" />
      <PageHeader title={`Welcome back, ${customer?.firstName}`} description={customer?.email} />

      <section className="mb-10">
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-sm font-semibold">Recent orders</h2>
          <Link to="/account/orders" className="text-sm text-primary hover:underline">View all</Link>
        </div>
        {isLoading ? <LoadingState /> : orders.length === 0 ? (
          <p className="text-sm text-muted-foreground">No orders yet.</p>
        ) : (
          <div className="divide-y divide-border rounded-[var(--radius)] border border-border">
            {orders.slice(0, 3).map(order => (
              <Link key={order.id} to={`/account/orders/${order.id}`} className="flex items-center justify-between p-4 text-sm hover:bg-secondary/50">
                <div>
                  <p className="font-medium">{order.orderNumber}</p>
                  <p className="text-xs text-muted-foreground">{formatDate(order.placedAt)}</p>
                </div>
                <div className="flex items-center gap-3">
                  <span>{formatMoney(order.total, order.currency)}</span>
                  <Badge variant="outline" className="capitalize">{order.status}</Badge>
                </div>
              </Link>
            ))}
          </div>
        )}
      </section>

      {wishlistProducts.length > 0 && (
        <section className="mb-10">
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-sm font-semibold">From your wishlist</h2>
            <Link to="/account/wishlist" className="text-sm text-primary hover:underline">View all</Link>
          </div>
          <ProductGrid products={wishlistProducts.slice(0, 4)} />
        </section>
      )}

      {recommended && recommended.items.length > 0 && (
        <section>
          <h2 className="mb-3 text-sm font-semibold">Recommended for you</h2>
          <ProductGrid products={recommended.items} />
        </section>
      )}
    </>
  )
}
