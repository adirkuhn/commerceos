import { describe, it, expect } from 'vitest'
import i18next from 'i18next'
import { formatDate, formatDateTime, formatTime, formatMonthDay, formatMonthYear, parseDateFormat, DEFAULT_DATE_FORMAT } from '../date'

const ams = (pattern: 'DD/MM/YYYY' | 'MM/DD/YYYY' | 'YYYY-MM-DD') => ({ pattern, timeZone: 'Europe/Amsterdam' })

describe('formatDate', () => {
  it.each([
    ['DD/MM/YYYY', '26/11/2027'],
    ['MM/DD/YYYY', '11/26/2027'],
    ['YYYY-MM-DD', '2027-11-26'],
  ] as const)('formats a date-only value as %s', (pattern, expected) => {
    expect(formatDate('2027-11-26', ams(pattern))).toBe(expected)
  })

  it('never shifts a date-only value by timezone', () => {
    expect(formatDate('2027-11-26', { pattern: 'YYYY-MM-DD', timeZone: 'America/Los_Angeles' })).toBe('2027-11-26')
    expect(formatDate('2027-11-26', { pattern: 'YYYY-MM-DD', timeZone: 'Pacific/Kiritimati' })).toBe('2027-11-26')
  })

  it("renders a timestamp's date in the tenant timezone", () => {
    // 23:30 UTC on the 25th is already the 26th in Amsterdam (UTC+1 in November).
    expect(formatDate('2027-11-25T23:30:00Z', ams('DD/MM/YYYY'))).toBe('26/11/2027')
    expect(formatDate('2027-11-25T23:30:00Z', { pattern: 'DD/MM/YYYY', timeZone: 'UTC' })).toBe('25/11/2027')
  })

  it('accepts a Date', () => {
    expect(formatDate(new Date('2027-11-26T12:00:00Z'), ams('DD/MM/YYYY'))).toBe('26/11/2027')
  })

  it('defaults to DD/MM/YYYY', () => {
    expect(DEFAULT_DATE_FORMAT.pattern).toBe('DD/MM/YYYY')
    expect(formatDate('2027-11-26')).toBe('26/11/2027')
  })

  it('renders a missing or unparseable value as an em dash', () => {
    expect(formatDate(null)).toBe('—')
    expect(formatDate(undefined)).toBe('—')
    expect(formatDate('')).toBe('—')
    expect(formatDate('not a date')).toBe('—')
  })

  it('falls back to the browser timezone for an unknown timezone instead of throwing', () => {
    expect(() => formatDate('2027-11-26T12:00:00Z', { pattern: 'DD/MM/YYYY', timeZone: 'Mars/Olympus' })).not.toThrow()
  })
})

describe('formatDateTime', () => {
  it('formats date and 24h time in the tenant timezone', () => {
    expect(formatDateTime('2026-09-25T14:30:00Z', ams('DD/MM/YYYY'))).toBe('25/09/2026 16:30')
    expect(formatDateTime('2026-09-25T14:30:00Z', ams('MM/DD/YYYY'))).toBe('09/25/2026 16:30')
    expect(formatDateTime('2026-09-25T14:30:00Z', ams('YYYY-MM-DD'))).toBe('2026-09-25 16:30')
  })

  it('renders midnight as 00:00, not 24:00', () => {
    expect(formatDateTime('2026-09-25T22:00:00Z', ams('YYYY-MM-DD'))).toBe('2026-09-26 00:00')
  })

  it('renders a missing value as an em dash', () => {
    expect(formatDateTime(null)).toBe('—')
  })
})

describe('formatTime', () => {
  it('formats a 24h time in the tenant timezone', () => {
    expect(formatTime('2026-09-25T14:30:05Z', ams('DD/MM/YYYY'))).toBe('16:30')
  })
})

describe('parseDateFormat', () => {
  it('accepts the three supported patterns', () => {
    expect(parseDateFormat('MM/DD/YYYY')).toBe('MM/DD/YYYY')
    expect(parseDateFormat('YYYY-MM-DD')).toBe('YYYY-MM-DD')
  })
  it('falls back to the default for an empty or unknown value', () => {
    expect(parseDateFormat('')).toBe('DD/MM/YYYY')
    expect(parseDateFormat('D.M.YY')).toBe('DD/MM/YYYY')
    expect(parseDateFormat(undefined)).toBe('DD/MM/YYYY')
  })
})

describe('formatMonthDay / formatMonthYear (chart and report labels)', () => {
  it('orders day and month like the tenant pattern', () => {
    expect(formatMonthDay('2026-09-03', ams('DD/MM/YYYY'))).toBe('3 Sep')
    expect(formatMonthDay('2026-09-03', ams('MM/DD/YYYY'))).toBe('Sep 3')
    expect(formatMonthDay('2026-09-03', ams('YYYY-MM-DD'))).toBe('3 Sep')
  })
  it('uses the tenant timezone for timestamps', () => {
    expect(formatMonthDay('2026-09-02T22:30:00Z', ams('DD/MM/YYYY'))).toBe('3 Sep')
  })
  it('formats month and year', () => {
    expect(formatMonthYear('2026-09-01', ams('DD/MM/YYYY'))).toBe('Sep 2026')
    expect(formatMonthYear('2026-09-01', ams('DD/MM/YYYY'), { shortYear: true })).toBe("Sep '26")
  })
  it('renders a missing value as an em dash', () => {
    expect(formatMonthDay(null)).toBe('—')
    expect(formatMonthYear(undefined)).toBe('—')
  })
})

describe('month names follow the UI language', () => {
  it('uses pt-BR month abbreviations', async () => {
    await i18next.changeLanguage('pt-BR')
    try {
      expect(formatMonthDay('2026-09-03', ams('DD/MM/YYYY'))).toBe('3 Set')
      expect(formatMonthYear('2026-12-01', ams('DD/MM/YYYY'))).toBe('Dez 2026')
    } finally {
      await i18next.changeLanguage('en')
    }
  })
})
