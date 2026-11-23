import { describe, expect, it } from 'vitest'
import { relativeTime } from '../relativeTime'

const now = new Date('2026-11-23T12:00:00Z')

describe('relativeTime', () => {
  it.each([
    ['2026-11-23T11:59:40Z', 'now'],
    ['2026-11-23T11:55:00Z', '5 minutes ago'],
    ['2026-11-23T09:00:00Z', '3 hours ago'],
    ['2026-11-22T12:00:00Z', 'yesterday'],
    ['2026-11-19T12:00:00Z', '4 days ago'],
  ])('%s is %s', (iso, want) => {
    expect(relativeTime(iso, 'en', now)).toBe(want)
  })

  it('a week or more falls back to the date', () => {
    expect(relativeTime('2026-11-01T12:00:00Z', 'en', now, () => '01-11-2026')).toBe('01-11-2026')
  })

  it('speaks the reader’s language', () => {
    expect(relativeTime('2026-11-23T11:55:00Z', 'pt-BR', now)).toBe('há 5 minutos')
    expect(relativeTime('2026-11-23T11:55:00Z', 'nl', now)).toBe('5 minuten geleden')
  })
})
