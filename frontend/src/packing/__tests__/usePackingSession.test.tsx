// @vitest-environment jsdom
import type { ReactNode } from 'react'
import { renderHook, act, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { usePackingSession } from '../usePackingSession'
import type { PackingSessionDetail } from '../../scanner/types'
import type { SalesOrder } from '../../oms/types'

vi.mock('../../scanner/api', () => ({
  getPackingSession: vi.fn(), startOrResumePacking: vi.fn(), touchPackingSession: vi.fn(),
  releasePackingSessionLock: vi.fn(), takeOverPackingSession: vi.fn(),
}))
vi.mock('../../oms/api', () => ({ getSalesOrder: vi.fn() }))
import * as packingApi from '../../scanner/api'

const order = { id: 'so-1' } as SalesOrder
const detail = { session: { id: 'pack-1', sales_order_id: 'so-1', status: 'in_progress', started_at: '', completed_by: '' }, line_items: [], scans: [] } as PackingSessionDetail
const wrapper = ({ children }: { children: ReactNode }) => <MemoryRouter initialEntries={['/packing']}>{children}</MemoryRouter>

function setup() {
  const onSelect = vi.fn()
  const onCleared = vi.fn()
  const hook = renderHook(() => usePackingSession({ token: 't', me: 'me', onSelect, onCleared }), { wrapper })
  return { ...hook, onSelect, onCleared }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(packingApi.releasePackingSessionLock).mockResolvedValue(null)
  vi.mocked(packingApi.startOrResumePacking).mockResolvedValue(detail.session)
  vi.mocked(packingApi.getPackingSession).mockResolvedValue(detail)
})

async function openOrder(result: ReturnType<typeof setup>['result']) {
  await act(() => result.current.selectOrder(order))
  await waitFor(() => expect(result.current.session).not.toBeNull())
}

describe('usePackingSession', () => {
  it('with no session in the URL, reports the screen cleared', () => {
    const { onCleared } = setup()
    expect(onCleared).toHaveBeenCalled()
  })

  it('selecting an order resets the page and loads its session', async () => {
    const { result, onSelect } = setup()
    await openOrder(result)
    expect(onSelect).toHaveBeenCalled()
    expect(result.current.activeOrder).toBe(order)
    expect(result.current.sessionLoading).toBe(false)
  })

  it('claim touches the lock and reloads; a refusal reports who has it', async () => {
    const { result } = setup()
    await openOrder(result)
    vi.mocked(packingApi.touchPackingSession).mockResolvedValueOnce(null)
    await act(async () => { expect(await result.current.claim()).toBe(true) })

    vi.mocked(packingApi.touchPackingSession).mockRejectedValueOnce(new Error('held by u-9'))
    await act(async () => { expect(await result.current.claim()).toBe(false) })
    expect(result.current.error).toBe('held by u-9')
  })

  it('a failed take-over is reported', async () => {
    const { result } = setup()
    await openOrder(result)
    vi.mocked(packingApi.takeOverPackingSession).mockRejectedValue(new Error('not allowed'))
    await act(() => result.current.takeOver())
    expect(result.current.error).toBe('not allowed')
  })
})
