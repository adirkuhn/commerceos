// @vitest-environment jsdom
import { renderHook, act, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useMoneybirdDocument } from '../useMoneybirdDocument'
import { CodedApiError } from '../../../api/codedApiError'
import type { MoneybirdDocument } from '../../types'

vi.mock('../../api', () => ({ getMoneybirdDocument: vi.fn(), sendToMoneybird: vi.fn() }))
import * as api from '../../api'

const booked: MoneybirdDocument = {
  source_type: 'sales_order', source_id: 'so-1', moneybird_id: 'inv-1', reference: '1234', status: 'synced',
  payment_registered: true, error: '', synced_at: '2026-10-29T10:00:00Z',
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(api.getMoneybirdDocument).mockResolvedValue({ document: null, administration_id: 'adm-1', auto_sync: false })
})

describe('useMoneybirdDocument', () => {
  it('loads where the order stands', async () => {
    const { result } = renderHook(() => useMoneybirdDocument('t', 'sales_order', 'so-1'))
    await waitFor(() => expect(result.current.connected).toBe(true))
    expect(result.current.state?.document).toBeNull()
    expect(api.getMoneybirdDocument).toHaveBeenCalledWith('t', 'sales_order', 'so-1')
  })

  // Shops without Moneybird don't see the card at all.
  it('not connected is not an error', async () => {
    vi.mocked(api.getMoneybirdDocument).mockRejectedValue(new CodedApiError('MONEYBIRD_NOT_CONNECTED', 'x', undefined))
    const { result } = renderHook(() => useMoneybirdDocument('t', 'sales_order', 'so-1'))
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.connected).toBe(false)
    expect(result.current.error).toBeNull()
  })

  it('sends, then shows the booked document', async () => {
    vi.mocked(api.sendToMoneybird).mockResolvedValue(booked)
    const { result } = renderHook(() => useMoneybirdDocument('t', 'sales_order', 'so-1'))
    await waitFor(() => expect(result.current.connected).toBe(true))

    await act(() => result.current.send())

    expect(api.sendToMoneybird).toHaveBeenCalledWith('t', 'sales_order', 'so-1')
    expect(result.current.state?.document).toEqual(booked)
    expect(result.current.sendError).toBeNull()
  })

  // The failure is recorded on the document too, so the state is reloaded with it.
  it('a failed send shows why and reloads the recorded failure', async () => {
    vi.mocked(api.sendToMoneybird).mockRejectedValue(new CodedApiError('MONEYBIRD_DUPLICATE_REFERENCE', 'x', undefined))
    const { result } = renderHook(() => useMoneybirdDocument('t', 'sales_order', 'so-1'))
    await waitFor(() => expect(result.current.connected).toBe(true))

    await act(() => result.current.send())

    expect(result.current.sendError).toBe('Moneybird already has an invoice with this number. It was not booked again.')
    expect(api.getMoneybirdDocument).toHaveBeenCalledTimes(2)
    expect(result.current.sending).toBe(false)
  })
})
