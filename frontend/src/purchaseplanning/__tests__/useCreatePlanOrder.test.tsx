// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook } from '@testing-library/react'
import type { ReactNode } from 'react'
import { MemoryRouter, useLocation } from 'react-router-dom'

vi.mock('../api', () => ({ createPurchaseOrderFromPlan: vi.fn() }))
import * as api from '../api'
import { useCreatePlanOrder } from '../hooks/useCreatePlanOrder'

const wrapper = ({ children }: { children: ReactNode }) => <MemoryRouter initialEntries={['/purchase-planner']}>{children}</MemoryRouter>

beforeEach(() => vi.clearAllMocks())

describe('useCreatePlanOrder', () => {
  it('creates the draft PO and opens it', async () => {
    vi.mocked(api.createPurchaseOrderFromPlan).mockResolvedValue({ purchase_order_id: 'po-9' })
    const { result } = renderHook(() => ({ create: useCreatePlanOrder('tok'), location: useLocation() }), { wrapper })

    await result.current.create('sup-1', [{ sku_id: 'sku-1', quantity: 3 }])

    expect(api.createPurchaseOrderFromPlan).toHaveBeenCalledWith('tok', { supplier_id: 'sup-1', lines: [{ sku_id: 'sku-1', quantity: 3 }] })
    await vi.waitFor(() => expect(result.current.location.pathname).toBe('/procurement/po-9'))
  })

  it('lets errors through for the caller to show', async () => {
    vi.mocked(api.createPurchaseOrderFromPlan).mockRejectedValue(new Error('insufficient role'))
    const { result } = renderHook(() => useCreatePlanOrder('tok'), { wrapper })

    await expect(result.current('sup-1', [])).rejects.toThrow('insufficient role')
  })

  it('fails instead of opening a PO without an id', async () => {
    vi.mocked(api.createPurchaseOrderFromPlan).mockResolvedValue({ purchase_order_id: '' })
    const { result } = renderHook(() => ({ create: useCreatePlanOrder('tok'), location: useLocation() }), { wrapper })

    await expect(result.current.create('sup-1', [])).rejects.toThrow()
    expect(result.current.location.pathname).toBe('/purchase-planner')
  })
})
