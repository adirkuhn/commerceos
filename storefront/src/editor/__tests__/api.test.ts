import { describe, it, expect, vi } from 'vitest'
import { EditorAPI } from '../api'
import { ApiError } from '../../api/storefront'

function fakeFetch(body: unknown, status = 200) {
  return vi.fn<typeof fetch>(async () => new Response(JSON.stringify(body), { status }))
}

describe('EditorAPI', () => {
  it('opens the draft with the session token', async () => {
    const f = fakeFetch({ data: { version: { revision: 2 } }, error: null })
    await new EditorAPI({ baseURL: 'https://api.test/api/v1', token: 'tok', fetch: f }).openDraft()
    const [url, init] = f.mock.calls[0]
    expect(url).toBe('https://api.test/api/v1/theme-editor/draft')
    expect(init?.method).toBe('POST')
    expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer tok')
  })

  it('saves a document with the expected revision', async () => {
    const f = fakeFetch({ data: { revision: 3, document: {} }, error: null })
    const root = { id: 'root', type: 'root' }
    const res = await new EditorAPI({ baseURL: 'https://api.test/api/v1', token: 't', fetch: f })
      .saveDocument('template', 'home', { expected_revision: 2, root })
    expect(res.revision).toBe(3)
    expect(f.mock.calls[0][0]).toBe('https://api.test/api/v1/theme-editor/draft/documents/template/home')
    expect(f.mock.calls[0][1]?.method).toBe('PUT')
    expect(JSON.parse(String(f.mock.calls[0][1]?.body))).toEqual({ expected_revision: 2, root })
  })

  it('surfaces conflicts and invalid nodes', async () => {
    const f = fakeFetch({ data: null, error: { code: 'VALIDATION_ERROR', message: 'bad', node_id: 'h1' } }, 422)
    const err = await new EditorAPI({ baseURL: 'https://api.test/api/v1', token: 't', fetch: f })
      .saveDocument('template', 'home', { expected_revision: 1, root: { id: 'root', type: 'root' } }).catch(e => e)
    expect(err).toBeInstanceOf(ApiError)
    expect(err).toMatchObject({ status: 422, nodeId: 'h1' })
  })
  it('uploads assets as multipart and reads list totals', async () => {
    const f = vi.fn<typeof fetch>()
      .mockResolvedValueOnce(new Response(JSON.stringify({ data: { id: 'a1' }, error: null }), { status: 201 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ data: [{ id: 'a1' }], total: 7, error: null })))
      .mockResolvedValueOnce(new Response(JSON.stringify({ data: null, error: null })))
    const api = new EditorAPI({ baseURL: 'https://api.test/api/v1', token: 't', fetch: f })

    await api.uploadAsset(new File(['x'], 'a.png', { type: 'image/png' }), 'Logo')
    const init = f.mock.calls[0][1]!
    expect(init.body).toBeInstanceOf(FormData)
    expect((init.body as FormData).get('alt')).toBe('Logo')
    expect(new Headers(init.headers).has('Content-Type')).toBe(false)

    await expect(api.listAssets()).resolves.toEqual({ items: [{ id: 'a1' }], total: 7 })
    await expect(api.deleteAsset('a1')).resolves.toBeUndefined()
  })
})
