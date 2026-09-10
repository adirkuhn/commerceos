import { Link } from 'react-router-dom'
import { listProducts } from '../../api/products'
import { useWishlist } from '../../wishlist/WishlistContext'
import { formatMoney } from '../../format'
import { useEffect, useState } from 'react'
import type { StorefrontProductSummary } from '../../types'

// *_GRID_COLS_CLASS map a column count to a literal, complete Tailwind
// class (including the sm: variant prefix on the desktop one) --
// Tailwind's JIT scanner extracts whole class tokens straight from this
// file's source text, so assembling "sm:" + a bare "grid-cols-N" at
// runtime would never produce a token that actually appears in source,
// and the corresponding CSS would silently never be generated. Clamped to
// this fixed 1-6 / 1-3 range for the same reason -- every value used at
// runtime must already exist here, literally, at build time.
const DESKTOP_GRID_COLS_CLASS: Record<number, string> = {
  1: 'sm:grid-cols-1', 2: 'sm:grid-cols-2', 3: 'sm:grid-cols-3', 4: 'sm:grid-cols-4', 5: 'sm:grid-cols-5', 6: 'sm:grid-cols-6',
}
const MOBILE_GRID_COLS_CLASS: Record<number, string> = {
  1: 'grid-cols-1', 2: 'grid-cols-2', 3: 'grid-cols-3',
}

function clamp(n: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, Math.round(n)))
}

interface Props {
  title: string
  subtitle?: string
  // Manual mode (productIds set, non-empty) wins over category mode --
  // a merchant picking specific products doesn't also pick a category.
  categoryId?: string
  productIds?: string[]
  excludeProductIds?: string[]
  limit: number
  itemsPerRowDesktop?: number
  itemsPerRowMobile?: number
}

// FeaturedProductsBlock is a compact, repeatable "N products" section --
// e.g. a home page stacking a "Snacks" section, a "Bebidas" section, a
// "Mate" section, each its own block instance with its own title/
// selection/limit. Distinct from ProductGridBlock, which is the full
// /products listing page (search, category sidebar, one category from
// the URL) -- that page can't be placed twice with two different fixed
// selections, this block can.
export function FeaturedProductsBlock({
  title, subtitle, categoryId, productIds, excludeProductIds, limit, itemsPerRowDesktop = 4, itemsPerRowMobile = 2,
}: Props) {
  const { isSaved, toggle } = useWishlist()
  const [products, setProducts] = useState<StorefrontProductSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const manualMode = !!productIds && productIds.length > 0

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    listProducts({
      categoryId: manualMode ? undefined : categoryId,
      productIds: manualMode ? productIds : undefined,
      excludeIds: excludeProductIds,
      perPage: limit,
    })
      .then(result => { if (!cancelled) setProducts(result.items) })
      .catch(e => { if (!cancelled) setError(e instanceof Error ? e.message : 'Unknown error') })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [manualMode, categoryId, (productIds ?? []).join(','), (excludeProductIds ?? []).join(','), limit])

  const desktopCols = DESKTOP_GRID_COLS_CLASS[clamp(itemsPerRowDesktop, 1, 6)]
  const mobileCols = MOBILE_GRID_COLS_CLASS[clamp(itemsPerRowMobile, 1, 3)]

  return (
    <div className="h-full w-full overflow-auto p-4">
      {(title || subtitle) && (
        <div className="mb-3">
          {title && <h2 className="text-xl font-semibold text-brand-700">{title}</h2>}
          {subtitle && <p className="text-sm text-brand-500">{subtitle}</p>}
        </div>
      )}
      {loading && <p className="text-sm text-gray-500">Loading products…</p>}
      {!loading && error && <p className="text-sm text-red-600">Could not load products: {error}</p>}
      {!loading && !error && products.length === 0 && <p className="text-sm text-gray-500">No products found.</p>}
      {!loading && !error && products.length > 0 && (
        <div className={`grid gap-4 ${mobileCols} ${desktopCols}`}>
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
  )
}
