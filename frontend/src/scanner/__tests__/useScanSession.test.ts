// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { act, renderHook, waitFor } from '@testing-library/react'
import { useScanSession, type ScanSessionApi } from '../useScanSession'

vi.mock('../../team/api', () => ({ listMembers: vi.fn().mockResolvedValue([]) }))

interface Detail { session: { status: string; locked_by?: string | null; locked_at?: string | null } }

const TOKEN = `x.${btoa(JSON.stringify({ user_id: 'me', role: 'customer_user' }))}.y`
const detail = (overrides: Partial<Detail['session']> = {}): Detail => ({ session: { status: 'in_progress', ...overrides } })

function fakeApi(): ScanSessionApi<Detail> & { [K in 'get' | 'touch' | 'release' | 'takeOver' | 'deleteScan' | 'reset']: ReturnType<typeof vi.fn> } {
  return {
    get: vi.fn().mockResolvedValue(detail()),
    touch: vi.fn().mockResolvedValue(null),
    release: vi.fn().mockResolvedValue(null),
    takeOver: vi.fn().mockResolvedValue(null),
    deleteScan: vi.fn().mockResolvedValue(null),
    reset: vi.fn().mockResolvedValue(null),
    messages: { loadFailed: 'Failed to load', lockLost: 'Lock lost', resetFailed: 'Failed to reset' },
  }
}

let sessionApi: ReturnType<typeof fakeApi>
let onAuthError: ReturnType<typeof vi.fn>

function setup() {
  return renderHook(() => useScanSession(TOKEN, 's-1', onAuthError, sessionApi))
}

beforeEach(() => {
  sessionApi = fakeApi()
  onAuthError = vi.fn()
})

describe('useScanSession loading', () => {
  it('loads the session detail on mount', async () => {
    const { result } = setup()
    expect(result.current.loading).toBe(true)
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(sessionApi.get).toHaveBeenCalledWith(TOKEN, 's-1')
    expect(result.current.detail).toEqual(detail())
  })

  it('shows the load failure message', async () => {
    sessionApi.get.mockRejectedValue('boom')
    const { result } = setup()
    await waitFor(() => expect(result.current.error).toBe('Failed to load'))
  })

  it('hands auth failures to onAuthError instead of showing them', async () => {
    sessionApi.get.mockRejectedValue(new Error('401 unauthorized'))
    const { result } = setup()
    await waitFor(() => expect(onAuthError).toHaveBeenCalled())
    expect(result.current.error).toBe('')
  })

  it('is read only while someone else freshly holds the lock', async () => {
    sessionApi.get.mockResolvedValue(detail({ locked_by: 'u-9', locked_at: new Date().toISOString() }))
    const { result } = setup()
    await waitFor(() => expect(result.current.heldByOther).toBe('u-9'))
    expect(result.current.readOnly).toBe(true)
    expect(result.current.me?.userId).toBe('me')
  })
})

describe('useScanSession actions', () => {
  it('run reloads after a successful action', async () => {
    const { result } = setup()
    await waitFor(() => expect(result.current.loading).toBe(false))
    const action = vi.fn().mockResolvedValue(null)

    let ok = false
    await act(async () => { ok = await result.current.run(action, 'Nope') })

    expect(ok).toBe(true)
    expect(sessionApi.get).toHaveBeenCalledTimes(2)
    expect(result.current.scanning).toBe(false)
  })

  it('run reports the error and returns false when the action fails', async () => {
    const { result } = setup()
    await waitFor(() => expect(result.current.loading).toBe(false))

    let ok = true
    await act(async () => { ok = await result.current.run(() => Promise.reject(new Error('Out of stock')), 'Nope') })
    expect(ok).toBe(false)
    expect(result.current.error).toBe('Out of stock')

    await act(async () => { await result.current.run(() => Promise.reject('x'), 'Nope') })
    expect(result.current.error).toBe('Nope')
  })

  it('run hands auth failures to onAuthError', async () => {
    const { result } = setup()
    await waitFor(() => expect(result.current.loading).toBe(false))
    await act(async () => { await result.current.run(() => Promise.reject(new Error('unauthorized')), 'Nope') })
    expect(onAuthError).toHaveBeenCalled()
    expect(result.current.error).toBe('')
  })

  it('deleteScan deletes then reloads, or shows the failure', async () => {
    const { result } = setup()
    await waitFor(() => expect(result.current.loading).toBe(false))
    await act(async () => { await result.current.deleteScan('scan-1') })
    expect(sessionApi.deleteScan).toHaveBeenCalledWith(TOKEN, 's-1', 'scan-1')
    expect(sessionApi.get).toHaveBeenCalledTimes(2)

    sessionApi.deleteScan.mockRejectedValue('x')
    await act(async () => { await result.current.deleteScan('scan-2') })
    expect(result.current.error).toBe('Failed to delete scan')
  })

  it('resetSession closes the dialog, runs onReset and reloads', async () => {
    const { result } = setup()
    await waitFor(() => expect(result.current.loading).toBe(false))
    act(() => result.current.setResetConfirmOpen(true))
    const onReset = vi.fn()

    await act(async () => { await result.current.resetSession(onReset) })

    expect(sessionApi.reset).toHaveBeenCalledWith(TOKEN, 's-1')
    expect(onReset).toHaveBeenCalled()
    expect(result.current.resetConfirmOpen).toBe(false)
    expect(result.current.resetting).toBe(false)
    expect(sessionApi.get).toHaveBeenCalledTimes(2)
  })

  it('a failed reset keeps the dialog open and shows why', async () => {
    sessionApi.reset.mockRejectedValue('x')
    const { result } = setup()
    await waitFor(() => expect(result.current.loading).toBe(false))
    act(() => result.current.setResetConfirmOpen(true))
    const onReset = vi.fn()

    await act(async () => { await result.current.resetSession(onReset) })

    expect(onReset).not.toHaveBeenCalled()
    expect(result.current.resetConfirmOpen).toBe(true)
    expect(result.current.error).toBe('Failed to reset')
  })

  it('takeOver takes the lock then reloads, or shows the failure', async () => {
    const { result } = setup()
    await waitFor(() => expect(result.current.loading).toBe(false))
    await act(async () => { await result.current.takeOver() })
    expect(sessionApi.takeOver).toHaveBeenCalledWith(TOKEN, 's-1')
    expect(sessionApi.get).toHaveBeenCalledTimes(2)

    sessionApi.takeOver.mockRejectedValue(new Error('Not allowed'))
    await act(async () => { await result.current.takeOver() })
    expect(result.current.error).toBe('Not allowed')
  })
})
