import { describe, it, expect } from 'vitest'
import { customDateRange, dateRange } from '../period'

describe('customDateRange', () => {
  it('covers exactly the picked days, inclusive, and says so', () => {
    const r = customDateRange('2026-03-01', '2026-03-31', 'UTC')
    expect(r).toEqual({
      firstDay: '2026-03-01', lastDay: '2026-03-31',
      from: '2026-03-01T00:00:00.000Z', to: '2026-04-01T00:00:00.000Z',
    })
  })

  it("sends the tenant's local midnights, not UTC ones", () => {
    const r = customDateRange('2026-03-01', '2026-03-31', 'Europe/Amsterdam')
    expect(r.from).toBe('2026-02-28T23:00:00.000Z')
    expect(r.to).toBe('2026-03-31T22:00:00.000Z') // DST began on the 29th
    expect(r.firstDay).toBe('2026-03-01')
    expect(r.lastDay).toBe('2026-03-31')
  })

  it('a single picked day is one whole tenant day', () => {
    const r = customDateRange('2026-09-26', '2026-09-26', 'America/New_York')
    expect(r.from).toBe('2026-09-26T04:00:00.000Z')
    expect(r.to).toBe('2026-09-27T04:00:00.000Z')
  })
})

describe('dateRange', () => {
  it('a 7D preset is the last 7 tenant days, today included', () => {
    const r = dateRange('7D', new Date('2026-09-26T10:00:00Z'), 'UTC')
    expect(r).toEqual({
      firstDay: '2026-09-20', lastDay: '2026-09-26',
      from: '2026-09-20T00:00:00.000Z', to: '2026-09-27T00:00:00.000Z',
    })
  })

  it("'today' is the tenant's today, not UTC's", () => {
    // 02:00 UTC on the 26th is still the 25th in New York.
    const r = dateRange('7D', new Date('2026-09-26T02:00:00Z'), 'America/New_York')
    expect(r.firstDay).toBe('2026-09-19')
    expect(r.lastDay).toBe('2026-09-25')
    expect(r.from).toBe('2026-09-19T04:00:00.000Z')
    expect(r.to).toBe('2026-09-26T04:00:00.000Z')
  })

  it.each([['30D', '2026-08-28'], ['90D', '2026-06-29'], ['12M', '2025-09-27']] as const)(
    '%s covers the right number of days', (period, firstDay) => {
      const r = dateRange(period, new Date('2026-09-26T10:00:00Z'), 'UTC')
      expect(r.firstDay).toBe(firstDay)
      expect(r.lastDay).toBe('2026-09-26')
    },
  )
})
