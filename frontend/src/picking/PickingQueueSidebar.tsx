import { useState } from 'react'
import { Search, Lock } from 'lucide-react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import type { QueueEntry } from '../oms/types'
import { customerName, salesOrderSourceLabel } from '../oms/salesOrderDisplay'
import { MiniLegs } from '../oms/FulfilmentLegs'
import { Money } from '../format/amountVisibility'
import { lockLabel } from '../packing/lockDisplay'
import { filterPickingQueue } from './pickingQueueFilter'

interface PickingQueueSidebarProps {
  entries: QueueEntry[]
  loading: boolean
  error: string
  activeOrderId: string | null
  // The active order's session is in progress (shows the "In progress" tag).
  activeInProgress: boolean
  // The order currently open read-only (?view=), if any.
  viewId: string | null
  nameFor: (userId: string) => string
  onSelect: (entry: QueueEntry) => void
}

// The picking queue: orders still to pick, searchable, with who is working
// each and whether packing already has it.
export function PickingQueueSidebar({ entries, loading, error, activeOrderId, activeInProgress, viewId, nameFor, onSelect }: PickingQueueSidebarProps) {
  const { t } = useTranslation(['picking', 'common'])
  const [search, setSearch] = useState('')
  const filteredEntries = filterPickingQueue(entries, search)

  return (
    <div className="w-64 border-r border-line bg-sunken flex flex-col overflow-hidden shrink-0">
      <div className="px-4 py-3 border-b border-line">
        <h2 className="caps-label-widest text-2xs font-bold text-ink">{t('queue.title')}</h2>
        <p className="text-xs font-semibold text-figure tabular-nums mt-1">{t('queue.count', { count: entries.length })}</p>
      </div>
      <div className="px-3 py-2 border-b border-line">
        <div className="relative">
          <Search size={12} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-ink-4" />
          <input
            value={search}
            onChange={e => setSearch(e.target.value)}
            placeholder={t('queue.search')}
            className="w-full pl-7 pr-2 h-8 text-xs rounded-md bg-surface border border-line text-ink placeholder-ink-4 outline-none focus:ring-1 focus:ring-focus"
          />
        </div>
      </div>
      <div className="flex-1 overflow-y-auto py-2">
        {loading && <p className="px-4 py-3 text-2xs text-ink-4">{t('common:status.loading')}</p>}
        {error && <p className="px-4 py-3 text-2xs text-danger">{error}</p>}
        {!loading && filteredEntries.length === 0 && (
          <p className="px-4 py-3 text-2xs text-ink-4">{t('queue.noUnfulfilled')}</p>
        )}
        {filteredEntries.map(entry => {
          const { sales_order: o, lock, in_packing } = entry
          return (
            <button aria-pressed={activeOrderId === o.id || viewId === o.id}
              key={o.id}
              onClick={() => onSelect(entry)}
              className={clsx(
                'w-full text-left px-4 py-3 border-b border-line transition-colors',
                activeOrderId === o.id
                  ? 'bg-accent/10 ring-1 ring-inset ring-accent'
                  : 'hover:bg-raised',
              )}
            >
              <div className="flex items-center justify-between mb-1">
                <span className="text-xs font-semibold text-ink tabular-nums truncate">#{o.number || o.external_order_id}</span>
                {activeOrderId === o.id && activeInProgress
                  ? <span className="text-2xs font-semibold text-accent">{t('queue.inProgress')}</span>
                  : <MiniLegs order={o} />}
              </div>
              <p className="text-2xs text-ink-3 truncate">{customerName(o)} · {salesOrderSourceLabel(o.source)}</p>
              <p className="text-2xs text-ink-4 mt-0.5"><Money minor={o.total_minor_units} currency={o.currency_code} /></p>
              {(lock || in_packing) && (
                <p className="flex items-center gap-1 mt-1 text-2xs text-ink-4">
                  <Lock size={10} className="shrink-0" />
                  <span className="truncate">{lock ? lockLabel(lock, nameFor(lock.user_id)) : t('queue.inPacking')}</span>
                </p>
              )}
            </button>
          )
        })}
      </div>
    </div>
  )
}
