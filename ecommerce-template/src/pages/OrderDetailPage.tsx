import { useParams, Link } from 'react-router-dom'
import { Download, PackageCheck, RotateCcw } from 'lucide-react'
import { Container } from '@/components/layout/Container'
import { PageHeader } from '@/components/layout/PageHeader'
import { Breadcrumbs } from '@/components/layout/Breadcrumbs'
import { Seo } from '@/components/seo/Seo'
import { Button } from '@/components/ui/Button'
import { Badge } from '@/components/ui/Badge'
import { LoadingState, NotFound } from '@/components/ui/States'
import { OrderSummary } from '@/components/ecommerce/OrderSummary'
import { useOrder, useCancelOrder } from '@/features/orders/hooks'
import { useAddCartItem } from '@/features/cart/hooks'
import { formatDate, formatMoney } from '@/lib/utils'
import { toast } from '@/components/ui/toast'
import type { OrderStatus } from '@/types'

const STATUS_VARIANT: Record<OrderStatus, 'default' | 'secondary' | 'destructive' | 'outline' | 'success'> = {
  pending: 'outline', processing: 'secondary', shipped: 'default', delivered: 'success', cancelled: 'destructive', refunded: 'outline',
}

function downloadInvoice(order: NonNullable<ReturnType<typeof useOrder>['data']>) {
  const lines = [
    `Invoice for order ${order.orderNumber}`,
    `Placed: ${formatDate(order.placedAt)}`,
    '',
    ...order.lines.map(l => `${l.name} x${l.quantity} — ${formatMoney(l.price * l.quantity, order.currency)}`),
    '',
    `Subtotal: ${formatMoney(order.subtotal, order.currency)}`,
    `Discount: -${formatMoney(order.discount, order.currency)}`,
    `Shipping: ${formatMoney(order.shipping, order.currency)}`,
    `Tax: ${formatMoney(order.tax, order.currency)}`,
    `Total: ${formatMoney(order.total, order.currency)}`,
  ]
  const blob = new Blob([lines.join('\n')], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `invoice-${order.orderNumber}.txt`
  a.click()
  URL.revokeObjectURL(url)
}

export default function OrderDetailPage() {
  const { id } = useParams<{ id: string }>()
  const { data: order, isLoading } = useOrder(id)
  const cancelOrder = useCancelOrder()
  const addItem = useAddCartItem()

  if (isLoading) return <LoadingState />
  if (!order) return <NotFound title="Order not found" description="Double-check the order link, or look it up from Order lookup." />

  const canCancel = order.status === 'pending' || order.status === 'processing'
  const canReturn = order.status === 'delivered'

  return (
    <>
      <Seo title={`Order ${order.orderNumber}`} />
      <Container className="pb-16">
        <Breadcrumbs items={[{ label: 'Orders', href: '/account/orders' }, { label: order.orderNumber }]} />
        <PageHeader
          title={`Order ${order.orderNumber}`}
          description={`Placed ${formatDate(order.placedAt)}`}
          action={<Badge variant={STATUS_VARIANT[order.status]} className="capitalize">{order.status}</Badge>}
        />

        <div className="grid gap-10 lg:grid-cols-[1fr_20rem]">
          <div className="space-y-8">
            {order.trackingEvents.length > 0 && (
              <div>
                <h2 className="mb-3 text-sm font-semibold">Tracking</h2>
                <ol className="space-y-3 border-l border-border pl-4">
                  {order.trackingEvents.map((event, i) => (
                    <li key={i} className="relative">
                      <span className="absolute -left-[1.4rem] top-1 h-2.5 w-2.5 rounded-full bg-primary" />
                      <p className="text-sm font-medium">{event.status}</p>
                      <p className="text-xs text-muted-foreground">{event.location ? `${event.location} · ` : ''}{formatDate(event.occurredAt)}</p>
                    </li>
                  ))}
                </ol>
                {order.trackingNumber && <p className="mt-3 text-sm text-muted-foreground">Tracking number: <span className="font-mono">{order.trackingNumber}</span></p>}
              </div>
            )}

            <div>
              <h2 className="mb-3 text-sm font-semibold">Items</h2>
              <OrderSummary order={order} />
            </div>

            <div>
              <h2 className="mb-3 text-sm font-semibold">Addresses</h2>
              <div className="grid gap-4 sm:grid-cols-2 text-sm text-muted-foreground">
                <div><p className="mb-1 font-medium text-foreground">Billing</p>{order.billingAddress.firstName} {order.billingAddress.lastName}<br />{order.billingAddress.line1}<br />{order.billingAddress.city}, {order.billingAddress.postalCode}</div>
                <div><p className="mb-1 font-medium text-foreground">Shipping</p>{order.shippingAddress.firstName} {order.shippingAddress.lastName}<br />{order.shippingAddress.line1}<br />{order.shippingAddress.city}, {order.shippingAddress.postalCode}</div>
              </div>
            </div>
          </div>

          <aside className="h-fit space-y-3 rounded-[var(--radius)] border border-border p-5">
            <Button variant="outline" className="w-full" onClick={() => downloadInvoice(order)}>
              <Download className="mr-2 h-4 w-4" /> Download invoice
            </Button>
            <Button
              variant="outline" className="w-full"
              onClick={() => { order.lines.forEach(l => addItem.mutate({ productId: l.productId, variantId: l.variantId, name: l.name, variantLabel: l.variantLabel, imageUrl: l.imageUrl, sku: l.sku, price: l.price, quantity: l.quantity })); toast({ title: 'Items added to cart' }) }}
            >
              <PackageCheck className="mr-2 h-4 w-4" /> Reorder
            </Button>
            {canReturn && (
              <Button asChild variant="outline" className="w-full">
                <Link to={`/account/returns?orderId=${order.id}`}><RotateCcw className="mr-2 h-4 w-4" /> Request return</Link>
              </Button>
            )}
            {canCancel && (
              <Button
                variant="destructive" className="w-full"
                disabled={cancelOrder.isPending}
                onClick={() => cancelOrder.mutate(order.id, { onSuccess: () => toast({ title: 'Order cancelled' }) })}
              >
                {cancelOrder.isPending ? 'Cancelling…' : 'Cancel order'}
              </Button>
            )}
          </aside>
        </div>
      </Container>
    </>
  )
}
