// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { ReportPageShell } from '../shared'

// A tenant in New York: at 02:00 UTC on 26 Sep it is still 25 Sep there.
vi.mock('../../../format/DateFormatProvider', async () => {
  const actual = await vi.importActual<typeof import('../../../format/DateFormatProvider')>('../../../format/DateFormatProvider')
  const formatter = actual.bindDateFormat({ pattern: 'YYYY-MM-DD', timeZone: 'America/New_York' })
  return { ...actual, useDateFormat: () => formatter }
})

beforeEach(() => {
  vi.useFakeTimers({ toFake: ['Date'] })
  vi.setSystemTime(new Date('2026-09-26T02:00:00Z'))
})

afterEach(() => {
  vi.useRealTimers()
})

describe('ReportPageShell', () => {
  it("shows the preset's days in the tenant's calendar, today included, nothing after", () => {
    render(<ReportPageShell title="Sales" period="7D" onPeriodChange={() => {}}><div /></ReportPageShell>)
    expect(screen.getByText('2026-09-19 – 2026-09-25')).toBeInTheDocument()
  })
})
