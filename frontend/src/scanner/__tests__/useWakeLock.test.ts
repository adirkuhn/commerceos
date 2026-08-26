// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook } from '@testing-library/react'
import { useWakeLock } from '../useWakeLock'

function mockWakeLockAPI() {
  const sentinel = { release: vi.fn().mockResolvedValue(undefined) }
  const request = vi.fn().mockResolvedValue(sentinel)
  vi.stubGlobal('navigator', { ...navigator, wakeLock: { request } })
  return { request, sentinel }
}

function setVisibility(state: DocumentVisibilityState) {
  vi.spyOn(document, 'visibilityState', 'get').mockReturnValue(state)
  document.dispatchEvent(new Event('visibilitychange'))
}

beforeEach(() => { vi.restoreAllMocks() })
afterEach(() => { vi.unstubAllGlobals() })

describe('useWakeLock', () => {
  it('requests a screen wake lock on mount', async () => {
    const { request } = mockWakeLockAPI()
    renderHook(() => useWakeLock())
    await vi.waitFor(() => expect(request).toHaveBeenCalledWith('screen'))
  })

  it('releases the lock on unmount', async () => {
    const { request, sentinel } = mockWakeLockAPI()
    const { unmount } = renderHook(() => useWakeLock())
    await vi.waitFor(() => expect(request).toHaveBeenCalled())

    unmount()

    expect(sentinel.release).toHaveBeenCalled()
  })

  it('reacquires the lock when the tab becomes visible again', async () => {
    const { request } = mockWakeLockAPI()
    renderHook(() => useWakeLock())
    await vi.waitFor(() => expect(request).toHaveBeenCalledTimes(1))

    setVisibility('visible')

    await vi.waitFor(() => expect(request).toHaveBeenCalledTimes(2))
  })

  it('does not reacquire when the tab is hidden', async () => {
    const { request } = mockWakeLockAPI()
    renderHook(() => useWakeLock())
    await vi.waitFor(() => expect(request).toHaveBeenCalledTimes(1))

    setVisibility('hidden')

    expect(request).toHaveBeenCalledTimes(1)
  })

  it('no-ops silently when the browser has no Wake Lock API', () => {
    vi.stubGlobal('navigator', { ...navigator, wakeLock: undefined })
    expect(() => renderHook(() => useWakeLock())).not.toThrow()
  })
})
