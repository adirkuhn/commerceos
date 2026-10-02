import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Copy } from 'lucide-react'
import { useDateFormat } from '../format/DateFormatProvider'
import { PaymentStatusBadge } from './PaymentStatusBadge'
import type { SalesOrder } from './types'

// The order detail's "Payment" card: did the payment succeed, how, its
// transaction id (to look it up at the payment provider) and when.
export function OrderPaymentCard({ order }: { order: SalesOrder }) {
  const { t } = useTranslation('orders')
  const { formatDateTime } = useDateFormat()
  const [copied, setCopied] = useState(false)
  const [copyError, setCopyError] = useState('')
  const method = order.payment_method_title || order.payment_method
  const reference = order.payment_reference

  const copyReference = async () => {
    if (!reference) return
    try {
      await navigator.clipboard.writeText(reference)
      setCopied(true)
      setCopyError('')
    } catch (e) {
      setCopyError(e instanceof Error ? e.message : String(e))
    }
  }

  return (
    <div className="rounded-xl border border-line p-4 space-y-1 text-xs">
      <h3 className="text-xs font-semibold text-ink mb-2">{t('detail.payment')}</h3>
      <p className="text-ink-3 flex items-center gap-1.5">{t('detail.paymentStatus')} <PaymentStatusBadge status={order.payment_status} /></p>
      <p className="text-ink-3">{t('detail.source')} <span className="text-ink capitalize">{order.source}</span></p>
      {method && <p className="text-ink-3">{t('detail.paymentLabel')} <span className="text-ink">{method}</span></p>}
      {reference && (
        <p className="text-ink-3 flex items-center gap-1.5 min-w-0">
          {t('detail.transaction')} <span className="text-ink font-mono truncate">{reference}</span>
          <button type="button" onClick={copyReference} aria-label={t('detail.copyTransaction')} title={t('detail.copyTransaction')} className="p-0.5 rounded text-ink-4 hover:text-ink-2">
            <Copy size={12} />
          </button>
          {copied && <span className="text-success">{t('detail.copied')}</span>}
        </p>
      )}
      {copyError && <p className="text-danger">{copyError}</p>}
      {order.date_paid && <p className="text-ink-3">{t('detail.paidOn')} <span className="text-ink">{formatDateTime(order.date_paid)}</span></p>}
      {order.shipping_method_title && (
        <p className="text-ink-3">{t('detail.shippingMethod')} <span className="text-ink">{order.shipping_method_title}</span></p>
      )}
    </div>
  )
}
