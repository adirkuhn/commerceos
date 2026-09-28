import { useDateFormat } from '../format/DateFormatProvider'

// The expiry date WooCommerce reported as sold for an order line
// (SalesOrderLineItem.sold_expiry_date). One component so every surface
// that lists order lines -- order detail, picking, packing, print --
// shows the same label, tooltip and tenant date format. Renders nothing
// for lines without one (every non-Woo line).
export const SOLD_EXPIRY_TOOLTIP = 'Reported by WooCommerce at checkout -- may not match the batch actually allocated'

interface SoldExpiryNoteProps {
  date: string | null | undefined
  className?: string
}

export function SoldExpiryNote({ date, className = 'text-2xs text-ink-4' }: SoldExpiryNoteProps) {
  const { formatDate } = useDateFormat()
  if (!date) return null
  return (
    <p className={className} title={SOLD_EXPIRY_TOOLTIP}>
      Sold exp. {formatDate(date)}
    </p>
  )
}
