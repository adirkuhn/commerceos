import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import { CreateSalesOrderPage } from './CreateSalesOrderPage'
import { editSalesOrder } from './api'
import { canEditOrder } from './create/editDraft'
import { useSalesOrderDetail } from './hooks/useSalesOrderDetail'

// /sales-orders/<id>/edit: the order form on an existing manual order.
export function EditSalesOrderPage({ salesOrderId, onClose }: { salesOrderId: string; onClose: () => void }) {
  const { token } = useAuth()
  const { t } = useTranslation(['orders', 'errors'])
  const { detail, error } = useSalesOrderDetail(token ?? '', salesOrderId)

  if (!detail || !canEditOrder(detail.sales_order)) {
    const text = error || (detail ? t('errors:SALES_ORDER_NOT_EDITABLE') : t('edit.loading'))
    return <p role={detail || error ? 'alert' : 'status'} className="m-6 text-xs text-ink-3">{text}</p>
  }
  return (
    <CreateSalesOrderPage onClose={onClose}
      edit={{ order: detail, onSave: input => editSalesOrder(token ?? '', salesOrderId, input) }} />
  )
}
