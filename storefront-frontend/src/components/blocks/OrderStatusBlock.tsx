import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { useLocation, useParams } from 'react-router-dom'
import { lookupOrder } from '../../api/orders'
import { formatMoney } from '../../format'
import type { StorefrontOrderStatus } from '../../types'

interface Props {
  pageSlug: 'order-confirmation' | 'order-lookup'
}

// The one functional block whose rendering genuinely differs by page:
// order-confirmation shows the order the shopper just placed (known
// order number, email carried via router state from checkout);
// order-lookup is a self-service form for a shopper returning later
// with no state at all, just their order number and email.
export function OrderStatusBlock({ pageSlug }: Props) {
  return pageSlug === 'order-lookup' ? <OrderLookupForm /> : <OrderConfirmation />
}

function OrderConfirmation() {
  const { orderNumber } = useParams<{ orderNumber: string }>()
  const location = useLocation()
  const email = (location.state as { email?: string } | null)?.email
  const [status, setStatus] = useState<StorefrontOrderStatus | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!orderNumber || !email) {
      setLoading(false)
      return
    }
    lookupOrder(orderNumber, email)
      .then(setStatus)
      .catch(e => setError(e instanceof Error ? e.message : 'Unknown error'))
      .finally(() => setLoading(false))
  }, [orderNumber, email])

  if (!orderNumber) return null

  // No email in router state means this page was opened directly (a
  // reload, a bookmark) rather than reached right after checkout -- we
  // never had a session to re-derive it from, so point the shopper at
  // the order-lookup form instead of guessing.
  if (!email) {
    return (
      <div className="p-6 text-center">
        <p className="text-lg font-semibold text-brand-700">Thanks for your order!</p>
        <p className="mt-1 text-brand-500">Order number: {orderNumber}</p>
        <p className="mt-4 text-sm text-gray-400">
          Reloaded this page? Use &ldquo;Track your order&rdquo; with your order number and email to check its status.
        </p>
      </div>
    )
  }
  if (loading) return <p className="p-6 text-center text-brand-500">Loading order…</p>
  if (error || !status) return <p className="p-6 text-center text-red-600">Could not load this order: {error}</p>

  return (
    <div className="p-6 text-center">
      <p className="text-lg font-semibold text-brand-700">Thanks for your order!</p>
      <p className="mt-1 text-brand-500">Order {status.order_number} — {status.status}</p>
      <p className="mt-1 text-brand-500">{formatMoney(status.total_minor_units, status.currency_code)}</p>
    </div>
  )
}

function OrderLookupForm() {
  const [orderNumber, setOrderNumber] = useState('')
  const [email, setEmail] = useState('')
  const [status, setStatus] = useState<StorefrontOrderStatus | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError(null)
    setStatus(null)
    setLoading(true)
    try {
      setStatus(await lookupOrder(orderNumber, email))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Order not found')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="mx-auto max-w-sm p-6">
      <form onSubmit={handleSubmit} className="space-y-3">
        <label className="block text-sm">
          <span className="mb-1 block text-brand-500">Order number</span>
          <input
            value={orderNumber}
            onChange={e => setOrderNumber(e.target.value)}
            required
            className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm outline-none focus:ring-1 focus:ring-brand-500"
          />
        </label>
        <label className="block text-sm">
          <span className="mb-1 block text-brand-500">Email</span>
          <input
            type="email"
            value={email}
            onChange={e => setEmail(e.target.value)}
            required
            className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm outline-none focus:ring-1 focus:ring-brand-500"
          />
        </label>
        <button
          type="submit"
          disabled={loading}
          className="w-full rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-brand-700 disabled:opacity-50"
        >
          {loading ? 'Looking up…' : 'Track order'}
        </button>
      </form>
      {error && <p className="mt-3 text-sm text-red-600">{error}</p>}
      {status && (
        <div className="mt-3 rounded-lg border border-gray-200 p-3 text-sm">
          <p className="text-brand-700">Status: {status.status}</p>
          <p className="text-brand-500">{formatMoney(status.total_minor_units, status.currency_code)}</p>
        </div>
      )}
    </div>
  )
}
