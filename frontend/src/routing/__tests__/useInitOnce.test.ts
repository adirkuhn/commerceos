// @vitest-environment jsdom
import { renderHook } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { useInitOnce } from '../useInitOnce'

describe('useInitOnce', () => {
  it('does nothing while key is null', () => {
    const init = vi.fn()
    const { result } = renderHook(() => useInitOnce(null, true, init))
    expect(init).not.toHaveBeenCalled()
    expect(result.current).toBe(false)
  })

  it('waits for ready, then runs once and reports initialised', () => {
    const init = vi.fn()
    const { result, rerender } = renderHook(({ ready }) => useInitOnce('edit:1', ready, init), { initialProps: { ready: false } })
    expect(init).not.toHaveBeenCalled()
    expect(result.current).toBe(false)

    rerender({ ready: true })
    expect(init).toHaveBeenCalledTimes(1)
    expect(result.current).toBe(true)

    rerender({ ready: true })
    expect(init).toHaveBeenCalledTimes(1)
  })

  it('runs again for a new key, and again for the same key after closing', () => {
    const init = vi.fn()
    const { rerender } = renderHook(({ k }) => useInitOnce(k, true, init), { initialProps: { k: 'edit:1' as string | null } })
    rerender({ k: 'edit:2' })
    expect(init).toHaveBeenCalledTimes(2)
    rerender({ k: null })
    rerender({ k: 'edit:2' })
    expect(init).toHaveBeenCalledTimes(3)
  })
})
