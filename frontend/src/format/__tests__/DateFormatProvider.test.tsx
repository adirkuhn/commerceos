// @vitest-environment jsdom
import { renderHook, waitFor, act } from '@testing-library/react'
import type { ReactNode } from 'react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { DateFormatProvider, useDateFormat, useApplyDateFormatSettings } from '../DateFormatProvider'

vi.mock('../../settings/api', () => ({ getTenantSettings: vi.fn() }))
import * as settingsApi from '../../settings/api'
import type { TenantSettings } from '../../settings/types'

const tenant = (over: Partial<TenantSettings>) => ({ timezone: 'Europe/Amsterdam', date_format: 'MM/DD/YYYY', ...over }) as TenantSettings

const wrapper = ({ children }: { children: ReactNode }) => <DateFormatProvider token="t">{children}</DateFormatProvider>

beforeEach(() => vi.clearAllMocks())

describe('DateFormatProvider', () => {
  it("formats with the tenant's date format and timezone once loaded", async () => {
    vi.mocked(settingsApi.getTenantSettings).mockResolvedValue(tenant({}))
    const { result } = renderHook(() => useDateFormat(), { wrapper })

    await waitFor(() => expect(result.current.formatDate('2027-11-26')).toBe('11/26/2027'))
    expect(result.current.formatDateTime('2026-09-25T14:30:00Z')).toBe('09/25/2026 16:30')
    expect(settingsApi.getTenantSettings).toHaveBeenCalledWith('t')
  })

  it('uses the default format until settings load, and if loading fails', async () => {
    vi.mocked(settingsApi.getTenantSettings).mockRejectedValue(new Error('boom'))
    const { result } = renderHook(() => useDateFormat(), { wrapper })

    expect(result.current.formatDate('2027-11-26')).toBe('26/11/2027')
    await waitFor(() => expect(settingsApi.getTenantSettings).toHaveBeenCalled())
    expect(result.current.formatDate('2027-11-26')).toBe('26/11/2027')
  })

  it('falls back to the default for an empty stored format', async () => {
    vi.mocked(settingsApi.getTenantSettings).mockResolvedValue(tenant({ date_format: '' }))
    const { result } = renderHook(() => useDateFormat(), { wrapper })

    await waitFor(() => expect(settingsApi.getTenantSettings).toHaveBeenCalled())
    expect(result.current.formatDate('2027-11-26')).toBe('26/11/2027')
  })

  it('applies freshly saved settings right away', async () => {
    vi.mocked(settingsApi.getTenantSettings).mockResolvedValue(tenant({}))
    const { result } = renderHook(() => ({ fmt: useDateFormat(), apply: useApplyDateFormatSettings() }), { wrapper })
    await waitFor(() => expect(result.current.fmt.formatDate('2027-11-26')).toBe('11/26/2027'))

    act(() => result.current.apply(tenant({ date_format: 'YYYY-MM-DD' })))

    expect(result.current.fmt.formatDate('2027-11-26')).toBe('2027-11-26')
  })

  it('works without a provider, using the default format', () => {
    const { result } = renderHook(() => useDateFormat())
    expect(result.current.formatDate('2027-11-26')).toBe('26/11/2027')
  })
})
