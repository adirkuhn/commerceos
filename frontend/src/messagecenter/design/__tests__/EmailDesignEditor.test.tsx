// @vitest-environment jsdom
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderAt } from '../../../test/renderAt'
import { EmailDesignEditor } from '../EmailDesignEditor'
import type { MessageTemplate, MessageTrigger } from '../../types'

vi.mock('../../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../api', () => ({
  listMessageTriggers: vi.fn(),
  previewMessageTemplate: vi.fn(),
  uploadMessageTemplateImage: vi.fn(),
}))

// GrapesJS needs a real browser; the hook is the seam.
const editor = {
  ready: true,
  content: vi.fn(() => ({ html: '<table><tr><td>Hi {{customer_name}}</td></tr></table>', design: { pages: [] } })),
  insertText: vi.fn(),
  insertHTML: vi.fn(),
}
let editorOpts: { blocks: { id: string; label: string; media: string }[] } | undefined
vi.mock('../useEmailEditor', () => ({
  useEmailEditor: (_c: unknown, opts: typeof editorOpts) => { editorOpts = opts; return editor },
}))

import * as api from '../../api'
import { acceptConfirm } from '../../../test/confirm'

const template: MessageTemplate = {
  id: 'tpl-1', name: 'Order confirmed', category: 'purchase', channels: ['email'],
  subject: 'Order {{order_number}}', body: '<p>Hi</p>', text_body: '', enabled: true,
  created_at: '2026-06-01T00:00:00Z', updated_at: '2026-06-01T00:00:00Z',
}

const triggers: MessageTrigger[] = [{
  subject: 'storefront.order.placed', audience: 'customer', has_order: true, sample_data: {},
  variables: [
    { name: 'customer_name', type: 'text', example: 'Maria', fields: [] },
    { name: 'items', type: 'list', example: null, fields: [{ name: 'name', type: 'text', example: 'Pão de queijo', fields: [] }] },
  ],
}]

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(api.listMessageTriggers).mockResolvedValue(triggers)
})

function renderEditor(onSave = vi.fn().mockResolvedValue(undefined)) {
  renderAt(<EmailDesignEditor template={template} onSave={onSave} onBack={vi.fn()} />, '/message-center/templates/tpl-1/design')
  return onSave
}

describe('EmailDesignEditor', () => {
  it('names the CommerceOS blocks in the editing language, each with an icon', async () => {
    renderEditor()
    await screen.findByText('{{customer_name}}')

    const header = editorOpts?.blocks.find(b => b.id === 'cos-header')
    expect(header?.label).toBe('Store header')
    expect(header?.media).toContain('<svg')
    expect(editorOpts?.blocks.every(b => !b.label.startsWith('design.'))).toBe(true)
  })

  it('lists the trigger variables and inserts them into the canvas', async () => {
    renderEditor()

    await userEvent.click(await screen.findByText('{{customer_name}}'))
    expect(editor.insertText).toHaveBeenCalledWith('{{customer_name}}')

    await userEvent.click(screen.getByText('{{#items}}'))
    expect(editor.insertHTML).toHaveBeenCalledWith(expect.stringContaining('data-each="items"'))
  })

  it('saves the subject, the inlined html and the design', async () => {
    const onSave = renderEditor()
    await screen.findByText('{{customer_name}}')

    const subject = screen.getByLabelText('Subject')
    await userEvent.clear(subject)
    await userEvent.type(subject, 'Thanks!')
    await userEvent.click(screen.getByText('Save'))

    await waitFor(() => expect(onSave).toHaveBeenCalledWith(template, {
      subject: 'Thanks!', body: '<table><tr><td>Hi {{customer_name}}</td></tr></table>', design: { pages: [] },
    }))
    expect(await screen.findByRole('status')).toBeInTheDocument()
  })

  it('shows a save error from the API', async () => {
    renderEditor(vi.fn().mockRejectedValue(new Error('boom')))
    await screen.findByText('{{customer_name}}')

    await userEvent.click(screen.getByText('Save'))

    expect(await screen.findByRole('alert')).toHaveTextContent('boom')
  })

  it('previews the current design against the chosen trigger', async () => {
    vi.mocked(api.previewMessageTemplate).mockResolvedValue({ subject: 'Order 1001', html: '<p>Hi Maria</p>', text: '' })
    renderEditor()
    await screen.findByText('{{customer_name}}')

    await userEvent.click(screen.getByText('Preview'))

    await waitFor(() => expect(api.previewMessageTemplate).toHaveBeenCalledWith('test-token', {
      trigger_subject: 'storefront.order.placed', subject: 'Order {{order_number}}',
      body: '<table><tr><td>Hi {{customer_name}}</td></tr></table>', text_body: '',
    }))
    expect(await screen.findByText('Order 1001')).toBeInTheDocument()
  })

  it('reset to default asks first, then calls onReset', async () => {
    const onReset = vi.fn().mockResolvedValue(undefined)
    renderAt(<EmailDesignEditor template={{ ...template, default_key: 'order_confirmation' }} onSave={vi.fn()} onReset={onReset} onBack={vi.fn()} />,
      '/message-center/templates/tpl-1/design')

    await userEvent.click(await screen.findByTitle('Reset to default'))
    expect(onReset).not.toHaveBeenCalled()
    await acceptConfirm()

    await waitFor(() => expect(onReset).toHaveBeenCalledWith({ ...template, default_key: 'order_confirmation' }))
  })

  it('has no reset for a template written from scratch', async () => {
    renderEditor()
    await screen.findByText('{{customer_name}}')

    expect(screen.queryByTitle('Reset to default')).not.toBeInTheDocument()
  })
})
