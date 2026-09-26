import { addDays, calendarDayInZone, zonedMidnight } from '../format/calendarDay'

export const REPORTS_PERIODS = ['7D', '30D', '90D', '12M'] as const
export type ReportsPeriod = typeof REPORTS_PERIODS[number]
export const DEFAULT_REPORTS_PERIOD: ReportsPeriod = '7D'

const PERIOD_DAYS: Record<ReportsPeriod, number> = {
  '7D': 7,
  '30D': 30,
  '90D': 90,
  '12M': 365,
}

// A report range is whole calendar days in the tenant's timezone:
// firstDay..lastDay (both inclusive, "yyyy-mm-dd") is what the page shows,
// and [from, to) -- the instants of the tenant's midnight starting
// firstDay and the one ending lastDay -- is what every report endpoint
// receives. The backend buckets days in the same timezone, so the range
// lines up with its day/week/month buckets. `to` is exclusive and must
// never be shown to the user: it is the day after the range.
export interface ReportRange {
  from: string
  to: string
  firstDay: string
  lastDay: string
}

// dateRange is a preset: the last N tenant days, today included. Reports
// that compare against a prior period (e.g. sales-summary,
// product-lifecycle) do that comparison server-side, so the caller only
// ever needs this one range.
export function dateRange(period: ReportsPeriod, now: Date, timeZone?: string): ReportRange {
  const lastDay = calendarDayInZone(now, timeZone)
  return customDateRange(addDays(lastDay, -(PERIOD_DAYS[period] - 1)), lastDay, timeZone)
}

// customDateRange is a user-picked [fromDate, toDate] calendar-day range
// (both "yyyy-mm-dd", both inclusive, as produced by <input type="date">).
export function customDateRange(fromDate: string, toDate: string, timeZone?: string): ReportRange {
  return {
    from: zonedMidnight(fromDate, timeZone).toISOString(),
    to: zonedMidnight(addDays(toDate, 1), timeZone).toISOString(),
    firstDay: fromDate,
    lastDay: toDate,
  }
}
