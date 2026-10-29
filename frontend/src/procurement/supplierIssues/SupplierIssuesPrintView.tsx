import { useEffect } from 'react'
import { X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { SupplierIssue } from '../types'
import { useDateFormat } from '../../format/DateFormatProvider'
import { SUPPLIER_ISSUE_KINDS, issueItemName } from './supplierIssueText'
import { PrintSheet } from '../../components/print/PrintSheet'

interface SupplierIssuesPrintViewProps {
  issues: SupplierIssue[]
  poRef: string
  supplierName: string | null
  orderDate: string
  onClose: () => void
}

// Printable list for the supplier -- "Save as PDF" in the browser's print
// dialog makes the PDF. Same zero-dependency approach as InvoicePrintView.
export function SupplierIssuesPrintView({ issues, poRef, supplierName, orderDate, onClose }: SupplierIssuesPrintViewProps) {
  const { t } = useTranslation(['procurement', 'common'])
  const { formatDate } = useDateFormat()
  useEffect(() => { window.print() }, [])

  return (
    <PrintSheet>
      <button
        aria-label={t('common:actions.close')} title={t('common:actions.close')} onClick={onClose}
        className="print:hidden fixed top-4 right-4 w-9 h-9 flex items-center justify-center rounded-lg bg-gray-100 hover:bg-gray-200 text-gray-600"
      >
        <X size={16} />
      </button>
      <div className="max-w-3xl mx-auto p-6 sm:p-10 print:p-0 print:max-w-none text-gray-900">
        <h1 className="text-xl font-bold">{t('supplierIssues.print.title', { ref: poRef })}</h1>
        <div className="mt-2 mb-6 text-sm text-gray-700 space-y-0.5">
          {supplierName && <p>{t('supplierIssues.print.supplier', { name: supplierName })}</p>}
          <p>{t('supplierIssues.print.orderDate', { date: formatDate(orderDate) })}</p>
          <p>{t('supplierIssues.print.printedOn', { date: formatDate(new Date().toISOString()) })}</p>
        </div>
        {SUPPLIER_ISSUE_KINDS.map(kind => {
          const ofKind = issues.filter(i => i.kind === kind)
          if (ofKind.length === 0) return null
          return (
            <section key={kind} className="mb-6 break-inside-avoid">
              <h2 className="text-sm font-semibold mb-1">{t(`supplierIssues.kind.${kind}`)}</h2>
              <table className="w-full text-sm border-collapse">
                <thead>
                  <tr className="border-b-2 border-gray-900">
                    <th className="text-left py-1.5">{t('supplierIssues.col.item')}</th>
                    <th className="text-left py-1.5">{t('supplierIssues.col.supplierCode')}</th>
                    <th className="text-right py-1.5">{t('supplierIssues.col.qty')}</th>
                    <th className="text-left py-1.5 pl-4">{t('supplierIssues.col.reason')}</th>
                  </tr>
                </thead>
                <tbody>
                  {ofKind.map(i => (
                    <tr key={`${i.source}:${i.source_id}`} className="border-b border-gray-300 align-top">
                      <td className="py-2 pr-2">{issueItemName(i)}</td>
                      <td className="py-2 pr-2">{i.supplier_sku || '—'}</td>
                      <td className="py-2 text-right tabular-nums">{i.quantity}</td>
                      <td className="py-2 pl-4">
                        {i.kind === 'missing' && i.expected_qty != null
                          ? t('supplierIssues.orderedReceived', { expected: i.expected_qty, received: i.received_qty ?? 0 })
                          : i.reason}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </section>
          )
        })}
      </div>
    </PrintSheet>
  )
}
