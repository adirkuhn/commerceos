import { useEffect } from 'react'
import { X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useDateFormat } from '../format/DateFormatProvider'
import { formatMoney } from '../format/money'

import type { ExportLine } from './quoteCsv'

// The quote for the customer: sku, product, quantity, price, line totals and the
// order total -- never cost or profit. "Save as PDF" in the browser's print
// dialog makes the PDF, same zero-dependency approach as InvoicePrintView.
// Amounts always print, even while they're hidden on screen.
export function QuotePrintView({ reference, name, currency, lines, onClose }: {
  reference: string | null
  name: string
  currency: string
  lines: ExportLine[]
  onClose: () => void
}) {
  const { t } = useTranslation(['quotes', 'common'])
  const { formatDate } = useDateFormat()
  useEffect(() => { window.print() }, [])
  const money = (minor: number) => formatMoney(minor, currency)
  const total = lines.reduce((sum, l) => sum + l.quantity * l.unitPriceMinor, 0)
  const title = [t('print.title'), reference].filter(Boolean).join(' ') + (name.trim() ? ` — ${name.trim()}` : '')

  return (
    <div className="fixed inset-0 z-50 bg-white overflow-y-auto">
      <button
        aria-label={t('common:actions.close')} title={t('common:actions.close')} onClick={onClose}
        className="print:hidden fixed top-4 right-4 w-9 h-9 flex items-center justify-center rounded-lg bg-gray-100 hover:bg-gray-200 text-gray-600"
      >
        <X size={16} />
      </button>
      <div className="max-w-3xl mx-auto p-6 sm:p-10 text-gray-900">
        <h1 className="text-xl font-bold">{title}</h1>
        <p className="mt-1 mb-6 text-sm text-gray-700">{formatDate(new Date().toISOString())}</p>
        <table className="w-full text-sm border-collapse">
          <thead>
            <tr className="border-b-2 border-gray-900">
              <th className="text-left py-1.5">{t('print.sku')}</th>
              <th className="text-left py-1.5">{t('print.product')}</th>
              <th className="text-right py-1.5">{t('print.quantity')}</th>
              <th className="text-right py-1.5">{t('print.unitPrice')}</th>
              <th className="text-right py-1.5">{t('print.lineTotal')}</th>
            </tr>
          </thead>
          <tbody>
            {lines.map(l => (
              <tr key={l.sku} className="border-b border-gray-300 break-inside-avoid">
                <td className="py-2 pr-2 font-mono text-xs">{l.sku}</td>
                <td className="py-2 pr-2">{l.product}</td>
                <td className="py-2 text-right tabular-nums">{l.quantity}</td>
                <td className="py-2 text-right tabular-nums">{money(l.unitPriceMinor)}</td>
                <td className="py-2 text-right tabular-nums">{money(l.quantity * l.unitPriceMinor)}</td>
              </tr>
            ))}
          </tbody>
          <tfoot>
            <tr className="border-t-2 border-gray-900 font-semibold">
              <td colSpan={4} className="py-2 text-right">{t('print.orderTotal')}</td>
              <td className="py-2 text-right tabular-nums">{money(total)}</td>
            </tr>
          </tfoot>
        </table>
      </div>
    </div>
  )
}
