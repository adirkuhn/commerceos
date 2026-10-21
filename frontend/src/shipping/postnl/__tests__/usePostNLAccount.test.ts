// @vitest-environment jsdom
import { renderHook, act, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { usePostNLAccount } from '../usePostNLAccount'
import { CodedApiError } from '../../../api/codedApiError'

vi.mock('../../api', () => ({ getPostNLConnection: vi.fn(), updatePostNLConnection: vi.fn() }))
import * as shippingApi from '../../api'

const account = { connection_id: 'pn', customer_code: 'DEVC', customer_number: '11223344', sandbox: true }

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(shippingApi.getPostNLConnection).mockResolvedValue(account)
})

describe('usePostNLAccount', () => {
  it('loads the account', async () => {
    const { result } = renderHook(() => usePostNLAccount('t', 'pn'))
    await waitFor(() => expect(result.current.account).toEqual(account))
    expect(shippingApi.getPostNLConnection).toHaveBeenCalledWith('t', 'pn')
  })

  it('shows why loading failed', async () => {
    vi.mocked(shippingApi.getPostNLConnection).mockRejectedValue(new CodedApiError('NOT_POSTNL_CONNECTION', 'x', undefined))
    const { result } = renderHook(() => usePostNLAccount('t', 'pn'))
    await waitFor(() => expect(result.current.error).toBe('This is not a PostNL connection.'))
  })

  it('saves and keeps the saved account', async () => {
    const live = { ...account, sandbox: false }
    vi.mocked(shippingApi.updatePostNLConnection).mockResolvedValue(live)
    const { result } = renderHook(() => usePostNLAccount('t', 'pn'))
    await waitFor(() => expect(result.current.account).not.toBeNull())
    const input = { api_key: '', customer_code: 'DEVC', customer_number: '11223344', sandbox: false }
    await act(() => result.current.save(input))
    expect(shippingApi.updatePostNLConnection).toHaveBeenCalledWith('t', 'pn', input)
    expect(result.current.account).toEqual(live)
  })
})
