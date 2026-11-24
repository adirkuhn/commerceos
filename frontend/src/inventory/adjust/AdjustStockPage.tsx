import { useCallback, useEffect, useState } from 'react'
import { ArrowLeft, History } from 'lucide-react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import type { SKU } from '../../pms/types'
import * as inventoryApi from '../api'
import type { StockMovement } from '../types'
import { useSKUStock } from '../hooks/useSKUStock'
import { stockUnitFor } from '../stockUnit'
import { formatSKUQuantity, movementTypeLabel } from '../inventoryFormat'
import { useDateFormat } from '../../format/DateFormatProvider'
import { apiErrorText } from '../../api/apiErrorText'
import { AdjustStockForm } from './AdjustStockForm'
import { stockSlots, type StockSlot } from './stockSlots'
import type { BatchRowRef } from '../useSkuActionRoute'

const RECENT = 8

// The stock page for one SKU: every place it is kept (warehouse + lot) as
// coupons to pick from, the adjust form, and the latest movements so the
// change just made shows up underneath.
export function AdjustStockPage({ token, productName, sku, warehouses, initialRow, onClose, onViewHistory, onWriteOff }: {
  token: string
  productName: string
  sku: SKU
  warehouses: { id: string; name: string }[]
  // The batch row the page was opened for (?batch= / ?ledger=).
  initialRow?: BatchRowRef | null
  onClose: () => void
  onViewHistory: () => void
  onWriteOff: () => void
}) {
  const { t } = useTranslation(['inventory', 'common'])
  const { level, batches, loading, error, adjust } = useSKUStock(token, sku.id)
  const recent = useRecentMovements(token, sku.id)
  const [saved, setSaved] = useState<string | null>(null)
  const unit = stockUnitFor(sku)
  const formatQuantity = (n: number) => formatSKUQuantity(sku, n)
  const slots = stockSlots(warehouses, batches)
  const loaded = !loading || batches.length > 0
  const initialKey = !initialRow ? undefined : slots.find(s => 'batch' in initialRow
    ? s.batchId === initialRow.batch
    : s.isLedger && s.warehouseId === initialRow.ledger)?.key

  return (
    <div className="flex h-full flex-col">
      <header className="flex flex-wrap items-end justify-between gap-4 border-b border-line bg-sunken px-6 py-4 shrink-0">
        <div className="min-w-0">
          <button type="button" onClick={onClose} className="mb-1 flex items-center gap-1 text-2xs font-medium text-ink-4 hover:text-ink-2">
            <ArrowLeft size={11} /> {t('adjust.back')}
          </button>
          <h1 className="text-xl font-semibold leading-tight text-ink">{t('adjust.title')}</h1>
          <p className="mt-0.5 truncate text-xs text-ink-3">
            {productName} · <span className="font-mono">{sku.sku_code}</span>
          </p>
        </div>
        {level && (
          <div className="text-right">
            <div className="caps-label-widest text-2xs font-semibold text-ink-4">{t('adjust.total')}</div>
            <div data-testid="stock-total" className="text-3xl font-bold leading-none tabular-nums text-figure">{formatQuantity(level.total)}</div>
          </div>
        )}
      </header>

      <div className="flex-1 overflow-auto">
        <div className="mx-auto flex max-w-5xl flex-col gap-8 px-6 py-6">
          {error && <p role="alert" className="text-xs text-danger">{t('adjust.loadFailed')} {error}</p>}
          {!loaded && <p className="text-xs text-ink-4">{t('common:status.loading')}</p>}
          {loaded && !error && (
            <section className="rounded-lg border border-line bg-raised p-5 shadow-panel">
              <AdjustStockForm
                key={`${sku.id}:${initialKey ?? ''}`}
                variant="page"
                slots={slots}
                unit={unit}
                formatQuantity={formatQuantity}
                initialKey={initialKey}
                onSave={async input => {
                  setSaved(null)
                  await adjust(input)
                  recent.reload()
                }}
                onRecorded={change => setSaved(t('adjust.saved', { before: formatQuantity(change.before), after: formatQuantity(change.after) }))}
                onWriteOff={onWriteOff}
              />
              {saved && <p role="status" className="mt-3 text-xs font-semibold text-success">{saved}</p>}
            </section>
          )}

          <section aria-labelledby="adjust-recent" className="flex flex-col gap-2">
            <div className="flex items-center justify-between">
              <h2 id="adjust-recent" className="caps-label-widest text-2xs font-semibold text-ink-4">{t('adjust.recent')}</h2>
              <button type="button" onClick={onViewHistory} className="flex items-center gap-1 text-2xs font-medium text-ink-4 hover:text-accent">
                <History size={11} /> {t('adjust.fullHistory')}
              </button>
            </div>
            <RecentMovements state={recent} slots={slots} formatQuantity={formatQuantity} />
          </section>
        </div>
      </div>
    </div>
  )
}

interface RecentState {
  rows: StockMovement[] | null
  error: string | null
  reload: () => void
}

function useRecentMovements(token: string, skuId: string): RecentState {
  const [rows, setRows] = useState<StockMovement[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [nonce, setNonce] = useState(0)
  const reload = useCallback(() => setNonce(n => n + 1), [])

  useEffect(() => {
    let cancelled = false
    inventoryApi.listStockMovements(token, skuId)
      .then(all => {
        if (cancelled) return
        const newest = [...all].sort((a, b) => b.occurred_at.localeCompare(a.occurred_at)).slice(0, RECENT)
        setRows(newest)
        setError(null)
      })
      .catch(e => { if (!cancelled) setError(apiErrorText(e) ?? 'error') })
    return () => { cancelled = true }
  }, [token, skuId, nonce])

  return { rows, error, reload }
}

function RecentMovements({ state, slots, formatQuantity }: {
  state: RecentState; slots: StockSlot[]; formatQuantity: (n: number) => string
}) {
  const { t } = useTranslation(['inventory', 'common'])
  const { formatDateTime } = useDateFormat()
  if (state.error) return <p className="text-xs text-danger">{t('errors.historyLoad')}</p>
  if (state.rows === null) return <p className="text-xs text-ink-4">{t('common:status.loading')}</p>
  if (state.rows.length === 0) return <p className="text-xs text-ink-4">{t('adjust.recentEmpty')}</p>

  const place = (m: StockMovement) => {
    const slot = slots.find(s => s.warehouseId === m.warehouse_id && s.batchId === (m.batch_id ?? ''))
    const lot = m.batch_id ? t('adjust.lot', { lot: slot?.lot || '—' }) : t('adjust.noLot')
    return `${slot?.warehouseName ?? m.warehouse_id} · ${lot}`
  }

  return (
    <ol className="divide-y divide-line rounded-lg border border-line bg-raised">
      {state.rows.map(m => (
        <li key={m.id} className="grid grid-cols-[auto_1fr_auto] items-baseline gap-x-4 gap-y-0.5 px-4 py-2.5 text-xs sm:grid-cols-[9rem_8rem_1fr_auto]">
          <span className="tabular-nums text-ink-4">{formatDateTime(m.occurred_at)}</span>
          <span className="font-semibold text-ink-2">{movementTypeLabel(m.type)}</span>
          <span className="col-span-3 row-start-2 min-w-0 truncate text-ink-3 sm:col-span-1 sm:row-start-auto">
            {place(m)}{m.reason ? ` — ${m.reason}` : ''}
          </span>
          <span className={clsx('col-start-3 row-start-1 text-right font-bold tabular-nums sm:col-start-auto sm:row-start-auto', m.quantity_delta < 0 ? 'text-danger' : 'text-figure')}>
            {m.quantity_delta > 0 ? '+' : '−'}{formatQuantity(Math.abs(m.quantity_delta))}
          </span>
        </li>
      ))}
    </ol>
  )
}
