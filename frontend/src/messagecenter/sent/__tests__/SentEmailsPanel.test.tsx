// @vitest-environment jsdom
import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderAt } from '../../../test/renderAt'
import { acceptConfirm } from '../../../test/confirm'
import SentEmailsPanel from '../SentEmailsPanel'
import type { SentEmail, SentEmailList, SentEmailSummary } from '../../types'

vi.mock('../../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'tok', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../api', () => ({
  listSentEmails: vi.fn(),
  getSentEmail: vi.fn(),
  deleteSentEmail: vi.fn(),
  deleteSentEmails: vi.fn(),
}))

import * as api from '../../api'

function entry(overrides: Partial<SentEmailSummary> = {}): SentEmailSummary {
  return {
    id: 'e1', source: 'automation', recipient: 'ana@example.com', subject: 'Order #1001 confirmed', status: 'sent', reason: '',
    rule_id: 'r1', template_id: 't1', trigger_subject: 'storefront.order.placed', created_at: '2026-10-12T09:30:00Z', ...overrides,
  }
}

function page(items: SentEmailSummary[], total = items.length): SentEmailList {
  return { items, total, page: 1, per_page: 50 }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(api.listSentEmails).mockResolvedValue(page([
    entry(),
    entry({ id: 'e2', status: 'failed', reason: '550 mailbox unavailable', subject: 'Stock report', source: 'notification' }),
    entry({ id: 'e3', status: 'skipped', reason: 'no_recipient', recipient: '', subject: 'Status update' }),
  ]))
})

describe('SentEmailsPanel', () => {
  it('lists the history with status, source and why an email failed or was skipped', async () => {
    renderAt(<SentEmailsPanel />, '/message-center/sent')

    expect(await screen.findByText('Order #1001 confirmed')).toBeInTheDocument()
    expect(screen.getByText('3 emails')).toBeInTheDocument()
    expect(screen.getByText('550 mailbox unavailable')).toBeInTheDocument()
    expect(screen.getByText('Nobody to send to (no customer email and no extra recipients)')).toBeInTheDocument()
    expect(screen.getByRole('cell', { name: 'Staff notification' })).toBeInTheDocument()
    expect(api.listSentEmails).toHaveBeenCalledWith('tok', expect.objectContaining({ page: 1, per_page: 50 }))
  })

  it('filters live in the URL and go to the API', async () => {
    const { router } = renderAt(<SentEmailsPanel />, '/message-center/sent', '/message-center/sent?page=2')
    await screen.findByText('Order #1001 confirmed')

    await userEvent.selectOptions(screen.getByLabelText('Status'), 'failed')

    await waitFor(() => expect(api.listSentEmails).toHaveBeenLastCalledWith('tok', expect.objectContaining({ status: 'failed', page: 1 })))
    expect(router.state.location.search).toBe('?status=failed')
  })

  it('a date filter covers whole days', async () => {
    renderAt(<SentEmailsPanel />, '/message-center/sent', '/message-center/sent?from=2026-10-01&to=2026-10-01')

    await waitFor(() => expect(api.listSentEmails).toHaveBeenCalled())
    const filter = vi.mocked(api.listSentEmails).mock.calls[0][1]
    expect(new Date(filter.to!).getTime() - new Date(filter.from!).getTime()).toBe(24 * 60 * 60 * 1000)
  })

  it('deletes the selected emails after asking', async () => {
    vi.mocked(api.deleteSentEmails).mockResolvedValue({ deleted: 2 })
    renderAt(<SentEmailsPanel />, '/message-center/sent')
    await screen.findByText('Order #1001 confirmed')

    await userEvent.click(screen.getByLabelText('Select “Order #1001 confirmed”'))
    await userEvent.click(screen.getByLabelText('Select “Stock report”'))
    await userEvent.click(screen.getByText('Delete 2'))
    await acceptConfirm()

    await waitFor(() => expect(api.deleteSentEmails).toHaveBeenCalledWith('tok', ['e1', 'e2']))
    await waitFor(() => expect(api.listSentEmails).toHaveBeenCalledTimes(2))
  })

  it('a list error is shown', async () => {
    vi.mocked(api.listSentEmails).mockRejectedValue(new Error('offline'))
    renderAt(<SentEmailsPanel />, '/message-center/sent')

    expect(await screen.findByRole('alert')).toHaveTextContent('offline')
  })

  it('opening a row shows the email as sent, and it can be deleted', async () => {
    const email: SentEmail = { ...entry(), html_body: '<p>Hi Ana</p>' }
    vi.mocked(api.getSentEmail).mockResolvedValue(email)
    vi.mocked(api.deleteSentEmail).mockResolvedValue(null)
    const { router } = renderAt(<SentEmailsPanel />, '/message-center/sent')

    await userEvent.click(await screen.findByText('Order #1001 confirmed'))

    expect(router.state.location.pathname).toBe('/message-center/sent/e1')
    const frame = await screen.findByTitle('Email as sent')
    expect(frame).toHaveAttribute('srcdoc', '<p>Hi Ana</p>')
    expect(frame).toHaveAttribute('sandbox', '')
    expect(screen.getByText(/Storefront order placed/)).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: /Delete/ }))
    await acceptConfirm()
    await waitFor(() => expect(api.deleteSentEmail).toHaveBeenCalledWith('tok', 'e1'))
    await waitFor(() => expect(router.state.location.pathname).toBe('/message-center/sent'))
  })

  it('a skipped email has no body to show', async () => {
    vi.mocked(api.getSentEmail).mockResolvedValue({ ...entry({ status: 'skipped', reason: 'template_disabled' }), html_body: '' })
    renderAt(<SentEmailsPanel />, '/message-center/sent', '/message-center/sent/e3')

    expect(await screen.findByText('Nothing was sent, so there is no message to show.')).toBeInTheDocument()
    const status = screen.getByText('Status').nextElementSibling as HTMLElement
    expect(within(status).getByText(/The template is switched off/)).toBeInTheDocument()
  })
})
