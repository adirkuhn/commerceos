// @vitest-environment jsdom
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderAt } from '../../test/renderAt'
import { act } from 'react'
import * as messagecenterApi from '../../messagecenter/api'
import * as whatsappApi from '../../whatsapp/api'
import MessageCenter from '../MessageCenter'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../messagecenter/api', () => ({
  listMessageTemplates: vi.fn().mockResolvedValue([]),
  createMessageTemplate: vi.fn(),
  updateMessageTemplate: vi.fn(),
  deleteMessageTemplate: vi.fn(),
  listAutomationRules: vi.fn().mockResolvedValue([]),
  createAutomationRule: vi.fn(),
  updateAutomationRule: vi.fn(),
  deleteAutomationRule: vi.fn(),
}))

vi.mock('../../whatsapp/api', () => ({
  listWhatsAppConversations: vi.fn().mockResolvedValue([]),
  listWhatsAppConversationMessages: vi.fn().mockResolvedValue([]),
}))

beforeEach(() => vi.clearAllMocks())

describe('MessageCenter', () => {
  it('shows the Templates panel by default', async () => {
    renderAt(<MessageCenter />, '/message-center')

    expect(await screen.findByPlaceholderText('Search templates...')).toBeInTheDocument()
  })

  it('switches to the Automation Rules panel', async () => {
    renderAt(<MessageCenter />, '/message-center')
    await screen.findByPlaceholderText('Search templates...')

    await userEvent.click(screen.getByText('Automation Rules'))

    expect(await screen.findByText('When an event fires and every condition matches, the chosen template is sent.')).toBeInTheDocument()
  })

  it('switches to the Conversations panel', async () => {
    renderAt(<MessageCenter />, '/message-center')
    await screen.findByPlaceholderText('Search templates...')

    await userEvent.click(screen.getByText('Conversations'))

    expect(await screen.findByText('No WhatsApp conversations yet.')).toBeInTheDocument()
  })
})

describe('MessageCenter URLs (refresh + Back)', () => {
  const template = { id: 'template-1', name: 'Order confirmed', category: 'purchase', channels: ['email'], subject: 'Hi', body: 'Thanks', created_at: '', updated_at: '' }

  it('each tab has its own URL, and Back returns to the previous tab', async () => {
    const { router } = renderAt(<MessageCenter />, '/message-center')
    await userEvent.click(await screen.findByText('Automation Rules'))
    expect(router.state.location.pathname).toBe('/message-center/rules')

    await act(() => router.navigate(-1))
    expect(router.state.location.pathname).toBe('/message-center')
    expect(await screen.findByPlaceholderText('Search templates...')).toBeInTheDocument()
  })

  it('/message-center/conversations opens that tab on load', async () => {
    renderAt(<MessageCenter />, '/message-center', '/message-center/conversations')
    expect(await screen.findByText('No WhatsApp conversations yet.')).toBeInTheDocument()
  })

  it('/message-center/templates/:id opens the template editor prefilled', async () => {
    vi.mocked(messagecenterApi.listMessageTemplates).mockResolvedValue([template] as never)
    renderAt(<MessageCenter />, '/message-center', '/message-center/templates/template-1')
    expect(await screen.findByText('Edit message template')).toBeInTheDocument()
    expect(screen.getByDisplayValue('Order confirmed')).toBeInTheDocument()
  })

  it('New template pushes /message-center/templates/new', async () => {
    const { router } = renderAt(<MessageCenter />, '/message-center')
    await userEvent.click(await screen.findByText('New template'))
    expect(router.state.location.pathname).toBe('/message-center/templates/new')
    expect(await screen.findByText('New message template')).toBeInTheDocument()
  })

  it('/message-center/rules/new opens the new rule form on load', async () => {
    renderAt(<MessageCenter />, '/message-center', '/message-center/rules/new')
    expect(await screen.findByText('New automation rule')).toBeInTheDocument()
  })

  it('/message-center/conversations/:phone selects that conversation', async () => {
    vi.mocked(whatsappApi.listWhatsAppConversations).mockResolvedValue([
      { phone_e164: '31612345678', customer_id: 'c1', customer_name: 'Jane Doe', last_direction: 'inbound', last_body: 'hi', last_message_at: '2026-08-23T10:00:00Z' },
    ] as never)
    renderAt(<MessageCenter />, '/message-center', '/message-center/conversations/31612345678')
    await waitFor(() => expect(whatsappApi.listWhatsAppConversationMessages).toHaveBeenCalledWith('test-token', '31612345678'))
  })
})
