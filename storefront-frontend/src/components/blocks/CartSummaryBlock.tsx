import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useCart } from '../../cart/CartContext'
import { formatMoney } from '../../format'

export function CartSummaryBlock() {
  const { cart, loading, setItemQuantity, removeItem, applyCoupon, error } = useCart()
  const [couponInput, setCouponInput] = useState('')
  const [couponBusy, setCouponBusy] = useState(false)
  const [couponError, setCouponError] = useState<string | null>(null)

  async function handleApplyCoupon() {
    if (!couponInput.trim()) return
    setCouponBusy(true)
    setCouponError(null)
    try {
      await applyCoupon(couponInput.trim())
      setCouponInput('')
    } catch (e) {
      setCouponError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setCouponBusy(false)
    }
  }

  async function handleRemoveCoupon() {
    setCouponBusy(true)
    setCouponError(null)
    try {
      await applyCoupon('')
    } catch (e) {
      setCouponError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setCouponBusy(false)
    }
  }

  if (loading) return <p className="p-4 text-sm text-gray-500">Loading cart…</p>
  if (!cart || cart.items.length === 0) {
    return (
      <div className="p-6 text-center">
        <p className="text-brand-500">Your cart is empty.</p>
        <Link to="/products" className="mt-2 inline-block text-sm text-brand-700 underline">Continue shopping</Link>
      </div>
    )
  }

  return (
    <div className="h-full overflow-auto p-6">
      {error && <p className="mb-3 text-sm text-red-600">{error}</p>}
      <div className="space-y-3">
        {cart.items.map(item => (
          <div key={item.sku_code} className="flex items-center justify-between border-b border-gray-100 pb-3">
            <div>
              <p className="text-sm font-medium text-brand-700">{item.product_name}</p>
              <p className="text-xs text-gray-400">{item.sku_code}</p>
            </div>
            <div className="flex items-center gap-3">
              <input
                type="number"
                min={1}
                aria-label={`Quantity for ${item.sku_code}`}
                value={item.quantity}
                onChange={e => setItemQuantity(item.sku_code, Math.max(1, parseInt(e.target.value, 10) || 1))}
                className="w-14 rounded border border-gray-200 px-2 py-1 text-sm"
              />
              <p className="w-20 text-right text-sm text-brand-700">
                {formatMoney(item.price_minor_units * item.quantity, item.currency_code)}
              </p>
              <button
                onClick={() => removeItem(item.sku_code)}
                aria-label={`Remove ${item.sku_code}`}
                className="text-xs text-red-500 hover:text-red-700"
              >
                Remove
              </button>
            </div>
          </div>
        ))}
      </div>
      <div className="mt-4 border-t border-gray-100 pt-4">
        {cart.coupon_code ? (
          <div className="flex items-center justify-between text-sm">
            <p className="text-brand-500">
              Coupon <span className="font-medium text-brand-700">{cart.coupon_code}</span> applied
            </p>
            <button onClick={handleRemoveCoupon} disabled={couponBusy} className="text-xs text-red-500 hover:text-red-700 disabled:opacity-50">
              Remove
            </button>
          </div>
        ) : (
          <div className="flex items-center gap-2">
            <input
              type="text"
              placeholder="Coupon code"
              aria-label="Coupon code"
              value={couponInput}
              onChange={e => setCouponInput(e.target.value)}
              className="flex-1 rounded border border-gray-200 px-2 py-1.5 text-sm"
            />
            <button
              onClick={handleApplyCoupon}
              disabled={couponBusy || !couponInput.trim()}
              className="shrink-0 rounded border border-brand-600 px-3 py-1.5 text-sm font-medium text-brand-700 transition-colors hover:bg-brand-50 disabled:opacity-50"
            >
              {couponBusy ? 'Applying…' : 'Apply'}
            </button>
          </div>
        )}
        {couponError && <p className="mt-1.5 text-xs text-red-600">{couponError}</p>}
      </div>
      <div className="mt-3 space-y-1.5">
        <div className="flex items-center justify-between">
          <p className="text-sm text-brand-500">Subtotal</p>
          <p className="text-sm text-brand-700">
            {formatMoney(cart.subtotal_minor_units, cart.items[0].currency_code)}
          </p>
        </div>
        {cart.discount_minor_units > 0 && (
          <div className="flex items-center justify-between">
            <p className="text-sm text-brand-500">Discount</p>
            <p className="text-sm text-brand-700">
              -{formatMoney(cart.discount_minor_units, cart.items[0].currency_code)}
            </p>
          </div>
        )}
        <div className="flex items-center justify-between">
          <p className="text-sm font-medium text-brand-700">Total</p>
          <p className="text-sm font-semibold text-brand-700">
            {formatMoney(cart.subtotal_minor_units - cart.discount_minor_units, cart.items[0].currency_code)}
          </p>
        </div>
      </div>
      <Link
        to="/checkout"
        className="mt-4 block rounded-lg bg-brand-600 px-4 py-2 text-center text-sm font-medium text-white transition-colors hover:bg-brand-700"
      >
        Checkout
      </Link>
    </div>
  )
}
