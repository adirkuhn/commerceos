// @vitest-environment jsdom
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { AIIntegrationCard } from '../AIIntegrationCard'
import type { AIConnection } from '../../integrations/types'
import { CodedApiError } from '../../api/codedApiError'

vi.mock('../../auth/AuthContext', () => ({ useAuth: () => ({ token: 'tok' }) }))
vi.mock('../../integrations/api', () => ({
  listAIConnections: vi.fn(),
  createAIConnection: vi.fn(),
  updateAIConnection: vi.fn(),
  deleteAIConnection: vi.fn(),
  reorderAIConnections: vi.fn(),
}))
import * as api from '../../integrations/api'

function conn(id: string, position: number, extra: Partial<AIConnection> = {}): AIConnection {
  return { id, provider: 'nvidia', position, label: '', enabled: true, has_api_key: true, status: 'connected', created_at: '', ...extra }
}

const list = [
  conn('a', 0, { label: 'NVIDIA key 1', model: 'meta/llama-3.3-70b-instruct' }),
  conn('b', 1, { provider: 'gemini' }),
  conn('c', 2, { provider: 'ollama', has_api_key: false, base_url: 'http://ollama:11434', enabled: false }),
]

// jsdom lays nothing out: give each row a real-looking box (stacked
// vertically, in DOM order) so dnd-kit's keyboard sensor can move between them.
const realRect = Element.prototype.getBoundingClientRect
beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(api.listAIConnections).mockResolvedValue(list)
  Element.prototype.getBoundingClientRect = function (this: Element) {
    const row = this.closest('[data-ai-row]')
    const rows = Array.from(document.querySelectorAll('[data-ai-row]'))
    const top = row ? rows.indexOf(row) * 50 : 0
    return { x: 0, y: top, top, left: 0, width: 300, height: 40, right: 300, bottom: top + 40, toJSON: () => ({}) } as DOMRect
  }
})
afterEach(() => { Element.prototype.getBoundingClientRect = realRect })

function rowNames(): string[] {
  return within(screen.getByRole('list', { name: /in the order they are tried/ })).getAllByRole('listitem')
    .map(li => li.querySelector('p')?.textContent ?? '')
}

describe('AIIntegrationCard', () => {
  it('lists the providers in routing order with their model, status and on/off', async () => {
    render(<AIIntegrationCard />)

    await waitFor(() => expect(rowNames()).toEqual(['NVIDIA key 1 · NVIDIA (free models)', 'Google Gemini', 'Ollama (self-hosted)']))
    expect(screen.getByText(/Tried in this order/)).toBeInTheDocument()
    expect(screen.getByText('meta/llama-3.3-70b-instruct')).toBeInTheDocument()
    expect(screen.getAllByText('Provider default model')).toHaveLength(2)
    expect(screen.getByRole('switch', { name: 'Use NVIDIA key 1 · NVIDIA (free models)' })).toBeChecked()
    expect(screen.getByRole('switch', { name: 'Use Ollama (self-hosted)' })).not.toBeChecked()
  })

  it('shows the empty state with an add button', async () => {
    vi.mocked(api.listAIConnections).mockResolvedValue([])
    render(<AIIntegrationCard />)

    expect(await screen.findByText(/No AI provider yet/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Add AI provider/ })).toBeInTheDocument()
    expect(screen.queryByRole('list')).not.toBeInTheDocument()
  })

  it('adds a provider at the end of the list', async () => {
    vi.mocked(api.listAIConnections).mockResolvedValue([])
    vi.mocked(api.createAIConnection).mockResolvedValue(conn('n', 0, { label: 'key 2' }))
    const user = userEvent.setup()
    render(<AIIntegrationCard />)

    await user.click(await screen.findByRole('button', { name: /Add AI provider/ }))
    await user.selectOptions(screen.getByLabelText('Provider'), 'nvidia')
    expect(screen.getByLabelText('API key')).toHaveAttribute('placeholder', 'nvapi-...')
    expect(screen.getByLabelText(/Model/)).toHaveAttribute('placeholder', 'mistralai/mistral-large-4-1t-a52b')
    await user.type(screen.getByLabelText(/Label/), 'key 2')
    await user.type(screen.getByLabelText('API key'), 'nvapi-x')
    await user.type(screen.getByLabelText(/Fallback models/), 'meta/llama-3.1-8b-instruct')
    await user.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(api.createAIConnection).toHaveBeenCalledWith('tok', {
      provider: 'nvidia', apiKey: 'nvapi-x', baseUrl: '', model: '', fallbackModels: 'meta/llama-3.1-8b-instruct', label: 'key 2',
    }))
    expect(await screen.findByText('key 2 · NVIDIA (free models)')).toBeInTheDocument()
    expect(screen.queryByLabelText('API key')).not.toBeInTheDocument()
  })

  it('hints the default fallback for NVIDIA', async () => {
    vi.mocked(api.listAIConnections).mockResolvedValue([])
    const user = userEvent.setup()
    render(<AIIntegrationCard />)
    await user.click(await screen.findByRole('button', { name: /Add AI provider/ }))
    await user.selectOptions(screen.getByLabelText('Provider'), 'nvidia')
    expect(screen.getByLabelText(/Fallback models/)).toHaveAttribute('placeholder', 'minimaxai/minimax-m3.1-flash, nvidia/nemotron-3.5-super-vl-120b-a12b')
  })

  it('asks for a server URL instead of a key for Ollama', async () => {
    const user = userEvent.setup()
    render(<AIIntegrationCard />)

    await user.click(await screen.findByRole('button', { name: /Add AI provider/ }))
    await user.selectOptions(screen.getByLabelText('Provider'), 'ollama')

    expect(screen.queryByLabelText('API key')).not.toBeInTheDocument()
    expect(screen.getByLabelText('Server URL')).toBeRequired()
  })

  it('edits without resending the stored key when the field stays empty', async () => {
    vi.mocked(api.updateAIConnection).mockResolvedValue(conn('a', 0, { label: 'NVIDIA key 1', model: 'nvidia/llama-3.3-nemotron-super-49b-v1' }))
    const user = userEvent.setup()
    render(<AIIntegrationCard />)

    await user.click(await screen.findByRole('button', { name: 'Edit NVIDIA key 1 · NVIDIA (free models)' }))
    const key = screen.getByLabelText('API key')
    expect(key).toHaveValue('')
    expect(key).toHaveAttribute('placeholder', 'Key saved — leave empty to keep')
    expect(key).not.toBeRequired()
    const model = screen.getByLabelText(/Model/)
    await user.clear(model)
    await user.type(model, 'nvidia/llama-3.3-nemotron-super-49b-v1')
    await user.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(api.updateAIConnection).toHaveBeenCalledWith('tok', 'a', { model: 'nvidia/llama-3.3-nemotron-super-49b-v1' }))
  })

  it('shows a save error inside the form', async () => {
    vi.mocked(api.createAIConnection).mockRejectedValue(new CodedApiError('AI_API_KEY_REQUIRED', 'an API key is required for this provider'))
    const user = userEvent.setup()
    render(<AIIntegrationCard />)

    await user.click(await screen.findByRole('button', { name: /Add AI provider/ }))
    await user.type(screen.getByLabelText('API key'), 'x')
    await user.click(screen.getByRole('button', { name: 'Save' }))

    expect(await screen.findByText('This provider needs an API key.')).toBeInTheDocument()
  })

  it('switches a provider off', async () => {
    vi.mocked(api.updateAIConnection).mockResolvedValue(conn('b', 1, { provider: 'gemini', enabled: false }))
    const user = userEvent.setup()
    render(<AIIntegrationCard />)

    await user.click(await screen.findByRole('switch', { name: 'Use Google Gemini' }))

    await waitFor(() => expect(api.updateAIConnection).toHaveBeenCalledWith('tok', 'b', { enabled: false }))
    await waitFor(() => expect(screen.getByRole('switch', { name: 'Use Google Gemini' })).not.toBeChecked())
  })

  it('removes a provider after confirming', async () => {
    vi.mocked(api.deleteAIConnection).mockResolvedValue(null)
    const user = userEvent.setup()
    render(<AIIntegrationCard />)

    await user.click(await screen.findByRole('button', { name: 'Remove Google Gemini' }))
    const dialog = await screen.findByRole('alertdialog')
    expect(within(dialog).getByText('Remove Google Gemini?')).toBeInTheDocument()
    vi.mocked(api.listAIConnections).mockResolvedValue([list[0], { ...list[2], position: 1 }])
    await user.click(within(dialog).getByRole('button', { name: 'Remove' }))

    await waitFor(() => expect(api.deleteAIConnection).toHaveBeenCalledWith('tok', 'b'))
    await waitFor(() => expect(rowNames()).toEqual(['NVIDIA key 1 · NVIDIA (free models)', 'Ollama (self-hosted)']))
  })

  it('surfaces a failed toggle', async () => {
    vi.mocked(api.updateAIConnection).mockRejectedValue(new Error('db down'))
    const user = userEvent.setup()
    render(<AIIntegrationCard />)

    await user.click(await screen.findByRole('switch', { name: 'Use Google Gemini' }))

    expect(await screen.findByText('db down')).toBeInTheDocument()
  })

  it('reorders with the keyboard and saves the new order', async () => {
    vi.mocked(api.reorderAIConnections).mockResolvedValue([
      { ...list[1], position: 0 }, { ...list[0], position: 1 }, list[2],
    ])
    const user = userEvent.setup()
    render(<AIIntegrationCard />)

    const handle = await screen.findByRole('button', { name: /Move NVIDIA key 1/ })
    handle.focus()
    await user.keyboard(' ')
    await user.keyboard('{ArrowDown}')
    await user.keyboard(' ')

    await waitFor(() => expect(api.reorderAIConnections).toHaveBeenCalledWith('tok', ['b', 'a', 'c']))
    await waitFor(() => expect(rowNames()).toEqual(['Google Gemini', 'NVIDIA key 1 · NVIDIA (free models)', 'Ollama (self-hosted)']))
  })

  it('rolls a failed reorder back and shows why', async () => {
    vi.mocked(api.reorderAIConnections).mockRejectedValue(new CodedApiError('AI_INVALID_ORDER', 'the new order must list every AI connection exactly once'))
    const user = userEvent.setup()
    render(<AIIntegrationCard />)

    const handle = await screen.findByRole('button', { name: /Move NVIDIA key 1/ })
    handle.focus()
    await user.keyboard(' ')
    await user.keyboard('{ArrowDown}')
    await user.keyboard(' ')

    expect(await screen.findByText(/must list every AI provider exactly once/)).toBeInTheDocument()
    expect(rowNames()[0]).toBe('NVIDIA key 1 · NVIDIA (free models)')
  })
})
