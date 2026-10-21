// @vitest-environment jsdom
import { renderHook, act, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useShippingAddresses } from '../useShippingAddresses'
import { CodedApiError } from '../../../api/codedApiError'

vi.mock('../../api', () => ({ getShippingAddresses: vi.fn(), updateShippingAddresses: vi.fn() }))
import * as shippingApi from '../../api'

const company = { company: 'Shop BV', street: 'Damrak', number: '1', postal_code: '1012LG', city: 'Amsterdam', cc: 'NL' }
const loaded = { sender_address: null, return_address: null, company_address: company }

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(shippingApi.getShippingAddresses).mockResolvedValue(loaded)
})

describe('useShippingAddresses', () => {
  it('loads the addresses', async () => {
    const { result } = renderHook(() => useShippingAddresses('t'))
    expect(result.current.loading).toBe(true)
    await waitFor(() => expect(result.current.addresses).toEqual(loaded))
    expect(result.current.loading).toBe(false)
    expect(shippingApi.getShippingAddresses).toHaveBeenCalledWith('t')
  })

  it('shows why loading failed', async () => {
    vi.mocked(shippingApi.getShippingAddresses).mockRejectedValue(new CodedApiError('SHIPPING_ADDRESS_INVALID', 'x', undefined))
    const { result } = renderHook(() => useShippingAddresses('t'))
    await waitFor(() => expect(result.current.error).toContain('company name'))
  })

  it('saves both addresses and keeps the saved result', async () => {
    const saved = { sender_address: company, return_address: null, company_address: company }
    vi.mocked(shippingApi.updateShippingAddresses).mockResolvedValue(saved)
    const { result } = renderHook(() => useShippingAddresses('t'))
    await waitFor(() => expect(result.current.addresses).not.toBeNull())
    await act(() => result.current.save({ sender_address: company, return_address: null }))
    expect(shippingApi.updateShippingAddresses).toHaveBeenCalledWith('t', { sender_address: company, return_address: null })
    expect(result.current.addresses).toEqual(saved)
  })

  it('a failed save rejects so the caller can show why', async () => {
    vi.mocked(shippingApi.updateShippingAddresses).mockRejectedValue(new Error('nope'))
    const { result } = renderHook(() => useShippingAddresses('t'))
    await waitFor(() => expect(result.current.addresses).not.toBeNull())
    await expect(result.current.save({ sender_address: null, return_address: null })).rejects.toThrow('nope')
    expect(result.current.addresses).toEqual(loaded)
  })
})
