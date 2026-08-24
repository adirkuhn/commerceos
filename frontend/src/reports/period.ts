export type ReportsPeriod = '7D' | '30D' | '90D' | '12M'

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
