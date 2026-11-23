// @vitest-environment jsdom
import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import Assistant from '../Assistant'
import * as api from '../../assistant/api'
import { CodedApiError } from '../../api/codedApiError'
import { aDetail, aMessage, aRequest } from '../../assistant/__tests__/fixtures'
import { renderAt } from '../../test/renderAt'

const auth = vi.hoisted(() => ({ role: 'customer_admin' }))
vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: { role: auth.role }, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))
vi.mock('../../assistant/api')

beforeEach(() => {
  auth.role = 'customer_admin'
  vi.mocked(api.listAssistantRequests).mockReset().mockResolvedValue({ requests: [], next_cursor: null })
  vi.mocked(api.getAssistantRequest).mockReset()
})

describe('Assistant home', () => {
  it('queues what was written and opens its page straight away', async () => {
    vi.mocked(api.createAssistantRequest).mockResolvedValue(aRequest({ id: 'r-new' }))
    const { router } = renderAt(<Assistant />, '/assistant')

    await userEvent.type(screen.getByLabelText('Ask or paste…'), 'Status of order CO-12?')
    await userEvent.click(screen.getByRole('button', { name: 'Ask' }))

    expect(api.createAssistantRequest).toHaveBeenCalledWith('test-token', 'Status of order CO-12?')
    await waitFor(() => expect(router.state.location.pathname).toBe('/assistant/r-new'))
  })

  it('shows why creating failed', async () => {
    vi.mocked(api.createAssistantRequest).mockRejectedValue(new CodedApiError('ASSISTANT_TEXT_TOO_LONG', 'too long'))
    renderAt(<Assistant />, '/assistant')

    await userEvent.type(screen.getByLabelText('Ask or paste…'), 'x')
    await userEvent.click(screen.getByRole('button', { name: 'Ask' }))

    expect(await screen.findByText('The text can be at most 20,000 characters.')).toBeInTheDocument()
  })

  it('lists requests with their status and why a queued one waits', async () => {
    vi.mocked(api.listAssistantRequests).mockResolvedValue({
      requests: [
        aRequest({ id: 'r-1', title: 'Status CO-12?', status: 'queued', wait_reason: 'model_busy', attempts: 1 }),
        aRequest({ id: 'r-2', title: 'Default tax?', status: 'answered' }),
      ],
      next_cursor: null,
    })
    renderAt(<Assistant />, '/assistant')

    const row = (await screen.findByText('Status CO-12?')).closest('li') as HTMLElement
    expect(within(row).getByText('Queued')).toBeInTheDocument()
    expect(within(row).getByText(/busy or out of quota/)).toBeInTheDocument()
    const answered = screen.getByText('Default tax?').closest('li') as HTMLElement
    expect(within(answered).getByText('Answered')).toBeInTheDocument()
    expect(within(answered).queryByRole('button', { name: 'Cancel' })).not.toBeInTheDocument()
  })

  it('filters by status through the URL', async () => {
    const { router } = renderAt(<Assistant />, '/assistant')

    await userEvent.selectOptions(await screen.findByLabelText('Status'), 'needs_input')

    await waitFor(() => expect(router.state.location.search).toBe('?status=needs_input'))
    await waitFor(() => expect(api.listAssistantRequests).toHaveBeenLastCalledWith('test-token', { status: 'needs_input' }))
  })

  it('cancels an unfinished request after confirming', async () => {
    vi.mocked(api.listAssistantRequests).mockResolvedValue({ requests: [aRequest({ id: 'r-1', title: 'Status CO-12?' })], next_cursor: null })
    vi.mocked(api.cancelAssistantRequest).mockResolvedValue(aRequest({ id: 'r-1', title: 'Status CO-12?', status: 'cancelled' }))
    renderAt(<Assistant />, '/assistant')

    const row = (await screen.findByText('Status CO-12?')).closest('li') as HTMLElement
    await userEvent.click(within(row).getByRole('button', { name: 'Cancel' }))
    await userEvent.click(within(screen.getByRole('alertdialog')).getByRole('button', { name: 'Cancel request' }))

    expect(api.cancelAssistantRequest).toHaveBeenCalledWith('test-token', 'r-1')
    expect(await within(row).findByText('Cancelled')).toBeInTheDocument()
  })

  it('loads more with the cursor', async () => {
    vi.mocked(api.listAssistantRequests)
      .mockResolvedValueOnce({ requests: [aRequest({ id: 'r-1', title: 'First' })], next_cursor: 'r-1' })
      .mockResolvedValueOnce({ requests: [aRequest({ id: 'r-2', title: 'Second' })], next_cursor: null })
    renderAt(<Assistant />, '/assistant')

    await userEvent.click(await screen.findByRole('button', { name: 'Load more' }))

    expect(await screen.findByText('Second')).toBeInTheDocument()
    expect(screen.getByText('First')).toBeInTheDocument()
  })

  it('is for admins only', () => {
    auth.role = 'customer_user'
    renderAt(<Assistant />, '/assistant')

    expect(screen.getByText('Only owners and admins can use the assistant.')).toBeInTheDocument()
    expect(api.listAssistantRequests).not.toHaveBeenCalled()
  })
})

describe('Assistant conversation', () => {
  it('shows the conversation: tool steps compact, question highlighted, notice translated, text not HTML', async () => {
    vi.mocked(api.getAssistantRequest).mockResolvedValue(aDetail({ status: 'needs_input' }, [
      aMessage({ seq: 1, content: 'Quote for <b>Ana</b>' }),
      aMessage({ seq: 2, role: 'assistant', content: '', tool_calls: [{ id: 'c1', name: 'search', arguments: '{}' }] }),
      aMessage({ seq: 3, role: 'tool', content: '', tool_name: 'search' }),
      aMessage({ seq: 4, role: 'assistant', content: 'Which Ana?\nAna Bakery or Ana Souza', question: true }),
      aMessage({ seq: 5, role: 'assistant', content: '', notice: 'too_many_steps', question: true }),
    ]))
    renderAt(<Assistant />, '/assistant', '/assistant/r-1')

    expect(await screen.findByText('Quote for <b>Ana</b>')).toBeInTheDocument()
    expect(screen.getByText('Looked up: search')).toBeInTheDocument()
    const question = screen.getByTestId('assistant-message-4')
    expect(question).toHaveTextContent('Which Ana?')
    expect(question).toHaveAttribute('data-question', 'true')
    expect(screen.getByText(/needed too many steps/)).toBeInTheDocument()
    expect(screen.getByLabelText('Your answer')).toBeInTheDocument()
  })

  it('while queued after a failed attempt: reason, next try and the last error', async () => {
    vi.mocked(api.getAssistantRequest).mockResolvedValue(aDetail({
      status: 'queued', attempts: 2, wait_reason: 'no_provider', last_error: 'no AI provider', next_attempt_at: '2026-11-23T12:05:00Z',
    }))
    renderAt(<Assistant />, '/assistant', '/assistant/r-1')

    expect(await screen.findByText(/No AI provider is connected/)).toBeInTheDocument()
    expect(screen.getByText(/Next try at/)).toBeInTheDocument()
    expect(screen.getByText('no AI provider')).toBeInTheDocument()
    expect(screen.queryByLabelText('Your answer')).not.toBeInTheDocument()
  })

  it('a follow-up on an answer queues it again', async () => {
    vi.mocked(api.getAssistantRequest).mockResolvedValue(aDetail({ status: 'answered' }, [
      aMessage(), aMessage({ seq: 2, role: 'assistant', content: 'CO-12 has shipped.' }),
    ]))
    vi.mocked(api.replyToAssistantRequest).mockResolvedValue(aRequest({ status: 'queued' }))
    renderAt(<Assistant />, '/assistant', '/assistant/r-1')

    await userEvent.type(await screen.findByLabelText('Ask a follow-up'), 'Which carrier?')
    await userEvent.click(screen.getByRole('button', { name: 'Send' }))

    expect(api.replyToAssistantRequest).toHaveBeenCalledWith('test-token', 'r-1', 'Which carrier?')
    expect(await screen.findByText('Waiting in the queue…')).toBeInTheDocument()
  })

  it('cancels after confirming', async () => {
    vi.mocked(api.getAssistantRequest).mockResolvedValue(aDetail({ status: 'needs_input' }))
    vi.mocked(api.cancelAssistantRequest).mockResolvedValue(aRequest({ status: 'cancelled' }))
    renderAt(<Assistant />, '/assistant', '/assistant/r-1')

    await userEvent.click(await screen.findByRole('button', { name: 'Cancel request' }))
    await userEvent.click(within(screen.getByRole('alertdialog')).getByRole('button', { name: 'Cancel request' }))

    expect(api.cancelAssistantRequest).toHaveBeenCalledWith('test-token', 'r-1')
    expect(await screen.findByText('Cancelled')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Cancel request' })).not.toBeInTheDocument()
  })

  it('shows a load error', async () => {
    vi.mocked(api.getAssistantRequest).mockRejectedValue(new CodedApiError('ASSISTANT_REQUEST_NOT_FOUND', 'nf'))
    renderAt(<Assistant />, '/assistant', '/assistant/r-1')

    expect(await screen.findByText('This assistant request doesn’t exist (any more).')).toBeInTheDocument()
  })

  it('links back to all requests', async () => {
    vi.mocked(api.getAssistantRequest).mockResolvedValue(aDetail({ status: 'answered' }))
    const { router } = renderAt(<Assistant />, '/assistant', '/assistant/r-1')

    await userEvent.click(await screen.findByRole('button', { name: 'All requests' }))

    await waitFor(() => expect(router.state.location.pathname).toBe('/assistant'))
  })
})
