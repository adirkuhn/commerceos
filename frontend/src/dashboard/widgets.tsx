import {
  AreaChart, Area, XAxis, YAxis, CartesianGrid,
  Tooltip, ResponsiveContainer, Legend,
} from 'recharts'
import clsx from 'clsx'
import i18next from 'i18next'
import { useTranslation } from 'react-i18next'
import { seriesColor, useChartTheme } from '../charts/chartTheme'
import { Money } from '../format/amountVisibility'
import type { SalesOrderSource } from '../oms/types'
import type { DailyStockFlow, TopProduct } from './types'

// Panels on the Today page.

// Typed over SalesOrderSource so a new source can't ship without a label.
function channelLabels(): Record<SalesOrderSource, string> {
  return {
    pos: 'POS', manual: i18next.t('dashboard:channelManual'), woocommerce: 'WooCommerce', bol: 'Bol.com', whatsapp: 'WhatsApp',
    internal: i18next.t('orders:list.sourceStorefront'),
  }
}

export function channelLabel(source: string): string {
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
