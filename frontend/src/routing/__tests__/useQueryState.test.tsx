// @vitest-environment jsdom
import type { ReactNode } from 'react'
import { renderHook, act } from '@testing-library/react'
import { MemoryRouter, useLocation, useNavigationType } from 'react-router-dom'
import { describe, it, expect } from 'vitest'
import { useQueryEnum, useQueryPage, useQuerySet, useQueryString } from '../useQueryState'

function wrapperAt(url: string) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <MemoryRouter initialEntries={[url]}>{children}</MemoryRouter>
  }
}

const TABS = ['sync', 'mapping', 'logs'] as const

describe('useQueryEnum', () => {
  it('reads the value from the URL so a refresh lands on the same tab', () => {
    const { result } = renderHook(() => useQueryEnum('tab', TABS, 'sync'), { wrapper: wrapperAt('/x?tab=logs') })
    expect(result.current[0]).toBe('logs')
  })

  it('falls back to the default when the param is absent', () => {
    const { result } = renderHook(() => useQueryEnum('tab', TABS, 'sync'), { wrapper: wrapperAt('/x') })
    expect(result.current[0]).toBe('sync')
  })

  it('falls back to the default when the param holds a value outside the allowed set', () => {
    const { result } = renderHook(() => useQueryEnum('tab', TABS, 'sync'), { wrapper: wrapperAt('/x?tab=bogus') })
    expect(result.current[0]).toBe('sync')
  })

  it('writes the value to the URL and keeps unrelated params', () => {
    const { result } = renderHook(
      () => ({ q: useQueryEnum('tab', TABS, 'sync'), loc: useLocation() }),
      { wrapper: wrapperAt('/x?page=2') },
    )
    act(() => result.current.q[1]('mapping'))
    expect(result.current.q[0]).toBe('mapping')
    const params = new URLSearchParams(result.current.loc.search)
    expect(params.get('tab')).toBe('mapping')
    expect(params.get('page')).toBe('2')
  })

  it('removes the param when set back to the default, keeping URLs clean', () => {
    const { result } = renderHook(
      () => ({ q: useQueryEnum('tab', TABS, 'sync'), loc: useLocation() }),
      { wrapper: wrapperAt('/x?tab=logs') },
    )
    act(() => result.current.q[1]('sync'))
    expect(result.current.loc.search).toBe('')
  })

  it('replaces the history entry by default so tab clicks do not flood Back', () => {
    const { result } = renderHook(
      () => ({ q: useQueryEnum('tab', TABS, 'sync'), nav: useNavigationType() }),
      { wrapper: wrapperAt('/x') },
    )
    act(() => result.current.q[1]('logs'))
    expect(result.current.nav).toBe('REPLACE')
  })

  it('pushes a history entry when asked, so Back returns to the previous value', () => {
    const { result } = renderHook(
      () => ({ q: useQueryEnum('tab', TABS, 'sync', { history: 'push' }), nav: useNavigationType() }),
      { wrapper: wrapperAt('/x') },
    )
    act(() => result.current.q[1]('logs'))
    expect(result.current.nav).toBe('PUSH')
  })
})

describe('useQueryString', () => {
  it('returns null when absent and the raw value when present', () => {
    const absent = renderHook(() => useQueryString('code'), { wrapper: wrapperAt('/x') })
    expect(absent.result.current[0]).toBeNull()
    const present = renderHook(() => useQueryString('code'), { wrapper: wrapperAt('/x?code=ABC%201') })
    expect(present.result.current[0]).toBe('ABC 1')
  })

  it('removes the param when set to null or empty string', () => {
    const { result } = renderHook(
      () => ({ q: useQueryString('code'), loc: useLocation() }),
      { wrapper: wrapperAt('/x?code=A&tab=b') },
    )
    act(() => result.current.q[1](null))
    expect(result.current.loc.search).toBe('?tab=b')
    act(() => result.current.q[1]('Z'))
    act(() => result.current.q[1](''))
    expect(result.current.loc.search).toBe('?tab=b')
  })
})

describe('resets option', () => {
  it('clears dependent params in the same URL write (e.g. a new search resets the page)', () => {
    const { result } = renderHook(
      () => ({ q: useQueryString('q', { resets: ['page'] }), loc: useLocation() }),
      { wrapper: wrapperAt('/x?page=4&tab=b') },
    )
    act(() => result.current.q[1]('abc'))
    const params = new URLSearchParams(result.current.loc.search)
    expect(params.get('q')).toBe('abc')
    expect(params.get('page')).toBeNull()
    expect(params.get('tab')).toBe('b')
  })
})

describe('useQueryPage', () => {
  it('reads a positive page number', () => {
    const { result } = renderHook(() => useQueryPage(), { wrapper: wrapperAt('/x?page=3') })
    expect(result.current[0]).toBe(3)
  })

  it.each(['0', '-2', 'abc', '1.5'])('treats invalid page %s as page 1', raw => {
    const { result } = renderHook(() => useQueryPage(), { wrapper: wrapperAt(`/x?page=${raw}`) })
    expect(result.current[0]).toBe(1)
  })

  it('omits page=1 from the URL', () => {
    const { result } = renderHook(
      () => ({ q: useQueryPage(), loc: useLocation() }),
      { wrapper: wrapperAt('/x?page=4') },
    )
    act(() => result.current.q[1](1))
    expect(result.current.loc.search).toBe('')
    act(() => result.current.q[1](2))
    expect(result.current.loc.search).toBe('?page=2')
  })
})

describe('useQuerySet', () => {
  it('reads a comma-separated list as a set', () => {
    const { result } = renderHook(() => useQuerySet('open'), { wrapper: wrapperAt('/x?open=a,b') })
    expect([...result.current.values].sort()).toEqual(['a', 'b'])
    expect(result.current.has('a')).toBe(true)
    expect(result.current.has('c')).toBe(false)
  })

  it('toggle adds and removes, dropping the param when empty', () => {
    const { result } = renderHook(
      () => ({ q: useQuerySet('open'), loc: useLocation() }),
      { wrapper: wrapperAt('/x?tab=t') },
    )
    act(() => result.current.q.toggle('a'))
    act(() => result.current.q.toggle('b'))
    expect(new URLSearchParams(result.current.loc.search).get('open')).toBe('a,b')
    act(() => result.current.q.toggle('a'))
    expect(new URLSearchParams(result.current.loc.search).get('open')).toBe('b')
    act(() => result.current.q.toggle('b'))
    expect(result.current.loc.search).toBe('?tab=t')
  })

  it('ids containing commas are encoded so they round-trip', () => {
    const { result } = renderHook(
      () => ({ q: useQuerySet('open'), loc: useLocation() }),
      { wrapper: wrapperAt('/x') },
    )
    act(() => result.current.q.toggle('a,b'))
    expect(result.current.q.has('a,b')).toBe(true)
    expect(result.current.q.has('a')).toBe(false)
  })
})
