import { useState } from 'react'
import { ClipboardList, Lock, Search, Truck } from 'lucide-react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import type { QueueEntry, SalesOrder } from '../oms/types'
import { orderNumber } from './orderDisplay'
import { lockLabel } from './lockDisplay'
import { customerName, salesOrderSourceLabel, shippingMethodLabel } from '../oms/salesOrderDisplay'
import { MiniLegs } from '../oms/FulfilmentLegs'
import { Money } from '../format/amountVisibility'

interface PackingOrderQueueProps {
  entries: QueueEntry[]
  loading: boolean
  error: string
  activeOrderId: string | null
  activeInProgress: boolean
  currentUserId: string
  isAdmin: boolean
  nameFor: (userId: string) => string
  onSelect: (order: SalesOrder) => void
  onOpenPickList: (orderIds: string[]) => void
}

// The packing page's left-hand order queue: every order still to be packed.
// Picking is optional, so unpicked orders are listed too ("Not picked").
// An order someone else is working is read-only, labelled with who and for
// how long; admins get a "Take over" button (the backend decides whether a
// takeover is allowed). Ticking orders opens the pick list for all of them
// -- the ticked set is throwaway UI state, the pick list itself gets its
// own URL (see Packing's ?pick=).
export function PackingOrderQueue({
  entries, loading, error, activeOrderId, activeInProgress, currentUserId, isAdmin, nameFor, onSelect, onOpenPickList,
}: PackingOrderQueueProps) {
  const { t } = useTranslation(['packing', 'common'])
  const [search, setSearch] = useState('')
  const [ticked, setTicked] = useState<ReadonlySet<string>>(new Set())

  const q = search.trim().toLowerCase()
  const visible = q
    ? entries.filter(({ sales_order: o }) => orderNumber(o).toLowerCase().includes(q) || customerName(o).toLowerCase().includes(q))
    : entries
  const tickedIds = entries.filter(e => ticked.has(e.sales_order.id)).map(e => e.sales_order.id)
  const allVisibleTicked = visible.length > 0 && visible.every(e => ticked.has(e.sales_order.id))

  function toggle(id: string) {
    setTicked(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  function toggleAllVisible() {
    setTicked(prev => {
      const next = new Set(prev)
      for (const e of visible) {
        if (allVisibleTicked) next.delete(e.sales_order.id)
        else next.add(e.sales_order.id)
      }
      return next
    })
  }

  return (
    <div className="w-64 border-r border-line bg-sunken flex flex-col overflow-hidden shrink-0">
      <div className="px-4 py-3 border-b border-line">
        <h2 className="caps-label-widest text-2xs font-bold text-ink">{t('queue.title')}</h2>
        <p className="text-xs font-semibold text-figure tabular-nums mt-1">{t('queue.count', { count: entries.length })}</p>
      </div>
      <div className="px-3 py-2 border-b border-line space-y-2">
        <div className="relative">
          <Search size={12} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-ink-4" />
          <input
            value={search}
            onChange={e => setSearch(e.target.value)}
            placeholder={t('queue.search')}
            className="w-full pl-7 pr-2 h-8 text-xs rounded-md bg-surface border border-line text-ink placeholder-ink-4 outline-none focus:ring-1 focus:ring-focus"
          />
        </div>
        <div className="flex items-center gap-2">
          <input
            type="checkbox"
            aria-label={t('queue.selectAll')}
            checked={allVisibleTicked}
            onChange={toggleAllVisible}
            disabled={visible.length === 0}
            className="accent-accent"
          />
          <button
            onClick={() => onOpenPickList(tickedIds)}
            disabled={tickedIds.length === 0}
            className="flex-1 flex items-center justify-center gap-1.5 px-2 h-8 text-xs font-semibold rounded-md bg-raised border border-line text-ink-2 hover:text-ink hover:bg-hover disabled:opacity-40"
          >
            <ClipboardList size={12} />
            {tickedIds.length > 0 ? t('queue.pickListCount', { n: tickedIds.length }) : t('queue.pickList')}
          </button>
        </div>
      </div>
      <ul className="flex-1 overflow-y-auto py-2">
        {loading && <li className="px-4 py-3 text-2xs text-ink-4">{t('common:status.loading')}</li>}
        {error && <li className="px-4 py-3 text-2xs text-danger">{error}</li>}
        {!loading && !error && visible.length === 0 && (
          <li className="px-4 py-3 text-2xs text-ink-4">{t('queue.empty')}</li>
        )}
        {visible.map(({ sales_order: o, lock }) => {
          const active = activeOrderId === o.id
          const lockedByOther = lock !== null && lock.user_id !== currentUserId
          const shippingMethod = shippingMethodLabel(o)
          const details = (
            <>
              <div className="flex items-center justify-between gap-2 mb-1">
                <span className="text-xs font-semibold text-ink tabular-nums truncate">#{orderNumber(o)}</span>
                {active && activeInProgress
                  ? <span className="shrink-0 whitespace-nowrap text-2xs font-semibold text-accent">{t('queue.inProgress')}</span>
                  : !o.fulfilled && <span className="shrink-0 whitespace-nowrap text-2xs font-semibold text-warning">{t('queue.notPicked')}</span>}
              </div>
              <p className="flex items-center justify-between gap-2 text-2xs text-ink-3">
                <span className="truncate">{customerName(o)} · {salesOrderSourceLabel(o.source)}</span>
                <MiniLegs order={o} />
              </p>
              {shippingMethod && (
                <p title={t('order.shippingMethod')} className="flex items-center gap-1 text-2xs text-ink-3 mt-0.5">
                  <Truck size={10} className="shrink-0" aria-hidden />
                  <span className="truncate">{shippingMethod}</span>
                </p>
              )}
              <p className="text-2xs text-ink-4 mt-0.5"><Money minor={o.total_minor_units} currency={o.currency_code} static /></p>
            </>
          )
          return (
            <li
              key={o.id}
              className={clsx(
                'flex items-start gap-2 pl-3 border-b border-line transition-colors',
                active ? 'bg-accent/10 ring-1 ring-inset ring-accent' : !lockedByOther && 'hover:bg-raised',
              )}
            >
              <input
                type="checkbox"
                aria-label={t('queue.addToPickList', { number: orderNumber(o) })}
                checked={ticked.has(o.id)}
                onChange={() => toggle(o.id)}
                className="mt-3.5 accent-accent"
              />
              {lockedByOther ? (
                <div className="flex-1 min-w-0 pr-4 py-3 opacity-70">
                  {details}
                  <p className="flex items-center gap-1 text-2xs text-ink-3 mt-1">
                    <Lock size={10} className="shrink-0" />
                    <span className="truncate">{lockLabel(lock, nameFor(lock.user_id))}</span>
                  </p>
                  {isAdmin && (
                    <button
                      aria-label={t('queue.takeOverOrder', { number: orderNumber(o) })}
                      onClick={() => onSelect(o)}
                      className="mt-1.5 px-2 py-0.5 text-2xs font-medium rounded-md border border-line text-ink-2 hover:text-ink hover:bg-raised"
                    >
                      {t('queue.takeOver')}
                    </button>
                  )}
                </div>
              ) : (
                <button aria-pressed={active} onClick={() => onSelect(o)} className="flex-1 min-w-0 text-left pr-4 py-3">
                  {details}
                </button>
              )}
            </li>
          )
        })}
      </ul>
    </div>
  )
}
