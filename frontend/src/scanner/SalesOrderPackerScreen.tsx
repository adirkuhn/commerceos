import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Search, LogOut, Package, ArrowLeft, Lock } from 'lucide-react'
import { usePackingQueue } from '../oms/hooks/usePackingQueue'
import { useMemberNames } from '../team/hooks/useMemberNames'
import { readClaims } from '../auth/jwtClaims'
import { canTakeOverLocks, lockLabel } from '../packing/lockDisplay'
import * as api from './api'
import type { SalesOrder } from '../oms/types'
import { useDateFormat } from '../format/DateFormatProvider'
import { customerName } from '../oms/salesOrderDisplay'
import { unknownCustomerLabel } from './scannerLabels'
import { apiErrorText } from '../api/apiErrorText'

interface SalesOrderPackerScreenProps {
  token: string
  onPackingStarted: (sessionId: string, order: SalesOrder) => void
  onLogout: () => void
  onBack: () => void
}

function formatPrice(minorUnits: number, currency: string): string {
  return `${currency} ${(minorUnits / 100).toFixed(2)}`
}

// Mirrors SalesOrderPickerScreen -- the packing-phase counterpart. Shows the
// packing queue: every unpacked order, picked or not ("Not picked" marks
// the latter). An order someone else is working is read-only, labelled with
// who and for how long -- admins may still tap it to take over (the backend
// decides).
export function SalesOrderPackerScreen({ token, onPackingStarted, onLogout, onBack }: SalesOrderPackerScreenProps) {
  const { formatDate } = useDateFormat()
  const { t } = useTranslation(['scanner', 'common'])
  const queue = usePackingQueue(token)
  const nameFor = useMemberNames(token)
  const me = readClaims(token)
  const isAdmin = canTakeOverLocks(me?.role)
  const [query, setQuery] = useState('')
  const [startError, setStartError] = useState('')
  const [startingId, setStartingId] = useState<string | null>(null)
  const loading = queue.loading
  const error = startError || queue.error

  const filtered = queue.entries.filter(({ sales_order: o }) =>
    query.trim() === ''
    || customerName(o, unknownCustomerLabel()).toLowerCase().includes(query.toLowerCase())
    || (o.number || o.external_order_id).toLowerCase().includes(query.toLowerCase()),
  )

  async function handleSelect(order: SalesOrder) {
    setStartError('')
    setStartingId(order.id)
    try {
      const session = await api.startOrResumePacking(token, order.id)
      onPackingStarted(session.id, order)
    } catch (err) {
      setStartError(apiErrorText(err) ?? t('errors.startPacking'))
      setStartingId(null)
    }
  }

  return (
    <div className="fixed inset-0 flex flex-col bg-night-950 text-gray-100">
      <div className="flex items-center gap-3 px-4 py-4 border-b border-night-700 shrink-0">
        <button aria-label={t('shared.backToApps')} onClick={onBack} className="w-11 h-11 -ml-3 -my-2 flex items-center justify-center rounded-lg shrink-0" title={t('shared.backToApps')}>
          <ArrowLeft size={18} className="text-gray-400" />
        </button>
        <h1 className="flex-1 text-base font-semibold">{t('orderPicker.packTitle')}</h1>
        <button aria-label={t('shared.unpair')} onClick={onLogout} className="p-2 rounded-lg hover:bg-night-800" title={t('shared.unpair')}>
          <LogOut size={16} className="text-gray-400" />
        </button>
      </div>
      <div className="px-4 py-3 shrink-0">
        <div className="relative">
          <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
          <input
            value={query}
            onChange={e => setQuery(e.target.value)}
            placeholder={t('orderPicker.search')}
            className="w-full pl-9 pr-3 py-2.5 text-sm rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus"
          />
        </div>
      </div>
      {error && <p className="mx-4 mb-2 text-sm text-red-400">{error}</p>}
      <div className="flex-1 overflow-y-auto px-4 pb-4 space-y-2">
        {loading && <p className="text-center text-sm text-gray-400 py-8">{t('common:status.loading')}</p>}
        {!loading && filtered.length === 0 && (
          <p className="text-center text-sm text-gray-400 py-8">{t('orderPicker.noneToPack')}</p>
        )}
        {filtered.map(({ sales_order: o, lock }) => {
          const lockedByOther = lock !== null && lock.user_id !== me?.userId
          return (
            <button
              key={o.id}
              onClick={() => handleSelect(o)}
              disabled={startingId !== null || (lockedByOther && !isAdmin)}
              className="w-full flex items-center gap-3 p-3.5 rounded-xl bg-night-800 border border-night-700 text-left hover:border-accent disabled:opacity-50 transition-colors"
            >
              <div className="w-9 h-9 rounded-lg bg-night-700 flex items-center justify-center shrink-0">
                <Package size={16} className="text-gray-400" />
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium truncate">{customerName(o, unknownCustomerLabel())}</p>
                <p className="text-xs text-gray-400 flex items-center gap-1.5">
                  <span>#{o.number || o.external_order_id}</span>
                  <span>·</span>
                  <span>{formatDate(o.occurred_at)}</span>
                  {!o.fulfilled && (
                    <>
                      <span>·</span>
                      <span className="text-amber-400">{t('orderPicker.notPicked')}</span>
                    </>
                  )}
                </p>
                {lockedByOther && (
                  <p className="text-xs text-gray-400 flex items-center gap-1 mt-0.5">
                    <Lock size={11} className="shrink-0" />
                    <span className="truncate">{lockLabel(lock, nameFor(lock.user_id))}</span>
                    {isAdmin && <span className="text-accent shrink-0">{t('orderPicker.takeOver')}</span>}
                  </p>
                )}
              </div>
              <p className="text-xs font-medium text-gray-300 shrink-0">{formatPrice(o.total_minor_units, o.currency_code)}</p>
              {startingId === o.id && <span className="text-xs text-accent shrink-0">{t('pickDone.starting')}</span>}
            </button>
          )
        })}
      </div>
    </div>
  )
}
