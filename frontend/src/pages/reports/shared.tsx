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
    return <span className="text-xs text-ink-4">— vs prev period</span>
  }
  const up = percent >= 0
  return (
    <div className={clsx('flex items-center gap-1 text-xs font-medium', up ? 'text-success' : 'text-danger')}>
      {up ? <TrendingUp size={12} /> : <TrendingDown size={12} />}
      <span>{`${up ? '+' : ''}${percent.toFixed(1)}%`}</span>
      <span> vs prev period</span>
    </div>
  )
}

export function ReportCard({ title, description, children, testId, overflowVisible }: { title: ReactNode; description?: ReactNode; children: ReactNode; testId?: string; overflowVisible?: boolean }) {
  return (
    <div data-testid={testId} className={clsx('bg-raised rounded-xl border border-line', overflowVisible ? 'overflow-visible' : 'overflow-hidden')}>
      <div className="px-4 py-3 border-b border-line">
        <h3 className="text-sm font-semibold text-ink">{title}</h3>
        {description && <p className="text-xs text-ink-4 mt-0.5">{description}</p>}
      </div>
      {children}
    </div>
  )
}

export function EmptyState({ children }: { children: ReactNode }) {
  return <p className="px-4 py-6 text-center text-xs text-ink-4">{children}</p>
}

export function PeriodSelector({ period, onChange }: { period: ReportsPeriod; onChange: (p: ReportsPeriod) => void }) {
  return (
    <div className="flex items-center gap-0.5 bg-raised border border-line rounded-lg p-0.5">
      {REPORTS_PERIODS.map(p => (
        <button aria-pressed={period === p}
          key={p}
          onClick={() => onChange(p)}
          className={clsx(
            'px-3 py-1 rounded-md text-xs font-medium transition-colors',
            period === p
              ? 'bg-accent-solid text-white'
              : 'text-ink-3 hover:text-ink-2',
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
      <div className="flex items-center gap-0.5 bg-raised border border-line rounded-lg p-0.5">
        {[...REPORTS_PERIODS, 'custom' as const].map(p => (
          <button aria-pressed={mode === p}
            key={p}
            onClick={() => onModeChange(p)}
            className={clsx(
              'px-3 py-1 rounded-md text-xs font-medium capitalize transition-colors',
              mode === p
                ? 'bg-accent-solid text-white'
                : 'text-ink-3 hover:text-ink-2',
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
            className="px-2 py-1 rounded-lg border border-line bg-surface text-ink focus:outline-none focus:ring-1 focus:ring-focus"
          />
          <span className="text-ink-4">–</span>
          <input
            type="date"
            value={customTo}
            min={customFrom || undefined}
            onChange={e => onCustomToChange(e.target.value)}
            className="px-2 py-1 rounded-lg border border-line bg-surface text-ink focus:outline-none focus:ring-1 focus:ring-focus"
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
    <div className="flex items-center gap-0.5 bg-raised border border-line rounded-lg p-0.5">
      {[{ value: false, label: 'Summary' }, { value: true, label: 'Daily' }].map(opt => (
        <button aria-pressed={showDaily === opt.value}
          key={opt.label}
          onClick={() => onChange(opt.value)}
          className={clsx(
            'px-3 py-1 rounded-md text-xs font-medium transition-colors',
            showDaily === opt.value
              ? 'bg-accent-solid text-white'
              : 'text-ink-3 hover:text-ink-2',
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
        <h2 className="text-sm font-semibold text-ink">
          {title}
          <span className="ml-2 text-xs font-normal text-ink-4">
            {range ? `${new Date(range.from).toLocaleDateString()} – ${new Date(range.to).toLocaleDateString()}` : subtitle}
          </span>
        </h2>
        {period && onPeriodChange && <PeriodSelector period={period} onChange={onPeriodChange} />}
      </div>
      {error && <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-3 py-2">{error}</p>}
      {loading && <p className="text-xs text-ink-4">Loading…</p>}
      {children}
    </div>
  )
}
