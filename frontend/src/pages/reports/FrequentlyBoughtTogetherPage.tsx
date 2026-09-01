import { useState, useEffect, useCallback } from 'react'
import { Search, X } from 'lucide-react'
import { useAuth } from '../../auth/AuthContext'
import * as reportsApi from '../../reports/api'
import * as pmsApi from '../../pms/api'
import type { Product, VariantWithSKUs, SKU } from '../../pms/types'
import type { ProductRecommendationsReport } from '../../reports/types'
import { useSingleReport } from '../../reports/hooks/useSingleReport'
import { dateRange, type ReportsPeriod } from '../../reports/period'
import { ReportPageShell, ReportCard, EmptyState } from './shared'

const SEARCH_DEBOUNCE_MS = 300
const SEARCH_MIN_LENGTH = 2

interface SKUOption {
  sku: SKU
  variant: VariantWithSKUs
  productName: string
}

export default function FrequentlyBoughtTogetherPage() {
  const { token } = useAuth()
  const [period, setPeriod] = useState<ReportsPeriod>('7D')
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
                <tr className="dark:bg-[#111827]/50 bg-gray-50">
                  <th className="px-4 py-2 text-left text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase">Items</th>
                  <th className="px-4 py-2 text-right text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase">Support</th>
                  <th className="px-4 py-2 text-right text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase">% of Orders</th>
                </tr>
              </thead>
              <tbody className="divide-y dark:divide-[#2D3748]/50 divide-gray-100">
                {frequentlyBoughtTogether.rows.map((row, i) => (
                  <tr key={i} data-testid="fbt-row" className="hover:dark:bg-[#2D3748]/30 hover:bg-gray-50">
                    <td className="px-4 py-2 dark:text-gray-200 text-gray-800">
                      {row.items.map(item => item.product_name || item.sku_code || item.sku_id).join(' + ')}
                    </td>
                    <td className="px-4 py-2 text-right dark:text-gray-400 text-gray-500">{row.support}</td>
                    <td className="px-4 py-2 text-right dark:text-gray-400 text-gray-500">{row.support_percent.toFixed(1)}%</td>
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
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<Product[]>([])
  const [searching, setSearching] = useState(false)
  const [searchError, setSearchError] = useState<string | null>(null)

  const [selectedProduct, setSelectedProduct] = useState<Product | null>(null)
  const [skuOptions, setSkuOptions] = useState<SKUOption[]>([])
  const [selectedSkuId, setSelectedSkuId] = useState<string | null>(null)
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
    setSelectedProduct(product)
    setResults([])
    setQuery('')
    setSelectedSkuId(null)
    setRecommendations(null)
    setLoadingSkus(true)
    pmsApi.getProduct(token, product.id)
      .then(detail => {
        const options: SKUOption[] = detail.variants.flatMap(variant =>
          (variant.skus ?? []).map(sku => ({ sku, variant, productName: detail.product.name })),
        )
        setSkuOptions(options)
        if (options.length === 1) {
          setSelectedSkuId(options[0].sku.id)
        }
      })
      .catch(e => setSearchError(e instanceof Error ? e.message : 'Unknown error'))
      .finally(() => setLoadingSkus(false))
  }

  function clearSelection() {
    setSelectedProduct(null)
    setSkuOptions([])
    setSelectedSkuId(null)
    setRecommendations(null)
  }

  useEffect(() => {
    if (!token || !selectedSkuId) return
    let cancelled = false
    setRecLoading(true)
    setRecError(null)
    const { from, to } = dateRange(period, new Date())
    reportsApi.getProductRecommendations(token, selectedSkuId, from, to)
      .then(r => { if (!cancelled) setRecommendations(r) })
      .catch(e => { if (!cancelled) setRecError(e instanceof Error ? e.message : 'Unknown error') })
      .finally(() => { if (!cancelled) setRecLoading(false) })
    return () => { cancelled = true }
  }, [token, selectedSkuId, period])

  return (
    <ReportCard title="Product Recommendations" description="Search a product to see what's frequently bought with it" overflowVisible>
      <div className="p-4 space-y-3">
        {!selectedProduct ? (
          <div className="relative">
            <div className="relative">
              <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 dark:text-gray-500 text-gray-400" />
              <input
                type="text"
                value={query}
                onChange={e => setQuery(e.target.value)}
                placeholder="Search products…"
                className="w-full pl-9 pr-3 py-2 text-xs rounded-lg border dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-white dark:text-gray-200 text-gray-800 focus:outline-none focus:ring-1 focus:ring-blue-500"
              />
            </div>
            {searching && <p className="text-[10px] dark:text-gray-500 text-gray-400 mt-1">Searching…</p>}
            {searchError && <p className="text-[10px] text-red-400 mt-1">{searchError}</p>}
            {results.length > 0 && (
              <div className="absolute z-10 mt-1 w-full rounded-lg border dark:border-[#2D3748] border-gray-200 dark:bg-[#1F2937] bg-white shadow-lg max-h-64 overflow-y-auto">
                {results.map(product => (
                  <button
                    key={product.id}
                    onClick={() => selectProduct(product)}
                    className="w-full text-left px-3 py-2 text-xs dark:text-gray-200 text-gray-800 hover:dark:bg-[#2D3748]/50 hover:bg-gray-50"
                  >
                    {product.name}
                  </button>
                ))}
              </div>
            )}
          </div>
        ) : (
          <div className="flex items-center justify-between gap-2">
            <p className="text-xs font-medium dark:text-gray-200 text-gray-800">{selectedProduct.name}</p>
            <button onClick={clearSelection} className="dark:text-gray-500 text-gray-400 hover:dark:text-gray-200 hover:text-gray-700">
              <X size={14} />
            </button>
          </div>
        )}

        {selectedProduct && loadingSkus && <p className="text-[10px] dark:text-gray-500 text-gray-400">Loading SKUs…</p>}

        {selectedProduct && !loadingSkus && skuOptions.length > 1 && !selectedSkuId && (
          <div className="space-y-1">
            <p className="text-[10px] dark:text-gray-500 text-gray-400">This product has multiple variants — pick one:</p>
            {skuOptions.map(opt => (
              <button
                key={opt.sku.id}
                onClick={() => setSelectedSkuId(opt.sku.id)}
                className="w-full text-left px-3 py-1.5 rounded-lg text-xs dark:text-gray-200 text-gray-800 border dark:border-[#2D3748] border-gray-200 hover:dark:bg-[#2D3748]/50 hover:bg-gray-50"
              >
                {opt.variant.name} — <span className="font-mono dark:text-gray-500 text-gray-400">{opt.sku.sku_code}</span>
              </button>
            ))}
          </div>
        )}

        {selectedSkuId && (
          <div>
            {recLoading && <p className="text-[10px] dark:text-gray-500 text-gray-400">Loading recommendations…</p>}
            {recError && <p className="text-[10px] text-red-400">{recError}</p>}
            {recommendations && recommendations.rows.length === 0 && !recLoading && (
              <p className="text-[10px] dark:text-gray-500 text-gray-400">No frequent co-purchases found for this product in the selected period</p>
            )}
            {recommendations && recommendations.rows.length > 0 && (
              <div data-testid="product-recommendations" className="space-y-1">
                {recommendations.rows.map(row => (
                  <div key={row.sku_id} className="flex items-center justify-between gap-2 text-xs px-3 py-1.5 rounded-lg border dark:border-[#2D3748] border-gray-200">
                    <span className="dark:text-gray-200 text-gray-800 truncate">{row.product_name || row.sku_code || row.sku_id}</span>
                    <span className="dark:text-gray-500 text-gray-400 shrink-0">support {row.support}</span>
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
