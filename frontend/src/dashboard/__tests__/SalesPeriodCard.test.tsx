// @vitest-environment jsdom
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { AmountVisibilityProvider } from '../../format/amountVisibility'
import type { SalesComparison, SalesFigures, SalesPeriod } from '../types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../api', () => ({ getSalesComparison: vi.fn() }))

import * as api from '../api'
import { SalesPeriodCard } from '../SalesPeriodCard'

function figures(overrides: Partial<SalesFigures> = {}): SalesFigures {
  return { from: '', to: '', revenue_minor_units: 0, order_count: 0, by_channel: [], ...overrides }
}

function comparison(overrides: Partial<SalesComparison> = {}): SalesComparison {
  return {
    period: 'this_month', currency_code: 'EUR', change_percent: null,
    current: figures(), previous: figures(), ...overrides,
  }
}

function Where() {
  const l = useLocation()
  return <p data-testid="where">{l.search}</p>
}

function renderCard({ url = '/', privacy = false } = {}) {
  const card = <><SalesPeriodCard /><Where /></>
  return render(
    <MemoryRouter initialEntries={[url]}>
      <Routes>
        <Route path="*" element={privacy ? <AmountVisibilityProvider>{card}</AmountVisibilityProvider> : card} />
      </Routes>
    </MemoryRouter>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(api.getSalesComparison).mockImplementation((_t: string, period: SalesPeriod) => Promise.resolve(comparison({ period })))
})

describe('SalesPeriodCard', () => {
  it('shows this month against the whole of last month by default', async () => {
    vi.mocked(api.getSalesComparison).mockResolvedValue(comparison({
      change_percent: 20,
      current: figures({ revenue_minor_units: 1018678, order_count: 3 }),
      previous: figures({ revenue_minor_units: 848898, order_count: 2 }),
    }))
    renderCard()

    expect(await screen.findByText('€10,186.78')).toBeInTheDocument()
    expect(api.getSalesComparison).toHaveBeenCalledWith('test-token', 'this_month')
    expect(screen.getByRole('button', { name: 'This month' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByText('+20% vs last month')).toBeInTheDocument()
    expect(screen.getByText('3 orders')).toBeInTheDocument()
    expect(screen.getByText(/Last month/)).toHaveTextContent('Last month: €8,488.98')
  })

  it('switches period and keeps it in the URL', async () => {
    renderCard()
    await screen.findByRole('button', { name: 'This week' })

    await userEvent.click(screen.getByRole('button', { name: 'This week' }))

    expect(screen.getByTestId('where')).toHaveTextContent('?sales=week')
    expect(api.getSalesComparison).toHaveBeenLastCalledWith('test-token', 'this_week')
    expect(screen.getByRole('button', { name: 'This week' })).toHaveAttribute('aria-pressed', 'true')
  })

  it('opens on the period in the URL', async () => {
    vi.mocked(api.getSalesComparison).mockResolvedValue(comparison({ period: 'today', change_percent: -12.4 }))
    renderCard({ url: '/?sales=today' })

    expect(await screen.findByText('-12% vs yesterday')).toBeInTheDocument()
    expect(api.getSalesComparison).toHaveBeenCalledWith('test-token', 'today')
  })

  it('says when there is nothing to compare with', async () => {
    renderCard({ url: '/?sales=week' })

    expect(await screen.findByText('No sales last week to compare with')).toBeInTheDocument()
  })

  it('compares each channel with the previous period, including channels that stopped selling', async () => {
    vi.mocked(api.getSalesComparison).mockResolvedValue(comparison({
      current: figures({ by_channel: [{ source: 'pos', total_minor_units: 710559 }] }),
      previous: figures({ by_channel: [{ source: 'pos', total_minor_units: 500000 }, { source: 'bol', total_minor_units: 5998 }] }),
    }))
    renderCard()

    const pos = await screen.findByRole('listitem', { name: 'POS' })
    expect(within(pos).getByText('€7,105.59')).toBeInTheDocument()
    expect(within(pos).getByText(/Last month/)).toHaveTextContent('Last month: €5,000.00')
    const bol = screen.getByRole('listitem', { name: 'Bol.com' })
    expect(within(bol).getByText('€0.00')).toBeInTheDocument()
    expect(within(bol).getByText(/Last month/)).toHaveTextContent('Last month: €59.98')
  })

  it('shows an empty state when no channel sold in either period', async () => {
    renderCard({ url: '/?sales=today' })

    expect(await screen.findByText('No sales yet today.')).toBeInTheDocument()
  })

  it('keeps amounts hidden but always shows the percentage', async () => {
    vi.mocked(api.getSalesComparison).mockResolvedValue(comparison({
      change_percent: 5, current: figures({ revenue_minor_units: 123456 }),
    }))
    renderCard({ privacy: true })

    expect(await screen.findByText('+5% vs last month')).toBeInTheDocument()
    expect(screen.queryByText(/1,234\.56/)).not.toBeInTheDocument()

    await userEvent.click(screen.getAllByRole('button', { name: 'Show amount' })[0])

    expect(screen.getByText(/1,234\.56/)).toBeInTheDocument()
  })

  it('shows an API error', async () => {
    vi.mocked(api.getSalesComparison).mockRejectedValue(new Error('boom'))
    renderCard()

    expect(await screen.findByRole('alert')).toBeInTheDocument()
  })
})
