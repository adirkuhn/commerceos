import { describe, it, expect, vi, beforeEach } from 'vitest'
import { listMessageTriggers, previewMessageTemplate, uploadMessageTemplateImage, listMessageTemplateDefaults, resetMessageTemplate, listSentEmails, getSentEmail, deleteSentEmail, deleteSentEmails } from '../api'
import { CodedApiError } from '../../api/codedApiError'

beforeEach(() => { vi.restoreAllMocks() })

function mockJSON(body: unknown, status = 200) {
  const fetchMock = vi.fn().mockResolvedValue({ ok: status < 300, status, json: () => Promise.resolve(body) })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

describe('sent email API', () => {
  it('lists with only the filters that are set', async () => {
    const fetchMock = mockJSON({ data: { items: [], total: 0, page: 1, per_page: 50 }, error: null })

    await listSentEmails('tok', { status: 'failed', q: '', page: 2, source: undefined })

    expect(fetchMock.mock.calls[0][0]).toMatch(/\/sent-emails\?status=failed&page=2$/)
  })

  it('gets, deletes one and deletes many', async () => {
    const fetchMock = mockJSON({ data: { deleted: 2 }, error: null })

    await getSentEmail('tok', 'e 1')
    await deleteSentEmail('tok', 'e1')
    expect(await deleteSentEmails('tok', ['a', 'b'])).toEqual({ deleted: 2 })

    expect(fetchMock.mock.calls[0][0]).toMatch(/\/sent-emails\/e%201$/)
    expect(fetchMock.mock.calls[1][1].method).toBe('DELETE')
    expect(fetchMock.mock.calls[2][0]).toMatch(/\/sent-emails\/bulk-delete$/)
    expect(JSON.parse(fetchMock.mock.calls[2][1].body)).toEqual({ ids: ['a', 'b'] })
  })
})

describe('design editor API', () => {
  it('lists the default templates', async () => {
    const fetchMock = mockJSON({ data: [{ key: 'order_confirmation' }], error: null })

    expect(await listMessageTemplateDefaults('tok')).toEqual([{ key: 'order_confirmation' }])
    expect(fetchMock.mock.calls[0][0]).toMatch(/\/message-template-defaults$/)
  })

  it('resets a template to its default', async () => {
    const fetchMock = mockJSON({ data: { id: 'tpl-1' }, error: null })

    expect(await resetMessageTemplate('tok', 'tpl-1')).toEqual({ id: 'tpl-1' })
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toMatch(/\/message-templates\/tpl-1\/reset$/)
    expect(init.method).toBe('POST')
  })

  it('surfaces a coded reset error', async () => {
    mockJSON({ data: null, error: { code: 'MESSAGE_TEMPLATE_HAS_NO_DEFAULT', message: 'x' } }, 422)

    await expect(resetMessageTemplate('tok', 'tpl-1')).rejects.toBeInstanceOf(CodedApiError)
  })

  it('lists triggers with their variables', async () => {
    const triggers = [{ subject: 'storefront.order.placed', audience: 'customer', has_order: true, variables: [], sample_data: {} }]
    const fetchMock = mockJSON({ data: triggers, error: null })

    expect(await listMessageTriggers('tok')).toEqual(triggers)
    expect(fetchMock.mock.calls[0][0]).toMatch(/\/message-triggers$/)
  })

  it('previews a template', async () => {
    const fetchMock = mockJSON({ data: { subject: 'S', html: '<p>H</p>', text: 'T' }, error: null })

    const out = await previewMessageTemplate('tok', { trigger_subject: 'storefront.order.placed', subject: 'S', body: '<p>H</p>' })

    expect(out).toEqual({ subject: 'S', html: '<p>H</p>', text: 'T' })
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toMatch(/\/message-templates\/preview$/)
    expect(init.method).toBe('POST')
    expect(JSON.parse(init.body)).toEqual({ trigger_subject: 'storefront.order.placed', subject: 'S', body: '<p>H</p>' })
  })

  it('uploads an image as multipart and returns its URL', async () => {
    const fetchMock = mockJSON({ data: { url: 'https://cdn/x.png' }, error: null })
    const file = new File(['png'], 'logo.png', { type: 'image/png' })

    expect(await uploadMessageTemplateImage('tok', file)).toBe('https://cdn/x.png')
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toMatch(/\/message-templates\/images$/)
    expect(init.body).toBeInstanceOf(FormData)
    expect((init.body as FormData).get('file')).toBe(file)
    expect(init.headers['Content-Type']).toBeUndefined()
  })

  it('keeps the error code of a failed upload', async () => {
    mockJSON({ data: null, error: { code: 'MESSAGE_TEMPLATE_IMAGE_TOO_LARGE', message: 'images can be at most 5 MB' } }, 422)
    await expect(uploadMessageTemplateImage('tok', new File(['x'], 'x.png'))).rejects.toMatchObject({ code: 'MESSAGE_TEMPLATE_IMAGE_TOO_LARGE' })
    await expect(uploadMessageTemplateImage('tok', new File(['x'], 'x.png'))).rejects.toBeInstanceOf(CodedApiError)
  })
})
