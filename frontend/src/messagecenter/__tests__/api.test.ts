import { describe, it, expect, vi, beforeEach } from 'vitest'
import * as api from '../api'
import type { MessageTemplate, APIResponse } from '../types'

const TOKEN = 'test-token'

function mockFetch<T>(data: T, status = 200) {
  const body: APIResponse<T> = { data, error: null }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  }))
}

function mockFetchError(code: string, message: string, status = 404) {
  const body: APIResponse<null> = { data: null, error: { code, message } }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: false,
    status,
    json: () => Promise.resolve(body),
  }))
}

const sampleTemplate: MessageTemplate = {
  id: 'template-1', name: 'Order confirmed', category: 'purchase', channels: ['email', 'whatsapp'],
  subject: 'Confirmed', body: 'Hi {{customer_name}}', enabled: true,
  created_at: '2026-06-01T00:00:00Z', updated_at: '2026-06-01T00:00:00Z',
}

beforeEach(() => { vi.restoreAllMocks() })

describe('listMessageTemplates', () => {
  it('returns array of templates', async () => {
    mockFetch([sampleTemplate])
    const result = await api.listMessageTemplates(TOKEN)
    expect(result).toEqual([sampleTemplate])
  })

  it('calls correct endpoint', async () => {
    mockFetch([])
    await api.listMessageTemplates(TOKEN)
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/message-templates',
      expect.objectContaining({ method: 'GET' }),
    )
  })

  it('throws on server error', async () => {
    mockFetchError('INTERNAL', 'internal server error', 500)
    await expect(api.listMessageTemplates(TOKEN)).rejects.toThrow('internal server error')
  })
})

describe('createMessageTemplate', () => {
  it('posts the input and returns the created template', async () => {
    mockFetch(sampleTemplate)
    const result = await api.createMessageTemplate(TOKEN, {
      name: 'Order confirmed', category: 'purchase', channels: ['email', 'whatsapp'], body: 'Hi {{customer_name}}',
    })
    expect(result).toEqual(sampleTemplate)
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/message-templates',
      expect.objectContaining({ method: 'POST' }),
    )
  })

  it('throws on validation error', async () => {
    mockFetchError('INVALID_REQUEST', 'template name is required', 422)
    await expect(api.createMessageTemplate(TOKEN, { name: '', category: 'purchase', channels: ['email'], body: 'x' }))
      .rejects.toThrow('template name is required')
  })
})

describe('updateMessageTemplate', () => {
  it('calls the correct endpoint with PATCH', async () => {
    mockFetch(sampleTemplate)
    await api.updateMessageTemplate(TOKEN, 'template-1', {
      name: 'Order confirmed', category: 'purchase', channels: ['email'], body: 'x', enabled: false,
    })
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/message-templates/template-1',
      expect.objectContaining({ method: 'PATCH' }),
    )
  })
})

describe('deleteMessageTemplate', () => {
  it('calls the correct endpoint with DELETE', async () => {
    mockFetch(null)
    await api.deleteMessageTemplate(TOKEN, 'template-1')
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/message-templates/template-1',
      expect.objectContaining({ method: 'DELETE' }),
    )
  })

  it('throws on not found', async () => {
    mockFetchError('NOT_FOUND', 'message template not found', 404)
    await expect(api.deleteMessageTemplate(TOKEN, 'missing')).rejects.toThrow('message template not found')
  })
})
