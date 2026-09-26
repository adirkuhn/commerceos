import { useState, useEffect, useCallback } from 'react'
import { Search, X } from 'lucide-react'
import { useAuth } from '../../auth/AuthContext'
import * as reportsApi from '../../reports/api'
import * as pmsApi from '../../pms/api'
import type { Product, VariantWithSKUs, SKU } from '../../pms/types'
import type { ProductRecommendationsReport } from '../../reports/types'
import { useSingleReport } from '../../reports/hooks/useSingleReport'
import { dateRange, type ReportsPeriod } from '../../reports/period'
import { useDateFormat } from '../../format/DateFormatProvider'
import { ReportPageShell, ReportCard, EmptyState } from './shared'
import { useReportPeriod } from '../../reports/hooks/useReportPeriod'
import { useQueryString } from '../../routing/useQueryState'

const SEARCH_DEBOUNCE_MS = 300
const SEARCH_MIN_LENGTH = 2

interface SKUOption {
  sku: SKU
  variant: VariantWithSKUs
  productName: string
}

export default function FrequentlyBoughtTogetherPage() {
  const { token } = useAuth()
  // In ?period= -- refresh and shared links keep the chosen range.
  const [period, setPeriod] = useReportPeriod()
  const getFrequentlyBoughtTogetherReport = useCallback(reportsApi.getFrequentlyBoughtTogetherReport, [])
  const { data: frequentlyBoughtTogether, loading, error } = useSingleReport(token ?? '', period, getFrequentlyBoughtTogetherReport)

  return (
    <ReportPageShell title="Frequently Bought Together" period={period} onPeriodChange={setPeriod} loading={loading} error={error}>
      <ProductRecommendationsSearch token={token ?? ''} period={period} />

      <ReportCard
        title="Frequently Bought Together"
        description="SKU sets that co-occurred most often in orders this period — support is the number of qualifying orders containing every item in the set"
      >
        {!frequentlyBoughtTogether || frequentlyBoughtTogether.rows.length === 0 ? (
          <EmptyState>No frequently co-occurring SKU sets in this period</EmptyState>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr className="dark:bg-night-900/50 bg-gray-50">
                  <th className="px-4 py-2 text-left text-2xs font-medium text-ink-4 uppercase">Items</th>
                  <th className="px-4 py-2 text-right text-2xs font-medium text-ink-4 uppercase">Support</th>
                  <th className="px-4 py-2 text-right text-2xs font-medium text-ink-4 uppercase">% of Orders</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-line/50">
                {frequentlyBoughtTogether.rows.map((row, i) => (
                  <tr key={i} data-testid="fbt-row" className="hover:bg-hover/30">
                    <td className="px-4 py-2 text-ink">
                      {row.items.map(item => item.product_name || item.sku_code || item.sku_id).join(' + ')}
                    </td>
                    <td className="px-4 py-2 text-right text-ink-3">{row.support}</td>
                    <td className="px-4 py-2 text-right text-ink-3">{row.support_percent.toFixed(1)}%</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </ReportCard>
    </ReportPageShell>
  )
}

// ProductRecommendationsSearch: type-ahead product search -> pick a
// product (and, if it has more than one SKU, pick which SKU) -> "given
// this product, what's frequently bought with it" via the per-SKU
// recommendations endpoint, reusing the same FP-Growth mining as the
// tenant-wide table above but scoped to one product.
function ProductRecommendationsSearch({ token, period }: { token: string; period: ReportsPeriod }) {
  const { timeZone } = useDateFormat().settings
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<Product[]>([])
  const [searching, setSearching] = useState(false)
  const [searchError, setSearchError] = useState<string | null>(null)

  // The chosen product and SKU live in the URL (?product=&sku=), so a
  // refresh or shared link reopens the same recommendations; the product
  // itself is reloaded from its id.
  const [productId, setProductId] = useQueryString('product', { resets: ['sku'] })
  const [selectedSkuId, setSelectedSkuId] = useQueryString('sku')
  const [selectedProduct, setSelectedProduct] = useState<Product | null>(null)
  const [skuOptions, setSkuOptions] = useState<SKUOption[]>([])
  const [loadingSkus, setLoadingSkus] = useState(false)

  const [recommendations, setRecommendations] = useState<ProductRecommendationsReport | null>(null)
  const [recLoading, setRecLoading] = useState(false)
  const [recError, setRecError] = useState<string | null>(null)

  useEffect(() => {
    if (!token || query.trim().length < SEARCH_MIN_LENGTH) {
      setResults([])
      return
    }
    let cancelled = false
    const timer = setTimeout(() => {
      setSearching(true)
      setSearchError(null)
      pmsApi.listProducts(token, { search: query.trim(), perPage: 8 })
        .then(res => { if (!cancelled) setResults(res.products) })
        .catch(e => { if (!cancelled) setSearchError(e instanceof Error ? e.message : 'Unknown error') })
        .finally(() => { if (!cancelled) setSearching(false) })
    }, SEARCH_DEBOUNCE_MS)
    return () => { cancelled = true; clearTimeout(timer) }
  }, [token, query])

  function selectProduct(product: Product) {
    setResults([])
    setQuery('')
    setProductId(product.id)
  }

  function clearSelection() {
    setProductId(null)
  }

  // Load the product in the URL and its SKUs; a single-SKU product picks
  // that SKU automatically.
  useEffect(() => {
    setRecommendations(null)
    if (!token || !productId) {
      setSelectedProduct(null)
      setSkuOptions([])
      return
    }
    let cancelled = false
    setLoadingSkus(true)
    pmsApi.getProduct(token, productId)
      .then(detail => {
        if (cancelled) return
        const options: SKUOption[] = detail.variants.flatMap(variant =>
          (variant.skus ?? []).map(sku => ({ sku, variant, productName: detail.product.name })),
        )
        setSelectedProduct(detail.product)
        setSkuOptions(options)
        if (options.length === 1) setSelectedSkuId(options[0].sku.id)
      })
      .catch(e => { if (!cancelled) setSearchError(e instanceof Error ? e.message : 'Unknown error') })
      .finally(() => { if (!cancelled) setLoadingSkus(false) })
    return () => { cancelled = true }
  }, [token, productId, setSelectedSkuId])

  useEffect(() => {
    if (!token || !selectedSkuId) return
    let cancelled = false
    setRecLoading(true)
    setRecError(null)
    const { from, to } = dateRange(period, new Date(), timeZone)
    reportsApi.getProductRecommendations(token, selectedSkuId, from, to)
      .then(r => { if (!cancelled) setRecommendations(r) })
      .catch(e => { if (!cancelled) setRecError(e instanceof Error ? e.message : 'Unknown error') })
      .finally(() => { if (!cancelled) setRecLoading(false) })
    return () => { cancelled = true }
  }, [token, selectedSkuId, period, timeZone])

  return (
    <ReportCard title="Product recommendations" description="Search a product to see what's frequently bought with it" overflowVisible>
      <div className="p-4 space-y-3">
        {!selectedProduct ? (
          <div className="relative">
            <div className="relative">
              <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-ink-4" />
              <input
                type="text"
                value={query}
                onChange={e => setQuery(e.target.value)}
                placeholder="Search products…"
                className="w-full pl-9 pr-3 py-2 text-xs rounded-lg border border-line bg-surface text-ink focus:outline-none focus:ring-1 focus:ring-focus"
              />
            </div>
            {searching && <p className="text-2xs text-ink-4 mt-1">Searching…</p>}
            {searchError && <p className="text-2xs text-danger mt-1">{searchError}</p>}
            {results.length > 0 && (
              <div className="absolute z-10 mt-1 w-full rounded-lg border border-line bg-raised shadow-lg max-h-64 overflow-y-auto">
                {results.map(product => (
                  <button
                    key={product.id}
                    onClick={() => selectProduct(product)}
                    className="w-full text-left px-3 py-2 text-xs text-ink hover:bg-hover/50"
                  >
                    {product.name}
                  </button>
                ))}
              </div>
            )}
          </div>
        ) : (
          <div className="flex items-center justify-between gap-2">
            <p className="text-xs font-medium text-ink">{selectedProduct.name}</p>
            <button onClick={clearSelection} aria-label="Clear selection" className="p-1 -m-1 rounded text-ink-4 hover:text-ink-2 hover:bg-hover">
              <X size={14} />
            </button>
          </div>
        )}

        {selectedProduct && loadingSkus && <p className="text-2xs text-ink-4">Loading SKUs…</p>}

        {selectedProduct && !loadingSkus && skuOptions.length > 1 && !selectedSkuId && (
          <div className="space-y-1">
            <p className="text-2xs text-ink-4">This product has multiple variants — pick one:</p>
            {skuOptions.map(opt => (
              <button
                key={opt.sku.id}
                onClick={() => setSelectedSkuId(opt.sku.id)}
                className="w-full text-left px-3 py-1.5 rounded-lg text-xs text-ink border border-line hover:bg-hover/50"
              >
                {opt.variant.name} — <span className="font-mono text-ink-4">{opt.sku.sku_code}</span>
              </button>
            ))}
          </div>
        )}

        {selectedSkuId && (
          <div>
            {recLoading && <p className="text-2xs text-ink-4">Loading recommendations…</p>}
            {recError && <p className="text-2xs text-danger">{recError}</p>}
            {recommendations && recommendations.rows.length === 0 && !recLoading && (
              <p className="text-2xs text-ink-4">No frequent co-purchases found for this product in the selected period</p>
            )}
            {recommendations && recommendations.rows.length > 0 && (
              <div data-testid="product-recommendations" className="space-y-1">
                {recommendations.rows.map(row => (
                  <div key={row.sku_id} className="flex items-center justify-between gap-2 text-xs px-3 py-1.5 rounded-lg border border-line">
                    <span className="text-ink truncate">{row.product_name || row.sku_code || row.sku_id}</span>
                    <span className="text-ink-4 shrink-0">support {row.support}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    </ReportCard>
  )
}
