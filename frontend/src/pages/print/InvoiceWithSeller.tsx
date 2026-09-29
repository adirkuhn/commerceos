import { useTranslation } from 'react-i18next'
import { useAuth } from '../../auth/AuthContext'
import { useCompanyProfile } from '../../settings/company/useCompanyProfile'
import type { SalesOrder, SalesOrderLineItem } from '../../oms/types'
import { InvoicePrintView } from './InvoicePrintView'

// Loads the tenant's company profile before mounting the invoice, which
// opens the print dialog as soon as it renders.
export function InvoiceWithSeller(props: { order: SalesOrder; lineItems: SalesOrderLineItem[]; onClose: () => void }) {
  const { token } = useAuth()
  const { t } = useTranslation('orders')
  const { profile, loading } = useCompanyProfile(token ?? '')
  if (loading) {
    return <div className="fixed inset-0 z-50 bg-white flex items-center justify-center text-sm text-gray-600">{t('print.preparing')}</div>
  }
  return <InvoicePrintView {...props} seller={profile} />
}
