import { useState } from 'react'
import { Link } from 'react-router-dom'
import { FileText, UserRoundPen, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { apiErrorText } from '../api/apiErrorText'
import { getCustomer } from '../customers/api'
import { CustomerPicker, type PickedCustomer } from '../customers/CustomerPicker'
import { orderCustomerChange } from './orderCustomer'
import { customerName } from './salesOrderDisplay'
import type { SalesOrder, UpdateSalesOrderInput } from './types'

// The order's customer, with picking another one (its addresses come along)
// or removing it, and the quote the order was created from.
export function OrderCustomerSection({ token, order, onUpdate }: {
  token: string
  order: SalesOrder
  onUpdate: (input: UpdateSalesOrderInput) => Promise<unknown>
}) {
  const { t } = useTranslation(['orders', 'customers', 'common'])
  const [picking, setPicking] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  async function change(picked: PickedCustomer | null) {
    setSaving(true)
    setError('')
    try {
      await onUpdate(orderCustomerChange(picked ? await getCustomer(token, picked.id) : null))
      setPicking(false)
    } catch (e) {
      setError(apiErrorText(e) ?? t('detail.customerChangeFailed'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-1.5">
        {order.customer_id ? (
          <Link to={`/customers/${order.customer_id}`} className="text-accent hover:underline">{customerName(order)}</Link>
        ) : (
          <span className="text-ink">{customerName(order)}</span>
        )}
        {order.customer_id && (
          <button onClick={() => change(null)} disabled={saving} aria-label={t('customers:picker.remove')} title={t('customers:picker.remove')}
            className="w-5 h-5 flex items-center justify-center rounded hover:bg-hover text-ink-4 disabled:opacity-40"><X size={11} /></button>
        )}
        {!picking && (
          <button onClick={() => setPicking(true)} disabled={saving} aria-label={t('detail.chooseCustomer')} title={t('detail.chooseCustomer')}
            className="w-5 h-5 flex items-center justify-center rounded hover:bg-hover text-ink-4 disabled:opacity-40"><UserRoundPen size={11} /></button>
        )}
      </div>
      {picking && (
        <div className="space-y-1">
          <CustomerPicker token={token} value={null} onChange={c => { if (c) void change(c) }} />
          <p className="text-2xs text-ink-4">{t('detail.chooseCustomerHint')}</p>
          <button onClick={() => setPicking(false)} className="text-2xs text-ink-4 hover:text-ink-2">{t('common:actions.cancel')}</button>
        </div>
      )}
      {error && <p role="alert" className="text-2xs text-danger">{error}</p>}
      {order.quote_id && (
        <Link to={`/quotes/${order.quote_id}`} className="inline-flex items-center gap-1 text-2xs text-accent hover:underline">
          <FileText size={11} /> {t('detail.fromQuote')}
        </Link>
      )}
    </div>
  )
}
