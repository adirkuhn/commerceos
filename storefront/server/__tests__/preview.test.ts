import { describe, it, expect } from 'vitest'
import { previewRedirect, previewTokenFrom } from '../preview.ts'

describe('previewRedirect', () => {
  it('moves ?preview=token into a cookie scoped to the store and strips it from the URL', () => {
    const r = previewRedirect('/store/key-1/pages/about?preview=tok&x=1')!
    expect(r.location).toBe('/store/key-1/pages/about?x=1')
    expect(r.cookie).toBe('cos_preview=tok; Path=/store/key-1; HttpOnly; SameSite=Lax; Max-Age=86400')
  })

  it('clears the cookie with ?preview=off', () => {
    expect(previewRedirect('/store/key-1/?preview=off')!.cookie).toBe('cos_preview=; Path=/store/key-1; HttpOnly; SameSite=Lax; Max-Age=0')
  })

  it('ignores URLs without a preview parameter or outside a store', () => {
    expect(previewRedirect('/store/key-1/')).toBeUndefined()
    expect(previewRedirect('/elsewhere?preview=tok')).toBeUndefined()
  })
})

describe('previewTokenFrom', () => {
  it('reads the cookie', () => {
    expect(previewTokenFrom('a=1; cos_preview=tok%2F1; b=2')).toBe('tok/1')
    expect(previewTokenFrom(undefined)).toBeUndefined()
    expect(previewTokenFrom('cos_preview=')).toBeUndefined()
  })
})
