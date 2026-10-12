import { describe, expect, it } from 'vitest'
import { formatLongDay } from '../date'

describe('formatLongDay', () => {
  const instant = new Date('2026-10-12T23:30:00Z')

  it('names the weekday and month in the given language', () => {
    expect(formatLongDay(instant, 'en', { pattern: 'DD/MM/YYYY', timeZone: 'UTC' })).toBe('Monday, October 12')
    expect(formatLongDay(instant, 'nl', { pattern: 'DD/MM/YYYY', timeZone: 'UTC' })).toBe('maandag 12 oktober')
  })

  it('uses the tenant timezone, not the browser one', () => {
    expect(formatLongDay(instant, 'en', { pattern: 'DD/MM/YYYY', timeZone: 'Europe/Amsterdam' })).toBe('Tuesday, October 13')
  })

  it('falls back to the browser zone for an unknown zone instead of throwing', () => {
    expect(() => formatLongDay(instant, 'en', { pattern: 'DD/MM/YYYY', timeZone: 'Mars/Base' })).not.toThrow()
  })
})
