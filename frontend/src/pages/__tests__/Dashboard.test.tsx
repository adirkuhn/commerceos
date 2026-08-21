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
  listPendingExpiredBatches: vi.fn(),
}))

vi.mock('../../pms/api', () => ({
  listProducts: vi.fn(),
}))

vi.mock('../../oms/api', () => ({
  listSalesOrders: vi.fn(),
}))

import * as inventoryApi from '../../inventory/api'
import * as pmsApi from '../../pms/api'
import * as omsApi from '../../oms/api'

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
  vi.mocked(inventoryApi.listExpiringBatches).mockResolvedValue([])
  vi.mocked(inventoryApi.listPendingExpiredBatches).mockResolvedValue([])
  vi.mocked(pmsApi.listProducts).mockResolvedValue({ products: [], total: 0 })
  vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [], total: 0 })
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

describe('Dashboard — stat cards', () => {
  it('renders real total product count', async () => {
    vi.mocked(pmsApi.listProducts).mockResolvedValue({ products: [], total: 2418 })
    render(<Dashboard />)
    expect(await screen.findByText('2,418')).toBeInTheDocument()
    expect(pmsApi.listProducts).toHaveBeenCalledWith('test-token', { perPage: 1 })
  })

  it('renders real total order count', async () => {
    vi.mocked(omsApi.listSalesOrders).mockImplementation(async (_token, params) => {
      if (params?.needsStockReview) return { salesOrders: [], total: 5 }
      return { salesOrders: [], total: 143 }
    })
    render(<Dashboard />)
    expect(await screen.findByText('143')).toBeInTheDocument()
    expect(omsApi.listSalesOrders).toHaveBeenCalledWith('test-token', { perPage: 1 })
  })

  it('renders real stock-issues count', async () => {
    vi.mocked(omsApi.listSalesOrders).mockImplementation(async (_token, params) => {
      if (params?.needsStockReview) return { salesOrders: [], total: 5 }
      return { salesOrders: [], total: 143 }
    })
    render(<Dashboard />)
    expect(await screen.findByText('5')).toBeInTheDocument()
    expect(omsApi.listSalesOrders).toHaveBeenCalledWith('test-token', { perPage: 1, needsStockReview: true })
  })

  it('renders real expiring-in-7-days count, distinct from the 30-day widget', async () => {
    vi.mocked(inventoryApi.listExpiringBatches).mockImplementation(async (_token, withinDays) =>
      withinDays === 7 ? [batch(), batch({ batch_id: 'batch-2' })] : [],
    )
    render(<Dashboard />)
    expect(await screen.findByText('2')).toBeInTheDocument()
    expect(inventoryApi.listExpiringBatches).toHaveBeenCalledWith('test-token', 7)
    expect(inventoryApi.listExpiringBatches).toHaveBeenCalledWith('test-token', 30)
  })

  it('renders real expired-batch count', async () => {
    vi.mocked(inventoryApi.listPendingExpiredBatches).mockResolvedValue([
      { batch_id: 'b1', sku_id: 's1', sku_code: 'SKU-1', product_name: 'Widget', warehouse_id: 'w1', warehouse_name: 'Main DC', lot_number: 'L1', expiry_date: '2026-01-01T00:00:00Z', quantity: 3 },
    ])
    render(<Dashboard />)
    expect(await screen.findByText('1')).toBeInTheDocument()
    expect(inventoryApi.listPendingExpiredBatches).toHaveBeenCalledWith('test-token')
  })

  it('shows "Coming soon" for widgets with no backend aggregate yet', async () => {
    render(<Dashboard />)
    const comingSoon = await screen.findAllByText('Coming soon')
    // Total Stock, Est. Revenue, Stock Flow, Sales by Channel, Top Products
    expect(comingSoon.length).toBe(5)
  })
})
