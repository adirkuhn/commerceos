import { describe, it, expect } from 'vitest'
import { addDays, calendarDayInZone, zonedMidnight } from '../calendarDay'

describe('calendarDayInZone', () => {
  it("is the instant's calendar day in the given zone", () => {
    // 23:30 UTC on the 25th is already the 26th in Amsterdam, still the 25th in New York.
    expect(calendarDayInZone(new Date('2027-11-25T23:30:00Z'), 'Europe/Amsterdam')).toBe('2027-11-26')
    expect(calendarDayInZone(new Date('2027-11-25T23:30:00Z'), 'UTC')).toBe('2027-11-25')
    expect(calendarDayInZone(new Date('2027-11-26T03:00:00Z'), 'America/New_York')).toBe('2027-11-25')
  })
})

describe('zonedMidnight', () => {
  it('is 00:00 local time in the given zone', () => {
    expect(zonedMidnight('2026-03-01', 'UTC').toISOString()).toBe('2026-03-01T00:00:00.000Z')
    expect(zonedMidnight('2026-03-01', 'Europe/Amsterdam').toISOString()).toBe('2026-02-28T23:00:00.000Z')
    expect(zonedMidnight('2026-07-01', 'Europe/Amsterdam').toISOString()).toBe('2026-06-30T22:00:00.000Z')
    expect(zonedMidnight('2026-03-01', 'America/New_York').toISOString()).toBe('2026-03-01T05:00:00.000Z')
    expect(zonedMidnight('2026-03-01', 'Asia/Kolkata').toISOString()).toBe('2026-02-28T18:30:00.000Z')
  })

  it('uses the offset in force at that midnight across a DST change', () => {
    // Amsterdam springs forward on 2026-03-29 at 02:00 -- that midnight is still UTC+1,
    // the next one is UTC+2.
    expect(zonedMidnight('2026-03-29', 'Europe/Amsterdam').toISOString()).toBe('2026-03-28T23:00:00.000Z')
    expect(zonedMidnight('2026-03-30', 'Europe/Amsterdam').toISOString()).toBe('2026-03-29T22:00:00.000Z')
    // New York falls back on 2026-11-01 at 02:00.
    expect(zonedMidnight('2026-11-01', 'America/New_York').toISOString()).toBe('2026-11-01T04:00:00.000Z')
    expect(zonedMidnight('2026-11-02', 'America/New_York').toISOString()).toBe('2026-11-02T05:00:00.000Z')
  })

  it('round-trips: the midnight of a day is on that day in the zone', () => {
    for (const zone of ['UTC', 'Europe/Amsterdam', 'America/Los_Angeles', 'Pacific/Kiritimati', 'Pacific/Pago_Pago']) {
      expect(calendarDayInZone(zonedMidnight('2026-09-26', zone), zone)).toBe('2026-09-26')
    }
  })
})

describe('addDays', () => {
  it('moves a calendar day across month and year ends', () => {
    expect(addDays('2026-01-31', 1)).toBe('2026-02-01')
    expect(addDays('2026-12-31', 1)).toBe('2027-01-01')
    expect(addDays('2026-03-01', -1)).toBe('2026-02-28')
    expect(addDays('2026-09-26', -6)).toBe('2026-09-20')
  })
})
