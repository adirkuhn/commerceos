import { useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts'
import clsx from 'clsx'
import { seriesColor, useChartTheme } from '../../charts/chartTheme'
import { useAuth } from '../../auth/AuthContext'
import * as reportsApi from '../../reports/api'
import { useReport } from '../../reports/hooks/useSingleReport'
import { customDateRange } from '../../reports/period'
import { SHIPMENT_BASES, SHIPMENT_GRANULARITIES, defaultShipmentsDays, quarterOf, withPeriodChange } from '../../reports/shipments'
import type { ShipmentBasis, ShipmentGranularity, ShipmentsShare } from '../../reports/types'
import { ReportPageShell, ReportCard, EmptyState } from './shared'
import { useQueryEnum, useQueryString } from '../../routing/useQueryState'
import { useDateFormat } from '../../format/DateFormatProvider'
import { calendarDayInZone } from '../../format/calendarDay'
import { channelLabel } from '../../dashboard/widgets'

const ISO_DAY = /^\d{4}-\d{2}-\d{2}$/
const isIsoDay = (v: string | null): v is string => v !== null && ISO_DAY.test(v)

interface SegmentedProps<T extends string> {
  label: string
  options: readonly T[]
  value: T
  onChange: (value: T) => void
  optionLabel: (value: T) => string
}

function Segmented<T extends string>({ label, options, value, onChange, optionLabel }: SegmentedProps<T>) {
  return (
    <div role="group" aria-label={label} className="flex items-center gap-0.5 bg-raised border border-line rounded-lg p-0.5">
      {options.map(o => (
        <button
          key={o} type="button" aria-pressed={value === o} onClick={() => onChange(o)}
          className={clsx('px-3 py-1 rounded-md text-xs font-medium transition-colors', value === o ? 'bg-accent-solid text-white' : 'text-ink-3 hover:text-ink-2')}
        >
          {optionLabel(o)}
        </button>
      ))}
    </div>
  )
}

function ShareList({ rows, total, label }: { rows: { key: string; label: string; shipments: number }[]; total: number; label: string }) {
  const { t } = useTranslation('reports')
  if (rows.length === 0) return <EmptyState>{t('shipments.empty')}</EmptyState>
  return (
    <ul aria-label={label} className="space-y-2">
      {rows.map(r => {
        const share = total > 0 ? (r.shipments / total) * 100 : 0
        return (
          <li key={r.key} className="text-xs">
            <div className="flex justify-between gap-2">
              <span className="text-ink truncate">{r.label}</span>
              <span className="tabular-nums text-ink-2">{r.shipments} <span className="text-ink-4">· {share.toFixed(0)}%</span></span>
            </div>
            <div className="h-1.5 mt-1 rounded-full bg-sunken overflow-hidden">
              <div className="h-full rounded-full bg-accent-solid" style={{ width: `${share}%` }} />
            </div>
          </li>
        )
      })}
    </ul>
  )
}

// Shipments per week/month/quarter/year: how many labels went out (or
// were first scanned by the carrier) per period, and where they went.
export default function ShipmentsPage() {
  const chart = useChartTheme()
  const { token } = useAuth()
  const { t, i18n } = useTranslation('reports')
  const { formatDate, formatMonthDay, formatMonthYear, settings: { timeZone } } = useDateFormat()
  // Everything lives in the URL: a refresh or a shared link shows the same
  // report. A new grouping drops a picked range for that grouping's default.
  const [granularity, setGranularity] = useQueryEnum<ShipmentGranularity>('granularity', SHIPMENT_GRANULARITIES, 'week', { resets: ['from', 'to'] })
  const [basis, setBasis] = useQueryEnum<ShipmentBasis>('basis', SHIPMENT_BASES, 'label_created')
  const [fromParam, setFrom] = useQueryString('from')
  const [toParam, setTo] = useQueryString('to')

  const today = calendarDayInZone(new Date(), timeZone)
  const defaults = defaultShipmentsDays(granularity, today)
  const firstDay = isIsoDay(fromParam) ? fromParam : defaults.firstDay
  const lastDay = isIsoDay(toParam) ? toParam : defaults.lastDay
  const { from, to } = useMemo(() => customDateRange(firstDay, lastDay, timeZone), [firstDay, lastDay, timeZone])

  const fetchReport = useCallback(
    (tok: string, f: string, t2: string) => reportsApi.getShipmentsReport(tok, f, t2, granularity, basis),
    [granularity, basis],
  )
  const { data, loading, error } = useReport(token ?? '', from, to, fetchReport)

  const periodLabel = useCallback((periodStart: string) => {
    const day = calendarDayInZone(new Date(periodStart), timeZone)
    switch (granularity) {
      case 'week': return t('shipments.weekOf', { date: formatMonthDay(periodStart) })
      case 'month': return formatMonthYear(periodStart)
      case 'quarter': return t('shipments.quarter', { quarter: quarterOf(day), year: day.slice(0, 4) })
      case 'year': return day.slice(0, 4)
    }
  }, [granularity, timeZone, t, formatMonthDay, formatMonthYear])

  const points = withPeriodChange(data?.points ?? []).map(p => ({ ...p, label: periodLabel(p.period_start) }))
  const total = data?.total_shipments ?? 0
  const countryName = useMemo(() => {
    try {
      const names = new Intl.DisplayNames([i18n.language], { type: 'region' })
      return (code: string) => (code ? names.of(code) ?? code : t('shipments.unknown'))
    } catch {
      return (code: string) => code || t('shipments.unknown')
    }
  }, [i18n.language, t])
  const shares = (rows: ShipmentsShare[], name: (key: string) => string) => rows.map(r => ({ key: r.key, label: name(r.key), shipments: r.shipments }))

  const granularityLabel = (g: ShipmentGranularity) => t(`shipments.granularity.${g}`)
  const basisLabel = (b: ShipmentBasis) => t(`shipments.basis.${b}`)

  return (
    <ReportPageShell title={t('shipments.title')} subtitle={`${formatDate(firstDay)} – ${formatDate(lastDay)}`} loading={loading} error={error}>
      <div className="flex items-center gap-2 flex-wrap">
        <Segmented label={t('shipments.groupBy')} options={SHIPMENT_GRANULARITIES} value={granularity} onChange={setGranularity} optionLabel={granularityLabel} />
        <Segmented label={t('shipments.countBy')} options={SHIPMENT_BASES} value={basis} onChange={setBasis} optionLabel={basisLabel} />
        <div className="flex items-center gap-1.5 text-xs">
          <input
            type="date" aria-label={t('shipments.from')} value={firstDay} max={lastDay} onChange={e => setFrom(e.target.value || null)}
            className="px-2 py-1 rounded-lg border border-line bg-surface text-ink focus:outline-none focus:ring-1 focus:ring-focus"
          />
          <span className="text-ink-4">–</span>
          <input
            type="date" aria-label={t('shipments.to')} value={lastDay} min={firstDay} onChange={e => setTo(e.target.value || null)}
            className="px-2 py-1 rounded-lg border border-line bg-surface text-ink focus:outline-none focus:ring-1 focus:ring-focus"
          />
        </div>
      </div>
      <p className="text-2xs text-ink-4">{t(`shipments.basisHint.${basis}`)}</p>

      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <ReportCard title={t('shipments.totalShipments')}><p className="text-2xl font-semibold tabular-nums" data-testid="total-shipments">{total}</p></ReportCard>
        <ReportCard title={t('shipments.totalOrders')} description={t('shipments.totalOrdersDesc')}><p className="text-2xl font-semibold tabular-nums">{data?.total_orders ?? 0}</p></ReportCard>
        <ReportCard title={t(`shipments.average.${granularity}`)}>
          <p className="text-2xl font-semibold tabular-nums">{points.length > 0 ? (total / points.length).toFixed(1) : '0'}</p>
        </ReportCard>
      </div>

      <ReportCard title={t(`shipments.per.${granularity}`)}>
        {total === 0 ? (
          <EmptyState>{t('shipments.empty')}</EmptyState>
        ) : (
          <ResponsiveContainer width="100%" height={260}>
            <BarChart data={points} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke={chart.grid} vertical={false} />
              <XAxis dataKey="label" tick={{ fontSize: 11, fill: chart.axis }} axisLine={false} tickLine={false} />
              <YAxis allowDecimals={false} tick={{ fontSize: 11, fill: chart.axis }} axisLine={false} tickLine={false} />
              <Tooltip contentStyle={chart.tooltip} formatter={(v: number) => [v, t('shipments.col.shipments')]} />
              <Bar dataKey="shipments" fill={seriesColor(chart, 0)} radius={4} />
            </BarChart>
          </ResponsiveContainer>
        )}
      </ReportCard>

      <ReportCard title={t('shipments.table')}>
        <div className="overflow-x-auto">
          <table className="w-full text-xs">
            <thead>
              <tr className="text-left text-ink-4">
                <th className="py-1.5 font-medium">{t('shipments.col.period')}</th>
                <th className="py-1.5 font-medium text-right">{t('shipments.col.shipments')}</th>
                <th className="py-1.5 font-medium text-right">{t('shipments.col.orders')}</th>
                <th className="py-1.5 font-medium text-right">{t('shipments.col.change')}</th>
              </tr>
            </thead>
            <tbody>
              {[...points].reverse().map(p => (
                <tr key={p.period_start} className="border-t border-line">
                  <td className="py-1.5 text-ink">{p.label}</td>
                  <td className="py-1.5 text-right tabular-nums">{p.shipments}</td>
                  <td className="py-1.5 text-right tabular-nums text-ink-2">{p.orders}</td>
                  <td className={clsx('py-1.5 text-right tabular-nums', p.change_percent === null ? 'text-ink-4' : p.change_percent >= 0 ? 'text-success' : 'text-danger')}>
                    {p.change_percent === null ? '—' : `${p.change_percent >= 0 ? '+' : ''}${p.change_percent.toFixed(1)}%`}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </ReportCard>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <ReportCard title={t('shipments.byCarrier')}>
          <ShareList label={t('shipments.byCarrier')} total={total} rows={(data?.by_carrier ?? []).map(c => ({ key: String(c.carrier), label: c.name, shipments: c.shipments }))} />
        </ReportCard>
        <ReportCard title={t('shipments.byChannel')}>
          <ShareList label={t('shipments.byChannel')} total={total} rows={shares(data?.by_channel ?? [], channelLabel)} />
        </ReportCard>
        <ReportCard title={t('shipments.byCountry')}>
          <ShareList label={t('shipments.byCountry')} total={total} rows={shares(data?.by_country ?? [], countryName)} />
        </ReportCard>
      </div>
    </ReportPageShell>
  )
}
