import type { ReactNode } from 'react'
import { TrendingUp, TrendingDown } from 'lucide-react'
import clsx from 'clsx'
import { dateRange, REPORTS_PERIODS, type ReportsPeriod } from '../../reports/period'


export const TOOLTIP_STYLE = {
  backgroundColor: '#1F2937',
  border: '1px solid #2D3748',
  borderRadius: '8px',
  fontSize: '11px',
  color: '#F9FAFB',
}

export function formatMoney(minorUnits: number, currencyCode: string): string {
  try {
    return new Intl.NumberFormat(undefined, { style: 'currency', currency: currencyCode || 'USD' }).format(minorUnits / 100)
  } catch {
    return `${(minorUnits / 100).toFixed(2)} ${currencyCode}`
  }
}

export function ChangeBadge({ percent }: { percent: number | null }) {
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

export function ReportCard({ title, description, children, testId, overflowVisible }: { title: ReactNode; description?: ReactNode; children: ReactNode; testId?: string; overflowVisible?: boolean }) {
  return (
    <div data-testid={testId} className={clsx('dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200', overflowVisible ? 'overflow-visible' : 'overflow-hidden')}>
      <div className="px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
        <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">{title}</h3>
        {description && <p className="text-xs dark:text-gray-500 text-gray-400 mt-0.5">{description}</p>}
      </div>
      {children}
    </div>
  )
}

export function EmptyState({ children }: { children: ReactNode }) {
  return <p className="px-4 py-6 text-center text-xs dark:text-gray-500 text-gray-400">{children}</p>
}

export function PeriodSelector({ period, onChange }: { period: ReportsPeriod; onChange: (p: ReportsPeriod) => void }) {
  return (
    <div className="flex items-center gap-0.5 dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 rounded-lg p-0.5">
      {REPORTS_PERIODS.map(p => (
        <button
          key={p}
          onClick={() => onChange(p)}
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
  )
}

export type ReportsRangeMode = ReportsPeriod | 'custom'

// DateRangeControl is the merged Reports Overview page's range picker --
// the usual period presets plus a "Custom" option that reveals two native
// date inputs. Kept separate from PeriodSelector (which every other report
// page still uses) so this page's custom-range affordance doesn't ripple
// into pages that only need presets.
export function DateRangeControl({
  mode, onModeChange, customFrom, customTo, onCustomFromChange, onCustomToChange,
}: {
  mode: ReportsRangeMode
  onModeChange: (mode: ReportsRangeMode) => void
  customFrom: string
  customTo: string
  onCustomFromChange: (value: string) => void
  onCustomToChange: (value: string) => void
}) {
  return (
    <div className="flex items-center gap-2 flex-wrap">
      <div className="flex items-center gap-0.5 dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 rounded-lg p-0.5">
        {[...REPORTS_PERIODS, 'custom' as const].map(p => (
          <button
            key={p}
            onClick={() => onModeChange(p)}
            className={clsx(
              'px-3 py-1 rounded-md text-xs font-medium capitalize transition-colors',
              mode === p
                ? 'bg-blue-600 text-white'
                : 'dark:text-gray-400 text-gray-500 hover:dark:text-gray-200 hover:text-gray-700',
            )}
          >
            {p === 'custom' ? 'Custom' : p}
          </button>
        ))}
      </div>
      {mode === 'custom' && (
        <div className="flex items-center gap-1.5 text-xs">
          <input
            type="date"
            value={customFrom}
            max={customTo || undefined}
            onChange={e => onCustomFromChange(e.target.value)}
            className="px-2 py-1 rounded-lg border dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-white dark:text-gray-200 text-gray-800 focus:outline-none focus:ring-1 focus:ring-blue-500"
          />
          <span className="dark:text-gray-500 text-gray-400">–</span>
          <input
            type="date"
            value={customTo}
            min={customFrom || undefined}
            onChange={e => onCustomToChange(e.target.value)}
            className="px-2 py-1 rounded-lg border dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-white dark:text-gray-200 text-gray-800 focus:outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
      )}
    </div>
  )
}

// ViewToggle is the merged Reports Overview page's "Summary / Daily"
// switch -- Daily swaps the Profitability and Revenue by Channel cards
// from whole-period aggregates to a per-day time series.
export function ViewToggle({ showDaily, onChange }: { showDaily: boolean; onChange: (showDaily: boolean) => void }) {
  return (
    <div className="flex items-center gap-0.5 dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 rounded-lg p-0.5">
      {[{ value: false, label: 'Summary' }, { value: true, label: 'Daily' }].map(opt => (
        <button
          key={opt.label}
          onClick={() => onChange(opt.value)}
          className={clsx(
            'px-3 py-1 rounded-md text-xs font-medium transition-colors',
            showDaily === opt.value
              ? 'bg-blue-600 text-white'
              : 'dark:text-gray-400 text-gray-500 hover:dark:text-gray-200 hover:text-gray-700',
          )}
        >
          {opt.label}
        </button>
      ))}
    </div>
  )
}

interface ReportPageShellProps {
  title: string
  period?: ReportsPeriod
  onPeriodChange?: (p: ReportsPeriod) => void
  subtitle?: string
  loading?: boolean
  error?: string | null
  children: ReactNode
}

// ReportPageShell is the standard wrapper every report page uses: title +
// date range (or a custom subtitle for non-period reports) + period
// selector, loading/error banners, then the report's own content.
export function ReportPageShell({ title, period, onPeriodChange, subtitle, loading, error, children }: ReportPageShellProps) {
  const range = period ? dateRange(period, new Date()) : null
  return (
    <div className="p-5 space-y-5 overflow-auto h-full">
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold dark:text-gray-100 text-gray-900">
          {title}
          <span className="ml-2 text-xs font-normal dark:text-gray-500 text-gray-400">
            {range ? `${new Date(range.from).toLocaleDateString()} – ${new Date(range.to).toLocaleDateString()}` : subtitle}
          </span>
        </h2>
        {period && onPeriodChange && <PeriodSelector period={period} onChange={onPeriodChange} />}
      </div>
      {error && <p className="text-xs text-red-400 bg-red-50 dark:bg-red-500/10 rounded-lg px-3 py-2">{error}</p>}
      {loading && <p className="text-xs dark:text-gray-500 text-gray-400">Loading…</p>}
      {children}
    </div>
  )
}
