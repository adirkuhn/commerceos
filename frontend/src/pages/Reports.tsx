import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts'
import { TrendingUp, TrendingDown } from 'lucide-react'
import clsx from 'clsx'
import { useState } from 'react'
import { useAuth } from '../auth/AuthContext'
import { useReportsData, dateRange, type ReportsPeriod } from '../reports/hooks/useReportsData'
import type { ProductPerformanceMatrixRow, ProductPerformanceQuadrant } from '../reports/types'

const TOOLTIP_STYLE = {
  backgroundColor: '#1F2937',
  border: '1px solid #2D3748',
  borderRadius: '8px',
  fontSize: '11px',
  color: '#F9FAFB',
}

const PERIODS: ReportsPeriod[] = ['7D', '30D', '90D', '12M']
const TOP_PRODUCTS_LIMIT = 5

const QUADRANTS: { id: ProductPerformanceQuadrant; label: string; hint: string }[] = [
  { id: 'stars', label: 'Stars', hint: 'High revenue, low margin — optimize pricing' },
  { id: 'winners', label: 'Winners', hint: 'High revenue, high margin — push harder' },
  { id: 'problems', label: 'Problems', hint: 'Low revenue, low margin — consider discontinuing' },
  { id: 'hidden_gems', label: 'Hidden Gems', hint: 'Low revenue, high margin — promote' },
]

function formatMoney(minorUnits: number, currencyCode: string): string {
  try {
    return new Intl.NumberFormat(undefined, { style: 'currency', currency: currencyCode || 'USD' }).format(minorUnits / 100)
  } catch {
    return `${(minorUnits / 100).toFixed(2)} ${currencyCode}`
  }
}

function ChangeBadge({ percent }: { percent: number | null }) {
  if (percent === null) {
    return <span className="text-xs dark:text-gray-500 text-gray-400">— vs prev period</span>
  }
  const up = percent >= 0
  return (
    <div className={clsx('flex items-center gap-1 text-xs font-medium', up ? 'text-green-400' : 'text-red-400')}>
      {up ? <TrendingUp size={12} /> : <TrendingDown size={12} />}
      <span>{`${up ? '+' : ''}${percent.toFixed(1)}%`}</span>
      <span> vs prev period</span>
    </div>
  )
}

export default function Reports() {
  const { token } = useAuth()
  const [period, setPeriod] = useState<ReportsPeriod>('7D')
  const { salesSummary, productPerformance, productPerformanceMatrix, channelRevenueThisMonth, loading, error } = useReportsData(token ?? '', period)

  const currency = salesSummary?.currency_code ?? 'USD'
  const { from, to } = dateRange(period, new Date())
  const topProducts = [...(productPerformance?.rows ?? [])]
    .sort((a, b) => b.revenue_minor_units - a.revenue_minor_units)
    .slice(0, TOP_PRODUCTS_LIMIT)

  const kpis = [
    { label: 'Total Revenue', value: salesSummary ? formatMoney(salesSummary.current.revenue_minor_units, currency) : '—', change: salesSummary?.revenue_change_percent ?? null },
    { label: 'Total Orders', value: salesSummary ? String(salesSummary.current.order_count) : '—', change: salesSummary?.order_count_change_percent ?? null },
    { label: 'Units Sold', value: salesSummary ? String(salesSummary.current.units_sold) : '—', change: salesSummary?.units_sold_change_percent ?? null },
    { label: 'Avg Order Value', value: salesSummary ? formatMoney(salesSummary.current.average_order_value_minor_units, currency) : '—', change: salesSummary?.aov_change_percent ?? null },
  ]

  return (
    <div className="p-5 space-y-5 overflow-auto h-full">
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold dark:text-gray-100 text-gray-900">
          Analytics &amp; Reports
          <span className="ml-2 text-xs font-normal dark:text-gray-500 text-gray-400">
            {new Date(from).toLocaleDateString()} – {new Date(to).toLocaleDateString()}
          </span>
        </h2>
        <div className="flex items-center gap-0.5 dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 rounded-lg p-0.5">
          {PERIODS.map(p => (
            <button
              key={p}
              onClick={() => setPeriod(p)}
              className={clsx(
                'px-3 py-1 rounded-md text-xs font-medium transition-colors',
                period === p
                  ? 'bg-blue-600 text-white'
                  : 'dark:text-gray-400 text-gray-500 hover:dark:text-gray-200 hover:text-gray-700',
              )}
            >
              {p}
            </button>
          ))}
        </div>
      </div>

      {error && <p className="text-xs text-red-400 bg-red-50 dark:bg-red-500/10 rounded-lg px-3 py-2">{error}</p>}
      {loading && !salesSummary && <p className="text-xs dark:text-gray-500 text-gray-400">Loading…</p>}

      <div className="grid grid-cols-4 gap-3">
        {kpis.map(kpi => (
          <div key={kpi.label} className="dark:bg-[#1F2937] bg-white rounded-xl p-4 border dark:border-[#2D3748] border-gray-200">
            <p className="text-xs dark:text-gray-500 text-gray-400 mb-2">{kpi.label}</p>
            <p className="text-2xl font-semibold dark:text-gray-100 text-gray-900 mb-1">{kpi.value}</p>
            <ChangeBadge percent={kpi.change} />
          </div>
        ))}
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 overflow-hidden">
          <div className="px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
            <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Revenue by Channel</h3>
            <p className="text-xs dark:text-gray-500 text-gray-400 mt-0.5">This month (not affected by the period selector above)</p>
          </div>
          {channelRevenueThisMonth.length === 0 ? (
            <p className="px-4 py-6 text-center text-xs dark:text-gray-500 text-gray-400">No revenue this month</p>
          ) : (
            <>
              <ResponsiveContainer width="100%" height={140}>
                <BarChart data={channelRevenueThisMonth} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#2D3748" />
                  <XAxis dataKey="source" tick={{ fontSize: 10, fill: '#6B7280' }} axisLine={false} tickLine={false} />
                  <YAxis tick={{ fontSize: 10, fill: '#6B7280' }} axisLine={false} tickLine={false} tickFormatter={v => formatMoney(v, currency)} />
                  <Tooltip contentStyle={TOOLTIP_STYLE} formatter={(v: number) => [formatMoney(v, currency), 'Revenue']} />
                  <Bar dataKey="total_minor_units" fill="#2563EB" radius={4} />
                </BarChart>
              </ResponsiveContainer>
              <table className="w-full text-xs border-t dark:border-[#2D3748] border-gray-200">
                <thead>
                  <tr className="dark:bg-[#111827]/50 bg-gray-50">
                    {['Channel', 'Revenue'].map(h => (
                      <th key={h} className="px-3 py-2 text-left text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase">{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y dark:divide-[#2D3748]/50 divide-gray-100">
                  {channelRevenueThisMonth.map(c => (
                    <tr key={c.source} className="hover:dark:bg-[#2D3748]/30 hover:bg-gray-50">
                      <td className="px-3 py-2 dark:text-gray-300 text-gray-700 font-medium">{c.source}</td>
                      <td className="px-3 py-2 dark:text-gray-200 text-gray-800 font-medium">{formatMoney(c.total_minor_units, currency)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </>
          )}
        </div>

        <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 overflow-hidden">
          <div className="px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
            <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Top Products by Revenue</h3>
            <p className="text-xs dark:text-gray-500 text-gray-400 mt-0.5">Selected period</p>
          </div>
          {topProducts.length === 0 ? (
            <p className="px-4 py-6 text-center text-xs dark:text-gray-500 text-gray-400">No sales in this period</p>
          ) : (
            <div className="divide-y dark:divide-[#2D3748]/50 divide-gray-100">
              {topProducts.map(p => (
                <div key={p.sku_id ?? p.sku_code} className="flex items-center gap-3 px-4 py-2.5">
                  <div className="flex-1 min-w-0">
                    <p className="text-xs dark:text-gray-200 text-gray-800 truncate">{p.product_name || p.sku_code}</p>
                    <p className="text-[10px] font-mono dark:text-gray-500 text-gray-400">{p.sku_code}</p>
                  </div>
                  <div className="text-right shrink-0">
                    <p className="text-xs font-medium dark:text-gray-200 text-gray-800">{formatMoney(p.revenue_minor_units, currency)}</p>
                    <p className={clsx('text-[10px] font-medium', p.margin_basis_points === null ? 'dark:text-gray-500 text-gray-400' : p.margin_basis_points >= 5000 ? 'text-green-400' : 'text-amber-400')}>
                      {p.margin_basis_points === null ? 'margin n/a' : `${(p.margin_basis_points / 100).toFixed(1)}%`}
                    </p>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 overflow-hidden">
        <div className="px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
          <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Product Performance Matrix</h3>
          {productPerformanceMatrix && productPerformanceMatrix.rows.length > 0 ? (
            <p className="text-xs dark:text-gray-500 text-gray-400 mt-0.5">
              Split around this period's own median revenue ({formatMoney(productPerformanceMatrix.median_revenue_minor_units, currency)}) and
              median margin ({(productPerformanceMatrix.median_margin_basis_points / 100).toFixed(1)}%)
            </p>
          ) : (
            <p className="text-xs dark:text-gray-500 text-gray-400 mt-0.5">Selected period</p>
          )}
        </div>
        {!productPerformanceMatrix || productPerformanceMatrix.rows.length === 0 ? (
          <p className="px-4 py-6 text-center text-xs dark:text-gray-500 text-gray-400">No SKUs with a known margin in this period</p>
        ) : (
          <div className="grid grid-cols-2 gap-3 p-4">
            {QUADRANTS.map(q => {
              const rows = productPerformanceMatrix.rows.filter(r => r.quadrant === q.id)
              return (
                <div key={q.id} data-testid={`quadrant-${q.id}`} className="rounded-lg border dark:border-[#2D3748] border-gray-200 p-3">
                  <div className="flex items-center justify-between mb-1.5">
                    <p className="text-xs font-semibold dark:text-gray-100 text-gray-900">{q.label}</p>
                    <span className="text-[10px] dark:text-gray-500 text-gray-400">{rows.length}</span>
                  </div>
                  <p className="text-[10px] dark:text-gray-500 text-gray-400 mb-2">{q.hint}</p>
                  {rows.length === 0 ? (
                    <p className="text-[10px] dark:text-gray-600 text-gray-300">None</p>
                  ) : (
                    <div className="space-y-1">
                      {rows.slice(0, 5).map((r: ProductPerformanceMatrixRow) => (
                        <div key={r.sku_id ?? r.sku_code} className="flex items-center justify-between gap-2 text-[11px]">
                          <span className="dark:text-gray-300 text-gray-700 truncate">{r.product_name || r.sku_code}</span>
                          <span className="dark:text-gray-500 text-gray-400 shrink-0">{formatMoney(r.revenue_minor_units, currency)}</span>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}
