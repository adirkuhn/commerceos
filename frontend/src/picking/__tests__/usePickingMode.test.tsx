// @vitest-environment jsdom
import { renderHook, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { usePickingMode } from '../usePickingMode'

vi.mock('../../settings/api', () => ({ getTenantSettings: vi.fn() }))
import * as settingsApi from '../../settings/api'

beforeEach(() => vi.clearAllMocks())

describe('usePickingMode', () => {
  it('is unresolved until settings load, then reports the mode', async () => {
    vi.mocked(settingsApi.getTenantSettings).mockResolvedValue({ picking_mode: 'list' } as Awaited<ReturnType<typeof settingsApi.getTenantSettings>>)

    const { result } = renderHook(() => usePickingMode('tok'))

    expect(result.current.mode).toBeNull()
    await waitFor(() => expect(result.current.mode).toBe('list'))
  })

  it('defaults to scan when the tenant never saved a mode', async () => {
    vi.mocked(settingsApi.getTenantSettings).mockResolvedValue({} as Awaited<ReturnType<typeof settingsApi.getTenantSettings>>)

    const { result } = renderHook(() => usePickingMode('tok'))

    await waitFor(() => expect(result.current.mode).toBe('scan'))
  })

  it('falls back to scan and reports the error when settings fail to load', async () => {
    vi.mocked(settingsApi.getTenantSettings).mockRejectedValue(new Error('boom'))

    const { result } = renderHook(() => usePickingMode('tok'))

    await waitFor(() => expect(result.current.error).toBe('boom'))
    expect(result.current.mode).toBe('scan')
  })
})
