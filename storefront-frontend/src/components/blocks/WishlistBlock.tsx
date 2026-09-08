import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { getProduct } from '../../api/products'
import { useWishlist } from '../../wishlist/WishlistContext'
import { formatMoney } from '../../format'
import type { StorefrontProduct } from '../../types'

export function WishlistBlock() {
  const { wishlist, loading: wishlistLoading, toggle } = useWishlist()
  const [products, setProducts] = useState<StorefrontProduct[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const productIds = wishlist?.product_ids ?? []

  useEffect(() => {
    if (wishlistLoading) return
    if (productIds.length === 0) {
      setProducts([])
      setLoading(false)
      return
    }
    let cancelled = false
    setLoading(true)
    setError(null)
    Promise.all(productIds.map(id => getProduct(id).catch(() => null)))
      .then(results => { if (!cancelled) setProducts(results.filter((p): p is StorefrontProduct => p !== null)) })
      .catch(e => { if (!cancelled) setError(e instanceof Error ? e.message : 'Unknown error') })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
    // productIds is derived from wishlist each render -- compare by its
    // joined value so this only re-fetches when the actual id set changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [wishlistLoading, productIds.join(',')])

  if (wishlistLoading || loading) return <p className="p-4 text-sm text-gray-500">Loading wishlist…</p>
  if (error) return <p className="p-4 text-sm text-red-600">Could not load your wishlist: {error}</p>
  if (products.length === 0) {
    return (
      <div className="p-6 text-center">
        <p className="text-brand-500">Your wishlist is empty.</p>
        <Link to="/products" className="mt-2 inline-block text-sm text-brand-700 underline">Continue shopping</Link>
      </div>
    )
  }

  return (
    <div className="grid h-full grid-cols-2 gap-4 overflow-auto p-4 sm:grid-cols-3 md:grid-cols-4">
      {products.map(p => {
        const sku = p.variants[0]?.skus[0]
        return (
          <div key={p.id} className="rounded-lg border border-gray-200 p-3">
            <Link to={`/products/${p.id}`} className="block hover:border-brand-500">
              {p.images[0] && <img src={p.images[0]} alt={p.name} className="mb-2 h-24 w-full rounded object-cover" />}
              <p className="truncate text-sm font-medium text-brand-700">{p.name}</p>
              {sku && (
                <p className="text-sm text-brand-500">
                  {sku.sale_price_minor_units != null ? (
                    <>
                      <span className="mr-1 text-gray-400 line-through">{formatMoney(sku.price_minor_units, sku.currency_code)}</span>
                      {formatMoney(sku.sale_price_minor_units, sku.currency_code)}
                    </>
                  ) : formatMoney(sku.price_minor_units, sku.currency_code)}
                </p>
              )}
            </Link>
            <button
              onClick={() => toggle(p.id)}
              className="mt-2 text-xs text-red-500 hover:text-red-700"
            >
              Remove
            </button>
          </div>
        )
      })}
    </div>
  )
}
