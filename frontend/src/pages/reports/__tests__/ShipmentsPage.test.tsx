// @vitest-environment jsdom
import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import ShipmentsPage from '../ShipmentsPage'
import { renderAt } from '../../../test/renderAt'
import type { ShipmentsReport } from '../../../reports/types'

vi.mock('../../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../../reports/api', () => ({
  getShipmentsReport: vi.fn(),
}))

import * as reportsApi from '../../../reports/api'

class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}

function report(overrides: Partial<ShipmentsReport> = {}): ShipmentsReport {
  return {
    granularity: 'month', basis: 'label_created',
    points: [
      { period_start: '2026-08-01T00:00:00Z', shipments: 10, orders: 9 },
      { period_start: '2026-09-01T00:00:00Z', shipments: 15, orders: 14 },
    ],
    total_shipments: 25, total_orders: 23,
    by_carrier: [{ carrier: 1, name: 'PostNL', shipments: 20 }, { carrier: 4, name: 'DPD', shipments: 5 }],
    by_channel: [{ key: 'bol', shipments: 15 }, { key: 'woocommerce', shipments: 10 }],
    by_country: [{ key: 'NL', shipments: 25 }],
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.stubGlobal('ResizeObserver', ResizeObserverStub)
  vi.mocked(reportsApi.getShipmentsReport).mockResolvedValue(report())
})

describe('ShipmentsPage', () => {
  it('fetches weeks by label date over the last 12 weeks by default', async () => {
    renderAt(<ShipmentsPage />, '/reports/shipments')

    await screen.findByTestId('total-shipments')
    const [, from, to, granularity, basis] = vi.mocked(reportsApi.getShipmentsReport).mock.calls[0]
    expect(granularity).toBe('week')
    expect(basis).toBe('label_created')
    const days = (new Date(to).getTime() - new Date(from).getTime()) / 86_400_000
    expect(days).toBeGreaterThan(77)
    expect(days).toBeLessThanOrEqual(85)
  })

  it('reads grouping, basis and range from the URL', async () => {
    renderAt(<ShipmentsPage />, '/reports/shipments', '/reports/shipments?granularity=quarter&basis=carrier_scan&from=2026-01-01&to=2026-06-30')

    await screen.findByTestId('total-shipments')
    expect(reportsApi.getShipmentsReport).toHaveBeenCalledWith('test-token', expect.stringMatching(/^2025-12-31T2|^2026-01-01T/), expect.stringMatching(/^2026-06-30T2|^2026-07-01T/), 'quarter', 'carrier_scan')
  })

  it('refetches when the grouping or the basis changes', async () => {
    const user = userEvent.setup()
    renderAt(<ShipmentsPage />, '/reports/shipments')
    await screen.findByTestId('total-shipments')

    await user.click(within(screen.getByRole('group', { name: 'Group by' })).getByRole('button', { name: 'Year' }))
    expect(reportsApi.getShipmentsReport).toHaveBeenLastCalledWith('test-token', expect.any(String), expect.any(String), 'year', 'label_created')

    await user.click(screen.getByRole('button', { name: 'Carrier scan' }))
    expect(reportsApi.getShipmentsReport).toHaveBeenLastCalledWith('test-token', expect.any(String), expect.any(String), 'year', 'carrier_scan')
  })

  it('shows totals, the per-period table with change, and the split', async () => {
    renderAt(<ShipmentsPage />, '/reports/shipments', '/reports/shipments?granularity=month')

    expect(await screen.findByTestId('total-shipments')).toHaveTextContent('25')
    expect(screen.getByText('23')).toBeInTheDocument()
    expect(screen.getByText('+50.0%')).toBeInTheDocument()
    expect(within(screen.getByRole('list', { name: 'By carrier' })).getByText('PostNL')).toBeInTheDocument()
    expect(within(screen.getByRole('list', { name: 'By channel' })).getByText('Bol.com')).toBeInTheDocument()
    expect(within(screen.getByRole('list', { name: 'By country' })).getByText('Netherlands')).toBeInTheDocument()
  })

  it('shows an empty state without shipments', async () => {
    vi.mocked(reportsApi.getShipmentsReport).mockResolvedValue(report({ points: [], total_shipments: 0, total_orders: 0, by_carrier: [], by_channel: [], by_country: [] }))
    renderAt(<ShipmentsPage />, '/reports/shipments')

    expect((await screen.findAllByText('No shipments in this period.')).length).toBeGreaterThan(0)
  })
})
