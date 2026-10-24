import { useState } from 'react'
import { Link } from 'react-router-dom'
import { FileText, Pencil } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { apiErrorText } from '../api/apiErrorText'
import { getCustomer } from '../customers/api'
import { CustomerPicker, type PickedCustomer } from '../customers/CustomerPicker'
import { orderCustomerChange } from './orderCustomer'
import { customerName } from './salesOrderDisplay'
import type { SalesOrder, UpdateSalesOrderInput } from './types'

// The order's customer and the quote it came from. Changing the customer is
// an edit: pick one, then Save (its addresses come along) or Cancel --
// nothing is written before Save.
export function OrderCustomerSection({ token, order, onUpdate }: {
  token: string
  order: SalesOrder
  onUpdate: (input: UpdateSalesOrderInput) => Promise<unknown>
}) {
  const { t } = useTranslation(['orders', 'common'])
  const [editing, setEditing] = useState(false)
  const [picked, setPicked] = useState<PickedCustomer | null>(null)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  function stopEditing() {
    setEditing(false)
    setPicked(null)
    setError('')
  }

  async function save() {
    if (!picked) return
    setSaving(true)
    setError('')
    try {
      await onUpdate(orderCustomerChange(await getCustomer(token, picked.id)))
      stopEditing()
    } catch (e) {
      setError(apiErrorText(e) ?? t('detail.customerChangeFailed'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-2">
      {!editing ? (
        <div className="flex items-center gap-1.5">
          {order.customer_id ? (
            <Link to={`/customers/${order.customer_id}`} className="text-accent hover:underline">{customerName(order)}</Link>
          ) : (
            <span className="text-ink">{customerName(order)}</span>
          )}
          <button type="button" onClick={() => setEditing(true)} aria-label={t('detail.editCustomer')} title={t('detail.editCustomer')}
            className="w-5 h-5 flex items-center justify-center rounded hover:bg-hover text-ink-4"><Pencil size={11} /></button>
        </div>
      ) : (
        <div className="space-y-2">
          <CustomerPicker token={token} value={picked} onChange={setPicked} />
          <p className="text-2xs text-ink-4">{t('detail.chooseCustomerHint')}</p>
          {error && <p role="alert" className="text-2xs text-danger">{error}</p>}
          <div className="flex gap-2">
            <button type="button" onClick={stopEditing} disabled={saving}
              className="px-2.5 py-1 rounded-lg border border-line text-2xs text-ink-2 hover:bg-hover">{t('common:actions.cancel')}</button>
            <button type="button" onClick={save} disabled={!picked || saving}
              className="px-2.5 py-1 rounded-lg bg-accent-solid text-white text-2xs font-medium hover:bg-accent-solid-hover disabled:opacity-40">{t('common:actions.save')}</button>
          </div>
        </div>
      )}
      {order.quote_id && (
        <Link to={`/quotes/${order.quote_id}`} className="inline-flex items-center gap-1 text-2xs text-accent hover:underline">
          <FileText size={11} /> {t('detail.fromQuote')}
        </Link>
      )}
    </div>
  )
}
