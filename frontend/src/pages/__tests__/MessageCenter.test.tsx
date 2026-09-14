// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import MessageCenter from '../MessageCenter'
import type { MessageTemplate } from '../../messagecenter/types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../messagecenter/api', () => ({
  listMessageTemplates: vi.fn(),
  createMessageTemplate: vi.fn(),
  updateMessageTemplate: vi.fn(),
  deleteMessageTemplate: vi.fn(),
}))

import * as messagecenterApi from '../../messagecenter/api'

function template(overrides: Partial<MessageTemplate> = {}): MessageTemplate {
  return {
    id: 'template-1', name: 'Order confirmed', category: 'purchase', channels: ['email', 'whatsapp'],
    subject: 'Confirmed', body: 'Hi {{customer_name}}', enabled: true,
    created_at: '2026-06-01T00:00:00Z', updated_at: '2026-06-01T00:00:00Z',
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(messagecenterApi.listMessageTemplates).mockResolvedValue([])
})

describe('MessageCenter', () => {
  it('shows an empty state when there are no templates', async () => {
    render(<MessageCenter />)

    expect(await screen.findByText('No message templates yet')).toBeInTheDocument()
  })

  it('lists templates with name, category, and body', async () => {
    vi.mocked(messagecenterApi.listMessageTemplates).mockResolvedValue([template()])

    render(<MessageCenter />)

    expect(await screen.findByText('Order confirmed')).toBeInTheDocument()
    expect(screen.getByText('Hi {{customer_name}}')).toBeInTheDocument()
    expect(screen.getByText('Purchase')).toBeInTheDocument()
  })

  it('filters by search', async () => {
    vi.mocked(messagecenterApi.listMessageTemplates).mockResolvedValue([
      template(), template({ id: 'template-2', name: 'Shipment delivered', body: 'It arrived' }),
    ])

    render(<MessageCenter />)
    await screen.findByText('Order confirmed')

    await userEvent.type(screen.getByPlaceholderText('Search templates...'), 'delivered')

    expect(screen.queryByText('Order confirmed')).not.toBeInTheDocument()
    expect(screen.getByText('Shipment delivered')).toBeInTheDocument()
  })

  it('creating a template calls createMessageTemplate with the form values', async () => {
    vi.mocked(messagecenterApi.createMessageTemplate).mockResolvedValue(template({ id: 'template-new' }))

    render(<MessageCenter />)
    await screen.findByText('No message templates yet')

    await userEvent.click(screen.getByText('New template'))
    await userEvent.type(screen.getByLabelText('Name'), 'Welcome message')
    await userEvent.click(screen.getByLabelText('WhatsApp'))
    await userEvent.type(screen.getByLabelText('Body'), 'Welcome!')
    await userEvent.click(screen.getByText('Create template'))

    await waitFor(() => expect(messagecenterApi.createMessageTemplate).toHaveBeenCalledWith(
      'test-token',
      expect.objectContaining({ name: 'Welcome message', channels: ['whatsapp'], body: 'Welcome!' }),
    ))
  })

  it('deleting a template calls deleteMessageTemplate', async () => {
    vi.mocked(messagecenterApi.listMessageTemplates).mockResolvedValue([template()])
    vi.stubGlobal('confirm', vi.fn(() => true))

    render(<MessageCenter />)
    await userEvent.click(await screen.findByTitle('Delete'))

    await waitFor(() => expect(messagecenterApi.deleteMessageTemplate).toHaveBeenCalledWith('test-token', 'template-1'))
  })
})
