import { useState, useEffect, useCallback } from 'react'
import { ChevronRight, ChevronDown, History, Image as ImageIcon } from 'lucide-react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import { useLocation } from 'react-router-dom'
import { parseInventoryPath, INVENTORY_PATH } from './inventoryRoutes'
import { useStackNavigation } from '../routing/useStackNavigation'
import { useQuerySet } from '../routing/useQueryState'
import { RecordNotFound } from '../components/common/RecordStatus'
import { useProductDetail } from '../pms/hooks/useProductDetail'
import * as inventoryApi from './api'
import type { SKU } from '../pms/types'
import type { Batch } from './types'
import { effectiveLowStockThreshold, stockTone } from './lowStock'
import { Money } from '../format/amountVisibility'
import { SKUStockPanel } from './SKUStockPanel'
import { useOnStockInvalidated } from './stockInvalidation'
import { useDateFormat } from '../format/DateFormatProvider'

// ── Product row ───────────────────────────────────────────────────────────────

// A product with more than one SKU (named variants) has no single
// "the" price/sale price to show at the product-row level — the price
// columns show a min–max range instead, and the sale-price column shows
// "-" rather than guessing which variant's sale is "the" one.
// shopLowStockThreshold is the shop's setting; the product's own line
// (from its detail) wins when set.
export function ProductStockRow({ token, productId, productName, productImageUrl, warehouses, shopLowStockThreshold, onViewHistory, onViewBatchHistory, onViewProductHistory }: {
  token: string; productId: string; productName: string; productImageUrl?: string | null; warehouses: { id: string; name: string }[]
  shopLowStockThreshold?: number
  onViewHistory: (sku: SKU) => void
  onViewBatchHistory: (sku: SKU, batch: Batch) => void
  onViewProductHistory: (productName: string, skus: SKU[]) => void
}) {
  const { formatDate } = useDateFormat()
  const { t } = useTranslation('inventory')
  // Expanded rows live in ?open= so a refresh, Back or a shared link keeps
  // them; a SKU action URL for this product (its modal) also expands it.
  const openRows = useQuerySet('open')
  const location = useLocation()
  const routeView = parseInventoryPath(location.pathname)
  const routeSkuId = routeView.kind === 'skuAction' && routeView.productId === productId ? routeView.skuId : null
  const expanded = openRows.has(productId) || routeSkuId !== null
  const toggleExpanded = () => openRows.toggle(productId)

  const { detail, loading, refresh } = useProductDetail(token, productId)
  const skus = (detail?.variants ?? []).flatMap(v => v.skus ?? [])
  const { exitToOpener } = useStackNavigation()
  const navigateToList = () => exitToOpener(INVENTORY_PATH)
  const skusKey = skus.map(s => s.id).join(',')
  // Kept as a separate lookup rather than threading a variantName field
  // through SKU itself -- every other consumer of `skus` here (movement
  // history, price range) has no use for it.
  const variantNameBySKU = Object.fromEntries(
    (detail?.variants ?? []).flatMap(v => (v.skus ?? []).map(s => [s.id, v.name] as const)),
  )

  const [stockBySKU, setStockBySKU] = useState<Record<string, number>>({})
  const [stockLoading, setStockLoading] = useState(true)

  // Also called directly (not just from the effect below) after any stock
  // mutation in a child SKUStockPanel, via onSKUUpdated -- so the collapsed
  // row's total never goes stale without a full page reload.
  const loadStock = useCallback(() => {
    if (!token || skus.length === 0) {
      setStockBySKU({})
      setStockLoading(false)
      return
    }
    setStockLoading(true)
    Promise.all(skus.map(sku => inventoryApi.getSKUStockLevel(token, sku.id).then(l => [sku.id, l.total] as const).catch(() => [sku.id, 0] as const)))
      .then(pairs => setStockBySKU(Object.fromEntries(pairs)))
      .finally(() => setStockLoading(false))
    // eslint-disable-next-line react-hooks/exhaustive-deps -- skusKey is the stable dependency; skus itself is a new array every render
  }, [token, skusKey])

  useEffect(() => { loadStock() }, [loadStock])
  // Another row's action (e.g. a case split landing units on one of these
  // SKUs) must refresh this row's total too.
  useOnStockInvalidated(skus.map(s => s.id), loadStock)

  const totalStock = Object.values(stockBySKU).reduce((sum, n) => sum + n, 0)
  const tone = stockTone(totalStock, effectiveLowStockThreshold(detail?.product.low_stock_threshold, shopLowStockThreshold))

  const singleSKU = skus.length === 1 ? skus[0] : null
  const prices = skus.map(s => s.price_minor_units)
  const priceRange = prices.length > 0 ? [Math.min(...prices), Math.max(...prices)] as const : null

  return (
    <>
      <tr className="hover:bg-hover/60 transition-colors cursor-pointer" onClick={toggleExpanded}>
        <td className="px-4 py-2.5 w-8">
          {expanded ? <ChevronDown size={13} className="text-ink-4" /> : <ChevronRight size={13} className="text-ink-4" />}
        </td>
        <td className="px-2 py-2.5">
          {productImageUrl ? (
            <img src={productImageUrl} alt="" className="w-6 h-6 rounded object-cover dark:bg-night-700 bg-gray-200" />
          ) : (
            <div className="w-6 h-6 rounded dark:bg-night-700 bg-gray-200 flex items-center justify-center">
              <ImageIcon size={12} className="text-ink-4" />
            </div>
          )}
        </td>
        <td className="px-3 py-2.5 font-medium text-ink">{productName || detail?.product.name}</td>
        <td className="px-3 py-2.5" onClick={e => e.stopPropagation()}>
          {singleSKU ? (
            <span className="text-xs text-ink-2"><Money minor={singleSKU.price_minor_units} currency={singleSKU.currency_code} /></span>
          ) : priceRange ? (
            <span className="text-xs text-ink-2">
              <Money minor={priceRange[0]} currency={skus[0].currency_code} />
              {priceRange[0] !== priceRange[1] && <> – <Money minor={priceRange[1]} currency={skus[0].currency_code} /></>}
            </span>
          ) : (
            <span className="text-xs text-ink-4">—</span>
          )}
        </td>
        <td className="px-3 py-2.5" onClick={e => e.stopPropagation()}>
          {singleSKU && singleSKU.sale_price_minor_units != null ? (
            <span className="flex items-center gap-1">
              <span className="text-xs font-medium text-success"><Money minor={singleSKU.sale_price_minor_units} currency={singleSKU.currency_code} /></span>
              {singleSKU.sale_end_date && (
                <span className="text-2xs text-ink-4">{t('row.saleUntil', { date: formatDate(singleSKU.sale_end_date) })}</span>
              )}
            </span>
          ) : (
            <span className="text-xs text-ink-4">—</span>
          )}
        </td>
        <td className="px-3 py-2.5">
          {stockLoading ? (
            <span className="text-xs text-ink-4">…</span>
          ) : (
            <span className={clsx(
              'text-xs font-medium tabular-nums',
              tone === 'out' ? 'text-danger' : tone === 'low' ? 'text-warning' : 'text-ink-2',
            )}>
              {totalStock}
            </span>
          )}
        </td>
        <td className="px-3 py-2.5" onClick={e => e.stopPropagation()}>
          <button
            type="button"
            onClick={() => onViewProductHistory(productName, skus)}
            aria-label={t('row.viewHistory')}
            title={t('row.viewHistory')}
            className="flex items-center gap-1 text-2xs text-ink-4 hover:text-accent"
          >
            <History size={11} /> {t('row.history')}
          </button>
        </td>
      </tr>
      {expanded && (
        <tr>
          <td colSpan={7} className="p-0">
            {loading && <p className="pl-8 py-3 text-xs text-ink-4">{t('row.loadingSkus')}</p>}
            {!loading && skus.length === 0 && <p className="pl-8 py-3 text-xs text-ink-4">{t('row.noSkus')}</p>}
            {!loading && detail && routeSkuId && !skus.some(s => s.id === routeSkuId) && (
              <div className="py-4"><RecordNotFound label={t('subject.sku')} onBack={() => navigateToList()} /></div>
            )}
            {!loading && skus.map(sku => (
              <SKUStockPanel key={sku.id} token={token} productId={productId} sku={sku} variantName={variantNameBySKU[sku.id]} warehouses={warehouses} onSKUUpdated={refresh} onStockChanged={loadStock} onViewHistory={onViewHistory} onViewBatchHistory={onViewBatchHistory} />
            ))}
          </td>
        </tr>
      )}
    </>
  )
}
