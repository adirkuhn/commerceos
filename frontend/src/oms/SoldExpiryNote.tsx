import { useTranslation } from 'react-i18next'
import { useDateFormat } from '../format/DateFormatProvider'

// The expiry date WooCommerce reported as sold for an order line
// (SalesOrderLineItem.sold_expiry_date). One component so every surface
// that lists order lines -- order detail, picking, packing, print --
// shows the same label, tooltip and tenant date format. Renders nothing
// for lines without one (every non-Woo line).
interface SoldExpiryNoteProps {
  date: string | null | undefined
  className?: string
}

export function SoldExpiryNote({ date, className = 'text-2xs text-ink-4' }: SoldExpiryNoteProps) {
  const { formatDate } = useDateFormat()
  const { t } = useTranslation('orders')
  if (!date) return null
  return (
    <p className={className} title={t('soldExpiryTooltip')}>
      {t('soldExpiry', { date: formatDate(date) })}
    </p>
  )
}
