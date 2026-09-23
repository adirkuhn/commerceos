// @vitest-environment jsdom
import { renderHook, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useMemberNames } from '../useMemberNames'

vi.mock('../../api', () => ({ listMembers: vi.fn() }))
import * as api from '../../api'

beforeEach(() => vi.clearAllMocks())

describe('useMemberNames', () => {
  it('resolves user ids to member names', async () => {
    vi.mocked(api.listMembers).mockResolvedValue([
      { membership_id: 'm1', user_id: 'u1', email: 'ann@x.io', first_name: 'Ann', last_name: 'Lee', role: 'customer_user', status: 'active', last_login_at: null },
      { membership_id: 'm2', user_id: 'u2', email: 'bo@x.io', first_name: '', last_name: '', role: 'customer_user', status: 'active', last_login_at: null },
    ])

    const { result } = renderHook(() => useMemberNames('tok'))

    await waitFor(() => expect(result.current('u1')).toBe('Ann Lee'))
    expect(result.current('u2')).toBe('bo@x.io')
  })

  it('falls back to a short id when the user is unknown or lm2 refuses the lookup', async () => {
    vi.mocked(api.listMembers).mockRejectedValue(new Error('forbidden'))
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})

    const { result } = renderHook(() => useMemberNames('tok'))

    await waitFor(() => expect(warn).toHaveBeenCalled())
    expect(result.current('3f2a9c1e-0000')).toBe('User 3f2a9c')
    warn.mockRestore()
  })

  it('does not call lm2 without a token', () => {
    renderHook(() => useMemberNames(''))
    expect(api.listMembers).not.toHaveBeenCalled()
  })
})
