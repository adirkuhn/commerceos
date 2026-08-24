import { useState, useEffect } from 'react'
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts'
import { useAuth } from '../../auth/AuthContext'
import * as dashboardApi from '../../dashboard/api'
import type { ChannelRevenue } from '../../dashboard/types'
import { ReportPageShell, ReportCard, EmptyState, formatMoney, TOOLTIP_STYLE } from './shared'

export default function RevenueByChannelPage() {
  const { token } = useAuth()
  const [channelRevenue, setChannelRevenue] = useState<ChannelRevenue[]>([])
  const [currency, setCurrency] = useState('USD')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!token) return
    let cancelled = false
    setLoading(true)
    setError(null)
    dashboardApi.getDashboardStats(token)
      .then(stats => {
        if (cancelled) return
        setChannelRevenue(stats.sales_by_channel_this_month)
        setCurrency(stats.currency_code)
      })
      .catch(e => { if (!cancelled) setError(e instanceof Error ? e.message : 'Unknown error') })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [token])

  return (
    <ReportPageShell title="Revenue by Channel" subtitle="This month" loading={loading} error={error}>
      <ReportCard title="Revenue by Channel" description="This month (not affected by a date range)">
        {channelRevenue.length === 0 ? (
          <EmptyState>No revenue this month</EmptyState>
        ) : (
          <>
            <ResponsiveContainer width="100%" height={180}>
              <BarChart data={channelRevenue} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
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
                {channelRevenue.map(c => (
                  <tr key={c.source} className="hover:dark:bg-[#2D3748]/30 hover:bg-gray-50">
                    <td className="px-3 py-2 dark:text-gray-300 text-gray-700 font-medium">{c.source}</td>
                    <td className="px-3 py-2 dark:text-gray-200 text-gray-800 font-medium">{formatMoney(c.total_minor_units, currency)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </>
        )}
      </ReportCard>
    </ReportPageShell>
  )
}
