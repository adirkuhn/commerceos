import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { Image as ImageIcon, Tag } from 'lucide-react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import { useDateFormat } from '../../format/DateFormatProvider'
import { Money } from '../../format/amountVisibility'
import { DaysLeftStamp } from './DaysLeftStamp'
import { ClearancePromotionDialog, type ClearanceCreated } from './ClearancePromotionDialog'
import { nextWindowOf } from './clearance'
import { useCloseToExpiry, type CloseToExpiryState } from './useCloseToExpiry'
import type { CloseToExpiryBatch, CloseToExpiryList } from './types'

// The "Expiring soon" tab: batches with stock left by close-to-expiry
// window (the same windows as the expiry email), to mark as seen, put on
// sale or discard. Batches marked as seen sit in their own list until they
// reach the next window.
// The tab's own state lives here, not on the Inventory page: its ?page=
// would otherwise be read from the product list's URL. reloadSignal
// changes when something outside (a discard) changed the list.
export function CloseToExpiryTab({ token, reloadSignal, onDiscard }: {
  token: string
  reloadSignal: number
  onDiscard: (batch: CloseToExpiryBatch) => void
}) {
  const state = useCloseToExpiry(token)
  const { refresh } = state
  const lastSignal = useRef(reloadSignal)
  useEffect(() => {
    if (lastSignal.current === reloadSignal) return
    lastSignal.current = reloadSignal
    refresh()
  }, [reloadSignal, refresh])
  return <CloseToExpiryPanel token={token} state={state} onDiscard={onDiscard} />
}

export function CloseToExpiryPanel({ token, state, onDiscard }: {
  token: string
  state: CloseToExpiryState
  onDiscard: (batch: CloseToExpiryBatch) => void
}) {
  const { t } = useTranslation(['inventory', 'common'])
  const { formatDate } = useDateFormat()
  const [saleFor, setSaleFor] = useState<CloseToExpiryBatch | null>(null)
  const [started, setStarted] = useState<ClearanceCreated | null>(null)
  const { data, loading, error, actionError, seen } = state

  function handleCreated(created: ClearanceCreated) {
    setSaleFor(null)
    setStarted(created)
    state.refresh()
  }

  return (
    <div className="px-6 py-4 space-y-4">
      {data && (
        <div className="flex flex-wrap items-end justify-between gap-3">
          <div className="space-y-2">
            <div role="group" aria-label={t('closeToExpiry.title')} className="inline-flex flex-wrap gap-1 p-1 rounded-xl bg-fill">
              {data.windows.map(w => {
                const current = w.window_days === data.window.window_days
                return (
                  <button key={w.window_days} type="button" aria-pressed={current} onClick={() => state.setWindow(w.window_days)}
                    className={clsx('flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors',
                      current ? 'bg-raised text-ink shadow-sm' : 'text-ink-3 hover:text-ink')}>
                    {t('closeToExpiry.windowDays', { count: w.window_days })}
                    <span className={clsx('min-w-5 px-1.5 rounded-full text-2xs tabular-nums',
                      w.open_count > 0 ? 'bg-accent-solid text-white' : 'bg-hover text-ink-4')}>
                      {w.open_count}
                    </span>
                  </button>
                )
              })}
            </div>
            <p className="text-xs text-ink-4">
              {t('closeToExpiry.windowRange', { from: formatDate(data.window.from), to: formatDate(data.window.to) })}
            </p>
          </div>
          <div className="inline-flex gap-1 text-xs">
            <ListToggle active={!seen} onClick={() => state.setSeen(false)} label={t('closeToExpiry.listOpen')} count={data.window.open_count} />
            <ListToggle active={seen} onClick={() => state.setSeen(true)} label={t('closeToExpiry.listSeen')} count={data.window.acknowledged_count} />
          </div>
        </div>
      )}

      {started && (
        <div role="status" className="rounded-lg border border-success/40 bg-success/10 px-3 py-2 text-xs text-ink-2">
          <Link to={`/promotions/${encodeURIComponent(started.promotionId)}`} className="font-medium underline-offset-2 hover:underline">
            {t('closeToExpiry.saleStarted', { name: started.name })}
          </Link>
          {started.warnings.map(w => <p key={w} className="mt-1 text-warning">{w}</p>)}
        </div>
      )}
      {error && <p className="text-xs text-danger">{error}</p>}
      {actionError && <p role="alert" className="text-xs text-danger">{actionError}</p>}
      {!data && loading && <p className="text-xs text-ink-4">{t('common:status.loading')}</p>}

      {data && data.items.length === 0 && !loading && (
        <p className="py-8 text-center text-xs text-ink-4">{seen ? t('closeToExpiry.emptySeen') : t('closeToExpiry.emptyOpen')}</p>
      )}

      {data && data.items.length > 0 && (
        <ul className={clsx('divide-y divide-line/60 rounded-xl border border-line bg-raised', loading && 'opacity-60')}>
          {data.items.map(b => (
            <BatchRow key={b.batch_id} batch={b} data={data} seen={seen} acting={state.actingId === b.batch_id}
              onMark={() => state.acknowledge(b)} onUndo={() => state.undo(b)}
              onSale={() => { setStarted(null); setSaleFor(b) }} onDiscard={() => onDiscard(b)} />
          ))}
        </ul>
      )}

      {data && <Pager data={data} onPage={state.setPage} />}

      {saleFor && <ClearancePromotionDialog token={token} batch={saleFor} onClose={() => setSaleFor(null)} onCreated={handleCreated} />}
    </div>
  )
}

function ListToggle({ active, onClick, label, count }: { active: boolean; onClick: () => void; label: string; count: number }) {
  return (
    <button type="button" aria-pressed={active} onClick={onClick}
      className={clsx('px-3 py-1.5 rounded-lg font-medium transition-colors',
        active ? 'bg-accent-solid text-white' : 'bg-raised text-ink-3 hover:bg-hover')}>
      {label} <span className="tabular-nums opacity-80">{count}</span>
    </button>
  )
}

function BatchRow({ batch: b, data, seen, acting, onMark, onUndo, onSale, onDiscard }: {
  batch: CloseToExpiryBatch
  data: CloseToExpiryList
  seen: boolean
  acting: boolean
  onMark: () => void
  onUndo: () => void
  onSale: () => void
  onDiscard: () => void
}) {
  const { t } = useTranslation('inventory')
  const { formatDate } = useDateFormat()
  const next = nextWindowOf(data.windows, data.window.window_days)
  const markHint = next ? t('closeToExpiry.markSeenHint', { days: next }) : t('closeToExpiry.markSeenHintLast')

  return (
    <li className="flex flex-wrap items-center gap-x-4 gap-y-3 px-4 py-3">
      <DaysLeftStamp daysLeft={b.days_left} />
      <div className="w-10 h-10 shrink-0 rounded-lg bg-sunken border border-line overflow-hidden flex items-center justify-center">
        {b.image_url
          ? <img src={b.image_url} alt="" className="w-full h-full object-cover" />
          : <ImageIcon size={14} className="text-ink-4" aria-hidden />}
      </div>
      <div className="min-w-0 flex-1 basis-48">
        <p className="text-sm font-medium text-ink truncate">{b.product_name}</p>
        <p className="text-xs text-ink-4 truncate">
          <span className="font-mono">{b.sku_code}</span>
          {b.lot_number && <>, {t('closeToExpiry.lot', { lot: b.lot_number })}</>}
          , {b.warehouse_name}, {formatDate(b.expiry_date)}
        </p>
        {seen && b.acknowledged_at && (
          <p className="text-2xs text-ink-4 mt-0.5">{t('closeToExpiry.seenOn', { date: formatDate(b.acknowledged_at) })}</p>
        )}
      </div>
      <div className="w-24 text-right">
        <p className="text-sm tabular-nums text-ink-2">{t('closeToExpiry.inStock', { count: b.quantity })}</p>
      </div>
      <div className="w-28 text-right text-sm tabular-nums">
        {b.sale_price_minor_units != null ? (
          <>
            <p className="text-ink"><Money minor={b.sale_price_minor_units} currency={b.currency_code} /></p>
            <p className="text-2xs text-ink-4 line-through"><Money minor={b.price_minor_units} currency={b.currency_code} /></p>
          </>
        ) : (
          <p className="text-ink-2"><Money minor={b.price_minor_units} currency={b.currency_code} /></p>
        )}
      </div>
      <div className="flex items-center gap-2 ml-auto">
        {seen ? (
          <button type="button" onClick={onUndo} disabled={acting}
            className="px-2.5 py-1.5 rounded-lg text-xs font-medium bg-fill text-ink-2 hover:bg-hover disabled:opacity-50">
            {t('closeToExpiry.showAgain')}
          </button>
        ) : (
          <button type="button" onClick={onMark} disabled={acting} title={markHint}
            className="px-2.5 py-1.5 rounded-lg text-xs font-medium bg-fill text-ink-2 hover:bg-hover disabled:opacity-50">
            {t('closeToExpiry.markSeen')}
          </button>
        )}
        {b.clearance_promotion ? (
          <Link to={`/promotions/${encodeURIComponent(b.clearance_promotion.id)}`}
            className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-medium text-success bg-success/10 hover:bg-success/15 max-w-48 truncate">
            <Tag size={12} aria-hidden /> {t('closeToExpiry.onSale', { name: b.clearance_promotion.name })}
          </Link>
        ) : (
          <button type="button" onClick={onSale}
            className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-semibold border border-accent text-accent hover:bg-accent/10">
            <Tag size={12} aria-hidden /> {t('closeToExpiry.putOnSale')}
          </button>
        )}
        <button type="button" onClick={onDiscard}
          className="px-2 py-1.5 rounded-lg text-xs font-medium text-danger hover:bg-danger/10">
          {t('expiry.discard')}
        </button>
      </div>
    </li>
  )
}

function Pager({ data, onPage }: { data: CloseToExpiryList; onPage: (page: number) => void }) {
  const { t } = useTranslation('inventory')
  const pages = Math.ceil(data.total / data.per_page)
  if (pages <= 1) return null
  return (
    <div className="flex items-center justify-end gap-3 text-xs text-ink-3">
      <button type="button" disabled={data.page <= 1} onClick={() => onPage(data.page - 1)}
        className="px-2.5 py-1.5 rounded-lg bg-fill hover:bg-hover disabled:opacity-40">{t('closeToExpiry.previous')}</button>
      <span className="tabular-nums">{t('closeToExpiry.pageOf', { page: data.page, pages })}</span>
      <button type="button" disabled={data.page >= pages} onClick={() => onPage(data.page + 1)}
        className="px-2.5 py-1.5 rounded-lg bg-fill hover:bg-hover disabled:opacity-40">{t('closeToExpiry.next')}</button>
    </div>
  )
}
