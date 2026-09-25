// @vitest-environment jsdom
import { renderHook, act } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { usePackingFinish } from '../usePackingFinish'
import type { PackingSessionDetail, CompletePackingResult, StockShortfall } from '../../scanner/types'

vi.mock('../../scanner/api', () => ({ resetPackingSession: vi.fn(), completePacking: vi.fn() }))
import * as packingApi from '../../scanner/api'

const session = { session: { id: 'pack-1', sales_order_id: 'so-1', status: 'in_progress', started_at: '', completed_by: '' }, line_items: [], scans: [] } as PackingSessionDetail

function setup() {
  const opts = { token: 't', session, loadSession: vi.fn().mockResolvedValue(undefined), setError: vi.fn(), closePanels: vi.fn(), onCompleted: vi.fn().mockResolvedValue(undefined) }
  return { ...renderHook(() => usePackingFinish(opts)), opts }
}

beforeEach(() => vi.clearAllMocks())

describe('usePackingFinish', () => {
  it('a stock shortfall asks for decisions, then completes with them', async () => {
    const shortfall = { sku_id: 'sku-1' } as StockShortfall
    vi.mocked(packingApi.completePacking)
      .mockResolvedValueOnce({ completed: false, stock_shortfalls: [shortfall] } as CompletePackingResult)
      .mockResolvedValueOnce({ completed: true, missing_items: [] } as unknown as CompletePackingResult)
    const { result, opts } = setup()
    act(() => result.current.startFinish())

    await act(() => result.current.complete())
    expect(result.current.stockShortfalls).toEqual([shortfall])
    expect(result.current.completeResult).toBeNull()
    expect(opts.onCompleted).not.toHaveBeenCalled()

    await act(() => result.current.complete({ 'sku-1': 'with_item' }))
    expect(packingApi.completePacking).toHaveBeenLastCalledWith('t', 'pack-1', { 'sku-1': 'with_item' })
    expect(result.current.stockShortfalls).toBeNull()
    expect(result.current.completeResult?.completed).toBe(true)
    expect(result.current.confirmingFinish).toBe(false)
    expect(opts.onCompleted).toHaveBeenCalled()
  })

  it('a failed complete is reported', async () => {
    vi.mocked(packingApi.completePacking).mockRejectedValue(new Error('nope'))
    const { result, opts } = setup()
    await act(() => result.current.complete())
    expect(opts.setError).toHaveBeenLastCalledWith('nope')
    expect(result.current.completing).toBe(false)
  })

  it('reset closes the confirm and the panels, then reloads', async () => {
    vi.mocked(packingApi.resetPackingSession).mockResolvedValue(session.session)
    const { result, opts } = setup()
    act(() => result.current.openResetConfirm())
    await act(() => result.current.resetSession())
    expect(result.current.resetConfirmOpen).toBe(false)
    expect(opts.closePanels).toHaveBeenCalled()
    expect(opts.loadSession).toHaveBeenCalledWith('pack-1')
  })

  it('a failed reset keeps the confirm open and reports it', async () => {
    vi.mocked(packingApi.resetPackingSession).mockRejectedValue(new Error('too late'))
    const { result, opts } = setup()
    act(() => result.current.openResetConfirm())
    await act(() => result.current.resetSession())
    expect(result.current.resetConfirmOpen).toBe(true)
    expect(result.current.resetting).toBe(false)
    expect(opts.setError).toHaveBeenLastCalledWith('too late')
  })
})
