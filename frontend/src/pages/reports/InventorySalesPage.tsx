import { useCallback } from 'react'
import { useAuth } from '../../auth/AuthContext'
import * as reportsApi from '../../reports/api'
import { useSingleReport } from '../../reports/hooks/useSingleReport'
import { ReportPageShell, ReportCard, EmptyState } from './shared'
import { useReportPeriod } from '../../reports/hooks/useReportPeriod'

export default function InventorySalesPage() {
  const { token } = useAuth()
  // In ?period= -- refresh and shared links keep the chosen range.
  const [period, setPeriod] = useReportPeriod()
  const getInventorySalesReport = useCallback(reportsApi.getInventorySalesReport, [])
  const { data: inventorySales, loading, error } = useSingleReport(token ?? '', period, getInventorySalesReport)
  const inventoryVsSales = [...(inventorySales?.rows ?? [])].sort((a, b) => (b.turnover_basis_points ?? -1) - (a.turnover_basis_points ?? -1))

  return (
    <ReportPageShell title="Inventory + Sales" period={period} onPeriodChange={setPeriod} loading={loading} error={error}>
      <ReportCard title="Inventory + Sales" description="Stock on hand vs. units sold in the selected period, sorted by turnover">
        {inventoryVsSales.length === 0 ? (
          <EmptyState>No stocked or sold SKUs in this period</EmptyState>
        ) : (
          <table className="w-full text-xs">
            <thead>
              <tr className="dark:bg-[#111827]/50 bg-gray-50">
                {['Product', 'SKU', 'Stock on Hand', 'Sold', 'Turnover'].map(h => (
                  <th key={h} className="px-4 py-2 text-left text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y dark:divide-[#2D3748]/50 divide-gray-100">
              {inventoryVsSales.map(row => (
                <tr key={row.sku_id} className="hover:dark:bg-[#2D3748]/30 hover:bg-gray-50">
                  <td className="px-4 py-2 dark:text-gray-200 text-gray-800 font-medium">{row.product_name || row.sku_code}</td>
                  <td className="px-4 py-2 font-mono dark:text-gray-500 text-gray-400">{row.sku_code}</td>
                  <td className="px-4 py-2 dark:text-gray-400 text-gray-500">{row.stock_on_hand}</td>
                  <td className="px-4 py-2 dark:text-gray-400 text-gray-500">{row.quantity_sold_in_period}</td>
                  <td className="px-4 py-2 dark:text-gray-200 text-gray-800 font-medium">
                    {row.turnover_basis_points === null ? 'n/a' : `${(row.turnover_basis_points / 100).toFixed(1)}%`}
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
