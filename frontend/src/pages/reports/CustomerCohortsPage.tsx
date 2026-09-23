import { useCallback } from 'react'
import { useAuth } from '../../auth/AuthContext'
import * as reportsApi from '../../reports/api'
import { useSingleReport } from '../../reports/hooks/useSingleReport'
import { ReportPageShell, ReportCard, EmptyState, formatMoney } from './shared'
import { useReportPeriod } from '../../reports/hooks/useReportPeriod'

export default function CustomerCohortsPage() {
  const { token } = useAuth()
  // In ?period= -- refresh and shared links keep the chosen range.
  const [period, setPeriod] = useReportPeriod()
  const getCustomerCohortsReport = useCallback(reportsApi.getCustomerCohortsReport, [])
  const { data: customerCohorts, loading, error } = useSingleReport(token ?? '', period, getCustomerCohortsReport)
  const currency = customerCohorts?.currency_code ?? 'USD'
  const cohorts = [...(customerCohorts?.cohorts ?? [])].sort((a, b) => a.cohort_month.localeCompare(b.cohort_month))
  const maxMonthsSinceCohort = cohorts.reduce((max, c) => c.retention.reduce((m, r) => Math.max(m, r.months_since_cohort), max), 0)
  const cohortMonthColumns = Array.from({ length: maxMonthsSinceCohort + 1 }, (_, i) => i)

  return (
    <ReportPageShell title="Customer Cohorts" period={period} onPeriodChange={setPeriod} loading={loading} error={error}>
      <ReportCard
        title="Customer Cohorts"
        description="Cohorts by month of each customer's first-ever order in this period, retention by month since acquisition"
      >
        {cohorts.length === 0 ? (
          <EmptyState>No new customers acquired in this period</EmptyState>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr className="dark:bg-[#111827]/50 bg-gray-50">
                  <th className="px-4 py-2 text-left text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase">Cohort</th>
                  <th className="px-4 py-2 text-left text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase">Size</th>
                  {cohortMonthColumns.map(m => (
                    <th key={m} className="px-3 py-2 text-center text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase whitespace-nowrap">
                      Month {m}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y dark:divide-[#2D3748]/50 divide-gray-100">
                {cohorts.map(cohort => {
                  const byMonth = new Map(cohort.retention.map(r => [r.months_since_cohort, r]))
                  return (
                    <tr key={cohort.cohort_month} className="hover:dark:bg-[#2D3748]/30 hover:bg-gray-50">
                      <td className="px-4 py-2 dark:text-gray-200 text-gray-800 font-medium whitespace-nowrap">
                        {new Date(cohort.cohort_month).toLocaleDateString(undefined, { month: 'short', year: 'numeric' })}
                      </td>
                      <td className="px-4 py-2 dark:text-gray-400 text-gray-500">{cohort.cohort_size}</td>
                      {cohortMonthColumns.map(m => {
                        const cell = byMonth.get(m)
                        if (!cell) {
                          return <td key={m} className="px-3 py-2 text-center dark:text-gray-700 text-gray-200">—</td>
                        }
                        const ratePercent = cell.retention_rate_basis_points / 100
                        return (
                          <td key={m} className="px-3 py-2 text-center" title={`${cell.retained_customers} customer(s), ${formatMoney(cell.revenue_minor_units, currency)}`}>
                            <span
                              className="inline-block px-2 py-0.5 rounded text-[10px] font-medium text-white"
                              style={{ backgroundColor: `rgba(37, 99, 235, ${Math.min(1, Math.max(0.15, ratePercent / 100))})` }}
                            >
                              {ratePercent.toFixed(0)}%
                            </span>
                          </td>
                        )
                      })}
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
      </ReportCard>
    </ReportPageShell>
  )
}
