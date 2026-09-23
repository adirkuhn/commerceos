// @vitest-environment jsdom
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderAt } from '../../test/renderAt'
import WhatsAppConversationsPanel from '../WhatsAppConversationsPanel'
import type { WhatsAppConversation, WhatsAppConversationMessage } from '../types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../api', () => ({
  listWhatsAppConversations: vi.fn(),
  listWhatsAppConversationMessages: vi.fn(),
  deleteWhatsAppConversation: vi.fn(),
}))

import * as whatsappApi from '../api'
import { acceptConfirm, dismissConfirm } from '../../test/confirm'

function conversation(overrides: Partial<WhatsAppConversation> = {}): WhatsAppConversation {
  return {
    phone_e164: '31612345678', customer_id: 'cust-1', customer_name: 'Jane Doe',
    last_direction: 'outbound', last_body: 'Widget - in stock: 7', last_message_at: '2026-08-23T10:01:00Z',
    ...overrides,
  }
}

function message(overrides: Partial<WhatsAppConversationMessage> = {}): WhatsAppConversationMessage {
  return {
    id: 'msg-1', phone_e164: '31612345678', direction: 'inbound', body: 'STOCK ABC123', created_at: '2026-08-23T10:00:00Z',
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(whatsappApi.listWhatsAppConversations).mockResolvedValue([])
  vi.mocked(whatsappApi.listWhatsAppConversationMessages).mockResolvedValue([])
  vi.mocked(whatsappApi.deleteWhatsAppConversation).mockResolvedValue(null)
})

describe('WhatsAppConversationsPanel', () => {
  it('shows an empty state when there are no conversations', async () => {
    renderAt(<WhatsAppConversationsPanel />, '/message-center')

    expect(await screen.findByText('No WhatsApp conversations yet.')).toBeInTheDocument()
  })

  it('lists conversations with customer name (or phone) and a last-message preview', async () => {
    vi.mocked(whatsappApi.listWhatsAppConversations).mockResolvedValue([
      conversation(),
      conversation({ phone_e164: '31699999999', customer_id: null, customer_name: undefined, last_direction: 'inbound', last_body: 'hello' }),
    ])

    renderAt(<WhatsAppConversationsPanel />, '/message-center')

    expect(await screen.findByText('Jane Doe')).toBeInTheDocument()
    expect(screen.getByText('31699999999')).toBeInTheDocument()
    expect(screen.getByText('hello')).toBeInTheDocument()
  })

  it('shows "Select a conversation" until one is clicked, then loads and renders its thread', async () => {
    vi.mocked(whatsappApi.listWhatsAppConversations).mockResolvedValue([conversation()])
    vi.mocked(whatsappApi.listWhatsAppConversationMessages).mockResolvedValue([
      message({ id: 'msg-2', direction: 'outbound', body: 'Widget - in stock: 7', created_at: '2026-08-23T10:01:00Z' }),
      message({ id: 'msg-1', direction: 'inbound', body: 'STOCK ABC123', created_at: '2026-08-23T10:00:00Z' }),
    ])

    renderAt(<WhatsAppConversationsPanel />, '/message-center')
    expect(await screen.findByText('Select a conversation')).toBeInTheDocument()

    await userEvent.click(await screen.findByText('Jane Doe'))

    expect(whatsappApi.listWhatsAppConversationMessages).toHaveBeenCalledWith('test-token', '31612345678')
    expect(await screen.findByText('STOCK ABC123')).toBeInTheDocument()
    expect(screen.getByText('Widget - in stock: 7')).toBeInTheDocument()
  })

  describe('delete chat', () => {
    beforeEach(() => {
      vi.mocked(whatsappApi.listWhatsAppConversations).mockResolvedValue([conversation()])
      vi.mocked(whatsappApi.listWhatsAppConversationMessages).mockResolvedValue([message()])
    })

    it('does nothing when the confirm dialog is declined', async () => {
      renderAt(<WhatsAppConversationsPanel />, '/message-center')
      await userEvent.click(await screen.findByText('Jane Doe'))

      await userEvent.click(await screen.findByLabelText('Delete chat'))

      expect(await screen.findByRole('alertdialog', { name: /Jane Doe/ })).toBeInTheDocument()
      await dismissConfirm()
      expect(whatsappApi.deleteWhatsAppConversation).not.toHaveBeenCalled()
      expect(await screen.findAllByText('Jane Doe')).toHaveLength(2)
    })

    it('confirms, deletes, clears the selection and removes it from the list', async () => {
      renderAt(<WhatsAppConversationsPanel />, '/message-center')
      await userEvent.click(await screen.findByText('Jane Doe'))
      await screen.findByText('STOCK ABC123')

      await userEvent.click(await screen.findByLabelText('Delete chat'))

      await acceptConfirm()
      await waitFor(() => expect(whatsappApi.deleteWhatsAppConversation).toHaveBeenCalledWith('test-token', '31612345678'))
      expect(await screen.findByText('Select a conversation')).toBeInTheDocument()
      expect(await screen.findByText('No WhatsApp conversations yet.')).toBeInTheDocument()
    })

    it('shows an inline error when the delete fails', async () => {
      vi.mocked(whatsappApi.deleteWhatsAppConversation).mockRejectedValue(new Error('db down'))
      renderAt(<WhatsAppConversationsPanel />, '/message-center')
      await userEvent.click(await screen.findByText('Jane Doe'))

      await userEvent.click(await screen.findByLabelText('Delete chat'))

      await acceptConfirm()
      expect(await screen.findByText('db down')).toBeInTheDocument()
      expect(screen.getAllByText('Jane Doe')).toHaveLength(2)
    })
  })
})
