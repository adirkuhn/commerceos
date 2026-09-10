import { useParams, Link } from 'react-router-dom'
import { CheckCircle2 } from 'lucide-react'
import { Container } from '@/components/layout/Container'
import { Seo } from '@/components/seo/Seo'
import { Button } from '@/components/ui/Button'
import { LoadingState, NotFound } from '@/components/ui/States'
import { OrderSummary } from '@/components/ecommerce/OrderSummary'
import { useOrder } from '@/features/orders/hooks'
import { formatDate } from '@/lib/utils'

export default function OrderConfirmationPage() {
  const { id } = useParams<{ id: string }>()
  const { data: order, isLoading } = useOrder(id)

  if (isLoading) return <LoadingState />
  if (!order) return <NotFound title="Order not found" description="We couldn't find a confirmation for this order." />

  return (
    <>
      <Seo title={`Order ${order.orderNumber} confirmed`} />
      <Container className="max-w-2xl py-16 text-center">
        <CheckCircle2 className="mx-auto h-12 w-12 text-emerald-600" />
        <h1 className="mt-4 text-2xl font-semibold tracking-tight">Thank you for your order!</h1>
        <p className="mt-1 text-muted-foreground">Order <strong className="text-foreground">{order.orderNumber}</strong> placed {formatDate(order.placedAt)}</p>
        <div className="mt-2 flex justify-center gap-4 text-sm">
          <span>Payment: <span className="font-medium capitalize">{order.paymentStatus}</span></span>
          {order.estimatedDeliveryLabel && <span>{order.estimatedDeliveryLabel}</span>}
        </div>

        <div className="mt-8 rounded-[var(--radius)] border border-border p-6 text-left">
          <OrderSummary order={order} />
        </div>

        <div className="mt-8 flex flex-col justify-center gap-3 sm:flex-row">
          <Button asChild><Link to="/">Continue shopping</Link></Button>
          <Button asChild variant="outline"><Link to={`/order/${order.id}`}>Track order</Link></Button>
        </div>
      </Container>
    </>
  )
}
