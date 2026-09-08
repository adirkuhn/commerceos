import { Link } from 'react-router-dom'
import { useCart } from '../../cart/CartContext'
import { formatMoney } from '../../format'

export function CartSummaryBlock() {
  const { cart, loading, setItemQuantity, removeItem, error } = useCart()

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
      <div className="mt-4 flex items-center justify-between">
        <p className="text-sm font-medium text-brand-700">Subtotal</p>
        <p className="text-sm font-semibold text-brand-700">
          {formatMoney(cart.subtotal_minor_units, cart.items[0].currency_code)}
        </p>
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
