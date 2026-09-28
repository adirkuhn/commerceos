// @vitest-environment jsdom
import { renderHook, act } from '@testing-library/react'
import type { ReactNode } from 'react'
import { createMemoryRouter, RouterProvider, useLocation } from 'react-router-dom'
import { describe, it, expect } from 'vitest'
import { useSalesChannelRoute } from '../useSalesChannelRoute'

function setup(url: string) {
  let router!: ReturnType<typeof createMemoryRouter>
  const hook = renderHook(() => ({ r: useSalesChannelRoute(), loc: useLocation() }), {
    wrapper: ({ children }: { children: ReactNode }) => {
      router ??= createMemoryRouter([{ path: '*', element: children }], { initialEntries: [url] })
      return <RouterProvider router={router} />
    },
  })
  return { hook, router: () => router }
}

describe('useSalesChannelRoute', () => {
  it.each([
    ['/sales-channels', { kind: 'list' }],
    ['/sales-channels/new', { kind: 'create' }],
    ['/sales-channels/c1', { kind: 'manage', channelId: 'c1' }],
    ['/sales-channels/c1/edit', { kind: 'edit', channelId: 'c1' }],
    ['/sales-channels/c1/api', { kind: 'api', channelId: 'c1' }],
    ['/sales-channels/c1/themes', { kind: 'themes', channelId: 'c1' }],
    ['/sales-channels/c1/bogus', { kind: 'manage', channelId: 'c1' }],
  ])('%s -> %o', (url, view) => {
    expect(setup(url).hook.result.current.r.view).toEqual(view)
  })

  it('open* push the matching URL', () => {
    const { hook } = setup('/sales-channels')
    act(() => hook.result.current.r.openEdit('c1'))
    expect(hook.result.current.loc.pathname).toBe('/sales-channels/c1/edit')
    act(() => hook.result.current.r.openApiDocs('c1'))
    expect(hook.result.current.loc.pathname).toBe('/sales-channels/c1/api')
    act(() => hook.result.current.r.openThemes('c1'))
    expect(hook.result.current.loc.pathname).toBe('/sales-channels/c1/themes')
    act(() => hook.result.current.r.openManage('c1'))
    expect(hook.result.current.loc.pathname).toBe('/sales-channels/c1')
    act(() => hook.result.current.r.openCreate())
    expect(hook.result.current.loc.pathname).toBe('/sales-channels/new')
  })

  it('close returns to the list', async () => {
    const { hook } = setup('/sales-channels/c1/edit')
    await act(async () => hook.result.current.r.close())
    expect(hook.result.current.loc.pathname).toBe('/sales-channels')
  })
})
