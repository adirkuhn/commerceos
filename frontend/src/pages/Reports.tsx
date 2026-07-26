import {
  AreaChart, Area, BarChart, Bar, LineChart, Line,
  XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend,
} from 'recharts'
import { TrendingUp, TrendingDown, Download } from 'lucide-react'
import clsx from 'clsx'
import { useState } from 'react'

const REVENUE = [
  { month: 'Dec', revenue: 84200, orders: 312, cost: 31400 },
  { month: 'Jan', revenue: 91800, orders: 338, cost: 34200 },
  { month: 'Feb', revenue: 88400, orders: 319, cost: 32900 },
  { month: 'Mar', revenue: 97600, orders: 371, cost: 36400 },
  { month: 'Apr', revenue: 107400, orders: 408, cost: 40100 },
  { month: 'May', revenue: 116100, orders: 443, cost: 43300 },
]

const CHANNEL_BREAKDOWN = [
  { channel: 'Shopify', revenue: 48200, orders: 189, returns: 8 },
  { channel: 'WooCommerce', revenue: 31500, orders: 124, returns: 5 },
  { channel: 'Bol.com', revenue: 22100, orders: 86, returns: 3 },
  { channel: 'Lightspeed', revenue: 14300, orders: 44, returns: 1 },
]

const STOCK_TURNS = [
  { month: 'Jan', turns: 3.2 },
  { month: 'Feb', turns: 3.4 },
  { month: 'Mar', turns: 3.8 },
  { month: 'Apr', turns: 4.1 },
  { month: 'May', turns: 4.3 },
]

const TOP_SKU = [
  { sku: 'WHP-001', name: 'Wireless Headphones Pro', units: 234, revenue: 58500, margin: 64.2 },
  { sku: 'WCM-4K', name: 'Webcam 4K Ultra', units: 143, revenue: 42900, margin: 67.3 },
  { sku: 'MKT-TKL', name: 'Mechanical Keyboard TKL', units: 156, revenue: 46800, margin: 58.7 },
  { sku: 'UCH-007', name: 'USB-C Hub 7-in-1', units: 198, revenue: 19800, margin: 68.0 },
  { sku: 'LSP-002', name: 'Laptop Stand Pro', units: 127, revenue: 12700, margin: 71.5 },
]

const TOOLTIP_STYLE = {
  backgroundColor: '#1F2937',
  border: '1px solid #2D3748',
  borderRadius: '8px',
  fontSize: '11px',
  color: '#F9FAFB',
}

const PERIODS = ['7D', '30D', '90D', '12M']

export default function Reports() {
  const [period, setPeriod] = useState('30D')

  const totalRevenue = REVENUE.reduce((s, r) => s + r.revenue, 0)
  const totalOrders = REVENUE.reduce((s, r) => s + r.orders, 0)
  const avgMargin = 65.1

  return (
    <div className="p-5 space-y-5 overflow-auto h-full">
      {/* Period selector */}
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Analytics & Reports</h2>
        <div className="flex items-center gap-2">
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
          <button className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 text-xs dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-100">
            <Download size={13} /> Export
          </button>
        </div>
      </div>

      {/* KPI row */}
      <div className="grid grid-cols-4 gap-3">
        {[
          { label: 'Total Revenue', value: `€${(totalRevenue / 1000).toFixed(0)}k`, change: '+8.3%', up: true },
          { label: 'Total Orders', value: totalOrders.toString(), change: '+12.1%', up: true },
          { label: 'Avg Margin', value: `${avgMargin}%`, change: '+1.2pp', up: true },
          { label: 'Return Rate', value: '4.2%', change: '-0.8pp', up: false },
        ].map(kpi => (
          <div key={kpi.label} className="dark:bg-[#1F2937] bg-white rounded-xl p-4 border dark:border-[#2D3748] border-gray-200">
            <p className="text-xs dark:text-gray-500 text-gray-400 mb-2">{kpi.label}</p>
            <p className="text-2xl font-semibold dark:text-gray-100 text-gray-900 mb-1">{kpi.value}</p>
            <div className={clsx('flex items-center gap-1 text-xs font-medium', kpi.up ? 'text-green-400' : 'text-red-400')}>
              {kpi.up ? <TrendingUp size={12} /> : <TrendingDown size={12} />}
              {kpi.change} vs prev period
            </div>
          </div>
        ))}
      </div>

      {/* Revenue chart */}
      <div className="dark:bg-[#1F2937] bg-white rounded-xl p-4 border dark:border-[#2D3748] border-gray-200">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Revenue vs Cost</h3>
            <p className="text-xs dark:text-gray-500 text-gray-400 mt-0.5">Monthly gross revenue and COGS</p>
          </div>
        </div>
        <ResponsiveContainer width="100%" height={200}>
          <AreaChart data={REVENUE} margin={{ top: 0, right: 0, left: -20, bottom: 0 }}>
            <defs>
              <linearGradient id="revGrad" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#2563EB" stopOpacity={0.3} />
                <stop offset="95%" stopColor="#2563EB" stopOpacity={0} />
              </linearGradient>
              <linearGradient id="costGrad" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#EF4444" stopOpacity={0.2} />
                <stop offset="95%" stopColor="#EF4444" stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke="#2D3748" />
            <XAxis dataKey="month" tick={{ fontSize: 11, fill: '#6B7280' }} axisLine={false} tickLine={false} />
            <YAxis tick={{ fontSize: 11, fill: '#6B7280' }} axisLine={false} tickLine={false} tickFormatter={v => `€${(v / 1000).toFixed(0)}k`} />
            <Tooltip contentStyle={TOOLTIP_STYLE} formatter={(v: number) => [`€${v.toLocaleString()}`, '']} />
            <Legend iconSize={10} wrapperStyle={{ fontSize: '11px' }} />
            <Area type="monotone" dataKey="revenue" name="Revenue" stroke="#2563EB" strokeWidth={2} fill="url(#revGrad)" />
            <Area type="monotone" dataKey="cost" name="COGS" stroke="#EF4444" strokeWidth={2} fill="url(#costGrad)" />
          </AreaChart>
        </ResponsiveContainer>
      </div>

      {/* Two col */}
      <div className="grid grid-cols-2 gap-4">
        {/* Channel breakdown */}
        <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 overflow-hidden">
          <div className="px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
            <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Revenue by Channel</h3>
          </div>
          <ResponsiveContainer width="100%" height={140}>
            <BarChart data={CHANNEL_BREAKDOWN} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#2D3748" />
              <XAxis dataKey="channel" tick={{ fontSize: 10, fill: '#6B7280' }} axisLine={false} tickLine={false} />
              <YAxis tick={{ fontSize: 10, fill: '#6B7280' }} axisLine={false} tickLine={false} tickFormatter={v => `€${(v / 1000).toFixed(0)}k`} />
              <Tooltip contentStyle={TOOLTIP_STYLE} formatter={(v: number) => [`€${v.toLocaleString()}`, 'Revenue']} />
              <Bar dataKey="revenue" fill="#2563EB" radius={4} />
            </BarChart>
          </ResponsiveContainer>
          <table className="w-full text-xs border-t dark:border-[#2D3748] border-gray-200">
            <thead>
              <tr className="dark:bg-[#111827]/50 bg-gray-50">
                {['Channel', 'Orders', 'Revenue', 'Returns'].map(h => (
                  <th key={h} className="px-3 py-2 text-left text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y dark:divide-[#2D3748]/50 divide-gray-100">
              {CHANNEL_BREAKDOWN.map(c => (
                <tr key={c.channel} className="hover:dark:bg-[#2D3748]/30 hover:bg-gray-50">
                  <td className="px-3 py-2 dark:text-gray-300 text-gray-700 font-medium">{c.channel}</td>
                  <td className="px-3 py-2 dark:text-gray-400 text-gray-500">{c.orders}</td>
                  <td className="px-3 py-2 dark:text-gray-200 text-gray-800 font-medium">€{c.revenue.toLocaleString()}</td>
                  <td className="px-3 py-2 dark:text-gray-400 text-gray-500">{c.returns}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Stock turns */}
        <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 overflow-hidden">
          <div className="px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
            <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Inventory Turnover</h3>
            <p className="text-xs dark:text-gray-500 text-gray-400 mt-0.5">Stock turns per month</p>
          </div>
          <ResponsiveContainer width="100%" height={140}>
            <LineChart data={STOCK_TURNS} margin={{ top: 10, right: 20, left: -20, bottom: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#2D3748" />
              <XAxis dataKey="month" tick={{ fontSize: 11, fill: '#6B7280' }} axisLine={false} tickLine={false} />
              <YAxis tick={{ fontSize: 11, fill: '#6B7280' }} axisLine={false} tickLine={false} domain={[2, 5]} />
              <Tooltip contentStyle={TOOLTIP_STYLE} />
              <Line type="monotone" dataKey="turns" stroke="#22C55E" strokeWidth={2} dot={{ fill: '#22C55E', r: 3 }} />
            </LineChart>
          </ResponsiveContainer>

          {/* Top SKUs */}
          <div className="border-t dark:border-[#2D3748] border-gray-200">
            <p className="px-4 py-2 text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Top SKUs by Revenue</p>
            <div className="divide-y dark:divide-[#2D3748]/50 divide-gray-100">
              {TOP_SKU.slice(0, 3).map(s => (
                <div key={s.sku} className="flex items-center gap-3 px-4 py-2">
                  <div className="flex-1 min-w-0">
                    <p className="text-xs dark:text-gray-200 text-gray-800 truncate">{s.name}</p>
                    <p className="text-[10px] font-mono dark:text-gray-500 text-gray-400">{s.sku}</p>
                  </div>
                  <div className="text-right shrink-0">
                    <p className="text-xs font-medium dark:text-gray-200 text-gray-800">€{s.revenue.toLocaleString()}</p>
                    <p className={clsx('text-[10px] font-medium', s.margin >= 65 ? 'text-green-400' : 'text-amber-400')}>
                      {s.margin.toFixed(1)}%
                    </p>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
