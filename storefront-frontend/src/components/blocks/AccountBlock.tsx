import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth, getSessionToken } from '../../auth/AuthContext'
import { listAccountOrders } from '../../api/auth'
import { formatMoney } from '../../format'
import type { AccountOrder } from '../../types'

export function AccountBlock() {
  const { account, loading, logout } = useAuth()
  const navigate = useNavigate()
  const [orders, setOrders] = useState<AccountOrder[]>([])
  const [ordersLoading, setOrdersLoading] = useState(true)
  const [ordersError, setOrdersError] = useState<string | null>(null)

  useEffect(() => {
    const token = getSessionToken()
    if (!token) {
      setOrdersLoading(false)
      return
    }
    listAccountOrders(token)
      .then(setOrders)
      .catch(e => setOrdersError(e instanceof Error ? e.message : 'Unknown error'))
      .finally(() => setOrdersLoading(false))
  }, [account])

  async function handleLogout() {
    await logout()
    navigate('/')
  }

  if (loading) return <p className="p-6 text-center text-sm text-gray-500">Loading account…</p>
  if (!account) {
    return (
      <div className="p-6 text-center">
        <p className="text-brand-500">You're not signed in.</p>
        <a href="/login" className="mt-2 inline-block text-sm text-brand-700 underline">Sign in</a>
      </div>
    )
  }

  return (
    <div className="p-6">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-xl font-semibold text-brand-700">
            {account.first_name ? `${account.first_name} ${account.last_name}`.trim() : account.email}
          </h1>
          <p className="text-sm text-brand-500">{account.email}</p>
        </div>
        <button onClick={handleLogout} className="text-sm text-brand-500 underline hover:text-brand-700">
          Sign out
        </button>
      </div>

      <h2 className="mt-8 text-lg font-semibold text-brand-700">Order history</h2>
      {ordersLoading && <p className="mt-2 text-sm text-gray-500">Loading orders…</p>}
      {!ordersLoading && ordersError && <p className="mt-2 text-sm text-red-600">{ordersError}</p>}
      {!ordersLoading && !ordersError && orders.length === 0 && (
        <p className="mt-2 text-sm text-brand-500">No orders yet.</p>
      )}
      {orders.length > 0 && (
        <div className="mt-3 divide-y divide-gray-100">
          {orders.map(o => (
            <div key={o.order_number} className="flex items-center justify-between py-3">
              <div>
                <p className="text-sm font-medium text-brand-700">{o.order_number}</p>
                <p className="text-xs text-gray-400">{new Date(o.placed_at).toLocaleDateString()} · {o.status}</p>
              </div>
              <p className="text-sm text-brand-700">{formatMoney(o.total_minor_units, o.currency_code)}</p>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
