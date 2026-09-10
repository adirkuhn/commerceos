import { Link } from 'react-router-dom'
import { Seo } from '@/components/seo/Seo'
import { PageHeader } from '@/components/layout/PageHeader'
import { DataTable, type DataTableColumn } from '@/components/ui/DataTable'
import { Badge } from '@/components/ui/Badge'
import { EmptyState, LoadingState } from '@/components/ui/States'
import { useAuthStore } from '@/stores/authStore'
import { useCustomerOrders } from '@/features/orders/hooks'
import { formatDate, formatMoney } from '@/lib/utils'
import type { Order } from '@/types'
import { PackageOpen } from 'lucide-react'

export default function AccountOrdersPage() {
  const customer = useAuthStore(s => s.session?.customer)
  const { data: orders = [], isLoading } = useCustomerOrders(customer?.id)

  const columns: DataTableColumn<Order>[] = [
    { key: 'orderNumber', header: 'Order', render: (o) => <Link to={`/account/orders/${o.id}`} className="font-medium text-primary hover:underline">{o.orderNumber}</Link> },
    { key: 'placedAt', header: 'Date', render: (o) => formatDate(o.placedAt) },
    { key: 'status', header: 'Status', render: (o) => <Badge variant="outline" className="capitalize">{o.status}</Badge> },
    { key: 'total', header: 'Total', render: (o) => formatMoney(o.total, o.currency) },
  ]

  return (
    <>
      <Seo title="Order history" />
      <PageHeader title="Order history" />
      {isLoading ? <LoadingState /> : orders.length === 0 ? (
        <EmptyState icon={<PackageOpen className="h-8 w-8" />} title="No orders yet" description="Orders you place will show up here." />
      ) : (
        <DataTable columns={columns} rows={orders} caption="Your order history" />
      )}
    </>
  )
}
