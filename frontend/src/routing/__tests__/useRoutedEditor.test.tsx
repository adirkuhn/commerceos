// @vitest-environment jsdom
import { renderHook, act } from '@testing-library/react'
import type { ReactNode } from 'react'
import { createMemoryRouter, RouterProvider, useLocation } from 'react-router-dom'
import { describe, it, expect } from 'vitest'
import { useRoutedEditor, NEW_RECORD } from '../useRoutedEditor'

function setup(url: string, initial: string[] = [url]) {
  let router!: ReturnType<typeof createMemoryRouter>
  const hook = renderHook(() => ({ ed: useRoutedEditor('/coupons'), loc: useLocation() }), {
    wrapper: ({ children }: { children: ReactNode }) => {
      router ??= createMemoryRouter([{ path: '*', element: children }], { initialEntries: initial, initialIndex: initial.length - 1 })
      return <RouterProvider router={router} />
    },
  })
  return { hook, router: () => router }
}

describe('useRoutedEditor', () => {
  it('is closed on the list URL', () => {
    const { hook } = setup('/coupons')
    expect(hook.result.current.ed.editingId).toBeNull()
  })

  it('reads "new" and a record id from the URL (refresh-safe)', () => {
    expect(setup('/coupons/new').hook.result.current.ed.editingId).toBe(NEW_RECORD)
    expect(setup('/coupons/c-1').hook.result.current.ed.editingId).toBe('c-1')
  })

  it('openNew / openEdit push the editor URL', () => {
    const { hook } = setup('/coupons')
    act(() => hook.result.current.ed.openNew())
    expect(hook.result.current.loc.pathname).toBe('/coupons/new')
    act(() => hook.result.current.ed.close())
    act(() => hook.result.current.ed.openEdit('c 1'))
    expect(hook.result.current.loc.pathname).toBe('/coupons/c%201')
    expect(hook.result.current.ed.editingId).toBe('c 1')
  })

  it('close returns to the list page it was opened from, keeping its query (page/search)', async () => {
    const { hook } = setup('/coupons?page=3')
    act(() => hook.result.current.ed.openEdit('c-1'))
    await act(async () => hook.result.current.ed.close())
    expect(hook.result.current.loc.pathname + hook.result.current.loc.search).toBe('/coupons?page=3')
  })

  it('close on a deep-linked editor replaces with the list URL', async () => {
    const { hook, router } = setup('/coupons/c-1', ['/elsewhere', '/coupons/c-1'])
    await act(async () => hook.result.current.ed.close())
    expect(hook.result.current.loc.pathname).toBe('/coupons')
    await act(() => router().navigate(-1))
    expect(hook.result.current.loc.pathname).toBe('/elsewhere')
  })

  it('showSaved replaces /new with the created record URL, so Back skips the blank form', async () => {
    const { hook, router } = setup('/coupons/new', ['/coupons', '/coupons/new'])
    await act(async () => hook.result.current.ed.showSaved('c-9'))
    expect(hook.result.current.loc.pathname).toBe('/coupons/c-9')
    expect(router().state.historyAction).toBe('REPLACE')
  })
})
