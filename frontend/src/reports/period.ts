export const REPORTS_PERIODS = ['7D', '30D', '90D', '12M'] as const
export type ReportsPeriod = typeof REPORTS_PERIODS[number]
export const DEFAULT_REPORTS_PERIOD: ReportsPeriod = '7D'

const PERIOD_DAYS: Record<ReportsPeriod, number> = {
  '7D': 7,
  '30D': 30,
  '90D': 90,
  '12M': 365,
}

// dateRange returns [from, to) for period, ending at "now" -- reports that
// compare against a prior period (e.g. sales-summary, product-lifecycle)
// do that comparison server-side, so the caller only ever needs this one
// range.
export function dateRange(period: ReportsPeriod, now: Date): { from: string; to: string } {
  const to = now
  const from = new Date(now.getTime() - PERIOD_DAYS[period] * 24 * 60 * 60 * 1000)
  return { from: from.toISOString(), to: to.toISOString() }
}

// customDateRange turns a user-picked [fromDate, toDate] calendar-day range
// (both "yyyy-mm-dd", both inclusive, as produced by <input type="date">)
// into the [from, to) ISO range every report endpoint expects -- toDate is
// made exclusive by advancing it one day, so the whole selected end day is
// included.
export function customDateRange(fromDate: string, toDate: string): { from: string; to: string } {
  const from = new Date(`${fromDate}T00:00:00.000Z`)
  const toExclusive = new Date(`${toDate}T00:00:00.000Z`)
  toExclusive.setUTCDate(toExclusive.getUTCDate() + 1)
  return { from: from.toISOString(), to: toExclusive.toISOString() }
}
