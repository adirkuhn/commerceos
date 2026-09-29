import { useEffect } from 'react'
import { X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { SalesOrder, SalesOrderLineItem, Address } from '../../oms/types'
import { useDateFormat } from '../../format/DateFormatProvider'
import { SoldExpiryNote } from '../../oms/SoldExpiryNote'

function addressName(address?: Address): string {
  const name = [address?.first_name, address?.last_name].filter(Boolean).join(' ')
  return name || address?.email || '—'
}

interface PickingListPrintViewProps {
  order: SalesOrder
  lineItems: SalesOrderLineItem[]
  onClose: () => void
}

// Plain-HTML printable view -- Tailwind's print: variants hide everything
// but the sheet itself, no PDF/print library dependency. Triggers the
// browser's own print dialog on mount; onClose (a print:hidden button,
// since the print dialog itself has no "back" affordance) returns to the
// normal detail chrome.
export function PickingListPrintView({ order, lineItems, onClose }: PickingListPrintViewProps) {
  const { formatDate } = useDateFormat()
  const { t } = useTranslation(['orders', 'common'])
  useEffect(() => { window.print() }, [])

  return (
    <div className="fixed inset-0 z-50 bg-white overflow-y-auto">
      <button aria-label={t('common:actions.close')}
        onClick={onClose}
        className="print:hidden fixed top-4 right-4 w-9 h-9 flex items-center justify-center rounded-lg bg-gray-100 hover:bg-gray-200 text-gray-600"
        title={t('common:actions.close')}
      >
        <X size={16} />
      </button>
      <div className="max-w-2xl mx-auto p-10 text-gray-900">
        <h1 className="text-2xl font-bold mb-1">{t('print.pickingList')}</h1>
        <p className="text-sm text-gray-600 mb-6">
          {t('print.orderLine', { number: order.number || order.external_order_id, date: formatDate(order.occurred_at) })}
        </p>

        <div className="mb-6 text-sm">
          <p className="font-medium">{t('print.shipTo')}</p>
          <p>{addressName(order.shipping)}</p>
          {order.shipping?.address_1 && <p>{order.shipping.address_1}</p>}
          {order.shipping?.address_2 && <p>{order.shipping.address_2}</p>}
          <p>{[order.shipping?.city, order.shipping?.state, order.shipping?.postcode].filter(Boolean).join(', ')}</p>
        </div>

        <table className="w-full text-sm border-collapse">
          <thead>
            <tr className="border-b-2 border-gray-900">
              <th className="text-left py-2">{t('common:fields.product')}</th>
              <th className="text-left py-2">{t('common:fields.sku')}</th>
              <th className="text-right py-2">{t('detail.qty')}</th>
              <th className="text-left py-2 pl-4">{t('print.picked')}</th>
            </tr>
          </thead>
          <tbody>
            {lineItems.map(li => (
              <tr key={li.id} className="border-b border-gray-300">
                <td className="py-2.5">
                  {li.name}
                  <SoldExpiryNote date={li.sold_expiry_date} className="text-xs text-gray-600" />
                </td>
                <td className="py-2.5 font-mono text-xs">{li.sku_code || '—'}</td>
                <td className="py-2.5 text-right">{li.quantity}</td>
                <td className="py-2.5 pl-4">
                  <span className="inline-block w-4 h-4 border border-gray-500" />
                </td>
              </tr>
            ))}
          </tbody>
        </table>

        {order.customer_note && (
          <div className="mt-6 text-sm">
            <p className="font-medium">{t('print.note')}</p>
            <p>{order.customer_note}</p>
          </div>
        )}
      </div>
    </div>
  )
}
