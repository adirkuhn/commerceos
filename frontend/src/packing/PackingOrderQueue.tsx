import { useState } from 'react'
import { ClipboardList, Lock, Search } from 'lucide-react'
import clsx from 'clsx'
import type { PackingQueueEntry, SalesOrder } from '../oms/types'
import { customerName, formatMoney, orderNumber } from './orderDisplay'
import { lockLabel } from './lockDisplay'

interface PackingOrderQueueProps {
  entries: PackingQueueEntry[]
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
        <p className="text-xs font-semibold text-ink">Orders to pack</p>
        <p className="text-2xs text-ink-4 mt-0.5">{entries.length} to pack</p>
      </div>
      <div className="px-3 py-2 border-b border-line space-y-2">
        <div className="relative">
          <Search size={12} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-ink-4" />
          <input
            value={search}
            onChange={e => setSearch(e.target.value)}
            placeholder="Search orders..."
            className="w-full pl-7 pr-2 py-1.5 text-2xs rounded-lg bg-raised border border-line text-ink placeholder-ink-4 outline-none focus:ring-1 focus:ring-focus"
          />
        </div>
        <div className="flex items-center gap-2">
          <input
            type="checkbox"
            aria-label="Select all orders"
            checked={allVisibleTicked}
            onChange={toggleAllVisible}
            disabled={visible.length === 0}
            className="accent-accent"
          />
          <button
            onClick={() => onOpenPickList(tickedIds)}
            disabled={tickedIds.length === 0}
            className="flex-1 flex items-center justify-center gap-1.5 px-2 py-1.5 text-2xs font-medium rounded-lg bg-raised border border-line text-ink-2 hover:text-ink disabled:opacity-40"
          >
            <ClipboardList size={12} />
            {tickedIds.length > 0 ? `Pick list (${tickedIds.length})` : 'Pick list'}
          </button>
        </div>
      </div>
      <ul className="flex-1 overflow-y-auto py-2">
        {loading && <li className="px-4 py-3 text-2xs text-ink-4">Loading…</li>}
        {error && <li className="px-4 py-3 text-2xs text-danger">{error}</li>}
        {!loading && !error && visible.length === 0 && (
          <li className="px-4 py-3 text-2xs text-ink-4">No orders to pack</li>
        )}
        {visible.map(({ sales_order: o, lock }) => {
          const active = activeOrderId === o.id
          const lockedByOther = lock !== null && lock.user_id !== currentUserId
          const details = (
            <>
              <div className="flex items-center justify-between gap-2 mb-1">
                <span className="text-xs font-medium text-ink">#{orderNumber(o)}</span>
                {active && activeInProgress
                  ? <span className="text-2xs font-medium text-accent">In progress</span>
                  : !o.fulfilled && <span className="text-2xs font-medium text-warning">Not picked</span>}
              </div>
              <p className="text-2xs text-ink-3 truncate">{customerName(o)}</p>
              <p className="text-2xs text-ink-4 mt-0.5">{formatMoney(o.total_minor_units, o.currency_code)}</p>
            </>
          )
          return (
            <li
              key={o.id}
              className={clsx(
                'flex items-start gap-2 pl-3 border-b border-line transition-colors',
                active ? 'bg-accent/10 border-l-2 border-l-accent' : !lockedByOther && 'hover:bg-raised',
              )}
            >
              <input
                type="checkbox"
                aria-label={`Add #${orderNumber(o)} to pick list`}
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
                      aria-label={`Take over #${orderNumber(o)}`}
                      onClick={() => onSelect(o)}
                      className="mt-1.5 px-2 py-0.5 text-2xs font-medium rounded-md border border-line text-ink-2 hover:text-ink hover:bg-raised"
                    >
                      Take over
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
