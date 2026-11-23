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
  vi.mocked(api.retryAssistantRequest).mockReset()
  vi.mocked(api.createAssistantRequest).mockReset()
  vi.mocked(api.replyToAssistantRequest).mockReset()
  vi.mocked(api.cancelAssistantRequest).mockReset()
})

function history(): HTMLElement {
  return screen.getByRole('navigation', { name: 'Chats' })
}

async function historyRow(title: string): Promise<HTMLElement> {
  return (await within(history()).findByText(title)).closest('li') as HTMLElement
}

describe('new chat', () => {
  it('Enter sends: the chat is created, opens at once and shows in the history', async () => {
    vi.mocked(api.createAssistantRequest).mockResolvedValue(aRequest({ id: 'r-new', title: 'Status of order CO-12?' }))
    vi.mocked(api.getAssistantRequest).mockResolvedValue(aDetail({ id: 'r-new', title: 'Status of order CO-12?' }))
    const { router } = renderAt(<Assistant />, '/assistant')

    await userEvent.type(screen.getByLabelText('Ask or paste…'), 'Status of order CO-12?{Enter}')

    expect(api.createAssistantRequest).toHaveBeenCalledWith('test-token', 'Status of order CO-12?')
    await waitFor(() => expect(router.state.location.pathname).toBe('/assistant/r-new'))
    expect(await within(history()).findByText('Status of order CO-12?')).toBeInTheDocument()
  })

  it('Shift+Enter adds a line instead of sending', async () => {
    renderAt(<Assistant />, '/assistant')
    const box = screen.getByLabelText('Ask or paste…')

    await userEvent.type(box, 'line one{Shift>}{Enter}{/Shift}line two')

    expect(box).toHaveValue('line one\nline two')
    expect(api.createAssistantRequest).not.toHaveBeenCalled()
  })

  it('shows why creating failed', async () => {
    vi.mocked(api.createAssistantRequest).mockRejectedValue(new CodedApiError('ASSISTANT_TEXT_TOO_LONG', 'too long'))
    renderAt(<Assistant />, '/assistant')

    await userEvent.type(screen.getByLabelText('Ask or paste…'), 'x')
    await userEvent.click(screen.getByRole('button', { name: 'Ask' }))

    expect(await screen.findByText('The text can be at most 20,000 characters.')).toBeInTheDocument()
  })

  it('is for admins only', () => {
    auth.role = 'customer_user'
    renderAt(<Assistant />, '/assistant')

    expect(screen.getByText('Only owners and admins can use the assistant.')).toBeInTheDocument()
    expect(api.listAssistantRequests).not.toHaveBeenCalled()
  })
})

describe('chat history', () => {
  it('lists chats newest first with their status and why a queued one waits', async () => {
    vi.mocked(api.listAssistantRequests).mockResolvedValue({
      requests: [
        aRequest({ id: 'r-1', title: 'Status CO-12?', status: 'queued', wait_reason: 'model_busy', attempts: 1 }),
        aRequest({ id: 'r-2', title: 'Default tax?', status: 'answered' }),
      ],
      next_cursor: null,
    })
    renderAt(<Assistant />, '/assistant')

    const row = await historyRow('Status CO-12?')
    expect(within(row).getByText('Queued')).toBeInTheDocument()
    expect(within(row).getByText(/busy or out of quota/)).toBeInTheDocument()
    expect(within(await historyRow('Default tax?')).getByText('Answered')).toBeInTheDocument()
    const titles = within(history()).getAllByRole('link').map(a => a.textContent)
    expect(titles[0]).toContain('Status CO-12?')
  })

  it('opens a chat, and New chat goes back to an empty one', async () => {
    vi.mocked(api.listAssistantRequests).mockResolvedValue({ requests: [aRequest({ id: 'r-1', title: 'Status CO-12?', status: 'answered' })], next_cursor: null })
    vi.mocked(api.getAssistantRequest).mockResolvedValue(aDetail({ status: 'answered' }))
    const { router } = renderAt(<Assistant />, '/assistant')

    await userEvent.click(await within(history()).findByRole('link', { name: /Status CO-12\?/ }))
    await waitFor(() => expect(router.state.location.pathname).toBe('/assistant/r-1'))

    await userEvent.click(screen.getByRole('button', { name: 'New chat' }))
    await waitFor(() => expect(router.state.location.pathname).toBe('/assistant'))
    expect(screen.getByLabelText('Ask or paste…')).toBeInTheDocument()
  })

  it('cancels an unfinished chat from its menu after confirming', async () => {
    vi.mocked(api.listAssistantRequests).mockResolvedValue({ requests: [aRequest({ id: 'r-1', title: 'Status CO-12?' })], next_cursor: null })
    vi.mocked(api.cancelAssistantRequest).mockResolvedValue(aRequest({ id: 'r-1', title: 'Status CO-12?', status: 'cancelled' }))
    renderAt(<Assistant />, '/assistant')

    const row = await historyRow('Status CO-12?')
    await userEvent.click(within(row).getByRole('button', { name: 'Actions for Status CO-12?' }))
    await userEvent.click(screen.getByRole('menuitem', { name: 'Cancel' }))
    await userEvent.click(within(screen.getByRole('alertdialog')).getByRole('button', { name: 'Cancel request' }))

    expect(api.cancelAssistantRequest).toHaveBeenCalledWith('test-token', 'r-1')
    expect(await within(row).findByText('Cancelled')).toBeInTheDocument()
  })

  it('retries a waiting chat from its menu; finished ones have no retry', async () => {
    vi.mocked(api.listAssistantRequests).mockResolvedValue({
      requests: [
        aRequest({ id: 'r-1', title: 'Waiting one', status: 'queued', wait_reason: 'model_busy', attempts: 2 }),
        aRequest({ id: 'r-2', title: 'Done one', status: 'answered' }),
      ],
      next_cursor: null,
    })
    vi.mocked(api.retryAssistantRequest).mockResolvedValue(aRequest({ id: 'r-1', title: 'Waiting one', status: 'queued', attempts: 0 }))
    renderAt(<Assistant />, '/assistant')

    await userEvent.click(within(await historyRow('Waiting one')).getByRole('button', { name: 'Actions for Waiting one' }))
    await userEvent.click(screen.getByRole('menuitem', { name: 'Retry now' }))
    expect(api.retryAssistantRequest).toHaveBeenCalledWith('test-token', 'r-1')

    expect(within(await historyRow('Done one')).queryByRole('button', { name: 'Actions for Done one' })).not.toBeInTheDocument()
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

  it('folds away on small screens behind a Chats button', async () => {
    renderAt(<Assistant />, '/assistant')
    const toggle = screen.getByRole('button', { name: 'Chats' })

    expect(toggle).toHaveAttribute('aria-expanded', 'false')
    await userEvent.click(toggle)
    expect(toggle).toHaveAttribute('aria-expanded', 'true')
  })
})

describe('chat thread', () => {
  it('user text stays plain, answers render markdown with admin links, tool steps fold into one line', async () => {
    vi.mocked(api.getAssistantRequest).mockResolvedValue(aDetail({ status: 'answered' }, [
      aMessage({ seq: 1, content: 'Quote for <b>Ana</b> **now**' }),
      aMessage({ seq: 2, role: 'assistant', content: '', tool_calls: [{ id: 'c1', name: 'search', arguments: '{}' }] }),
      aMessage({ seq: 3, role: 'tool', content: '', tool_name: 'search' }),
      aMessage({ seq: 4, role: 'assistant', content: '', tool_calls: [{ id: 'c2', name: 'get_sales_order', arguments: '{}' }] }),
      aMessage({ seq: 5, role: 'tool', content: '', tool_name: 'get_sales_order' }),
      aMessage({ seq: 6, role: 'assistant', content: '**[CO-12](/sales-orders/so-1)** has shipped. [evil](https://evil.example)' }),
    ]))
    renderAt(<Assistant />, '/assistant', '/assistant/r-1')

    expect(await screen.findByText('Quote for <b>Ana</b> **now**')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'CO-12' })).toHaveAttribute('href', '/sales-orders/so-1')
    expect(screen.queryByRole('link', { name: 'evil' })).not.toBeInTheDocument()

    const steps = screen.getByRole('button', { name: 'Looked up 2 things' })
    expect(steps).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByText('get_sales_order')).not.toBeInTheDocument()
    await userEvent.click(steps)
    expect(screen.getByText('search')).toBeInTheDocument()
    expect(screen.getByText('get_sales_order')).toBeInTheDocument()
  })

  it('highlights a question and translates app notices', async () => {
    vi.mocked(api.getAssistantRequest).mockResolvedValue(aDetail({ status: 'needs_input' }, [
      aMessage({ seq: 1 }),
      aMessage({ seq: 2, role: 'assistant', content: 'Which Ana?', question: true }),
      aMessage({ seq: 3, role: 'assistant', content: '', notice: 'too_many_steps', question: true }),
    ]))
    renderAt(<Assistant />, '/assistant', '/assistant/r-1')

    expect(await screen.findByTestId('assistant-message-2')).toHaveAttribute('data-question', 'true')
    expect(screen.getByText(/needed too many steps/)).toBeInTheDocument()
    expect(screen.getByLabelText('Your answer')).toBeEnabled()
  })

  it('shows the assistant thinking while it works, and the composer waits', async () => {
    vi.mocked(api.getAssistantRequest).mockResolvedValue(aDetail({ status: 'processing' }))
    renderAt(<Assistant />, '/assistant', '/assistant/r-1')

    expect(await screen.findByText('Assistant is thinking…')).toBeInTheDocument()
    expect(screen.getByRole('textbox', { name: /Ask a follow-up|Your answer|Ask or paste/ })).toBeDisabled()
    expect(screen.getByText('You can write again when it answers.')).toBeInTheDocument()
  })

  it('while waiting after a failed attempt: reason, next try, details and Retry now', async () => {
    vi.mocked(api.getAssistantRequest)
      .mockResolvedValueOnce(aDetail({
        status: 'queued', attempts: 2, wait_reason: 'no_provider', last_error: 'no AI provider', next_attempt_at: '2026-11-23T12:05:00Z',
      }))
      .mockResolvedValue(aDetail({ status: 'processing' }))
    vi.mocked(api.retryAssistantRequest).mockResolvedValue(aRequest({ status: 'queued', attempts: 0 }))
    renderAt(<Assistant />, '/assistant', '/assistant/r-1')

    expect(await screen.findByText(/No AI provider is connected/)).toBeInTheDocument()
    expect(screen.getByText(/Next try at/)).toBeInTheDocument()
    expect(screen.queryByText('no AI provider')).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Show details' }))
    expect(screen.getByText('no AI provider')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Retry now' }))

    expect(api.retryAssistantRequest).toHaveBeenCalledWith('test-token', 'r-1')
    expect(await screen.findByText('Assistant is thinking…')).toBeInTheDocument()
  })

  it('surfaces a failed retry', async () => {
    vi.mocked(api.getAssistantRequest).mockResolvedValue(aDetail({ status: 'queued', wait_reason: 'model_busy', attempts: 1 }))
    vi.mocked(api.retryAssistantRequest).mockRejectedValue(new CodedApiError('ASSISTANT_INVALID_TRANSITION', 'x'))
    renderAt(<Assistant />, '/assistant', '/assistant/r-1')

    await userEvent.click(await screen.findByRole('button', { name: 'Retry now' }))

    expect(await screen.findByText('The request can’t do that right now. Refresh to see its current state.')).toBeInTheDocument()
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
    expect(await screen.findByText('This chat was cancelled. Start a new chat to ask again.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Cancel request' })).not.toBeInTheDocument()
  })

  it('shows a load error', async () => {
    vi.mocked(api.getAssistantRequest).mockRejectedValue(new CodedApiError('ASSISTANT_REQUEST_NOT_FOUND', 'nf'))
    renderAt(<Assistant />, '/assistant', '/assistant/r-1')

    expect(await screen.findByText('This assistant request doesn’t exist (any more).')).toBeInTheDocument()
  })
})
