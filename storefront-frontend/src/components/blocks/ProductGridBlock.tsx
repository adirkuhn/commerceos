import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { listProducts } from '../../api/products'
import { useWishlist } from '../../wishlist/WishlistContext'
import { formatMoney } from '../../format'
import type { StorefrontCategoryFacet, StorefrontProductSummary } from '../../types'

export function ProductGridBlock() {
  const { isSaved, toggle } = useWishlist()
  const [searchParams, setSearchParams] = useSearchParams()
  const categoryId = searchParams.get('category') ?? undefined
  const urlSearch = searchParams.get('q') ?? ''

  const [searchInput, setSearchInput] = useState(urlSearch)
  const [products, setProducts] = useState<StorefrontProductSummary[]>([])
  const [facets, setFacets] = useState<StorefrontCategoryFacet[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => setSearchInput(urlSearch), [urlSearch])

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    listProducts({ categoryId, search: urlSearch || undefined })
      .then(result => { if (!cancelled) { setProducts(result.items); setFacets(result.category_facets) } })
      .catch(e => { if (!cancelled) setError(e instanceof Error ? e.message : 'Unknown error') })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [categoryId, urlSearch])

  function submitSearch(e: React.FormEvent) {
    e.preventDefault()
    const next = new URLSearchParams(searchParams)
    if (searchInput.trim()) next.set('q', searchInput.trim())
    else next.delete('q')
    setSearchParams(next)
  }

  function selectCategory(id: string | null) {
    const next = new URLSearchParams(searchParams)
    if (id) next.set('category', id)
    else next.delete('category')
    setSearchParams(next)
  }

  return (
    <div className="flex h-full gap-4 overflow-auto p-4">
      <aside className="w-48 shrink-0">
        <form onSubmit={submitSearch} className="mb-4">
          <input
            type="search"
            value={searchInput}
            onChange={e => setSearchInput(e.target.value)}
            placeholder="Search products…"
            aria-label="Search products"
            className="w-full rounded border border-gray-200 px-2 py-1.5 text-sm"
          />
        </form>
        {facets.length > 0 && (
          <nav aria-label="Categories">
            <p className="mb-2 text-xs font-medium uppercase tracking-wide text-brand-500">Categories</p>
            <ul className="space-y-1 text-sm">
              <li>
                <button
                  onClick={() => selectCategory(null)}
                  className={!categoryId ? 'font-semibold text-brand-700' : 'text-brand-500 hover:text-brand-700'}
                >
                  All
                </button>
              </li>
              {facets.map(f => (
                <li key={f.category_id}>
                  <button
                    onClick={() => selectCategory(f.category_id)}
                    className={categoryId === f.category_id ? 'font-semibold text-brand-700' : 'text-brand-500 hover:text-brand-700'}
                  >
                    {f.category_name} ({f.count})
                  </button>
                </li>
              ))}
            </ul>
          </nav>
        )}
      </aside>

      <div className="flex-1">
        {loading && <p className="p-4 text-sm text-gray-500">Loading products…</p>}
        {!loading && error && <p className="p-4 text-sm text-red-600">Could not load products: {error}</p>}
        {!loading && !error && products.length === 0 && <p className="p-4 text-sm text-gray-500">No products found.</p>}
        {!loading && !error && products.length > 0 && (
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4">
            {products.map(p => (
              <div key={p.id} className="relative rounded-lg border border-gray-200 p-3 hover:border-brand-500">
                <button
                  onClick={e => { e.preventDefault(); toggle(p.id) }}
                  aria-label={isSaved(p.id) ? 'Remove from wishlist' : 'Save to wishlist'}
                  aria-pressed={isSaved(p.id)}
                  className="absolute right-2 top-2 z-10 text-sm text-brand-500 hover:text-brand-700"
                >
                  {isSaved(p.id) ? '♥' : '♡'}
                </button>
                <Link to={`/products/${p.id}`} className="block">
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
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
