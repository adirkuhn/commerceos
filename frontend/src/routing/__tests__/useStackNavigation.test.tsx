// @vitest-environment jsdom
import { renderHook, act } from '@testing-library/react'
import type { ReactNode } from 'react'
import { createMemoryRouter, RouterProvider, useLocation } from 'react-router-dom'
import { describe, it, expect } from 'vitest'
import { useStackNavigation } from '../useStackNavigation'

// renderHook needs the router to be the wrapper; a data router lets the test
// inspect the full history index after each navigation.
function setup(initial: string[]) {
  let router!: ReturnType<typeof createMemoryRouter>
  const hook = renderHook(() => ({ nav: useStackNavigation(), loc: useLocation() }), {
    wrapper: ({ children }: { children: ReactNode }) => {
      router ??= createMemoryRouter([{ path: '*', element: children }], { initialEntries: initial, initialIndex: initial.length - 1 })
      return <RouterProvider router={router} />
    },
  })
  return { hook, router: () => router }
}

describe('useStackNavigation', () => {
  it('open pushes a new entry', () => {
    const { hook } = setup(['/list'])
    act(() => hook.result.current.nav.open('/list/1'))
    expect(hook.result.current.loc.pathname).toBe('/list/1')
  })

  it('exitTo pops back when the screen was opened from that parent, so Back does not re-enter the child', async () => {
    const { hook, router } = setup(['/home', '/list'])
    act(() => hook.result.current.nav.open('/list/1'))
    await act(async () => hook.result.current.nav.exitTo('/list'))
    expect(hook.result.current.loc.pathname).toBe('/list')
    await act(() => router().navigate(-1))
    expect(hook.result.current.loc.pathname).toBe('/home')
  })

  it('exitTo replaces when the screen was deep-linked (no parent in history)', async () => {
    const { hook, router } = setup(['/elsewhere', '/list/1'])
    await act(async () => hook.result.current.nav.exitTo('/list'))
    expect(hook.result.current.loc.pathname).toBe('/list')
    await act(() => router().navigate(-1))
    expect(hook.result.current.loc.pathname).toBe('/elsewhere')
  })

  it('exitTo compares the parent including its query string', async () => {
    const { hook } = setup(['/list?page=2'])
    act(() => hook.result.current.nav.open('/list/1'))
    await act(async () => hook.result.current.nav.exitTo('/list'))
    // opened from /list?page=2, asked to exit to /list -- not the same entry,
    // so it navigates rather than popping to a different page of results.
    expect(hook.result.current.loc.pathname + hook.result.current.loc.search).toBe('/list')
  })

  it('exitToOpener pops back to wherever the screen was opened from, keeping its query', async () => {
    const { hook } = setup(['/list?page=2'])
    act(() => hook.result.current.nav.open('/list/1'))
    await act(async () => hook.result.current.nav.exitToOpener('/list'))
    expect(hook.result.current.loc.pathname + hook.result.current.loc.search).toBe('/list?page=2')
  })

  it('exitToOpener falls back to the given path when deep-linked', async () => {
    const { hook } = setup(['/list/1'])
    await act(async () => hook.result.current.nav.exitToOpener('/list'))
    expect(hook.result.current.loc.pathname).toBe('/list')
  })
})
