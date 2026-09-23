// @vitest-environment jsdom
import { renderHook, act } from '@testing-library/react'
import type { ReactNode } from 'react'
import { createMemoryRouter, RouterProvider, useLocation } from 'react-router-dom'
import { describe, it, expect } from 'vitest'
import { useIntegrationsRoute } from '../useIntegrationsRoute'

function setup(url: string) {
  let router!: ReturnType<typeof createMemoryRouter>
  const hook = renderHook(() => ({ r: useIntegrationsRoute(), loc: useLocation() }), {
    wrapper: ({ children }: { children: ReactNode }) => {
      router ??= createMemoryRouter([{ path: '*', element: children }], { initialEntries: [url] })
      return <RouterProvider router={router} />
    },
  })
  return { hook, router: () => router }
}

describe('useIntegrationsRoute', () => {
  it.each([
    ['/integrations', { kind: 'list' }],
    ['/integrations/connect', { kind: 'browse' }],
    ['/integrations/connect/stripe', { kind: 'connect', provider: 'stripe' }],
    ['/integrations/connect/bogus', { kind: 'browse' }],
    ['/integrations/new/bol', { kind: 'new', provider: 'bol' }],
    ['/integrations/new/myparcel', { kind: 'new', provider: 'myparcel' }],
    ['/integrations/new/bogus', { kind: 'list' }],
    ['/integrations/c1', { kind: 'connection', connectionId: 'c1', panel: 'settings' }],
    ['/integrations/c1/import', { kind: 'connection', connectionId: 'c1', panel: 'import' }],
    ['/integrations/c1/test-mode', { kind: 'connection', connectionId: 'c1', panel: 'test-mode' }],
    ['/integrations/c1/bogus', { kind: 'connection', connectionId: 'c1', panel: 'settings' }],
  ])('%s -> %o', (url, view) => {
    expect(setup(url).hook.result.current.r.view).toEqual(view)
  })

  it('navigators push the matching URL', () => {
    const { hook } = setup('/integrations')
    act(() => hook.result.current.r.openBrowse())
    expect(hook.result.current.loc.pathname).toBe('/integrations/connect')
    act(() => hook.result.current.r.openConnect('whatsapp'))
    expect(hook.result.current.loc.pathname).toBe('/integrations/connect/whatsapp')
    act(() => hook.result.current.r.openNew('bol'))
    expect(hook.result.current.loc.pathname).toBe('/integrations/new/bol')
    act(() => hook.result.current.r.openPanel('c 1', 'settings'))
    expect(hook.result.current.loc.pathname).toBe('/integrations/c%201')
    act(() => hook.result.current.r.openPanel('c1', 'delete-products'))
    expect(hook.result.current.loc.pathname).toBe('/integrations/c1/delete-products')
  })

  it('close returns to the list', async () => {
    const { hook } = setup('/integrations/c1/import')
    await act(async () => hook.result.current.r.close())
    expect(hook.result.current.loc.pathname).toBe('/integrations')
  })

  it('backToBrowse from a connect form returns to the catalog', async () => {
    const { hook } = setup('/integrations/connect/email')
    await act(async () => hook.result.current.r.backToBrowse())
    expect(hook.result.current.loc.pathname).toBe('/integrations/connect')
  })
})
