// @vitest-environment jsdom
import { renderHook, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useChannelConnections } from '../useChannelConnections'
import type { Connection } from '../../../integrations/types'

vi.mock('../../../integrations/api', () => ({ listConnections: vi.fn() }))
import * as integrationsApi from '../../../integrations/api'

const conn = (provider: string, status: Connection['status']): Connection =>
  ({ id: `${provider}-${status}`, provider, store_url: '', status, created_at: '' })

beforeEach(() => vi.clearAllMocks())

describe('useChannelConnections', () => {
  it('picks the connected WooCommerce and Bol connections, ignoring errored ones', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([
      conn('woocommerce', 'error'), conn('woocommerce', 'connected'), conn('bol', 'connected'), conn('shopify', 'connected'),
    ])
    const { result } = renderHook(() => useChannelConnections('tok'))
    await waitFor(() => expect(result.current.woo?.id).toBe('woocommerce-connected'))
    expect(result.current.bol?.id).toBe('bol-connected')
    expect(integrationsApi.listConnections).toHaveBeenCalledWith('tok')
  })

  it('is empty when none are connected or the lookup fails', async () => {
    vi.mocked(integrationsApi.listConnections).mockRejectedValue(new Error('boom'))
    const { result } = renderHook(() => useChannelConnections('tok'))
    await waitFor(() => expect(integrationsApi.listConnections).toHaveBeenCalled())
    expect(result.current).toEqual({ woo: null, bol: null })
  })

  it('does not fetch without a token', () => {
    const { result } = renderHook(() => useChannelConnections(null))
    expect(integrationsApi.listConnections).not.toHaveBeenCalled()
    expect(result.current).toEqual({ woo: null, bol: null })
  })
})
