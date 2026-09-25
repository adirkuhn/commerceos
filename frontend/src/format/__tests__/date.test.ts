import { describe, it, expect } from 'vitest'
import { formatDate, formatDateTime } from '../date'

const iso = '2026-09-25T14:30:00Z'

describe('formatDate', () => {
  it('formats an ISO timestamp as a localized date', () => {
    expect(formatDate(iso)).toBe(new Date(iso).toLocaleDateString())
  })
  it('renders a missing value as an em dash', () => {
    expect(formatDate(null)).toBe('—')
    expect(formatDate(undefined)).toBe('—')
    expect(formatDate('')).toBe('—')
  })
})

describe('formatDateTime', () => {
  it('formats an ISO timestamp as a localized date and time', () => {
    expect(formatDateTime(iso)).toBe(new Date(iso).toLocaleString())
  })
  it('renders a missing value as an em dash', () => {
    expect(formatDateTime(null)).toBe('—')
  })
})
