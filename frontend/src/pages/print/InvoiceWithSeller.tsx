import { useAuth } from '../../auth/AuthContext'
import { useCompanyProfile } from '../../settings/company/useCompanyProfile'
import type { SalesOrder, SalesOrderLineItem } from '../../oms/types'
import { InvoicePrintView } from './InvoicePrintView'

// Loads the tenant's company profile before mounting the invoice, which
// opens the print dialog as soon as it renders.
export function InvoiceWithSeller(props: { order: SalesOrder; lineItems: SalesOrderLineItem[]; onClose: () => void }) {
  const { token } = useAuth()
  const { profile, loading } = useCompanyProfile(token ?? '')
  if (loading) {
    return <div className="fixed inset-0 z-50 bg-white flex items-center justify-center text-sm text-gray-600">Preparing invoice…</div>
  }
  return <InvoicePrintView {...props} seller={profile} />
}
