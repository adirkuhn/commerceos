import { useTranslation } from 'react-i18next'
import { Money } from '../format/amountVisibility'
import type { SalesOrderPayment } from './types'

interface SalesOrderPaymentsCardProps {
  payments: SalesOrderPayment[]
  currency: string
}

// The order detail's "Payments" card: every part that paid the order (more
// than one for a split payment). Renders nothing for an order whose source
// sent no breakdown -- OrderPaymentCard already shows its payment method.
export function SalesOrderPaymentsCard({ payments, currency }: SalesOrderPaymentsCardProps) {
  const { t } = useTranslation('orders')
  if (payments.length === 0) return null

  return (
    <div className="rounded-xl border border-line p-4 space-y-2 text-xs">
      <h3 className="text-xs font-semibold text-ink">{t('detail.payments')}</h3>
      <ul className="divide-y divide-line/60">
        {payments.map((payment, index) => (
          <li key={index} className="py-1.5 flex justify-between gap-3">
            <div className="min-w-0">
              <p className="text-ink">{payment.method}</p>
              {payment.transaction_id && (
                <p className="text-2xs text-ink-4 truncate">
                  {t('detail.transaction')} <span className="font-mono">{payment.transaction_id}</span>
                </p>
              )}
            </div>
            <Money minor={payment.amount_minor_units} currency={currency} className="font-medium text-ink shrink-0" />
          </li>
        ))}
      </ul>
    </div>
  )
}
