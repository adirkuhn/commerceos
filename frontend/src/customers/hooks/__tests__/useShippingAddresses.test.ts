// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'

vi.mock('../../api', () => ({
  listShippingAddresses: vi.fn(), addShippingAddress: vi.fn(), updateShippingAddress: vi.fn(),
  removeShippingAddress: vi.fn(), setDefaultShippingAddress: vi.fn(),
}))
import * as api from '../../api'
import { useShippingAddresses } from '../useShippingAddresses'
import type { ShippingAddress } from '../../types'

const main: ShippingAddress = {
  id: 'sa-1', customer_id: 'c-1', label: 'Main', first_name: '', last_name: '', company: '', address_1: 'Main 1', address_2: '',
  city: 'Utrecht', state: '', postcode: '', country: 'NL', phone: '', is_default: true,
}
const dock: ShippingAddress = { ...main, id: 'sa-2', label: 'Dock', address_1: 'Dock 2', is_default: false }

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(api.listShippingAddresses).mockResolvedValue([main])
})

describe('useShippingAddresses', () => {
  it('loads a customer\'s shipping addresses', async () => {
    const { result } = renderHook(() => useShippingAddresses('tok', 'c-1'))
    await waitFor(() => expect(result.current.addresses).toEqual([main]))
    expect(api.listShippingAddresses).toHaveBeenCalledWith('tok', 'c-1')
  })

  it('reloads after add, update and remove: the default can move on the server', async () => {
    const { result } = renderHook(() => useShippingAddresses('tok', 'c-1'))
    await waitFor(() => expect(result.current.addresses).toHaveLength(1))

    vi.mocked(api.addShippingAddress).mockResolvedValue(dock)
    vi.mocked(api.listShippingAddresses).mockResolvedValue([main, dock])
    await act(() => result.current.add({ address_1: 'Dock 2' }))
    expect(api.addShippingAddress).toHaveBeenCalledWith('tok', 'c-1', { address_1: 'Dock 2' })
    expect(result.current.addresses.map(a => a.id)).toEqual(['sa-1', 'sa-2'])

    vi.mocked(api.updateShippingAddress).mockResolvedValue({ ...dock, city: 'Delft' })
    vi.mocked(api.listShippingAddresses).mockResolvedValue([main, { ...dock, city: 'Delft' }])
    await act(() => result.current.update('sa-2', { address_1: 'Dock 2', city: 'Delft' }))
    expect(result.current.addresses[1].city).toBe('Delft')

    vi.mocked(api.removeShippingAddress).mockResolvedValue(null)
    vi.mocked(api.listShippingAddresses).mockResolvedValue([{ ...dock, is_default: true }])
    await act(() => result.current.remove('sa-1'))
    expect(api.removeShippingAddress).toHaveBeenCalledWith('tok', 'c-1', 'sa-1')
    expect(result.current.addresses).toEqual([{ ...dock, is_default: true }])
  })

  it('makes an address the default from the returned list', async () => {
    const { result } = renderHook(() => useShippingAddresses('tok', 'c-1'))
    await waitFor(() => expect(result.current.addresses).toHaveLength(1))

    vi.mocked(api.setDefaultShippingAddress).mockResolvedValue([{ ...main, is_default: false }, { ...dock, is_default: true }])
    await act(() => result.current.makeDefault('sa-2'))
    expect(api.setDefaultShippingAddress).toHaveBeenCalledWith('tok', 'c-1', 'sa-2')
    expect(result.current.addresses.find(a => a.is_default)?.id).toBe('sa-2')
  })

  it('shows a load error', async () => {
    vi.mocked(api.listShippingAddresses).mockRejectedValue(new Error('boom'))
    const { result } = renderHook(() => useShippingAddresses('tok', 'c-1'))
    await waitFor(() => expect(result.current.error).toBe('boom'))
  })

  it('loads nothing without a customer', () => {
    const { result } = renderHook(() => useShippingAddresses('tok', null))
    expect(api.listShippingAddresses).not.toHaveBeenCalled()
    expect(result.current.addresses).toEqual([])
  })
})
