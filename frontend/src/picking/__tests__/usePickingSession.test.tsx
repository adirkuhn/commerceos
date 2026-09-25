// @vitest-environment jsdom
import type { ReactNode } from 'react'
import { renderHook, act, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { usePickingSession } from '../usePickingSession'
import type { SalesOrder } from '../../oms/types'
import type { PickingSessionDetail } from '../../scanner/types'

vi.mock('../../oms/api', () => ({ getSalesOrder: vi.fn() }))
vi.mock('../../scanner/api', () => ({
  startOrResumePicking: vi.fn(),
  getPickingSession: vi.fn(),
  pausePicking: vi.fn(),
  completePicking: vi.fn(),
  resetPickingSession: vi.fn(),
  touchPickingSession: vi.fn(),
  releasePickingSessionLock: vi.fn(),
  takeOverPickingSession: vi.fn(),
}))

import * as pickingApi from '../../scanner/api'

function order(id: string): SalesOrder {
  return { id, number: id, external_order_id: id } as SalesOrder
}

function detail(orderId: string, sessionId: string): PickingSessionDetail {
  return {
    session: { id: sessionId, sales_order_id: orderId, status: 'in_progress', started_at: '', completed_by: '' },
    line_items: [], scans: [],
  }
}

const wrapper = ({ children }: { children: ReactNode }) => <MemoryRouter initialEntries={['/picking']}>{children}</MemoryRouter>

function setup() {
  const refreshQueue = vi.fn().mockResolvedValue(undefined)
  const onSelectionReset = vi.fn()
  const hook = renderHook(() => usePickingSession({ token: 'tok', userId: 'me', refreshQueue, onSelectionReset }), { wrapper })
  return { ...hook, refreshQueue, onSelectionReset }
}

async function openOrder(result: ReturnType<typeof setup>['result'], id: string, sessionId: string) {
  vi.mocked(pickingApi.startOrResumePicking).mockResolvedValueOnce(detail(id, sessionId).session)
  vi.mocked(pickingApi.getPickingSession).mockResolvedValueOnce(detail(id, sessionId))
  await act(() => result.current.selectOrder(order(id)))
  await waitFor(() => expect(result.current.session?.session.id).toBe(sessionId))
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(pickingApi.releasePickingSessionLock).mockResolvedValue(null)
})

describe('usePickingSession', () => {
  it('opens an order by starting/resuming its session, resetting the pick sheet', async () => {
    const { result, onSelectionReset } = setup()
    await openOrder(result, 'so-1', 'ps-1')

    expect(result.current.activeOrder?.id).toBe('so-1')
    expect(pickingApi.startOrResumePicking).toHaveBeenCalledWith('tok', 'so-1')
    expect(onSelectionReset).toHaveBeenCalled()
  })

  it('pauses the in-progress session before switching to another order', async () => {
    const { result } = setup()
    await openOrder(result, 'so-1', 'ps-1')
    vi.mocked(pickingApi.pausePicking).mockResolvedValue(detail('so-1', 'ps-1').session)

    await openOrder(result, 'so-2', 'ps-2')

    expect(pickingApi.pausePicking).toHaveBeenCalledWith('tok', 'ps-1')
    expect(result.current.activeOrder?.id).toBe('so-2')
  })

  it('reports a failed start', async () => {
    vi.mocked(pickingApi.startOrResumePicking).mockRejectedValue(new Error('nope'))
    const { result } = setup()

    await act(() => result.current.selectOrder(order('so-1')))

    expect(result.current.error).toBe('nope')
    expect(result.current.sessionLoading).toBe(false)
  })

  it('keeps the review open with shortfalls when completion is short on stock', async () => {
    const { result, refreshQueue } = setup()
    await openOrder(result, 'so-1', 'ps-1')
    act(() => result.current.setConfirmingFinish(true))
    const shortfalls = [{ sku_id: 'sku-1' }] as NonNullable<Awaited<ReturnType<typeof pickingApi.completePicking>>['stock_shortfalls']>
    vi.mocked(pickingApi.completePicking).mockResolvedValue({ completed: false, stock_shortfalls: shortfalls } as Awaited<ReturnType<typeof pickingApi.completePicking>>)

    await act(() => result.current.handleConfirmComplete())

    expect(result.current.stockShortfalls).toEqual(shortfalls)
    expect(result.current.confirmingFinish).toBe(true)
    expect(result.current.completeResult).toBeNull()
    expect(refreshQueue).not.toHaveBeenCalled()
  })

  it('completes, closes the review and refreshes the queue', async () => {
    const { result, refreshQueue } = setup()
    await openOrder(result, 'so-1', 'ps-1')
    act(() => result.current.setConfirmingFinish(true))
    const done = { completed: true } as Awaited<ReturnType<typeof pickingApi.completePicking>>
    vi.mocked(pickingApi.completePicking).mockResolvedValue(done)

    await act(() => result.current.handleConfirmComplete())

    expect(pickingApi.completePicking).toHaveBeenCalledWith('tok', 'ps-1', undefined)
    expect(result.current.completeResult).toBe(done)
    expect(result.current.confirmingFinish).toBe(false)
    expect(refreshQueue).toHaveBeenCalled()
  })

  it('starting over resets the session and the pick sheet', async () => {
    const { result, onSelectionReset } = setup()
    await openOrder(result, 'so-1', 'ps-1')
    act(() => result.current.setResetConfirmOpen(true))
    onSelectionReset.mockClear()
    vi.mocked(pickingApi.resetPickingSession).mockResolvedValue(detail('so-1', 'ps-1').session)
    vi.mocked(pickingApi.getPickingSession).mockResolvedValueOnce(detail('so-1', 'ps-1'))

    await act(() => result.current.handleResetSession())

    expect(pickingApi.resetPickingSession).toHaveBeenCalledWith('tok', 'ps-1')
    expect(result.current.resetConfirmOpen).toBe(false)
    expect(onSelectionReset).toHaveBeenCalled()
  })
})
