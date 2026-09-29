import { useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../../auth/AuthContext'
import * as reportsApi from '../../reports/api'
import { useSingleReport } from '../../reports/hooks/useSingleReport'
import { ReportPageShell, ReportCard, EmptyState } from './shared'
import { useReportPeriod } from '../../reports/hooks/useReportPeriod'

export default function InventorySalesPage() {
  const { token } = useAuth()
  const { t } = useTranslation('reports')
  // In ?period= -- refresh and shared links keep the chosen range.
  const [period, setPeriod] = useReportPeriod()
  const getInventorySalesReport = useCallback(reportsApi.getInventorySalesReport, [])
  const { data: inventorySales, loading, error } = useSingleReport(token ?? '', period, getInventorySalesReport)
  const inventoryVsSales = [...(inventorySales?.rows ?? [])].sort((a, b) => (b.turnover_basis_points ?? -1) - (a.turnover_basis_points ?? -1))

  return (
    <ReportPageShell title={t('inventorySales.title')} period={period} onPeriodChange={setPeriod} loading={loading} error={error}>
      <ReportCard title={t('inventorySales.title')} description={t('inventorySales.desc')}>
        {inventoryVsSales.length === 0 ? (
          <EmptyState>{t('inventorySales.empty')}</EmptyState>
        ) : (
          <table className="w-full text-xs">
            <thead>
              <tr className="dark:bg-night-900/50 bg-gray-50">
                {[t('col.product'), t('col.sku'), t('col.stockOnHand'), t('col.sold'), t('col.turnover')].map(h => (
                  <th key={h} className="px-4 py-2 text-left text-2xs font-medium text-ink-4 uppercase">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-line/50">
              {inventoryVsSales.map(row => (
                <tr key={row.sku_id} className="hover:bg-hover/30">
                  <td className="px-4 py-2 text-ink font-medium">{row.product_name || row.sku_code}</td>
                  <td className="px-4 py-2 font-mono text-ink-4">{row.sku_code}</td>
                  <td className="px-4 py-2 text-ink-3">{row.stock_on_hand}</td>
                  <td className="px-4 py-2 text-ink-3">{row.quantity_sold_in_period}</td>
                  <td className="px-4 py-2 text-ink font-medium">
                    {row.turnover_basis_points === null ? t('na') : `${(row.turnover_basis_points / 100).toFixed(1)}%`}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </ReportCard>
    </ReportPageShell>
  )
}
