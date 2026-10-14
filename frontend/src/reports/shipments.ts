import { addDays } from '../format/calendarDay'
import type { ShipmentGranularity, ShipmentsPoint } from './types'

export const SHIPMENT_GRANULARITIES = ['week', 'month', 'quarter', 'year'] as const satisfies readonly ShipmentGranularity[]
export const SHIPMENT_BASES = ['label_created', 'carrier_scan'] as const

// How many periods the report shows by default, the current one included.
const DEFAULT_PERIODS: Record<ShipmentGranularity, number> = { week: 12, month: 12, quarter: 8, year: 5 }

function parts(day: string): [number, number, number] {
  const [y, m, d] = day.split('-').map(Number)
  return [y, m, d]
}

function iso(y: number, m: number, d: number): string {
  return new Date(Date.UTC(y, m - 1, d)).toISOString().slice(0, 10)
}

export function quarterOf(day: string): number {
  return Math.floor((parts(day)[1] - 1) / 3) + 1
}

// The first day of the period holding day ("yyyy-mm-dd"). Weeks start on
// Sunday, like the backend's buckets and the dashboard.
export function periodStartDay(day: string, granularity: ShipmentGranularity): string {
  const [y, m, d] = parts(day)
  switch (granularity) {
    case 'week': return addDays(day, -new Date(Date.UTC(y, m - 1, d)).getUTCDay())
    case 'month': return iso(y, m, 1)
    case 'quarter': return iso(y, m - ((m - 1) % 3), 1)
    case 'year': return iso(y, 1, 1)
  }
}

function previousPeriods(start: string, granularity: ShipmentGranularity, n: number): string {
  const [y, m] = parts(start)
  switch (granularity) {
    case 'week': return addDays(start, -7 * n)
    case 'month': return iso(y, m - n, 1)
    case 'quarter': return iso(y, m - 3 * n, 1)
    case 'year': return iso(y - n, 1, 1)
  }
}

// The default range: whole periods ending with the current (running) one.
export function defaultShipmentsDays(granularity: ShipmentGranularity, today: string): { firstDay: string; lastDay: string } {
  const current = periodStartDay(today, granularity)
  return { firstDay: previousPeriods(current, granularity, DEFAULT_PERIODS[granularity] - 1), lastDay: today }
}

export interface ShipmentsPointWithChange extends ShipmentsPoint {
  // % change in shipments against the period before; null for the first
  // period or when the one before had none.
  change_percent: number | null
}

export function withPeriodChange(points: ShipmentsPoint[]): ShipmentsPointWithChange[] {
  return points.map((p, i) => {
    const before = i > 0 ? points[i - 1].shipments : 0
    return { ...p, change_percent: before > 0 ? Math.round(((p.shipments - before) / before) * 1000) / 10 : null }
  })
}
