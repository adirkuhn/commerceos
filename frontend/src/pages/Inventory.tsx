import { useState, useEffect } from 'react'
import { Search, AlertTriangle, Trash2 } from 'lucide-react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import { useLocation } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { parseInventoryPath, inventoryPaths, INVENTORY_PATH } from '../inventory/inventoryRoutes'
import { useInventoryHistoryTarget } from '../inventory/useInventoryHistoryTarget'
import { useStackNavigation } from '../routing/useStackNavigation'
import { useQueryEnum, useQueryPage, useQueryString } from '../routing/useQueryState'
import { RecordLoading, RecordNotFound } from '../components/common/RecordStatus'
import Pagination from '../components/common/Pagination'
import { useProducts } from '../pms/hooks/useProducts'
import { useWarehouses } from '../inventory/hooks/useWarehouses'
import { useExpiredBatchReview } from '../inventory/hooks/useExpiredBatchReview'
import { useExpiringBatches } from '../inventory/hooks/useExpiringBatches'
import { useDiscards } from '../inventory/hooks/useDiscards'
import { useDiscardBatch } from '../inventory/hooks/useDiscardBatch'
import { LOW_STOCK_THRESHOLD } from '../inventory/inventoryFormat'
import { ProductStockRow, ProductStockSummary } from '../inventory/ProductStockRow'
import { BatchMovementHistoryPage, ProductMovementHistoryPage, SKUMovementHistoryPage } from '../inventory/MovementHistoryPages'
import { ExpiryDiscardPage } from '../inventory/ExpiryDiscardPage'
import { StockInvalidationProvider } from '../inventory/stockInvalidation'

const PAGE_SIZE = 20

const STOCK_FILTERS = ['all', 'low', 'out'] as const
type StockFilter = typeof STOCK_FILTERS[number]

// ── Page ──────────────────────────────────────────────────────────────────────

// The provider lets a stock action in one product row refresh every other
// row showing a SKU it touched (see stockInvalidation).
export default function Inventory() {
  return (
    <StockInvalidationProvider>
      <InventoryPage />
    </StockInvalidationProvider>
  )
}

function InventoryPage() {
  const { token } = useAuth()
  const { t } = useTranslation(['inventory', 'common'])
  // Search/page/stock filter (?q=&page=&stock=) and every full-screen view
  // (history, expiry review -- see inventoryRoutes) live in the URL, so a
  // refresh or Back keeps the user where they were.
  const location = useLocation()
  const view = parseInventoryPath(location.pathname)
  const { open, exitToOpener } = useStackNavigation()
  const backToList = () => exitToOpener(INVENTORY_PATH)
  const [urlSearch, setUrlSearch] = useQueryString('q', { resets: ['page'] })
  const search = urlSearch ?? ''
  const [searchInput, setSearchInput] = useState(search)
  const [page, setPage] = useQueryPage()
  const [stockFilter, setStockFilter] = useQueryEnum<StockFilter>('stock', STOCK_FILTERS, 'all')
  const [summaryByProduct, setSummaryByProduct] = useState<Record<string, ProductStockSummary>>({})
  const expiredReview = useExpiredBatchReview(token ?? '')
  const expiringBatches = useExpiringBatches(token ?? '')
  const discardsReport = useDiscards(token ?? '')
  const discardHook = useDiscardBatch(token ?? '')

  // Debounced into ?q=; only writes a real change, so landing on a URL
  // with ?page= doesn't get reset by the search write (which clears page).
  useEffect(() => {
    if (searchInput === search) return
    const timeout = setTimeout(() => setUrlSearch(searchInput), 300)
    return () => clearTimeout(timeout)
  }, [searchInput, search, setUrlSearch])
  useEffect(() => { setSearchInput(search) }, [search])

  const { products, total, loading, error } = useProducts(token ?? '', { page, perPage: PAGE_SIZE, search })
  const { warehouses } = useWarehouses(token ?? '')

  // Rows report their own stock total once loaded — low-stock/out-of-stock
  // filtering is client-side over the current page (the product search API
  // has no stock-aware filtering yet), so a row not yet reporting a
  // summary is still shown rather than hidden while it resolves.
  const handleSummary = (productId: string, summary: ProductStockSummary) => {
    setSummaryByProduct(prev => {
      const existing = prev[productId]
      if (existing && existing.totalStock === summary.totalStock && existing.skuCount === summary.skuCount) return prev
      return { ...prev, [productId]: summary }
    })
  }

  const visibleProducts = products.filter(p => {
    if (stockFilter === 'all') return true
    const summary = summaryByProduct[p.id]
    if (!summary) return true
    if (stockFilter === 'out') return summary.totalStock <= 0
    return summary.totalStock > 0 && summary.totalStock <= LOW_STOCK_THRESHOLD
  })

  if (view.kind === 'productHistory' || view.kind === 'skuHistory' || view.kind === 'batchHistory') {
    return <InventoryHistoryView token={token ?? ''} view={view} warehouses={warehouses} onClose={backToList} />
  }

  if (view.kind === 'expiry') {
    return (
      <ExpiryDiscardPage
        onClose={backToList}
        expiredReview={expiredReview}
        expiringBatches={expiringBatches}
        discardsReport={discardsReport}
        discardHook={discardHook}
      />
    )
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-3 px-5 py-3 border-b border-line bg-sunken shrink-0">
        <div className="relative flex-1 max-w-sm">
          <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 text-ink-4" />
          <input
            value={searchInput}
            onChange={e => setSearchInput(e.target.value)}
            placeholder={t('list.searchPlaceholder')}
            className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg bg-raised border border-line text-ink placeholder-ink-4 outline-none focus:ring-1 focus:ring-focus"
          />
        </div>
        <div className="flex items-center rounded-lg border border-line overflow-hidden shrink-0">
          {([
            ['all', t('list.filterAll')],
            ['low', t('list.filterLow')],
            ['out', t('list.filterOut')],
          ] as const).map(([value, label]) => (
            <button aria-pressed={stockFilter === value}
              key={value}
              type="button"
              onClick={() => setStockFilter(value)}
              className={clsx(
                'px-2.5 py-1.5 text-2xs font-medium transition-colors',
                stockFilter === value
                  ? 'bg-accent-solid text-white'
                  : 'bg-raised text-ink-3 hover:bg-hover',
              )}
            >
              {label}
            </button>
          ))}
        </div>
        {expiredReview.batches.length > 0 && (
          <button
            onClick={() => open(inventoryPaths.expiry)}
            className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg dark:bg-red-500/10 bg-red-50 text-danger text-2xs font-medium hover:dark:bg-red-500/20 hover:bg-red-100 transition-colors shrink-0"
          >
            <AlertTriangle size={12} /> {t('list.expiredBatches', { n: expiredReview.batches.length })}
          </button>
        )}
        <button
          onClick={() => open(`${inventoryPaths.expiry}?tab=discarded`)}
          className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg bg-raised border border-line text-ink-3 text-2xs font-medium hover:bg-hover transition-colors shrink-0"
        >
          <Trash2 size={12} /> {t('list.discarded')}
        </button>
        <span className="text-xs text-ink-4 ml-auto">{t('list.productCount', { count: total })}</span>
      </div>

      {error && (
        <div className="flex items-center gap-1.5 px-5 py-2 text-xs text-danger">
          <AlertTriangle size={12} /> {error}
        </div>
      )}

      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
            <tr>
              <th className="px-2 py-2.5 w-8" />
              <th className="px-2 py-2.5 w-8" />
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 caps-label">{t('list.colProduct')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 caps-label">{t('list.colPrice')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 caps-label">{t('list.colSalePrice')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 caps-label">{t('list.colStock')}</th>
              <th className="px-3 py-2.5 w-8" />
            </tr>
          </thead>
          <tbody className="divide-y divide-line/60">
            {loading && (
              <tr><td colSpan={7} className="px-4 py-6 text-center text-ink-4">{t('common:status.loading')}</td></tr>
            )}
            {!loading && visibleProducts.length === 0 && (
              <tr><td colSpan={7} className="px-4 py-6 text-center text-ink-4">{t('list.empty')}</td></tr>
            )}
            {/* A SKU action URL for a product not on this list page (shared
                link, or the list moved on) still opens -- pinned on top. */}
            {!loading && view.kind === 'skuAction' && !visibleProducts.some(p => p.id === view.productId) && (
              <ProductStockRow
                key={`pinned-${view.productId}`} token={token ?? ''} productId={view.productId} productName="" warehouses={warehouses}
                onSummary={handleSummary} onViewHistory={sku => open(inventoryPaths.skuHistory(view.productId, sku.id))}
                onViewBatchHistory={(sku, batch) => open(inventoryPaths.batchHistory(view.productId, sku.id, batch.id))}
                onViewProductHistory={() => open(inventoryPaths.productHistory(view.productId))}
              />
            )}
            {!loading && visibleProducts.map(p => (
              <ProductStockRow
                key={p.id} token={token ?? ''} productId={p.id} productName={p.name} productImageUrl={p.image_url} warehouses={warehouses}
                onSummary={handleSummary} onViewHistory={sku => open(inventoryPaths.skuHistory(p.id, sku.id))}
                onViewBatchHistory={(sku, batch) => open(inventoryPaths.batchHistory(p.id, sku.id, batch.id))}
                onViewProductHistory={() => open(inventoryPaths.productHistory(p.id))}
              />
            ))}
          </tbody>
        </table>
      </div>

      <Pagination page={page} pageSize={PAGE_SIZE} totalItems={total} onPageChange={setPage} />
    </div>
  )
}

// Movement history page for the product / SKU / batch named in the URL
// (see inventoryRoutes) -- reloads its subject from the ids, so it works
// straight from a refresh or a shared link.
function InventoryHistoryView({ token, view, warehouses, onClose }: {
  token: string
  view: Parameters<typeof useInventoryHistoryTarget>[1]
  warehouses: { id: string; name: string }[]
  onClose: () => void
}) {
  const target = useInventoryHistoryTarget(token, view)
  const { t } = useTranslation('inventory')
  const data = target.data
  if (!data) {
    if (target.loading) return <RecordLoading />
    // A missing SKU/batch reports itself; anything else means the product
    // lookup (the first request) failed.
    // (Matched on the error's own English text, which is never shown.)
    const subject = target.error === 'SKU not found' ? t('subject.sku') : target.error === 'Batch not found' ? t('subject.batch') : t('subject.product')
    return <RecordNotFound label={subject} onBack={onClose} />
  }
  if (view.kind === 'batchHistory' && data.sku && data.batch) {
    return <BatchMovementHistoryPage token={token} sku={data.sku} batch={data.batch} warehouses={warehouses} onClose={onClose} />
  }
  if (view.kind === 'skuHistory' && data.sku) {
    return <SKUMovementHistoryPage token={token} sku={data.sku} warehouses={warehouses} onClose={onClose} />
  }
  return <ProductMovementHistoryPage token={token} productName={data.productName} skus={data.skus} warehouses={warehouses} onClose={onClose} />
}
