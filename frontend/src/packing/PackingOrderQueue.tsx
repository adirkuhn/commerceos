import { useState } from 'react'
import { ClipboardList, Search } from 'lucide-react'
import clsx from 'clsx'
import type { SalesOrder } from '../oms/types'
import { customerName, formatMoney, orderNumber } from './orderDisplay'

interface PackingOrderQueueProps {
  orders: SalesOrder[]
  loading: boolean
  error: string
  activeOrderId: string | null
  activeInProgress: boolean
  onSelect: (order: SalesOrder) => void
  onOpenPickList: (orderIds: string[]) => void
}

// The packing page's left-hand order queue. Picking is optional, so it lists
// picked and unpicked orders alike ("Not picked" marks the latter); ticking
// orders and opening the pick list shows what to fetch off the shelves for
// all of them at once. The ticked set is throwaway UI state -- the pick
// list itself gets its own URL (see Packing's ?pick=).
export function PackingOrderQueue({ orders, loading, error, activeOrderId, activeInProgress, onSelect, onOpenPickList }: PackingOrderQueueProps) {
  const [search, setSearch] = useState('')
  const [ticked, setTicked] = useState<ReadonlySet<string>>(new Set())

  const q = search.trim().toLowerCase()
  const visible = q
    ? orders.filter(o => orderNumber(o).toLowerCase().includes(q) || customerName(o).toLowerCase().includes(q))
    : orders
  const tickedIds = orders.filter(o => ticked.has(o.id)).map(o => o.id)
  const allVisibleTicked = visible.length > 0 && visible.every(o => ticked.has(o.id))

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
      for (const o of visible) {
        if (allVisibleTicked) next.delete(o.id)
        else next.add(o.id)
      }
      return next
    })
  }

  return (
    <div className="w-64 border-r border-line bg-sunken flex flex-col overflow-hidden shrink-0">
      <div className="px-4 py-3 border-b border-line">
        <p className="text-xs font-semibold text-ink">Orders to pack</p>
        <p className="text-2xs text-ink-4 mt-0.5">{orders.length} ready to pack</p>
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
      <div className="flex-1 overflow-y-auto py-2">
        {loading && <p className="px-4 py-3 text-2xs text-ink-4">Loading…</p>}
        {error && <p className="px-4 py-3 text-2xs text-danger">{error}</p>}
        {!loading && !error && visible.length === 0 && (
          <p className="px-4 py-3 text-2xs text-ink-4">No orders ready to pack</p>
        )}
        {visible.map(o => {
          const active = activeOrderId === o.id
          return (
            <div
              key={o.id}
              className={clsx(
                'flex items-start gap-2 pl-3 border-b border-line transition-colors',
                active ? 'bg-accent/10 border-l-2 border-l-accent' : 'hover:bg-raised',
              )}
            >
              <input
                type="checkbox"
                aria-label={`Add #${orderNumber(o)} to pick list`}
                checked={ticked.has(o.id)}
                onChange={() => toggle(o.id)}
                className="mt-3.5 accent-accent"
              />
              <button aria-pressed={active} onClick={() => onSelect(o)} className="flex-1 min-w-0 text-left pr-4 py-3">
                <div className="flex items-center justify-between gap-2 mb-1">
                  <span className="text-xs font-medium text-ink">#{orderNumber(o)}</span>
                  {active && activeInProgress
                    ? <span className="text-2xs font-medium text-accent">In progress</span>
                    : !o.fulfilled && <span className="text-2xs font-medium text-warning">Not picked</span>}
                </div>
                <p className="text-2xs text-ink-3 truncate">{customerName(o)}</p>
                <p className="text-2xs text-ink-4 mt-0.5">{formatMoney(o.total_minor_units, o.currency_code)}</p>
              </button>
            </div>
          )
        })}
      </div>
    </div>
  )
}
