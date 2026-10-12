// @vitest-environment jsdom
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { AppProvider } from '../../context/AppContext'
import { AmountVisibilityProvider } from '../../format/amountVisibility'
import type { DashboardStatsReport, SalesComparison, TodayQueue } from '../types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../api', () => ({
  getTodayQueue: vi.fn(),
  getDashboardStats: vi.fn(),
  getSalesComparison: vi.fn(),
}))

vi.mock('../../inventory/api', () => ({
  listExpiringBatches: vi.fn().mockResolvedValue([]),
}))

import * as api from '../api'
import TodayPage from '../TodayPage'

class ResizeObserverStub { observe() {} unobserve() {} disconnect() {} }

function queue(overrides: Partial<TodayQueue> = {}): TodayQueue {
  return { to_pick: 0, to_pack: 0, stock_issues: 0, unmatched_skus: 0, expiring_batches: 0, expired_batches: 0, ship_today: 0, ...overrides }
}

function report(overrides: Partial<DashboardStatsReport> = {}): DashboardStatsReport {
  return {
    total_stock_units: 0, currency_code: 'EUR', revenue_this_month_minor_units: 0,
    stock_flow_last_7_days: [], sales_by_channel_this_month: [], top_products_this_month: [],
    ...overrides,
  }
}

function sales(overrides: Partial<SalesComparison> = {}): SalesComparison {
  const figures = { from: '', to: '', revenue_minor_units: 0, order_count: 0, by_channel: [] }
  return { period: 'this_month', currency_code: 'EUR', change_percent: null, current: figures, previous: figures, ...overrides }
}

function Where() {
  const l = useLocation()
  return <p data-testid="where">{l.pathname + l.search}</p>
}

function renderToday({ privacy = false } = {}) {
  const page = <><TodayPage /><Where /></>
  return render(
    <MemoryRouter initialEntries={['/']}>
      <AppProvider>
        <Routes>
          <Route path="*" element={privacy ? <AmountVisibilityProvider>{page}</AmountVisibilityProvider> : page} />
        </Routes>
      </AppProvider>
    </MemoryRouter>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.stubGlobal('ResizeObserver', ResizeObserverStub)
  vi.mocked(api.getDashboardStats).mockResolvedValue(report())
  vi.mocked(api.getSalesComparison).mockResolvedValue(sales())
})

describe('TodayPage work queue', () => {
  it('shows how many orders wait to be picked, packed and shipped today', async () => {
    vi.mocked(api.getTodayQueue).mockResolvedValue(queue({ to_pick: 14, to_pack: 6, ship_today: 9 }))
    renderToday()

    const board = await screen.findByRole('region', { name: 'Work queue' })
    expect(await within(board).findByText('14')).toBeInTheDocument()
    expect(within(board).getByText('6')).toBeInTheDocument()
    expect(within(board).getByText('9')).toBeInTheDocument()
    expect(within(board).getByText('orders to ship today')).toBeInTheDocument()
  })

  it('opens the sales orders due to ship today', async () => {
    vi.mocked(api.getTodayQueue).mockResolvedValue(queue({ ship_today: 1 }))
    renderToday()

    const board = await screen.findByRole('region', { name: 'Work queue' })
    expect(await within(board).findByText('order to ship today')).toBeInTheDocument()
    await userEvent.click(within(board).getByRole('button', { name: 'View' }))

    expect(screen.getByTestId('where').textContent).toBe('/sales-orders?ship_due=true')
  })

  it('starts picking from the queue', async () => {
    vi.mocked(api.getTodayQueue).mockResolvedValue(queue({ to_pick: 3 }))
    renderToday()

    await userEvent.click(await screen.findByRole('button', { name: 'Start picking' }))

    expect(screen.getByTestId('where').textContent).toBe('/picking')
  })

  it('starts packing from the queue', async () => {
    vi.mocked(api.getTodayQueue).mockResolvedValue(queue({ to_pack: 2 }))
    renderToday()

    await userEvent.click(await screen.findByRole('button', { name: 'Start packing' }))

    expect(screen.getByTestId('where').textContent).toBe('/packing')
  })
})

describe('TodayPage needs attention', () => {
  it('lists only what needs doing, each with its action', async () => {
    vi.mocked(api.getTodayQueue).mockResolvedValue(queue({ expiring_batches: 3, stock_issues: 1 }))
    renderToday()

    const list = await screen.findByRole('region', { name: 'Needs attention' })
    expect(await within(list).findByText('3 batches expire within 7 days')).toBeInTheDocument()
    expect(within(list).getByText('1 order waiting on stock')).toBeInTheDocument()
    expect(within(list).queryByText(/unmatched/i)).not.toBeInTheDocument()
    expect(within(list).queryByText(/expired/i)).not.toBeInTheDocument()
  })

  it('opens the expiring tab of the expiry review', async () => {
    vi.mocked(api.getTodayQueue).mockResolvedValue(queue({ expiring_batches: 3 }))
    renderToday()

    await userEvent.click(await screen.findByRole('button', { name: 'Review expiring batches' }))

    expect(screen.getByTestId('where').textContent).toBe('/inventory/expiry?tab=expiring')
  })

  it('routes each kind of problem to the page that fixes it', async () => {
    vi.mocked(api.getTodayQueue).mockResolvedValue(queue({ expired_batches: 2, unmatched_skus: 4, stock_issues: 1 }))
    renderToday()

    await userEvent.click(await screen.findByRole('button', { name: 'Write off expired batches' }))
    expect(screen.getByTestId('where').textContent).toBe('/inventory/expiry?tab=expired')
  })

  it('says so when nothing needs attention', async () => {
    vi.mocked(api.getTodayQueue).mockResolvedValue(queue())
    renderToday()

    expect(await screen.findByText('Nothing needs attention.')).toBeInTheDocument()
  })

  it('shows the error when the counts cannot load', async () => {
    vi.mocked(api.getTodayQueue).mockRejectedValue(new Error('boom'))
    renderToday()

    expect(await screen.findByRole('alert')).toBeInTheDocument()
  })
})

describe('TodayPage money', () => {
  it('hides the sales card\'s revenue until revealed, but shows the change', async () => {
    vi.mocked(api.getTodayQueue).mockResolvedValue(queue())
    vi.mocked(api.getSalesComparison).mockResolvedValue(sales({
      change_percent: 8,
      current: { from: '', to: '', revenue_minor_units: 123456, order_count: 2, by_channel: [] },
    }))
    renderToday({ privacy: true })

    expect(await screen.findByText('+8% vs last month')).toBeInTheDocument()
    expect(screen.queryByText(/1,234\.56/)).not.toBeInTheDocument()

    await userEvent.click(screen.getAllByRole('button', { name: 'Show amount' })[0])

    expect(screen.getByText(/1,234\.56/)).toBeInTheDocument()
  })
})
