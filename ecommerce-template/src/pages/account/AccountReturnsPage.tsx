import { useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Seo } from '@/components/seo/Seo'
import { PageHeader } from '@/components/layout/PageHeader'
import { Select } from '@/components/ui/Select'
import { Textarea } from '@/components/ui/Textarea'
import { Label } from '@/components/ui/Label'
import { Button } from '@/components/ui/Button'
import { Checkbox } from '@/components/ui/Checkbox'
import { Badge } from '@/components/ui/Badge'
import { EmptyState } from '@/components/ui/States'
import { RotateCcw } from 'lucide-react'
import { useAuthStore } from '@/stores/authStore'
import { useCustomerOrders, useRequestReturn } from '@/features/orders/hooks'
import { toast } from '@/components/ui/toast'

const RETURN_REASONS = ['Wrong size', 'Changed my mind', 'Item damaged', 'Not as described', 'Other']

interface SubmittedReturn {
  orderId: string
  orderNumber: string
  reason: string
  productNames: string[]
}

export default function AccountReturnsPage() {
  const [params] = useSearchParams()
  const customer = useAuthStore(s => s.session?.customer)
  const { data: orders = [] } = useCustomerOrders(customer?.id)
  const requestReturn = useRequestReturn()
  const deliveredOrders = orders.filter(o => o.status === 'delivered')

  const [orderId, setOrderId] = useState(params.get('orderId') ?? deliveredOrders[0]?.id ?? '')
  const [selectedProductIds, setSelectedProductIds] = useState<string[]>([])
  const [reason, setReason] = useState(RETURN_REASONS[0])
  const [submitted, setSubmitted] = useState<SubmittedReturn[]>([])

  const selectedOrder = deliveredOrders.find(o => o.id === orderId)

  return (
    <>
      <Seo title="Returns" />
      <PageHeader title="Returns" />

      {deliveredOrders.length === 0 ? (
        <EmptyState icon={<RotateCcw className="h-8 w-8" />} title="No eligible orders" description="Only delivered orders can be returned." />
      ) : (
        <form
          className="max-w-md space-y-4"
          onSubmit={(e) => {
            e.preventDefault()
            if (!selectedOrder) return
            requestReturn.mutate(
              { orderId: selectedOrder.id, lineProductIds: selectedProductIds, reason },
              {
                onSuccess: () => {
                  setSubmitted(prev => [...prev, {
                    orderId: selectedOrder.id, orderNumber: selectedOrder.orderNumber, reason,
                    productNames: selectedOrder.lines.filter(l => selectedProductIds.includes(l.productId)).map(l => l.name),
                  }])
                  setSelectedProductIds([])
                  toast({ title: 'Return requested', variant: 'success' })
                },
                onError: (err) => toast({ title: 'Could not submit return', description: err instanceof Error ? err.message : undefined, variant: 'destructive' }),
              },
            )
          }}
        >
          <div>
            <Label htmlFor="return-order">Order</Label>
            <Select id="return-order" className="mt-1.5" value={orderId} onChange={(e) => { setOrderId(e.target.value); setSelectedProductIds([]) }}>
              {deliveredOrders.map(o => <option key={o.id} value={o.id}>{o.orderNumber}</option>)}
            </Select>
          </div>

          {selectedOrder && (
            <div>
              <Label>Items to return</Label>
              <div className="mt-1.5 space-y-2">
                {selectedOrder.lines.map(line => (
                  <label key={line.productId} className="flex items-center gap-2 text-sm">
                    <Checkbox
                      checked={selectedProductIds.includes(line.productId)}
                      onCheckedChange={(checked) => setSelectedProductIds(prev => (checked ? [...prev, line.productId] : prev.filter(id => id !== line.productId)))}
                    />
                    {line.name}
                  </label>
                ))}
              </div>
            </div>
          )}

          <div>
            <Label htmlFor="return-reason-select">Reason</Label>
            <Select id="return-reason-select" className="mt-1.5" value={reason} onChange={(e) => setReason(e.target.value)}>
              {RETURN_REASONS.map(r => <option key={r} value={r}>{r}</option>)}
            </Select>
          </div>
          <div>
            <Label htmlFor="return-notes">Additional details (optional)</Label>
            <Textarea id="return-notes" className="mt-1.5" rows={3} />
          </div>

          <Button type="submit" disabled={selectedProductIds.length === 0 || requestReturn.isPending}>
            {requestReturn.isPending ? 'Submitting…' : 'Request return'}
          </Button>
        </form>
      )}

      {submitted.length > 0 && (
        <div className="mt-10">
          <h2 className="mb-3 text-sm font-semibold">Return requests</h2>
          <ul className="space-y-2">
            {submitted.map((r, i) => (
              <li key={i} className="flex items-center justify-between rounded-[var(--radius)] border border-border p-3 text-sm">
                <span>{r.orderNumber} -- {r.productNames.join(', ')} ({r.reason})</span>
                <Badge variant="outline">Requested</Badge>
              </li>
            ))}
          </ul>
        </div>
      )}
    </>
  )
}
