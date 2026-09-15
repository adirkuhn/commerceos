// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
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
    render(<MessageCenter />)

    expect(await screen.findByPlaceholderText('Search templates...')).toBeInTheDocument()
  })

  it('switches to the Automation Rules panel', async () => {
    render(<MessageCenter />)
    await screen.findByPlaceholderText('Search templates...')

    await userEvent.click(screen.getByText('Automation Rules'))

    expect(await screen.findByText('When an event fires and every condition matches, the chosen template is sent.')).toBeInTheDocument()
  })

  it('switches to the Conversations panel', async () => {
    render(<MessageCenter />)
    await screen.findByPlaceholderText('Search templates...')

    await userEvent.click(screen.getByText('Conversations'))

    expect(await screen.findByText('No WhatsApp conversations yet.')).toBeInTheDocument()
  })
})
