// @vitest-environment jsdom
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import DeadLetters from '../DeadLetters'
import { CodedApiError } from '../../api/codedApiError'
import type { DeadLetter, DeadLetterPage, DeadLetterSummary } from '../../deadletters/types'

let mockRole = 'super_admin'
vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'tok', user: { role: mockRole }, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../deadletters/api', () => ({
  listDeadLetters: vi.fn(),
  getDeadLetter: vi.fn(),
  retryDeadLetter: vi.fn(),
  discardDeadLetter: vi.fn(),
}))

import * as api from '../../deadletters/api'

function summary(overrides: Partial<DeadLetterSummary> = {}): DeadLetterSummary {
  return {
    stream: 'events', sequence: 7, subject: 'oms.sales_order.status_changed',
    consumer: 'sales-order-status-changed-scheduler', consumer_known: true, event_id: 'e-1', tenant_id: 't-1',
    error: 'woo: 403 forbidden', num_delivered: 5, dead_lettered_at: '2026-11-03T10:00:00Z', stored_at: '2026-11-03T10:00:01Z',
    ...overrides,
  }
}

function detail(overrides: Partial<DeadLetter> = {}): DeadLetter {
  return { ...summary(), data: '{"event_id":"e-1","sales_order_id":"so-1"}', ...overrides }
}

function pageOf(items: DeadLetterSummary[], extra: Partial<DeadLetterPage> = {}): DeadLetterPage {
  return { items, next_cursor: null, total: items.length, ...extra }
}

// Shows the current URL so tests can assert navigation.
function Where() {
  const { pathname, search } = useLocation()
  return <div data-testid="where">{pathname + search}</div>
}

function renderAt(url: string) {
  return render(
    <MemoryRouter initialEntries={[url]}>
      <Routes>
        <Route path="/dead-letters/*" element={<><DeadLetters /><Where /></>} />
      </Routes>
    </MemoryRouter>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  mockRole = 'super_admin'
  vi.mocked(api.listDeadLetters).mockImplementation(async (_t, f) => {
    if (f.limit === 1) return pageOf([], { total: f.stream === 'events' ? 3 : 0 })
    return pageOf([summary(), summary({ sequence: 8, consumer: '', consumer_known: false, error: '' })], { total: 3 })
  })
})

describe('DeadLetters list', () => {
  it('is super_admin only and never calls the API otherwise', async () => {
    mockRole = 'customer_owner'

    renderAt('/dead-letters')

    expect(await screen.findByText('Super admin access required.')).toBeInTheDocument()
    expect(api.listDeadLetters).not.toHaveBeenCalled()
  })

  it('lists dead letters with a count per stream, unknown consumers marked', async () => {
    renderAt('/dead-letters')

    expect(await screen.findByText('woo: 403 forbidden')).toBeInTheDocument()
    expect(screen.getByText('unknown')).toBeInTheDocument()
    expect(await screen.findByTestId('count-events')).toHaveTextContent('3')
    expect(screen.getByTestId('count-webhooks')).toHaveTextContent('0')
    expect(api.listDeadLetters).toHaveBeenCalledWith('tok', expect.objectContaining({ stream: 'events' }))
  })

  it('reads stream and filters from the URL', async () => {
    renderAt('/dead-letters?stream=webhooks&consumer=c1&cursor=5')

    await screen.findByText('woo: 403 forbidden')
    expect(api.listDeadLetters).toHaveBeenCalledWith('tok', expect.objectContaining({ stream: 'webhooks', consumer: 'c1', cursor: 5 }))
    expect(screen.getByRole('tab', { name: /Webhooks/ })).toHaveAttribute('aria-selected', 'true')
  })

  it('applies filters to the URL and starts from the first page', async () => {
    renderAt('/dead-letters?cursor=5')
    await screen.findByText('woo: 403 forbidden')

    await userEvent.type(screen.getByLabelText('Tenant ID'), 't-9')
    await userEvent.click(screen.getByRole('button', { name: 'Filter' }))

    await waitFor(() => expect(screen.getByTestId('where')).toHaveTextContent('/dead-letters?tenant_id=t-9'))
    expect(api.listDeadLetters).toHaveBeenLastCalledWith('tok', expect.objectContaining({ tenantId: 't-9', cursor: undefined }))
  })

  it('pages with the next cursor', async () => {
    vi.mocked(api.listDeadLetters).mockImplementation(async (_t, f) =>
      f.limit === 1 ? pageOf([], { total: 60 }) : pageOf([summary()], { next_cursor: 57, total: 60 }))
    renderAt('/dead-letters')

    await userEvent.click(await screen.findByRole('button', { name: 'Next page' }))

    await waitFor(() => expect(screen.getByTestId('where')).toHaveTextContent('/dead-letters?cursor=57'))
  })

  it('opens a dead letter on its own URL', async () => {
    vi.mocked(api.getDeadLetter).mockResolvedValue(detail())
    renderAt('/dead-letters')

    await userEvent.click(await screen.findByRole('link', { name: '7' }))

    expect(screen.getByTestId('where')).toHaveTextContent('/dead-letters/events/7')
    expect(await screen.findByText('Dead letter #7')).toBeInTheDocument()
  })

  it('shows a load failure', async () => {
    vi.mocked(api.listDeadLetters).mockRejectedValue(new Error('network down'))

    renderAt('/dead-letters')

    expect(await screen.findAllByText('network down')).not.toHaveLength(0)
  })

  it('says when nothing is waiting', async () => {
    vi.mocked(api.listDeadLetters).mockResolvedValue(pageOf([]))

    renderAt('/dead-letters')

    expect(await screen.findByText('No dead letters. Nothing is waiting.')).toBeInTheDocument()
  })
})

describe('DeadLetters detail', () => {
  it('shows the failure details and the pretty-printed data', async () => {
    vi.mocked(api.getDeadLetter).mockResolvedValue(detail())

    renderAt('/dead-letters/events/7')

    expect(await screen.findByText('Dead letter #7')).toBeInTheDocument()
    expect(api.getDeadLetter).toHaveBeenCalledWith('tok', 'events', 7)
    expect(screen.getByTestId('dead-letter-data').textContent).toBe('{\n  "event_id": "e-1",\n  "sales_order_id": "so-1"\n}')
    expect(screen.getByText('woo: 403 forbidden')).toBeInTheDocument()
    expect(screen.getAllByText(/re-read the current state/).length).toBeGreaterThan(0)
  })

  it('retries for the failed consumer after confirming, then goes back to the list', async () => {
    vi.mocked(api.getDeadLetter).mockResolvedValue(detail())
    vi.mocked(api.retryDeadLetter).mockResolvedValue(null)
    renderAt('/dead-letters/events/7')

    await userEvent.click(await screen.findByRole('button', { name: 'Retry' }))
    const dialog = screen.getByRole('alertdialog')
    expect(within(dialog).getByText(/for sales-order-status-changed-scheduler only/)).toBeInTheDocument()
    await userEvent.click(within(dialog).getByRole('button', { name: 'Retry event' }))

    expect(api.retryDeadLetter).toHaveBeenCalledWith('tok', 'events', 7)
    await waitFor(() => expect(screen.getByTestId('where')).toHaveTextContent('/dead-letters?stream=events'))
  })

  it('warns that a retry with an unknown consumer reaches every consumer', async () => {
    vi.mocked(api.getDeadLetter).mockResolvedValue(detail({ consumer: '', consumer_known: false }))
    renderAt('/dead-letters/events/7')

    await userEvent.click(await screen.findByRole('button', { name: 'Retry' }))

    expect(within(screen.getByRole('alertdialog')).getByText(/for every consumer on oms.sales_order.status_changed/)).toBeInTheDocument()
  })

  it('does nothing when the confirmation is cancelled', async () => {
    vi.mocked(api.getDeadLetter).mockResolvedValue(detail())
    renderAt('/dead-letters/events/7')

    await userEvent.click(await screen.findByRole('button', { name: 'Discard' }))
    await userEvent.click(within(screen.getByRole('alertdialog')).getByRole('button', { name: 'Cancel' }))

    expect(api.discardDeadLetter).not.toHaveBeenCalled()
    expect(screen.getByTestId('where')).toHaveTextContent('/dead-letters/events/7')
  })

  it('discards after confirming', async () => {
    vi.mocked(api.getDeadLetter).mockResolvedValue(detail())
    vi.mocked(api.discardDeadLetter).mockResolvedValue(null)
    renderAt('/dead-letters/events/7')

    await userEvent.click(await screen.findByRole('button', { name: 'Discard' }))
    await userEvent.click(within(screen.getByRole('alertdialog')).getByRole('button', { name: 'Discard event' }))

    expect(api.discardDeadLetter).toHaveBeenCalledWith('tok', 'events', 7)
    await waitFor(() => expect(screen.getByTestId('where')).toHaveTextContent('/dead-letters?stream=events'))
  })

  it('keeps the page and shows the error when a retry fails', async () => {
    vi.mocked(api.getDeadLetter).mockResolvedValue(detail())
    vi.mocked(api.retryDeadLetter).mockRejectedValue(new CodedApiError('DEAD_LETTER_RETRY_FAILED', 'dead letter could not be retried', 'nats: timeout'))
    renderAt('/dead-letters/events/7')

    await userEvent.click(await screen.findByRole('button', { name: 'Retry' }))
    await userEvent.click(within(screen.getByRole('alertdialog')).getByRole('button', { name: 'Retry event' }))

    expect(await screen.findByText(/The event could not be published again/)).toBeInTheDocument()
    expect(screen.getByTestId('where')).toHaveTextContent('/dead-letters/events/7')
  })

  it('shows not found for a dead letter that is gone', async () => {
    vi.mocked(api.getDeadLetter).mockRejectedValue(new CodedApiError('DEAD_LETTER_NOT_FOUND', 'dead letter not found'))

    renderAt('/dead-letters/events/7')

    expect(await screen.findByText(/Dead letter not found/)).toBeInTheDocument()
  })
})
