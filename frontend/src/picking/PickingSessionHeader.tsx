import { CheckCircle, PauseCircle, Undo2, Lock } from 'lucide-react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import type { SalesOrder } from '../oms/types'
import { customerName, shippingAddressLine } from '../oms/salesOrderDisplay'
import { Money } from '../format/amountVisibility'
import { formatLockAge } from '../packing/lockDisplay'

interface PickingSessionHeaderProps {
  order: SalesOrder
  pausing: boolean
  readOnly: boolean
  onPause: () => void
  onStartOver: () => void
  onComplete: () => void
}

// The open order's header: who/where it ships to, and pause / start over /
// complete.
export function PickingSessionHeader({ order, pausing, readOnly, onPause, onStartOver, onComplete }: PickingSessionHeaderProps) {
  const { t } = useTranslation('picking')
  return (
    <div className={clsx(
      'flex items-center gap-4 px-6 py-3 border-b border-line shrink-0',
    )}>
      <div className="flex-1">
        <div className="flex items-center gap-2 mb-0.5">
          <h2 className="text-sm font-semibold text-ink">#{order.number || order.external_order_id}</h2>
          <span className="text-xs text-ink-4">·</span>
          <span className="text-xs text-ink-3">{customerName(order)}</span>
        </div>
        <p className="text-xs text-ink-4"><Money minor={order.total_minor_units} currency={order.currency_code} /></p>
        {shippingAddressLine(order.shipping) && (
          <p className="text-xs text-ink-4 mt-0.5">{shippingAddressLine(order.shipping)}</p>
        )}
      </div>
      <div className="flex items-center gap-2 shrink-0">
        <button
          type="button" onClick={onPause} disabled={pausing || readOnly}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-line bg-raised text-xs font-medium text-ink-2 hover:bg-hover disabled:opacity-50"
        >
          <PauseCircle size={13} /> {pausing ? t('header.pausing') : t('header.pause')}
        </button>
        <button
          type="button" onClick={onStartOver} disabled={readOnly}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-line bg-raised text-xs font-medium text-danger hover:bg-hover disabled:opacity-50"
        >
          <Undo2 size={13} /> {t('header.startOver')}
        </button>
        <button
          type="button" onClick={onComplete} disabled={readOnly}
          className="flex items-center gap-1.5 px-4 py-1.5 rounded-lg bg-green-600 text-white text-xs font-semibold hover:bg-green-700 disabled:opacity-50"
        >
          <CheckCircle size={13} /> {t('header.complete')}
        </button>
      </div>
    </div>
  )
}

interface PickingLockBannerProps {
  holderName: string
  lockedSince?: string | null
  canTakeOver: boolean
  onTakeOver: () => void
}

// Someone else holds this session's lock: view only, with take-over for
// roles allowed to.
export function PickingLockBanner({ holderName, lockedSince, canTakeOver, onTakeOver }: PickingLockBannerProps) {
  const { t } = useTranslation('picking')
  return (
    <div role="status" className="flex items-center gap-2 px-6 py-2 border-b border-line bg-warning/10 text-xs text-ink-2">
      <Lock size={12} className="shrink-0" />
      <span className="flex-1">
        {t('lock.message', { name: holderName, since: lockedSince ? ` · ${formatLockAge(lockedSince)}` : '' })}
      </span>
      {canTakeOver && (
        <button type="button" onClick={onTakeOver} className="px-2 py-0.5 rounded-md border border-line bg-raised font-medium hover:bg-hover">
          {t('lock.takeOver')}
        </button>
      )}
    </div>
  )
}
