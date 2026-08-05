// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import Dashboard from '../Dashboard'
import type { ExpiringBatch } from '../../inventory/types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../inventory/api', () => ({
  listExpiringBatches: vi.fn(),
}))

import * as inventoryApi from '../../inventory/api'

function batch(overrides: Partial<ExpiringBatch> = {}): ExpiringBatch {
  return {
    batch_id: 'batch-1', sku_id: 'sku-1', sku_code: 'SKU-1', product_name: 'Widget',
    warehouse_id: 'wh-1', warehouse_name: 'Main DC', lot_number: 'LOT-1',
    expiry_date: '2026-09-01T00:00:00Z', quantity: 10,
    ...overrides,
  }
}

// jsdom doesn't implement ResizeObserver, which recharts' ResponsiveContainer
// (used elsewhere on this page) requires to mount at all.
class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.stubGlobal('ResizeObserver', ResizeObserverStub)
})

describe('Dashboard — Expiring Soon widget', () => {
  it('shows the empty state when nothing is expiring', async () => {
    vi.mocked(inventoryApi.listExpiringBatches).mockResolvedValue([])
    render(<Dashboard />)
    expect(await screen.findByText('Nothing expiring in the next 30 days.')).toBeInTheDocument()
  })

  it('renders real batches from the API, not mock data', async () => {
    vi.mocked(inventoryApi.listExpiringBatches).mockResolvedValue([batch()])
    render(<Dashboard />)

    expect(await screen.findByText('Widget')).toBeInTheDocument()
    expect(screen.getByText(/SKU-1/)).toBeInTheDocument()
    expect(screen.getByText(/lot LOT-1/)).toBeInTheDocument()
    expect(screen.getByText('10 units')).toBeInTheDocument()
    expect(inventoryApi.listExpiringBatches).toHaveBeenCalledWith('test-token', 30)
  })

  it('surfaces an API error', async () => {
    vi.mocked(inventoryApi.listExpiringBatches).mockRejectedValue(new Error('internal server error'))
    render(<Dashboard />)
    expect(await screen.findByText('internal server error')).toBeInTheDocument()
  })
})
