// @vitest-environment jsdom
import { renderHook, act } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { useSessionLock, HEARTBEAT_MS } from '../useSessionLock'

const now = () => new Date().toISOString()
const stale = () => new Date(Date.now() - 3 * 60_000).toISOString()

function setup(props: Partial<Parameters<typeof useSessionLock>[0]> = {}) {
  const touch = vi.fn().mockResolvedValue(null)
  const release = vi.fn().mockResolvedValue(null)
  const onLost = vi.fn()
  const initial = { sessionId: 's1', me: 'me', lockedBy: null as string | null, lockedAt: null as string | null, active: true, touch, release, onLost, ...props }
  const hook = renderHook(p => useSessionLock(p), { initialProps: initial })
  return { ...hook, touch, release, onLost, initial }
}

beforeEach(() => vi.useFakeTimers())
afterEach(() => vi.useRealTimers())

describe('useSessionLock', () => {
  it('viewing an unlocked session neither heartbeats nor releases', () => {
    const { result, touch, release, unmount } = setup()
    expect(result.current.holding).toBe(false)
    expect(result.current.heldByOther).toBeNull()

    act(() => { vi.advanceTimersByTime(HEARTBEAT_MS * 2) })
    unmount()

    expect(touch).not.toHaveBeenCalled()
    expect(release).not.toHaveBeenCalled()
  })

  it('heartbeats while holding the lock', () => {
    const { result, touch } = setup({ lockedBy: 'me', lockedAt: now() })
    expect(result.current.holding).toBe(true)

    act(() => { vi.advanceTimersByTime(HEARTBEAT_MS) })

    expect(touch).toHaveBeenCalledWith('s1')
  })

  it('releases the lock when leaving the session', () => {
    const { release, rerender, initial } = setup({ lockedBy: 'me', lockedAt: now() })

    rerender({ ...initial, sessionId: 's2', lockedBy: null, lockedAt: null })

    expect(release).toHaveBeenCalledWith('s1')
  })

  it('releases the lock on unmount', () => {
    const { release, unmount } = setup({ lockedBy: 'me', lockedAt: now() })
    unmount()
    expect(release).toHaveBeenCalledWith('s1')
  })

  it('releases the lock when the page is closed', () => {
    const { release } = setup({ lockedBy: 'me', lockedAt: now() })
    act(() => { window.dispatchEvent(new Event('pagehide')) })
    expect(release).toHaveBeenCalledWith('s1')
  })

  it('reports who else holds a fresh lock', () => {
    const since = now()
    const { result } = setup({ lockedBy: 'other', lockedAt: since })
    expect(result.current.holding).toBe(false)
    expect(result.current.heldByOther).toBe('other')
  })

  it('a stale lock is free', () => {
    const { result } = setup({ lockedBy: 'other', lockedAt: stale() })
    expect(result.current.heldByOther).toBeNull()
  })

  it('a heartbeat refused because someone took over reports the loss', async () => {
    const { touch, onLost } = setup({ lockedBy: 'me', lockedAt: now() })
    touch.mockRejectedValue(new Error('packing session is locked by another user'))

    await act(async () => { vi.advanceTimersByTime(HEARTBEAT_MS) })

    expect(onLost).toHaveBeenCalled()
  })

  it('a finished session stops heartbeating and does not release', () => {
    const { touch, release, unmount } = setup({ lockedBy: 'me', lockedAt: now(), active: false })
    act(() => { vi.advanceTimersByTime(HEARTBEAT_MS) })
    unmount()
    expect(touch).not.toHaveBeenCalled()
    expect(release).not.toHaveBeenCalled()
  })
})
