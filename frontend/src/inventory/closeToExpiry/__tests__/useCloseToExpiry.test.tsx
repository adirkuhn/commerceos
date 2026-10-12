// @vitest-environment jsdom
import { renderHook, act, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { createMemoryRouter, RouterProvider, useLocation } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useCloseToExpiry } from '../useCloseToExpiry'
import * as api from '../api'
import { CodedApiError } from '../../../api/codedApiError'
import type { CloseToExpiryBatch, CloseToExpiryList } from '../types'

vi.mock('../api')

const batch = { batch_id: 'b-1', days_left: 20 } as CloseToExpiryBatch

function list(overrides: Partial<CloseToExpiryList> = {}): CloseToExpiryList {
  const window = { window_days: 30, min_days_left: 16, from: '2026-10-28', to: '2026-11-11', open_count: 1, acknowledged_count: 0 }
  return { today: '2026-10-12', window, windows: [window], items: [batch], page: 1, per_page: 50, total: 1, ...overrides }
}

function setup(url = '/inventory/expiry?tab=expiring') {
  let router!: ReturnType<typeof createMemoryRouter>
  const hook = renderHook(() => ({ h: useCloseToExpiry('tok'), loc: useLocation() }), {
    wrapper: ({ children }: { children: ReactNode }) => {
      router ??= createMemoryRouter([{ path: '*', element: children }], { initialEntries: [url] })
      return <RouterProvider router={router} />
    },
  })
  return hook
}

beforeEach(() => {
  vi.mocked(api.listCloseToExpiry).mockReset().mockResolvedValue(list())
  vi.mocked(api.acknowledgeBatchExpiry).mockReset().mockResolvedValue()
  vi.mocked(api.unacknowledgeBatchExpiry).mockReset().mockResolvedValue()
})

describe('useCloseToExpiry', () => {
  it('loads the window, list and page from the URL', async () => {
    const { result } = setup('/inventory/expiry?tab=expiring&window=30&seen=1&page=2')
    await waitFor(() => expect(result.current.h.data).not.toBeNull())
    expect(api.listCloseToExpiry).toHaveBeenCalledWith('tok', { window: 30, acknowledged: true, page: 2 })
    expect(result.current.h.seen).toBe(true)
  })

  it('ignores a malformed window', async () => {
    setup('/inventory/expiry?window=abc')
    await waitFor(() => expect(api.listCloseToExpiry).toHaveBeenCalledWith('tok', { window: undefined, acknowledged: false, page: 1 }))
  })

  it('choosing a window or list resets the page and keeps the tab', async () => {
    const { result } = setup('/inventory/expiry?tab=expiring&page=3')
    await waitFor(() => expect(result.current.h.data).not.toBeNull())
    act(() => result.current.h.setWindow(15))
    expect(result.current.loc.search).toBe('?tab=expiring&window=15')
    act(() => result.current.h.setSeen(true))
    expect(result.current.loc.search).toBe('?tab=expiring&window=15&seen=1')
  })

  it('acknowledges for the listed window, then reloads', async () => {
    const { result } = setup()
    await waitFor(() => expect(result.current.h.data).not.toBeNull())
    await act(() => result.current.h.acknowledge(batch))
    expect(api.acknowledgeBatchExpiry).toHaveBeenCalledWith('tok', 'b-1', 30)
    expect(api.listCloseToExpiry).toHaveBeenCalledTimes(2)
    expect(result.current.h.actionError).toBe('')
  })

  it('undo removes the listed window acknowledgement', async () => {
    const { result } = setup()
    await waitFor(() => expect(result.current.h.data).not.toBeNull())
    await act(() => result.current.h.undo(batch))
    expect(api.unacknowledgeBatchExpiry).toHaveBeenCalledWith('tok', 'b-1', 30)
  })

  it('surfaces a failed action and a failed load', async () => {
    vi.mocked(api.acknowledgeBatchExpiry).mockRejectedValue(new CodedApiError('BATCH_NOT_IN_EXPIRY_WINDOW', 'moved'))
    const { result } = setup()
    await waitFor(() => expect(result.current.h.data).not.toBeNull())
    await act(() => result.current.h.acknowledge(batch))
    expect(result.current.h.actionError).not.toBe('')

    vi.mocked(api.listCloseToExpiry).mockRejectedValue(new Error('down'))
    await act(() => result.current.h.refresh())
    expect(result.current.h.error).not.toBeNull()
  })
})
