// @vitest-environment jsdom
import { renderHook, act } from '@testing-library/react'
import type { ReactNode } from 'react'
import { createMemoryRouter, RouterProvider, useLocation } from 'react-router-dom'
import { describe, it, expect } from 'vitest'
import { useUsersRoute } from '../useUsersRoute'

function setup(url: string) {
  let router!: ReturnType<typeof createMemoryRouter>
  const hook = renderHook(() => ({ r: useUsersRoute(), loc: useLocation() }), {
    wrapper: ({ children }: { children: ReactNode }) => {
      router ??= createMemoryRouter([{ path: '*', element: children }], { initialEntries: [url] })
      return <RouterProvider router={router} />
    },
  })
  return { hook, router: () => router }
}

describe('useUsersRoute', () => {
  it.each([
    ['/users', { kind: 'list' }],
    ['/users/invite', { kind: 'invite' }],
    ['/users/groups', { kind: 'groups', groupId: null, editing: false }],
    ['/users/groups/new', { kind: 'groups', groupId: null, editing: true }],
    ['/users/groups/g1', { kind: 'groups', groupId: 'g1', editing: false }],
    ['/users/groups/g1/edit', { kind: 'groups', groupId: 'g1', editing: true }],
    ['/users/u-1', { kind: 'member', userId: 'u-1' }],
  ])('%s -> %o', (url, view) => {
    expect(setup(url).hook.result.current.r.view).toEqual(view)
  })

  it('navigators push the matching URL', () => {
    const { hook } = setup('/users')
    act(() => hook.result.current.r.openGroups())
    expect(hook.result.current.loc.pathname).toBe('/users/groups')
    act(() => hook.result.current.r.openGroup('g1'))
    expect(hook.result.current.loc.pathname).toBe('/users/groups/g1')
    act(() => hook.result.current.r.editGroup('g1'))
    expect(hook.result.current.loc.pathname).toBe('/users/groups/g1/edit')
    act(() => hook.result.current.r.newGroup())
    expect(hook.result.current.loc.pathname).toBe('/users/groups/new')
    act(() => hook.result.current.r.openMember('u 1'))
    expect(hook.result.current.loc.pathname).toBe('/users/u%201')
    act(() => hook.result.current.r.openInvite())
    expect(hook.result.current.loc.pathname).toBe('/users/invite')
  })

  it('closeGroupForm returns to the group (or the groups list when creating)', async () => {
    const edit = setup('/users/groups/g1/edit')
    await act(async () => edit.hook.result.current.r.closeGroupForm())
    expect(edit.hook.result.current.loc.pathname).toBe('/users/groups/g1')

    const create = setup('/users/groups/new')
    await act(async () => create.hook.result.current.r.closeGroupForm())
    expect(create.hook.result.current.loc.pathname).toBe('/users/groups')
  })

  it('close returns to the list', async () => {
    const { hook } = setup('/users/u-1')
    await act(async () => hook.result.current.r.close())
    expect(hook.result.current.loc.pathname).toBe('/users')
  })
})
