// @vitest-environment jsdom
import { renderHook, waitFor, act } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { useResource } from '../useResource'

describe('useResource', () => {
  it('loads data and clears loading', async () => {
    const { result } = renderHook(() => useResource(() => Promise.resolve(42), ['a']))
    expect(result.current.loading).toBe(true)
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.data).toBe(42)
    expect(result.current.error).toBe('')
  })

  it('surfaces the error message on failure', async () => {
    const { result } = renderHook(() => useResource(() => Promise.reject(new Error('not found')), ['a']))
    await waitFor(() => expect(result.current.error).toBe('not found'))
    expect(result.current.data).toBeNull()
  })

  it('reloads when deps change and ignores a stale response that resolves late', async () => {
    let resolveFirst: (v: string) => void = () => {}
    const load = vi.fn()
      .mockImplementationOnce(() => new Promise<string>(r => { resolveFirst = r }))
      .mockImplementationOnce(() => Promise.resolve('second'))
    const { result, rerender } = renderHook(({ id }) => useResource(load, [id]), { initialProps: { id: '1' } })
    rerender({ id: '2' })
    await waitFor(() => expect(result.current.data).toBe('second'))
    await act(async () => { resolveFirst('first') })
    expect(result.current.data).toBe('second')
  })

  it('lets the caller replace the data after a local update', async () => {
    const { result } = renderHook(() => useResource(() => Promise.resolve({ n: 1 }), []))
    await waitFor(() => expect(result.current.data).toEqual({ n: 1 }))
    act(() => result.current.setData({ n: 2 }))
    expect(result.current.data).toEqual({ n: 2 })
  })
})
