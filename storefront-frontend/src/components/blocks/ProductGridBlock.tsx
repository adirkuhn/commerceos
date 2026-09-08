import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { listProducts } from '../../api/products'
import { formatMoney } from '../../format'
import type { StorefrontProductSummary } from '../../types'

export function ProductGridBlock() {
  const [products, setProducts] = useState<StorefrontProductSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    listProducts()
      .then(result => { if (!cancelled) setProducts(result.items) })
      .catch(e => { if (!cancelled) setError(e instanceof Error ? e.message : 'Unknown error') })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [])

  if (loading) return <p className="p-4 text-sm text-gray-500">Loading products…</p>
  if (error) return <p className="p-4 text-sm text-red-600">Could not load products: {error}</p>
  if (products.length === 0) return <p className="p-4 text-sm text-gray-500">No products yet.</p>

  return (
    <div className="grid h-full grid-cols-2 gap-4 overflow-auto p-4 sm:grid-cols-3 md:grid-cols-4">
      {products.map(p => (
        <Link key={p.id} to={`/products/${p.id}`} className="block rounded-lg border border-gray-200 p-3 hover:border-brand-500">
          {p.image_url && <img src={p.image_url} alt={p.name} className="mb-2 h-24 w-full rounded object-cover" />}
          <p className="truncate text-sm font-medium text-brand-700">{p.name}</p>
          {p.price_minor_units != null && p.currency_code && (
            <p className="text-sm text-brand-500">
              {p.sale_price_minor_units != null ? (
                <>
                  <span className="mr-1 text-gray-400 line-through">{formatMoney(p.price_minor_units, p.currency_code)}</span>
                  {formatMoney(p.sale_price_minor_units, p.currency_code)}
                </>
              ) : formatMoney(p.price_minor_units, p.currency_code)}
            </p>
          )}
        </Link>
      ))}
    </div>
  )
}
