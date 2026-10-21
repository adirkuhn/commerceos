// @vitest-environment jsdom
import { renderHook, act, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useShippingConnections } from '../useShippingConnections'
import { shippingConnectionsOf } from '../../test/shippingConnections'

vi.mock('../api', () => ({ listShippingConnections: vi.fn(), updateShippingPreferences: vi.fn() }))
import * as shippingApi from '../api'

const TWO = [{ id: 'c1', provider: 'myparcel' }, { id: 'c2', provider: 'postnl' }]

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(shippingApi.listShippingConnections).mockResolvedValue(shippingConnectionsOf(TWO, 'c2', 'c2'))
})

describe('useShippingConnections', () => {
  it('loads the connections and the default', async () => {
    const { result } = renderHook(() => useShippingConnections('t'))
    expect(result.current.loading).toBe(true)
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(shippingApi.listShippingConnections).toHaveBeenCalledWith('t')
    expect(result.current.connections.map(c => c.id)).toEqual(['c1', 'c2'])
    expect(result.current.defaultConnection?.id).toBe('c2')
    expect(result.current.configuredConnectionId).toBe('c2')
    expect(result.current.error).toBeNull()
  })

  it('has no default without connections', async () => {
    vi.mocked(shippingApi.listShippingConnections).mockResolvedValue(shippingConnectionsOf([]))
    const { result } = renderHook(() => useShippingConnections('t'))
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.connections).toEqual([])
    expect(result.current.defaultConnection).toBeNull()
  })

  it('does not load without a token', () => {
    const { result } = renderHook(() => useShippingConnections(''))
    expect(result.current.loading).toBe(false)
    expect(shippingApi.listShippingConnections).not.toHaveBeenCalled()
  })

  it('surfaces a load failure', async () => {
    vi.mocked(shippingApi.listShippingConnections).mockRejectedValue(new Error('down'))
    const { result } = renderHook(() => useShippingConnections('t'))
    await waitFor(() => expect(result.current.error).toBe('down'))
    expect(result.current.loading).toBe(false)
  })

  it('saving the default takes the refreshed list', async () => {
    vi.mocked(shippingApi.updateShippingPreferences).mockResolvedValue(shippingConnectionsOf(TWO, 'c1', null))
    const { result } = renderHook(() => useShippingConnections('t'))
    await waitFor(() => expect(result.current.loading).toBe(false))
    await act(() => result.current.saveDefault(null))
    expect(shippingApi.updateShippingPreferences).toHaveBeenCalledWith('t', null)
    expect(result.current.defaultConnection?.id).toBe('c1')
    expect(result.current.configuredConnectionId).toBeNull()
  })

  it('a failed save rejects and keeps the list', async () => {
    vi.mocked(shippingApi.updateShippingPreferences).mockRejectedValue(new Error('forbidden'))
    const { result } = renderHook(() => useShippingConnections('t'))
    await waitFor(() => expect(result.current.loading).toBe(false))
    await act(() => expect(result.current.saveDefault('c1')).rejects.toThrow('forbidden'))
    expect(result.current.defaultConnection?.id).toBe('c2')
  })
})
