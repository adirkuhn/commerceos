import { CheckCircle, PackagePlus, Tag, Undo2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { SalesOrder } from '../oms/types'
import { customerName, shippingAddressLine } from '../oms/salesOrderDisplay'
import { formatMoney } from '../format/money'

interface PackingSessionHeaderProps {
  order: SalesOrder
  showCreateLabel: boolean
  readOnly: boolean
  onCreateLabel: () => void
  onAddItem: () => void
  onStartOver: () => void
  onComplete: () => void
}

// The open order's summary and the session-wide actions.
export function PackingSessionHeader({ order, showCreateLabel, readOnly, onCreateLabel, onAddItem, onStartOver, onComplete }: PackingSessionHeaderProps) {
  const { t } = useTranslation('packing')
  return (
    <div className="flex items-center gap-4 px-6 py-3 border-b border-line shrink-0">
      <div className="flex-1">
        <div className="flex items-center gap-2 mb-0.5">
          <h2 className="text-sm font-semibold text-ink">#{order.number || order.external_order_id}</h2>
          <span className="text-xs text-ink-4">·</span>
          <span className="text-xs text-ink-3">{customerName(order)}</span>
        </div>
        <p className="text-xs text-ink-4">{formatMoney(order.total_minor_units, order.currency_code)}</p>
        {shippingAddressLine(order.shipping) && (
          <p className="text-xs text-ink-4 mt-0.5">{shippingAddressLine(order.shipping)}</p>
        )}
      </div>
      <div className="flex items-center gap-2 shrink-0">
        {showCreateLabel && (
          <button
            type="button" onClick={onCreateLabel} disabled={readOnly}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-line bg-raised text-xs font-medium text-ink-2 hover:bg-hover disabled:opacity-50"
          >
            <Tag size={13} /> {t('header.createLabel')}
          </button>
        )}
        <button
          type="button" onClick={onAddItem} disabled={readOnly}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-line bg-raised text-xs font-medium text-ink-2 hover:bg-hover disabled:opacity-50"
        >
          <PackagePlus size={13} /> {t('header.addItem')}
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
