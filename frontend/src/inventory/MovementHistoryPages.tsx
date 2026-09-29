import { useState, useEffect } from 'react'
import { ArrowLeft } from 'lucide-react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import * as inventoryApi from './api'
import type { SKU } from '../pms/types'
import type { Batch, StockMovement } from './types'
import { movementTypeLabel } from './inventoryFormat'
import { useDateFormat } from '../format/DateFormatProvider'

type ProductMovementRow = StockMovement & { skuCode: string; balance: number }

export function ProductMovementHistoryPage({ token, productName, skus, warehouses, onClose }: {
  token: string; productName: string; skus: SKU[]; warehouses: { id: string; name: string }[]; onClose: () => void
}) {
  const { formatDateTime } = useDateFormat()
  const { t } = useTranslation(['inventory', 'common'])
  const [rows, setRows] = useState<ProductMovementRow[] | null>(null)
  const [batches, setBatches] = useState<Batch[]>([])
  const [error, setError] = useState<string | null>(null)
  const skusKey = skus.map(s => s.id).join(',')

  useEffect(() => {
    let cancelled = false
    Promise.all(skus.map(sku => Promise.all([
      inventoryApi.listStockMovements(token, sku.id),
      inventoryApi.listBatches(token, sku.id),
    ]).then(([movements, batches]) => ({ sku, movements, batches }))))
      .then(perSKU => {
        if (cancelled) return
        // One running balance across every SKU of this product -- an
        // account book for the product as a whole, not per-variant --
        // computed oldest-first, then reversed for newest-first display.
        const combined = perSKU.flatMap(({ sku, movements }) => movements.map(m => ({ ...m, skuCode: sku.sku_code })))
        combined.sort((a, b) => a.occurred_at.localeCompare(b.occurred_at))
        let balance = 0
        const withBalance: ProductMovementRow[] = combined.map(m => {
          balance += m.quantity_delta
          return { ...m, balance }
        })
        setRows(withBalance.reverse())
        setBatches(perSKU.flatMap(p => p.batches))
      })
      .catch(e => { if (!cancelled) setError(e instanceof Error ? e.message : t('errors.historyLoad')) })
    return () => { cancelled = true }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- skusKey is the stable dependency; skus itself is a new array every render
  }, [token, skusKey])

  const warehouseName = (id: string) => warehouses.find(w => w.id === id)?.name ?? id
  const batchLabel = (batchId: string | null | undefined) => {
    if (!batchId) return t('history.generalLedger')
    const batch = batches.find(b => b.id === batchId)
    return batch?.lot_number || batchId
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b border-line bg-sunken shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-2xs font-medium text-ink-4 hover:text-ink-2 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> {t('history.backToInventory')}
          </button>
          <h1 className="text-xl font-semibold text-ink leading-tight">{t('history.title')}</h1>
          <p className="text-xs text-ink-4 mt-0.5">{t('history.productSubtitle', { name: productName })}</p>
        </div>
      </div>

      <div className="flex-1 overflow-auto">
        {error && <p className="px-6 py-4 text-xs text-danger">{error}</p>}
        {!error && rows === null && <p className="px-6 py-4 text-xs text-ink-4">{t('common:status.loading')}</p>}
        {!error && rows !== null && rows.length === 0 && (
          <p className="px-6 py-4 text-xs text-ink-4">{t('history.emptyProduct')}</p>
        )}
        {!error && rows !== null && rows.length > 0 && (
          <table className="w-full text-xs">
            <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
              <tr>
                {[t('columns.date'), t('columns.sku'), t('columns.type'), t('columns.warehouse'), t('columns.batch'), t('columns.quantity'), t('columns.balance'), t('columns.reason')].map(h => (
                  <th key={h} className="px-4 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-line/60">
              {rows.map(m => (
                <tr key={m.id} className="hover:bg-hover/60 transition-colors">
                  <td className="px-4 py-2.5 text-ink-3 whitespace-nowrap">{formatDateTime(m.occurred_at)}</td>
                  <td className="px-4 py-2.5 font-mono text-ink-2">{m.skuCode}</td>
                  <td className="px-4 py-2.5 text-ink">{movementTypeLabel(m.type)}</td>
                  <td className="px-4 py-2.5 text-ink-3">{warehouseName(m.warehouse_id)}</td>
                  <td className="px-4 py-2.5 text-ink-3">{batchLabel(m.batch_id)}</td>
                  <td className={clsx('px-4 py-2.5 font-medium tabular-nums', m.quantity_delta >= 0 ? 'text-success' : 'text-danger')}>
                    {m.quantity_delta >= 0 ? '+' : ''}{m.quantity_delta}
                  </td>
                  <td className="px-4 py-2.5 font-medium tabular-nums text-ink">{m.balance}</td>
                  <td className="px-4 py-2.5 text-ink-4">{m.reason || '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}

export function SKUMovementHistoryPage({ token, sku, warehouses, onClose }: {
  token: string; sku: SKU; warehouses: { id: string; name: string }[]; onClose: () => void
}) {
  const { formatDateTime } = useDateFormat()
  const { t } = useTranslation(['inventory', 'common'])
  const [movements, setMovements] = useState<StockMovement[] | null>(null)
  const [batches, setBatches] = useState<Batch[]>([])
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    Promise.all([inventoryApi.listStockMovements(token, sku.id), inventoryApi.listBatches(token, sku.id)])
      .then(([m, b]) => { if (!cancelled) { setMovements(m); setBatches(b) } })
      .catch(e => { if (!cancelled) setError(e instanceof Error ? e.message : t('errors.historyLoad')) })
    return () => { cancelled = true }
  }, [token, sku.id])

  const warehouseName = (id: string) => warehouses.find(w => w.id === id)?.name ?? id
  const batchLabel = (batchId: string | null | undefined) => {
    if (!batchId) return t('history.generalLedger')
    const batch = batches.find(b => b.id === batchId)
    return batch?.lot_number || batchId
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b border-line bg-sunken shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-2xs font-medium text-ink-4 hover:text-ink-2 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> {t('history.backToInventory')}
          </button>
          <h1 className="text-xl font-semibold text-ink leading-tight">{t('history.title')}</h1>
          <p className="text-xs font-mono text-ink-4 mt-0.5">{sku.sku_code}</p>
        </div>
      </div>

      <div className="flex-1 overflow-auto">
        {error && <p className="px-6 py-4 text-xs text-danger">{error}</p>}
        {!error && movements === null && <p className="px-6 py-4 text-xs text-ink-4">{t('common:status.loading')}</p>}
        {!error && movements !== null && movements.length === 0 && (
          <p className="px-6 py-4 text-xs text-ink-4">{t('history.emptySku')}</p>
        )}
        {!error && movements !== null && movements.length > 0 && (
          <table className="w-full text-xs">
            <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
              <tr>
                {[t('columns.date'), t('columns.type'), t('columns.warehouse'), t('columns.batch'), t('columns.quantity'), t('columns.reason')].map(h => (
                  <th key={h} className="px-4 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-line/60">
              {[...movements].reverse().map(m => (
                <tr key={m.id} className="hover:bg-hover/60 transition-colors">
                  <td className="px-4 py-2.5 text-ink-3 whitespace-nowrap">{formatDateTime(m.occurred_at)}</td>
                  <td className="px-4 py-2.5 text-ink">{movementTypeLabel(m.type)}</td>
                  <td className="px-4 py-2.5 text-ink-3">{warehouseName(m.warehouse_id)}</td>
                  <td className="px-4 py-2.5 text-ink-3">{batchLabel(m.batch_id)}</td>
                  <td className={clsx('px-4 py-2.5 font-medium tabular-nums', m.quantity_delta >= 0 ? 'text-success' : 'text-danger')}>
                    {m.quantity_delta >= 0 ? '+' : ''}{m.quantity_delta}
                  </td>
                  <td className="px-4 py-2.5 text-ink-4">{m.reason || '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}

export function BatchMovementHistoryPage({ token, sku, batch, warehouses, onClose }: {
  token: string; sku: SKU; batch: Batch; warehouses: { id: string; name: string }[]; onClose: () => void
}) {
  const { formatDateTime } = useDateFormat()
  const { t } = useTranslation(['inventory', 'common'])
  const [movements, setMovements] = useState<StockMovement[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    inventoryApi.listBatchMovements(token, batch.id)
      .then(m => { if (!cancelled) setMovements(m) })
      .catch(e => { if (!cancelled) setError(e instanceof Error ? e.message : t('errors.historyLoad')) })
    return () => { cancelled = true }
  }, [token, batch.id])

  const warehouseName = (id: string) => warehouses.find(w => w.id === id)?.name ?? id

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b border-line bg-sunken shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-2xs font-medium text-ink-4 hover:text-ink-2 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> {t('history.backToInventory')}
          </button>
          <h1 className="text-xl font-semibold text-ink leading-tight">{t('history.batchTitle')}</h1>
          <p className="text-xs font-mono text-ink-4 mt-0.5">{t('history.lot', { sku: sku.sku_code, lot: batch.lot_number || '—' })}</p>
        </div>
      </div>

      <div className="flex-1 overflow-auto">
        {error && <p className="px-6 py-4 text-xs text-danger">{error}</p>}
        {!error && movements === null && <p className="px-6 py-4 text-xs text-ink-4">{t('common:status.loading')}</p>}
        {!error && movements !== null && movements.length === 0 && (
          <p className="px-6 py-4 text-xs text-ink-4">{t('history.emptyBatch')}</p>
        )}
        {!error && movements !== null && movements.length > 0 && (
          <table className="w-full text-xs">
            <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
              <tr>
                {[t('columns.date'), t('columns.type'), t('columns.warehouse'), t('columns.quantity'), t('columns.reason')].map(h => (
                  <th key={h} className="px-4 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-line/60">
              {[...movements].reverse().map(m => (
                <tr key={m.id} className="hover:bg-hover/60 transition-colors">
                  <td className="px-4 py-2.5 text-ink-3 whitespace-nowrap">{formatDateTime(m.occurred_at)}</td>
                  <td className="px-4 py-2.5 text-ink">{movementTypeLabel(m.type)}</td>
                  <td className="px-4 py-2.5 text-ink-3">{warehouseName(m.warehouse_id)}</td>
                  <td className={clsx('px-4 py-2.5 font-medium tabular-nums', m.quantity_delta >= 0 ? 'text-success' : 'text-danger')}>
                    {m.quantity_delta >= 0 ? '+' : ''}{m.quantity_delta}
                  </td>
                  <td className="px-4 py-2.5 text-ink-4">{m.reason || '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}
