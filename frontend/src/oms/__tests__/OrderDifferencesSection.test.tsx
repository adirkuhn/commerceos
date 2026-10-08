// @vitest-environment jsdom
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { OrderDifferencesSection } from '../OrderDifferencesSection'
import { renderAt } from '../../test/renderAt'
import type { OrderDifference } from '../orderDifferences'
import type { Connection } from '../../integrations/types'

vi.mock('../orderDifferences', async importOriginal => ({
  ...(await importOriginal<typeof import('../orderDifferences')>()),
  listOrderDifferences: vi.fn(),
  dismissOrderDifference: vi.fn(),
  auditWooOrderDifferences: vi.fn(),
}))
vi.mock('../../integrations/api', () => ({ listConnections: vi.fn(), disconnectConnection: vi.fn() }))
import { listOrderDifferences, dismissOrderDifference, auditWooOrderDifferences } from '../orderDifferences'
import * as integrationsApi from '../../integrations/api'

function difference(overrides: Partial<OrderDifference> = {}): OrderDifference {
  return {
    id: 'd-1', sales_order_id: 'so-1', order_number: '25922', customer_name: 'Ana Silva',
    order_occurred_at: '2026-10-06T10:20:00Z', kind: 'missing', sku_code: '7896015976606',
    name: 'Mistura de Pão de Queijo - Pinduca - 500g', channel_quantity: 1, local_quantity: 0,
    channel_total_minor_units: 0, local_total_minor_units: 0, currency_code: 'EUR', detected_at: '2026-10-09T10:00:00Z',
    ...overrides,
  }
}

const woo = { id: 'conn-1', provider: 'woocommerce', store_url: 'https://shop.example', status: 'connected' } as Connection

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(integrationsApi.listConnections).mockResolvedValue([woo])
  vi.mocked(listOrderDifferences).mockResolvedValue({ differences: [], total: 0 })
})

describe('OrderDifferencesSection', () => {
  it('lists each difference with its order, customer and what is missing', async () => {
    vi.mocked(listOrderDifferences).mockResolvedValue({ differences: [difference()], total: 1 })

    renderAt(<OrderDifferencesSection token="tok" />, '/stock-issues')

    expect(await screen.findByText('Missing in CommerceOS: Mistura de Pão de Queijo - Pinduca - 500g (7896015976606) × 1')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: '#25922' })).toHaveAttribute('href', '/sales-orders/so-1')
    expect(screen.getByText('Ana Silva')).toBeInTheDocument()
  })

  it('describes a quantity difference and a total gap', async () => {
    vi.mocked(listOrderDifferences).mockResolvedValue({ total: 2, differences: [
      difference({ id: 'q', kind: 'quantity', channel_quantity: 3, local_quantity: 2, name: 'Apple', sku_code: 'A' }),
      difference({ id: 't', kind: 'total', channel_total_minor_units: 3641, local_total_minor_units: 3294 }),
    ] })

    renderAt(<OrderDifferencesSection token="tok" />, '/stock-issues')

    expect(await screen.findByText('Quantity differs: Apple (A) — channel 3, CommerceOS 2')).toBeInTheDocument()
    expect(screen.getByText(/Channel order total €36\.41, but its lines add up to €32\.94/)).toBeInTheDocument()
  })

  it('shows an empty state', async () => {
    renderAt(<OrderDifferencesSection token="tok" />, '/stock-issues')

    expect(await screen.findByText('No order differences — every compared order matches the channel.')).toBeInTheDocument()
  })

  it('dismisses a difference', async () => {
    vi.mocked(listOrderDifferences).mockResolvedValueOnce({ differences: [difference()], total: 1 }).mockResolvedValue({ differences: [], total: 0 })
    vi.mocked(dismissOrderDifference).mockResolvedValue(null)
    renderAt(<OrderDifferencesSection token="tok" />, '/stock-issues')

    await userEvent.click(await screen.findByRole('button', { name: 'Dismiss' }))

    expect(dismissOrderDifference).toHaveBeenCalledWith('tok', 'd-1')
    expect(await screen.findByText('No order differences — every compared order matches the channel.')).toBeInTheDocument()
  })

  it('compares all imported orders with Woo and shows progress', async () => {
    vi.mocked(auditWooOrderDifferences)
      .mockResolvedValueOnce({ checked: 25, errors: [], next_cursor: 'c-2' })
      .mockResolvedValueOnce({ checked: 4, errors: [], next_cursor: '' })
    renderAt(<OrderDifferencesSection token="tok" />, '/stock-issues')

    await userEvent.click(await screen.findByRole('button', { name: 'Compare all orders with Woo (https://shop.example)' }))

    expect(await screen.findByText('Compared 29 orders with Woo.')).toBeInTheDocument()
    expect(auditWooOrderDifferences).toHaveBeenCalledTimes(2)
  })

  it('surfaces an audit failure', async () => {
    vi.mocked(auditWooOrderDifferences).mockRejectedValue(new Error('connection not found'))
    renderAt(<OrderDifferencesSection token="tok" />, '/stock-issues')

    await userEvent.click(await screen.findByRole('button', { name: /Compare all orders with Woo/ }))

    await waitFor(() => expect(screen.getByText('connection not found')).toHaveClass('text-danger'))
  })
})
