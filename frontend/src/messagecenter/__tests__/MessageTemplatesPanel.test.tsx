// @vitest-environment jsdom
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderAt } from '../../test/renderAt'
import MessageTemplatesPanel from '../MessageTemplatesPanel'
import type { MessageTemplate } from '../types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../api', () => ({
  listMessageTemplates: vi.fn(),
  createMessageTemplate: vi.fn(),
  updateMessageTemplate: vi.fn(),
  deleteMessageTemplate: vi.fn(),
}))

vi.mock('../design/EmailDesignEditor', () => ({
  EmailDesignEditor: ({ template }: { template: MessageTemplate }) => <div data-testid="design-editor">{template.id}</div>,
}))

import * as messagecenterApi from '../api'
import { acceptConfirm } from '../../test/confirm'

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

describe('MessageTemplatesPanel', () => {
  it('shows an empty state when there are no templates', async () => {
    renderAt(<MessageTemplatesPanel />, '/message-center')

    expect(await screen.findByText('No message templates yet')).toBeInTheDocument()
  })

  it('lists templates with name, category, and body', async () => {
    vi.mocked(messagecenterApi.listMessageTemplates).mockResolvedValue([template()])

    renderAt(<MessageTemplatesPanel />, '/message-center')

    expect(await screen.findByText('Order confirmed')).toBeInTheDocument()
    expect(screen.getByText('Hi {{customer_name}}')).toBeInTheDocument()
    expect(screen.getByText('Purchase')).toBeInTheDocument()
  })

  it('filters by search', async () => {
    vi.mocked(messagecenterApi.listMessageTemplates).mockResolvedValue([
      template(), template({ id: 'template-2', name: 'Shipment delivered', body: 'It arrived' }),
    ])

    renderAt(<MessageTemplatesPanel />, '/message-center')
    await screen.findByText('Order confirmed')

    await userEvent.type(screen.getByPlaceholderText('Search templates...'), 'delivered')

    expect(screen.queryByText('Order confirmed')).not.toBeInTheDocument()
    expect(screen.getByText('Shipment delivered')).toBeInTheDocument()
  })

  it('creating a template calls createMessageTemplate with the form values', async () => {
    vi.mocked(messagecenterApi.createMessageTemplate).mockResolvedValue(template({ id: 'template-new' }))

    renderAt(<MessageTemplatesPanel />, '/message-center')
    await screen.findByText('No message templates yet')

    await userEvent.click(screen.getByText('New template'))
    await userEvent.type(await screen.findByLabelText('Name'), 'Welcome message')
    await userEvent.click(screen.getByLabelText('WhatsApp'))
    expect(screen.queryByLabelText('Body')).not.toBeInTheDocument()
    await userEvent.type(await screen.findByLabelText('WhatsApp / SMS text'), 'Welcome!')
    await userEvent.click(screen.getByText('Create template'))

    await waitFor(() => expect(messagecenterApi.createMessageTemplate).toHaveBeenCalledWith(
      'test-token',
      expect.objectContaining({ name: 'Welcome message', channels: ['whatsapp'], text_body: 'Welcome!' }),
    ))
  })

  it('lists an email body as plain text, not markup', async () => {
    vi.mocked(messagecenterApi.listMessageTemplates).mockResolvedValue([template({ body: '<table><tr><td>Hello <b>there</b></td></tr></table>' })])

    renderAt(<MessageTemplatesPanel />, '/message-center')

    expect(await screen.findByText('Hello there')).toBeInTheDocument()
  })

  it('editing keeps the text body and enabled flag', async () => {
    vi.mocked(messagecenterApi.listMessageTemplates).mockResolvedValue([template({ text_body: 'Hi on WhatsApp' })])
    vi.mocked(messagecenterApi.updateMessageTemplate).mockResolvedValue(template())

    renderAt(<MessageTemplatesPanel />, '/message-center/templates/template-1')
    expect(await screen.findByLabelText('WhatsApp / SMS text')).toHaveValue('Hi on WhatsApp')
    await userEvent.click(screen.getByText('Save changes'))

    await waitFor(() => expect(messagecenterApi.updateMessageTemplate).toHaveBeenCalledWith(
      'test-token', 'template-1',
      expect.objectContaining({ body: 'Hi {{customer_name}}', text_body: 'Hi on WhatsApp', enabled: true }),
    ))
  })

  it('the design button opens the visual editor for email templates only', async () => {
    vi.mocked(messagecenterApi.listMessageTemplates).mockResolvedValue([
      template(), template({ id: 'template-2', name: 'Text only', channels: ['sms'], body: '', text_body: 'Hi' }),
    ])

    renderAt(<MessageTemplatesPanel />, '/message-center')

    expect(await screen.findAllByTitle('Design email')).toHaveLength(1)
    await userEvent.click(screen.getByTitle('Design email'))
    expect(await screen.findByTestId('design-editor')).toHaveTextContent('template-1')
  })

  it('a design URL for a missing template says so', async () => {
    renderAt(<MessageTemplatesPanel />, '/message-center/templates/gone/design')

    expect(await screen.findByText('This template doesn’t exist any more.')).toBeInTheDocument()
  })

  it('deleting a template calls deleteMessageTemplate', async () => {
    vi.mocked(messagecenterApi.listMessageTemplates).mockResolvedValue([template()])

    renderAt(<MessageTemplatesPanel />, '/message-center')
    await userEvent.click(await screen.findByTitle('Delete'))

    await acceptConfirm()
    await waitFor(() => expect(messagecenterApi.deleteMessageTemplate).toHaveBeenCalledWith('test-token', 'template-1'))
  })

  it('the editor explains loops and conditional sections', async () => {
    renderAt(<MessageTemplatesPanel />, '/message-center/templates/new')

    expect(await screen.findByText(/Repeat a list with \{\{#items\}\}…\{\{\/items\}\}/)).toBeInTheDocument()
  })
})
