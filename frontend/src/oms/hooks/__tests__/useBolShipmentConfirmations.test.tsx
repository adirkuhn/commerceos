// @vitest-environment jsdom
import { renderHook, waitFor, act } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { useBolShipmentConfirmations, BOL_CONFIRMATION_POLL_MS } from '../useBolShipmentConfirmations'
import type { BolShipmentConfirmation } from '../../bolConfirmations'

vi.mock('../../bolConfirmations', async importOriginal => ({
  ...(await importOriginal<typeof import('../../bolConfirmations')>()),
  listBolShipmentConfirmations: vi.fn(),
}))
import { listBolShipmentConfirmations } from '../../bolConfirmations'

function confirmation(status: BolShipmentConfirmation['status']): BolShipmentConfirmation {
  return {
    id: 'conf-1', shipment_id: 'ship-1', external_order_id: 'C1', transporter_code: 'TNT', track_and_trace: '3S1',
    process_status_id: 'ps-1', status, error_message: '', needs_manual_action: status === 'failure' || status === 'timeout',
    created_at: '2026-10-07T14:11:00Z', resolved_at: null,
  }
}

beforeEach(() => vi.clearAllMocks())
afterEach(() => vi.useRealTimers())

describe('useBolShipmentConfirmations', () => {
  it('loads the order\'s confirmations', async () => {
    vi.mocked(listBolShipmentConfirmations).mockResolvedValue([confirmation('success')])

    const { result } = renderHook(() => useBolShipmentConfirmations('tok', 'so-1', true))

    await waitFor(() => expect(result.current.confirmations).toEqual([confirmation('success')]))
    expect(listBolShipmentConfirmations).toHaveBeenCalledWith('tok', 'so-1')
    expect(result.current.error).toBeNull()
  })

  it('does nothing when disabled (not a Bol order)', () => {
    const { result } = renderHook(() => useBolShipmentConfirmations('tok', 'so-1', false))

    expect(listBolShipmentConfirmations).not.toHaveBeenCalled()
    expect(result.current.confirmations).toEqual([])
  })

  it('surfaces the API error', async () => {
    vi.mocked(listBolShipmentConfirmations).mockRejectedValue(new Error('sales order not found'))

    const { result } = renderHook(() => useBolShipmentConfirmations('tok', 'so-1', true))

    await waitFor(() => expect(result.current.error).toBe('sales order not found'))
  })

  it('checks again while Bol is still processing, and stops once Bol answered', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
    vi.mocked(listBolShipmentConfirmations)
      .mockResolvedValueOnce([confirmation('pending')])
      .mockResolvedValueOnce([confirmation('success')])

    const { result } = renderHook(() => useBolShipmentConfirmations('tok', 'so-1', true))
    await waitFor(() => expect(result.current.confirmations[0]?.status).toBe('pending'))

    await act(async () => { await vi.advanceTimersByTimeAsync(BOL_CONFIRMATION_POLL_MS) })
    await waitFor(() => expect(result.current.confirmations[0]?.status).toBe('success'))

    await act(async () => { await vi.advanceTimersByTimeAsync(BOL_CONFIRMATION_POLL_MS * 3) })
    expect(listBolShipmentConfirmations).toHaveBeenCalledTimes(2)
  })
})
