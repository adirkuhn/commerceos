import { describe, it, expect } from 'vitest'
import { defaultShipmentsDays, periodStartDay, withPeriodChange, quarterOf } from '../shipments'

describe('periodStartDay', () => {
  it('starts weeks on Sunday', () => {
    expect(periodStartDay('2026-10-14', 'week')).toBe('2026-10-11') // Wednesday
    expect(periodStartDay('2026-10-11', 'week')).toBe('2026-10-11') // Sunday
    expect(periodStartDay('2026-10-17', 'week')).toBe('2026-10-11') // Saturday
  })

  it('starts months, quarters and years on their first day', () => {
    expect(periodStartDay('2026-08-20', 'month')).toBe('2026-08-01')
    expect(periodStartDay('2026-08-20', 'quarter')).toBe('2026-07-01')
    expect(periodStartDay('2026-12-31', 'quarter')).toBe('2026-10-01')
    expect(periodStartDay('2026-08-20', 'year')).toBe('2026-01-01')
  })
})

describe('defaultShipmentsDays', () => {
  it('covers whole periods up to today: 12 weeks, 12 months, 8 quarters, 5 years', () => {
    expect(defaultShipmentsDays('week', '2026-10-14')).toEqual({ firstDay: '2026-07-26', lastDay: '2026-10-14' })
    expect(defaultShipmentsDays('month', '2026-10-14')).toEqual({ firstDay: '2025-11-01', lastDay: '2026-10-14' })
    expect(defaultShipmentsDays('quarter', '2026-10-14')).toEqual({ firstDay: '2025-01-01', lastDay: '2026-10-14' })
    expect(defaultShipmentsDays('year', '2026-10-14')).toEqual({ firstDay: '2022-01-01', lastDay: '2026-10-14' })
  })
})

describe('quarterOf', () => {
  it('names the quarter of a day', () => {
    expect(quarterOf('2026-01-01')).toBe(1)
    expect(quarterOf('2026-07-01')).toBe(3)
    expect(quarterOf('2026-12-31')).toBe(4)
  })
})

describe('withPeriodChange', () => {
  it('adds the change against the previous period, none for the first or after an empty one', () => {
    const points = [
      { period_start: 'a', shipments: 0, orders: 0 },
      { period_start: 'b', shipments: 10, orders: 9 },
      { period_start: 'c', shipments: 15, orders: 12 },
      { period_start: 'd', shipments: 12, orders: 12 },
    ]
    expect(withPeriodChange(points).map(p => p.change_percent)).toEqual([null, null, 50, -20])
  })
})
