// @vitest-environment jsdom
import { renderHook, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { useSingleReport } from '../useSingleReport'

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

describe('useSingleReport', () => {
  it("requests the preset's whole tenant-local days", async () => {
    const fetchFn = vi.fn().mockResolvedValue({})
    renderHook(() => useSingleReport('t', '7D', fetchFn))
    await waitFor(() => expect(fetchFn).toHaveBeenCalledWith('t', '2026-09-19T04:00:00.000Z', '2026-09-26T04:00:00.000Z'))
  })
})
