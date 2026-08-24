import { BarChart, Bar, LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts'
import { TrendingUp, TrendingDown, AlertTriangle, Info } from 'lucide-react'
import clsx from 'clsx'
import { useState } from 'react'
import { useAuth } from '../auth/AuthContext'
import { useReportsData, dateRange, type ReportsPeriod } from '../reports/hooks/useReportsData'
import type { ProductPerformanceMatrixRow, ProductPerformanceQuadrant, SalesTrendGranularity, ProductLifecycleStage, RFMSegment } from '../reports/types'

const TOOLTIP_STYLE = {
  backgroundColor: '#1F2937',
  border: '1px solid #2D3748',
  borderRadius: '8px',
  fontSize: '11px',
  color: '#F9FAFB',
}

const PERIODS: ReportsPeriod[] = ['7D', '30D', '90D', '12M']
const TOP_PRODUCTS_LIMIT = 5
const TREND_GRANULARITIES: SalesTrendGranularity[] = ['day', 'week', 'month']
const WEEKDAY_LABELS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']

const QUADRANTS: { id: ProductPerformanceQuadrant; label: string; hint: string }[] = [
  { id: 'stars', label: 'Stars', hint: 'High revenue, low margin — optimize pricing' },
  { id: 'winners', label: 'Winners', hint: 'High revenue, high margin — push harder' },
  { id: 'problems', label: 'Problems', hint: 'Low revenue, low margin — consider discontinuing' },
  { id: 'hidden_gems', label: 'Hidden Gems', hint: 'Low revenue, high margin — promote' },
]

const LIFECYCLE_STAGES: { id: ProductLifecycleStage; label: string; hint: string }[] = [
  { id: 'new', label: 'New', hint: 'Sold now, not in the previous period' },
  { id: 'growing', label: 'Growing', hint: 'Sold more than the previous period' },
  { id: 'stable', label: 'Stable', hint: 'Sold about the same as the previous period' },
  { id: 'declining', label: 'Declining', hint: 'Sold less than the previous period' },
  { id: 'dormant', label: 'Dormant', hint: 'Sold last period, nothing now' },
]

const RFM_SEGMENTS: { id: RFMSegment; label: string; hint: string }[] = [
  { id: 'champions', label: 'Champions', hint: 'Recent, frequent, highest spend' },
  { id: 'loyal_customers', label: 'Loyal Customers', hint: 'Frequent and recent' },
  { id: 'potential_loyalists', label: 'Potential Loyalists', hint: 'Recent, not yet frequent' },
  { id: 'at_risk', label: 'At Risk', hint: 'Used to order often, hasn’t recently' },
  { id: 'lost', label: 'Lost', hint: 'Neither recent nor frequent' },
  { id: 'needs_attention', label: 'Needs Attention', hint: 'Mid-range on every axis' },
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
  const [trendGranularity, setTrendGranularity] = useState<SalesTrendGranularity>('day')
  const { salesSummary, productPerformance, productPerformanceMatrix, customerPerformance, categoryPerformance, slowMovingProducts, profitability, inventorySales, businessInsights, salesTrends, productLifecycle, customerCohorts, customerRFM, frequentlyBoughtTogether, stockoutRisk, channelRevenueThisMonth, loading, error } = useReportsData(token ?? '', period, trendGranularity)

  const currency = salesSummary?.currency_code ?? 'USD'
  const { from, to } = dateRange(period, new Date())
  const topProducts = [...(productPerformance?.rows ?? [])]
    .sort((a, b) => b.revenue_minor_units - a.revenue_minor_units)
    .slice(0, TOP_PRODUCTS_LIMIT)
  const topCustomers = [...(customerPerformance?.rows ?? [])]
    .sort((a, b) => b.revenue_minor_units - a.revenue_minor_units)
    .slice(0, TOP_PRODUCTS_LIMIT)
  const categories = [...(categoryPerformance?.rows ?? [])].sort((a, b) => b.revenue_minor_units - a.revenue_minor_units)
  const slowMovers = [...(slowMovingProducts?.rows ?? [])].sort((a, b) => b.stock_on_hand - a.stock_on_hand)
  const inventoryVsSales = [...(inventorySales?.rows ?? [])].sort((a, b) => (b.turnover_basis_points ?? -1) - (a.turnover_basis_points ?? -1))
  const trendPoints = (salesTrends?.points ?? []).map(p => ({
    ...p,
    label: new Date(p.period_start).toLocaleDateString(undefined, trendGranularity === 'month' ? { month: 'short', year: '2-digit' } : { month: 'short', day: 'numeric' }),
  }))
  const weekdaySales = [...(salesTrends?.by_weekday ?? [])].sort((a, b) => a.weekday - b.weekday)
  const cohorts = [...(customerCohorts?.cohorts ?? [])].sort((a, b) => a.cohort_month.localeCompare(b.cohort_month))
  const maxMonthsSinceCohort = cohorts.reduce((max, c) => c.retention.reduce((m, r) => Math.max(m, r.months_since_cohort), max), 0)
  const cohortMonthColumns = Array.from({ length: maxMonthsSinceCohort + 1 }, (_, i) => i)

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

      {businessInsights && businessInsights.insights.length > 0 && (
        <div data-testid="business-insights" className="space-y-2">
          {businessInsights.insights.map(insight => (
            <div
              key={insight.code}
              className={clsx(
                'flex items-start gap-2 rounded-lg px-3 py-2 text-xs',
                insight.severity === 'warning'
                  ? 'bg-amber-50 dark:bg-amber-500/10 text-amber-700 dark:text-amber-400'
                  : 'bg-blue-50 dark:bg-blue-500/10 text-blue-700 dark:text-blue-400',
              )}
            >
              {insight.severity === 'warning' ? <AlertTriangle size={14} className="shrink-0 mt-0.5" /> : <Info size={14} className="shrink-0 mt-0.5" />}
              <span>{insight.message}</span>
            </div>
          ))}
        </div>
      )}

      <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 overflow-hidden">
        <div className="px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
          <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Profitability</h3>
          <p className="text-xs dark:text-gray-500 text-gray-400 mt-0.5">
            Selected period — profit/margin cover only the revenue with a known SKU cost price
          </p>
        </div>
        {!profitability || profitability.summary.cost_known_revenue_minor_units === 0 ? (
          <p className="px-4 py-6 text-center text-xs dark:text-gray-500 text-gray-400">No cost-known revenue in this period</p>
        ) : (
          <div data-testid="profitability-summary" className="grid grid-cols-4 gap-3 p-4">
            <div>
              <p className="text-xs dark:text-gray-500 text-gray-400 mb-1">Revenue</p>
              <p className="text-sm font-semibold dark:text-gray-100 text-gray-900">{formatMoney(profitability.summary.revenue_minor_units, currency)}</p>
              {profitability.summary.cost_known_revenue_minor_units < profitability.summary.revenue_minor_units && (
                <p className="text-[10px] dark:text-gray-500 text-gray-400 mt-0.5">
                  {formatMoney(profitability.summary.cost_known_revenue_minor_units, currency)} with known cost
                </p>
              )}
            </div>
            <div>
              <p className="text-xs dark:text-gray-500 text-gray-400 mb-1">COGS</p>
              <p className="text-sm font-semibold dark:text-gray-100 text-gray-900">{formatMoney(profitability.summary.cost_minor_units, currency)}</p>
            </div>
            <div>
              <p className="text-xs dark:text-gray-500 text-gray-400 mb-1">Profit</p>
              <p className="text-sm font-semibold dark:text-gray-100 text-gray-900">{formatMoney(profitability.summary.profit_minor_units, currency)}</p>
            </div>
            <div>
              <p className="text-xs dark:text-gray-500 text-gray-400 mb-1">Margin</p>
              <p className="text-sm font-semibold dark:text-gray-100 text-gray-900">{(profitability.summary.margin_basis_points / 100).toFixed(1)}%</p>
            </div>
          </div>
        )}
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

      <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 overflow-hidden">
        <div className="px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
          <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Top Customers by Revenue</h3>
          <p className="text-xs dark:text-gray-500 text-gray-400 mt-0.5">Selected period — guest orders excluded</p>
        </div>
        {topCustomers.length === 0 ? (
          <p className="px-4 py-6 text-center text-xs dark:text-gray-500 text-gray-400">No customer orders in this period</p>
        ) : (
          <div className="divide-y dark:divide-[#2D3748]/50 divide-gray-100">
            {topCustomers.map(c => (
              <div key={c.customer_id} className="flex items-center gap-3 px-4 py-2.5">
                <div className="flex-1 min-w-0">
                  <p className="text-xs dark:text-gray-200 text-gray-800 truncate">{c.customer_name || 'Unnamed customer'}</p>
                  <p className="text-[10px] dark:text-gray-500 text-gray-400">
                    {c.order_count} order{c.order_count === 1 ? '' : 's'} · {formatMoney(c.average_order_value_minor_units, currency)} AOV
                  </p>
                </div>
                <p className="text-xs font-medium dark:text-gray-200 text-gray-800 shrink-0">{formatMoney(c.revenue_minor_units, currency)}</p>
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 overflow-hidden">
        <div className="px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
          <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Revenue by Category</h3>
          <p className="text-xs dark:text-gray-500 text-gray-400 mt-0.5">Selected period</p>
        </div>
        {categories.length === 0 ? (
          <p className="px-4 py-6 text-center text-xs dark:text-gray-500 text-gray-400">No category sales in this period</p>
        ) : (
          <table className="w-full text-xs">
            <thead>
              <tr className="dark:bg-[#111827]/50 bg-gray-50">
                {['Category', 'Units', 'Orders', 'Revenue'].map(h => (
                  <th key={h} className="px-4 py-2 text-left text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y dark:divide-[#2D3748]/50 divide-gray-100">
              {categories.map(cat => (
                <tr key={cat.category_id ?? 'uncategorized'} className="hover:dark:bg-[#2D3748]/30 hover:bg-gray-50">
                  <td className="px-4 py-2 dark:text-gray-200 text-gray-800 font-medium">{cat.category_name}</td>
                  <td className="px-4 py-2 dark:text-gray-400 text-gray-500">{cat.quantity_sold}</td>
                  <td className="px-4 py-2 dark:text-gray-400 text-gray-500">{cat.order_count}</td>
                  <td className="px-4 py-2 dark:text-gray-200 text-gray-800 font-medium">{formatMoney(cat.revenue_minor_units, currency)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 overflow-hidden">
        <div className="px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
          <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Slow-Moving Products</h3>
          <p className="text-xs dark:text-gray-500 text-gray-400 mt-0.5">In stock with zero units sold in the selected period</p>
        </div>
        {slowMovers.length === 0 ? (
          <p className="px-4 py-6 text-center text-xs dark:text-gray-500 text-gray-400">No slow-moving products in this period</p>
        ) : (
          <table className="w-full text-xs">
            <thead>
              <tr className="dark:bg-[#111827]/50 bg-gray-50">
                {['Product', 'SKU', 'Stock on Hand'].map(h => (
                  <th key={h} className="px-4 py-2 text-left text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y dark:divide-[#2D3748]/50 divide-gray-100">
              {slowMovers.map(p => (
                <tr key={p.sku_id} className="hover:dark:bg-[#2D3748]/30 hover:bg-gray-50">
                  <td className="px-4 py-2 dark:text-gray-200 text-gray-800 font-medium">{p.product_name || p.sku_code}</td>
                  <td className="px-4 py-2 font-mono dark:text-gray-500 text-gray-400">{p.sku_code}</td>
                  <td className="px-4 py-2 dark:text-amber-400 text-amber-600 font-medium">{p.stock_on_hand}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 overflow-hidden">
        <div className="px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
          <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Inventory + Sales</h3>
          <p className="text-xs dark:text-gray-500 text-gray-400 mt-0.5">Stock on hand vs. units sold in the selected period, sorted by turnover</p>
        </div>
        {inventoryVsSales.length === 0 ? (
          <p className="px-4 py-6 text-center text-xs dark:text-gray-500 text-gray-400">No stocked or sold SKUs in this period</p>
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
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 overflow-hidden">
          <div className="px-4 py-3 border-b dark:border-[#2D3748] border-gray-200 flex items-center justify-between">
            <div>
              <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Sales Trend</h3>
              <p className="text-xs dark:text-gray-500 text-gray-400 mt-0.5">Revenue over the selected period</p>
            </div>
            <div className="flex items-center gap-0.5 dark:bg-[#111827] bg-gray-50 border dark:border-[#2D3748] border-gray-200 rounded-lg p-0.5">
              {TREND_GRANULARITIES.map(g => (
                <button
                  key={g}
                  onClick={() => setTrendGranularity(g)}
                  className={clsx(
                    'px-2 py-1 rounded-md text-[10px] font-medium capitalize transition-colors',
                    trendGranularity === g
                      ? 'bg-blue-600 text-white'
                      : 'dark:text-gray-400 text-gray-500 hover:dark:text-gray-200 hover:text-gray-700',
                  )}
                >
                  {g}
                </button>
              ))}
            </div>
          </div>
          {trendPoints.length === 0 ? (
            <p className="px-4 py-6 text-center text-xs dark:text-gray-500 text-gray-400">No sales in this period</p>
          ) : (
            <ResponsiveContainer width="100%" height={220}>
              <LineChart data={trendPoints} margin={{ top: 10, right: 16, left: -20, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#2D3748" />
                <XAxis dataKey="label" tick={{ fontSize: 10, fill: '#6B7280' }} axisLine={false} tickLine={false} />
                <YAxis tick={{ fontSize: 10, fill: '#6B7280' }} axisLine={false} tickLine={false} tickFormatter={v => formatMoney(v, currency)} />
                <Tooltip contentStyle={TOOLTIP_STYLE} formatter={(v: number) => [formatMoney(v, currency), 'Revenue']} />
                <Line type="monotone" dataKey="revenue_minor_units" stroke="#2563EB" strokeWidth={2} dot={false} />
              </LineChart>
            </ResponsiveContainer>
          )}
        </div>

        <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 overflow-hidden">
          <div className="px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
            <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Seasonality by Weekday</h3>
            <p className="text-xs dark:text-gray-500 text-gray-400 mt-0.5">Revenue summed across every occurrence of each weekday</p>
          </div>
          {weekdaySales.length === 0 ? (
            <p className="px-4 py-6 text-center text-xs dark:text-gray-500 text-gray-400">No sales to break down by weekday in this period</p>
          ) : (
            <ResponsiveContainer width="100%" height={220}>
              <BarChart data={weekdaySales.map(w => ({ ...w, label: WEEKDAY_LABELS[w.weekday] }))} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#2D3748" />
                <XAxis dataKey="label" tick={{ fontSize: 10, fill: '#6B7280' }} axisLine={false} tickLine={false} />
                <YAxis tick={{ fontSize: 10, fill: '#6B7280' }} axisLine={false} tickLine={false} tickFormatter={v => formatMoney(v, currency)} />
                <Tooltip contentStyle={TOOLTIP_STYLE} formatter={(v: number) => [formatMoney(v, currency), 'Revenue']} />
                <Bar dataKey="revenue_minor_units" fill="#2563EB" radius={4} />
              </BarChart>
            </ResponsiveContainer>
          )}
        </div>
      </div>

      <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 overflow-hidden">
        <div className="px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
          <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Product Lifecycle</h3>
          <p className="text-xs dark:text-gray-500 text-gray-400 mt-0.5">Compared against the immediately preceding period of equal length</p>
        </div>
        {!productLifecycle || productLifecycle.rows.length === 0 ? (
          <p className="px-4 py-6 text-center text-xs dark:text-gray-500 text-gray-400">No sales in this or the previous period</p>
        ) : (
          <div className="grid grid-cols-3 gap-3 p-4">
            {LIFECYCLE_STAGES.map(stage => {
              const rows = productLifecycle.rows.filter(r => r.stage === stage.id)
              return (
                <div key={stage.id} data-testid={`lifecycle-${stage.id}`} className="rounded-lg border dark:border-[#2D3748] border-gray-200 p-3">
                  <div className="flex items-center justify-between mb-1.5">
                    <p className="text-xs font-semibold dark:text-gray-100 text-gray-900">{stage.label}</p>
                    <span className="text-[10px] dark:text-gray-500 text-gray-400">{rows.length}</span>
                  </div>
                  <p className="text-[10px] dark:text-gray-500 text-gray-400 mb-2">{stage.hint}</p>
                  {rows.length === 0 ? (
                    <p className="text-[10px] dark:text-gray-600 text-gray-300">None</p>
                  ) : (
                    <div className="space-y-1">
                      {rows.slice(0, 5).map(r => (
                        <div key={r.sku_id ?? r.sku_code} className="flex items-center justify-between gap-2 text-[11px]">
                          <span className="dark:text-gray-300 text-gray-700 truncate">{r.product_name || r.sku_code}</span>
                          <span className="dark:text-gray-500 text-gray-400 shrink-0">{r.current_quantity_sold} sold</span>
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

      <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 overflow-hidden">
        <div className="px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
          <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Customer Cohorts</h3>
          <p className="text-xs dark:text-gray-500 text-gray-400 mt-0.5">
            Cohorts by month of each customer's first-ever order in this period, retention by month since acquisition
          </p>
        </div>
        {cohorts.length === 0 ? (
          <p className="px-4 py-6 text-center text-xs dark:text-gray-500 text-gray-400">No new customers acquired in this period</p>
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
      </div>

      <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 overflow-hidden">
        <div className="px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
          <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Customer Segments (RFM)</h3>
          <p className="text-xs dark:text-gray-500 text-gray-400 mt-0.5">
            Recency/Frequency/Monetary quintiles, relative to this period's own customers — guest orders excluded
          </p>
        </div>
        {!customerRFM || customerRFM.rows.length === 0 ? (
          <p className="px-4 py-6 text-center text-xs dark:text-gray-500 text-gray-400">No customer orders in this period</p>
        ) : (
          <div className="grid grid-cols-3 gap-3 p-4">
            {RFM_SEGMENTS.map(segment => {
              const rows = customerRFM.rows.filter(r => r.segment === segment.id)
              return (
                <div key={segment.id} data-testid={`rfm-${segment.id}`} className="rounded-lg border dark:border-[#2D3748] border-gray-200 p-3">
                  <div className="flex items-center justify-between mb-1.5">
                    <p className="text-xs font-semibold dark:text-gray-100 text-gray-900">{segment.label}</p>
                    <span className="text-[10px] dark:text-gray-500 text-gray-400">{rows.length}</span>
                  </div>
                  <p className="text-[10px] dark:text-gray-500 text-gray-400 mb-2">{segment.hint}</p>
                  {rows.length === 0 ? (
                    <p className="text-[10px] dark:text-gray-600 text-gray-300">None</p>
                  ) : (
                    <div className="space-y-1">
                      {rows.slice(0, 5).map(r => (
                        <div key={r.customer_id} className="flex items-center justify-between gap-2 text-[11px]">
                          <span className="dark:text-gray-300 text-gray-700 truncate">{r.customer_name || 'Unnamed customer'}</span>
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

      <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 overflow-hidden">
        <div className="px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
          <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Frequently Bought Together</h3>
          <p className="text-xs dark:text-gray-500 text-gray-400 mt-0.5">
            SKU sets that co-occurred most often in orders this period — support is the number of qualifying orders containing every item in the set
          </p>
        </div>
        {!frequentlyBoughtTogether || frequentlyBoughtTogether.rows.length === 0 ? (
          <p className="px-4 py-6 text-center text-xs dark:text-gray-500 text-gray-400">No frequently co-occurring SKU sets in this period</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr className="dark:bg-[#111827]/50 bg-gray-50">
                  <th className="px-4 py-2 text-left text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase">Items</th>
                  <th className="px-4 py-2 text-right text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase">Support</th>
                  <th className="px-4 py-2 text-right text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase">% of Orders</th>
                </tr>
              </thead>
              <tbody className="divide-y dark:divide-[#2D3748]/50 divide-gray-100">
                {frequentlyBoughtTogether.rows.map((row, i) => (
                  <tr key={i} data-testid="fbt-row" className="hover:dark:bg-[#2D3748]/30 hover:bg-gray-50">
                    <td className="px-4 py-2 dark:text-gray-200 text-gray-800">
                      {row.items.map(item => item.product_name || item.sku_code || item.sku_id).join(' + ')}
                    </td>
                    <td className="px-4 py-2 text-right dark:text-gray-400 text-gray-500">{row.support}</td>
                    <td className="px-4 py-2 text-right dark:text-gray-400 text-gray-500">{row.support_percent.toFixed(1)}%</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 overflow-hidden">
        <div className="px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
          <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Stockout Risk</h3>
          <p className="text-xs dark:text-gray-500 text-gray-400 mt-0.5">
            SKUs projected to run out before a reorder placed today (14-day lead time) could arrive, ranked soonest-first
          </p>
        </div>
        {(() => {
          const atRisk = (stockoutRisk?.rows ?? [])
            .filter(r => r.at_risk)
            .sort((a, b) => (a.days_of_stock_remaining ?? 0) - (b.days_of_stock_remaining ?? 0))
          if (atRisk.length === 0) {
            return <p className="px-4 py-6 text-center text-xs dark:text-gray-500 text-gray-400">No SKUs at risk of stocking out</p>
          }
          return (
            <div className="overflow-x-auto">
              <table className="w-full text-xs">
                <thead>
                  <tr className="dark:bg-[#111827]/50 bg-gray-50">
                    <th className="px-4 py-2 text-left text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase">Product</th>
                    <th className="px-4 py-2 text-right text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase">Stock</th>
                    <th className="px-4 py-2 text-right text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase">Days Left</th>
                    <th className="px-4 py-2 text-right text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase">Suggested Reorder</th>
                  </tr>
                </thead>
                <tbody className="divide-y dark:divide-[#2D3748]/50 divide-gray-100">
                  {atRisk.map(row => (
                    <tr key={row.sku_id} data-testid="stockout-risk-row" className="hover:dark:bg-[#2D3748]/30 hover:bg-gray-50">
                      <td className="px-4 py-2 dark:text-gray-200 text-gray-800">{row.product_name || row.sku_code}</td>
                      <td className="px-4 py-2 text-right dark:text-gray-400 text-gray-500">{row.stock_on_hand}</td>
                      <td className="px-4 py-2 text-right text-red-500 font-medium">{row.days_of_stock_remaining?.toFixed(1) ?? '—'}</td>
                      <td className="px-4 py-2 text-right dark:text-gray-400 text-gray-500">{row.suggested_reorder_quantity}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )
        })()}
      </div>
    </div>
  )
}
