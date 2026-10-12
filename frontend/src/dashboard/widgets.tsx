import { Calendar } from 'lucide-react'
import {
  AreaChart, Area, BarChart, Bar, Cell, XAxis, YAxis, CartesianGrid,
  Tooltip, ResponsiveContainer, Legend,
} from 'recharts'
import clsx from 'clsx'
import i18next from 'i18next'
import { useTranslation } from 'react-i18next'
import { channelSeriesColor, seriesColor, useChartTheme } from '../charts/chartTheme'
import { useAuth } from '../auth/AuthContext'
import { useExpiringBatches } from '../inventory/hooks/useExpiringBatches'
import { useDateFormat } from '../format/DateFormatProvider'
import { Money, useMoneyFormat } from '../format/amountVisibility'
import type { SalesOrderSource } from '../oms/types'
import type { ChannelRevenue, DailyStockFlow, TopProduct } from './types'

// Panels shared by the classic Dashboard and the depot Today page.

// Typed over SalesOrderSource so a new source can't ship without a label.
function channelLabels(): Record<SalesOrderSource, string> {
  return {
    pos: 'POS', manual: i18next.t('dashboard:channelManual'), woocommerce: 'WooCommerce', bol: 'Bol.com', whatsapp: 'WhatsApp',
    internal: i18next.t('orders:list.sourceStorefront'),
  }
}

function channelLabel(source: string): string {
  return channelLabels()[source as SalesOrderSource] ?? source
}

export function StockFlowPanel({ flow, className }: { flow: DailyStockFlow[]; className?: string }) {
  const { t } = useTranslation('dashboard')
  const chart = useChartTheme()
  const data = flow.map(f => ({ day: f.date.slice(5), in: f.in, out: f.out }))
  return (
    <div className={clsx('bg-raised rounded-xl p-4 border border-line', className)}>
      <div className="flex items-center justify-between mb-4">
        <div>
          <h3 className="text-sm font-semibold text-ink">{t('stockFlow.title')}</h3>
          <p className="text-xs text-ink-4 mt-0.5">{t('stockFlow.sub')}</p>
        </div>
      </div>
      <ResponsiveContainer width="100%" height={180}>
        <AreaChart data={data} margin={{ top: 0, right: 0, left: -20, bottom: 0 }}>
          <defs>
            <linearGradient id="inGrad" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={seriesColor(chart, 0)} stopOpacity={0.3} />
              <stop offset="95%" stopColor={seriesColor(chart, 0)} stopOpacity={0} />
            </linearGradient>
            <linearGradient id="outGrad" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={seriesColor(chart, 1)} stopOpacity={0.3} />
              <stop offset="95%" stopColor={seriesColor(chart, 1)} stopOpacity={0} />
            </linearGradient>
          </defs>
          <CartesianGrid strokeDasharray="3 3" stroke={chart.grid} vertical={false} />
          <XAxis dataKey="day" tick={{ fontSize: 11, fill: chart.axis }} axisLine={false} tickLine={false} />
          <YAxis tick={{ fontSize: 11, fill: chart.axis }} axisLine={false} tickLine={false} />
          <Tooltip contentStyle={chart.tooltip} />
          <Legend iconSize={10} wrapperStyle={{ fontSize: '11px', color: chart.label }} />
          <Area type="monotone" dataKey="in" name={t('stockFlow.inbound')} stroke={seriesColor(chart, 0)} strokeWidth={2} fill="url(#inGrad)" />
          <Area type="monotone" dataKey="out" name={t('stockFlow.outbound')} stroke={seriesColor(chart, 1)} strokeWidth={2} fill="url(#outGrad)" />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  )
}

export function ChannelSalesPanel({ channels, currency, className }: { channels: ChannelRevenue[]; currency: string; className?: string }) {
  const { t } = useTranslation('dashboard')
  const chart = useChartTheme()
  const money = useMoneyFormat()
  const data = channels.map(c => ({ ...c, label: channelLabel(c.source), color: channelSeriesColor(chart, c.source) }))
  return (
    <div className={clsx('bg-raised rounded-xl p-4 border border-line', className)}>
      <h3 className="text-sm font-semibold text-ink mb-1">{t('channels.title')}</h3>
      <p className="text-xs text-ink-4 mb-4">{t('channels.sub')}</p>
      {data.length === 0 ? (
        <p className="text-xs text-ink-4">{t('channels.empty')}</p>
      ) : (
        <>
          <ResponsiveContainer width="100%" height={100}>
            <BarChart data={data} layout="vertical" margin={{ top: 0, right: 10, left: 0, bottom: 0 }}>
              <XAxis type="number" hide />
              <YAxis type="category" dataKey="label" tick={{ fontSize: 11, fill: chart.axis }} axisLine={false} tickLine={false} width={80} />
              <Tooltip contentStyle={chart.tooltip} formatter={(v: number) => [money(v, currency), t('channels.revenue')]} />
              <Bar dataKey="total_minor_units" radius={4}>
                {data.map(c => <Cell key={c.source} fill={c.color} />)}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
          <div className="mt-3 space-y-2">
            {data.map(c => (
              <div key={c.source} className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <div className="w-2 h-2 rounded-full" style={{ background: c.color }} />
                  <span className="text-xs text-ink-3">{c.label}</span>
                </div>
                <Money minor={c.total_minor_units} currency={currency} className="text-xs font-medium text-ink-2" />
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  )
}

export function TopProductsPanel({ products, currency, className }: { products: TopProduct[]; currency: string; className?: string }) {
  const { t } = useTranslation(['dashboard', 'common'])
  return (
    <div className={clsx('bg-raised rounded-xl border border-line overflow-hidden', className)}>
      <div className="flex items-center justify-between px-4 py-3 border-b border-line">
        <h3 className="text-sm font-semibold text-ink">{t('topProducts.title')}</h3>
      </div>
      {products.length === 0 ? (
        <p className="px-4 py-4 text-xs text-ink-4">{t('topProducts.empty')}</p>
      ) : (
        <table className="w-full">
          <thead>
            <tr className="border-b border-line">
              <th className="text-left text-2xs font-medium text-ink-4 caps-label px-4 py-2">{t('common:fields.product')}</th>
              <th className="text-right text-2xs font-medium text-ink-4 caps-label px-4 py-2">{t('topProducts.sold')}</th>
              <th className="text-right text-2xs font-medium text-ink-4 caps-label px-4 py-2">{t('topProducts.revenue')}</th>
            </tr>
          </thead>
          <tbody>
            {products.map(p => (
              <tr key={p.sku_id ?? p.sku_code} className="border-b dark:border-night-700/50 border-gray-50 last:border-0 hover:bg-hover/30 transition-colors">
                <td className="px-4 py-2.5">
                  <p className="text-xs font-medium text-ink">{p.product_name}</p>
                  <p className="text-2xs text-ink-4 font-mono">{p.sku_code}</p>
                </td>
                <td className="px-4 py-2.5 text-right">
                  <span className="text-xs text-ink-2">{p.quantity_sold}</span>
                </td>
                <td className="px-4 py-2.5 text-right">
                  <Money minor={p.revenue_minor_units} currency={currency} className="text-xs font-medium text-ink" />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}

export function ExpiringSoonWidget() {
  const { formatDate } = useDateFormat()
  const { t } = useTranslation(['dashboard', 'common'])
  const { token } = useAuth()
  const { batches, loading, error } = useExpiringBatches(token ?? '', 30)

  return (
    <div className="bg-raised rounded-xl border border-line overflow-hidden">
      <div className="flex items-center justify-between px-4 py-3 border-b border-line">
        <h3 className="text-sm font-semibold text-ink">{t('expiring.title')}</h3>
        <span className={clsx('text-2xs px-2 py-0.5 rounded-full font-medium', batches.length > 0 ? 'bg-amber-500/20 text-warning' : 'bg-fill text-ink-3')}>
          {t('expiring.within', { n: batches.length })}
        </span>
      </div>
      {loading && <p className="px-4 py-4 text-xs text-ink-4">{t('common:status.loading')}</p>}
      {error && <p className="px-4 py-4 text-xs text-danger">{error}</p>}
      {!loading && !error && batches.length === 0 && (
        <p className="px-4 py-4 text-xs text-ink-4">{t('expiring.empty')}</p>
      )}
      {!loading && !error && batches.length > 0 && (
        <div className="divide-y divide-line max-h-64 overflow-auto">
          {batches.map(b => {
            const expired = new Date(b.expiry_date).getTime() < Date.now()
            return (
              <div key={b.batch_id} className="flex items-start gap-3 px-4 py-3 hover:bg-hover/30 transition-colors">
                <Calendar size={13} className={clsx('mt-0.5 shrink-0', expired ? 'text-danger' : 'text-warning')} />
                <div className="flex-1 min-w-0">
                  <p className="text-xs text-ink-2 truncate">{b.product_name}</p>
                  <p className="text-2xs text-ink-4 font-mono">
                    {t('expiring.lot', { sku: b.sku_code, lot: b.lot_number || '—', warehouse: b.warehouse_name })}
                  </p>
                </div>
                <div className="text-right shrink-0">
                  <p className="text-xs font-medium text-ink">{t('expiring.units', { count: b.quantity })}</p>
                  <p className={clsx('text-2xs font-mono', expired ? 'text-danger' : 'text-warning')}>
                    {formatDate(b.expiry_date)}
                  </p>
                </div>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
