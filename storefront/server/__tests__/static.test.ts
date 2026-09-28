import { describe, it, expect } from 'vitest'
import { staticFileFor } from '../static.ts'

describe('staticFileFor', () => {
  const root = '/srv/dist/client'

  it('maps hashed assets with a long cache', () => {
    expect(staticFileFor(root, '/store/assets/runtime-abc123.js')).toEqual({
      path: '/srv/dist/client/assets/runtime-abc123.js', contentType: 'text/javascript; charset=utf-8', cacheControl: 'public, max-age=31536000, immutable',
    })
  })

  it('serves the editor page uncached', () => {
    expect(staticFileFor(root, '/store/editor')).toMatchObject({ path: '/srv/dist/client/editor.html', cacheControl: 'no-cache' })
  })

  it('refuses path traversal and unknown paths', () => {
    expect(staticFileFor(root, '/store/assets/../../etc/passwd')).toBeUndefined()
    expect(staticFileFor(root, '/store/assets/%2e%2e/secret')).toBeUndefined()
    expect(staticFileFor(root, '/store/key-1/')).toBeUndefined()
  })
})
