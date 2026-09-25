// @vitest-environment jsdom
import { renderHook, act } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'

vi.mock('../../api', () => ({ testConnection: vi.fn() }))

import * as api from '../../api'
import { useTestConnection } from '../useTestConnection'

beforeEach(() => vi.clearAllMocks())

describe('useTestConnection', () => {
  it('records a success result and reloads', async () => {
    vi.mocked(api.testConnection).mockResolvedValue(undefined as never)
    const onTested = vi.fn()
    const { result } = renderHook(() => useTestConnection('tok', onTested))
    await act(() => result.current.test('c1'))
    expect(result.current.testResult).toEqual({ id: 'c1', ok: true, message: 'Connection OK' })
    expect(result.current.testingId).toBeNull()
    expect(onTested).toHaveBeenCalledOnce()
  })

  it('records a failure result and still reloads', async () => {
    vi.mocked(api.testConnection).mockRejectedValue(new Error('unreachable'))
    const onTested = vi.fn()
    const { result } = renderHook(() => useTestConnection('tok', onTested))
    await act(() => result.current.test('c1'))
    expect(result.current.testResult).toEqual({ id: 'c1', ok: false, message: 'unreachable' })
    expect(onTested).toHaveBeenCalledOnce()
  })

  it('does nothing without a token', async () => {
    const onTested = vi.fn()
    const { result } = renderHook(() => useTestConnection('', onTested))
    await act(() => result.current.test('c1'))
    expect(api.testConnection).not.toHaveBeenCalled()
    expect(onTested).not.toHaveBeenCalled()
  })
})
