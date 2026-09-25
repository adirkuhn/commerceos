import type { ReactNode } from 'react'
import { Undo2 } from 'lucide-react'
import type { SalesOrder } from '../oms/types'
import { customerName, shippingAddressLine } from '../oms/salesOrderDisplay'
import { UNKNOWN_CUSTOMER_LABEL } from './scannerLabels'

// The order info band under a session screen's header: number, customer
// and ship-to address, plus whatever extra lines the screen adds (billing,
// shipping method, the start-over button).
export function ScanSessionOrderBand({ order, children }: { order: SalesOrder; children?: ReactNode }) {
  return (
    <div className="px-4 py-3 border-b border-night-700 shrink-0 bg-night-900">
      <div className="flex items-baseline gap-1.5 flex-wrap">
        <span className="text-base font-semibold">#{order.number || order.external_order_id}</span>
        <span className="text-gray-400">·</span>
        <span className="text-base text-gray-300">{customerName(order, UNKNOWN_CUSTOMER_LABEL)}</span>
      </div>
      {shippingAddressLine(order.shipping) && (
        <p className="text-sm text-gray-400 mt-0.5">{shippingAddressLine(order.shipping)}</p>
      )}
      {children}
    </div>
  )
}

export function StartOverButton({ onClick, disabled }: { onClick: () => void; disabled: boolean }) {
  return (
    <button
      type="button" onClick={onClick} disabled={disabled}
      className="mt-0.5 -mb-2 py-2.5 flex items-center gap-1.5 text-sm text-red-400 disabled:opacity-50"
    >
      <Undo2 size={12} /> Start over
    </button>
  )
}
