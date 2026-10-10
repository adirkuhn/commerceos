// @vitest-environment jsdom
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderAt } from '../../test/renderAt'
import AutomationRulesPanel from '../AutomationRulesPanel'
import type { AutomationRule, MessageTemplate } from '../types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../api', () => ({
  listMessageTemplates: vi.fn(),
  createMessageTemplate: vi.fn(),
  updateMessageTemplate: vi.fn(),
  deleteMessageTemplate: vi.fn(),
  listAutomationRules: vi.fn(),
  createAutomationRule: vi.fn(),
  updateAutomationRule: vi.fn(),
  deleteAutomationRule: vi.fn(),
}))

import * as messagecenterApi from '../api'
import { acceptConfirm } from '../../test/confirm'

function template(overrides: Partial<MessageTemplate> = {}): MessageTemplate {
  return {
    id: 'template-1', name: 'Collect point notice', category: 'tracking', channels: ['whatsapp'],
    subject: '', body: 'Your package is at a collection point', enabled: true,
    created_at: '2026-06-01T00:00:00Z', updated_at: '2026-06-01T00:00:00Z',
    ...overrides,
  }
}

function rule(overrides: Partial<AutomationRule> = {}): AutomationRule {
  return {
    id: 'rule-1', name: 'Collect point rule', trigger_subject: 'shipping.shipment.fulfillment_signal',
    conditions: [{ field: 'status_code', operator: 'in', value: [8, 19] }], template_id: 'template-1',
    extra_recipients: [], enabled: true, created_at: '2026-06-01T00:00:00Z', updated_at: '2026-06-01T00:00:00Z',
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(messagecenterApi.listAutomationRules).mockResolvedValue([])
  vi.mocked(messagecenterApi.listMessageTemplates).mockResolvedValue([template()])
})

describe('AutomationRulesPanel', () => {
  it('shows an empty state when there are no rules', async () => {
    renderAt(<AutomationRulesPanel />, '/message-center')

    expect(await screen.findByText('No automation rules yet')).toBeInTheDocument()
  })

  it('lists rules with name, trigger, and conditions', async () => {
    vi.mocked(messagecenterApi.listAutomationRules).mockResolvedValue([rule()])

    renderAt(<AutomationRulesPanel />, '/message-center')

    expect(await screen.findByText('Collect point rule')).toBeInTheDocument()
    expect(screen.getByText('Shipment status update')).toBeInTheDocument()
    expect(screen.getByText('status_code is one of 8, 19')).toBeInTheDocument()
  })

  it('creating a rule calls createAutomationRule with the form values', async () => {
    vi.mocked(messagecenterApi.createAutomationRule).mockResolvedValue(rule({ id: 'rule-new' }))

    renderAt(<AutomationRulesPanel />, '/message-center')
    await screen.findByText('No automation rules yet')

    await userEvent.click(screen.getByText('New rule'))
    await userEvent.type(await screen.findByLabelText('Name'), 'Notify on collect point')
    await userEvent.selectOptions(screen.getByLabelText('Send this template'), 'template-1')
    await userEvent.click(screen.getByText('Create rule'))

    await waitFor(() => expect(messagecenterApi.createAutomationRule).toHaveBeenCalledWith(
      'test-token',
      expect.objectContaining({ name: 'Notify on collect point', template_id: 'template-1' }),
    ))
  })

  it('deleting a rule calls deleteAutomationRule', async () => {
    vi.mocked(messagecenterApi.listAutomationRules).mockResolvedValue([rule()])

    renderAt(<AutomationRulesPanel />, '/message-center')
    await userEvent.click(await screen.findByTitle('Delete'))

    await acceptConfirm()
    await waitFor(() => expect(messagecenterApi.deleteAutomationRule).toHaveBeenCalledWith('test-token', 'rule-1'))
  })

  it('creating a rule sends its extra email recipients', async () => {
    vi.mocked(messagecenterApi.createAutomationRule).mockResolvedValue(rule({ id: 'rule-new' }))

    renderAt(<AutomationRulesPanel />, '/message-center')
    await screen.findByText('No automation rules yet')

    await userEvent.click(screen.getByText('New rule'))
    await userEvent.type(await screen.findByLabelText('Name'), 'Copy to the shop')
    await userEvent.type(screen.getByLabelText('Also email'), 'shop@example.nl{enter}buyer@example.nl')
    await userEvent.click(screen.getByText('Create rule'))

    await waitFor(() => expect(messagecenterApi.createAutomationRule).toHaveBeenCalledWith(
      'test-token',
      expect.objectContaining({ extra_recipients: ['shop@example.nl', 'buyer@example.nl'] }),
    ))
  })

  it('editing a rule shows and keeps its extra recipients', async () => {
    vi.mocked(messagecenterApi.listAutomationRules).mockResolvedValue([rule({ extra_recipients: ['shop@example.nl'] })])
    vi.mocked(messagecenterApi.updateAutomationRule).mockResolvedValue(rule())

    renderAt(<AutomationRulesPanel />, '/message-center/rules/rule-1')

    expect(await screen.findByLabelText('Also email')).toHaveValue('shop@example.nl')
    await userEvent.click(screen.getByText('Save changes'))

    await waitFor(() => expect(messagecenterApi.updateAutomationRule).toHaveBeenCalledWith(
      'test-token', 'rule-1', expect.objectContaining({ extra_recipients: ['shop@example.nl'] }),
    ))
  })

  it('the stock close-to-expiry trigger explains who gets it', async () => {
    renderAt(<AutomationRulesPanel />, '/message-center')
    await screen.findByText('No automation rules yet')

    await userEvent.click(screen.getByText('New rule'))
    await userEvent.selectOptions(await screen.findByLabelText('When this happens'), 'inventory.stock.close_to_expiry_reported')

    expect(screen.getByText(/Stock email recipients in Settings › Notifications/)).toBeInTheDocument()
  })

  it('lists the stock close-to-expiry trigger by name', async () => {
    vi.mocked(messagecenterApi.listAutomationRules).mockResolvedValue([rule({ trigger_subject: 'inventory.stock.close_to_expiry_reported', conditions: [] })])

    renderAt(<AutomationRulesPanel />, '/message-center')

    expect(await screen.findByText('Stock close to expiry report')).toBeInTheDocument()
  })
})
