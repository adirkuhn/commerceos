import { describe, it, expect, vi } from 'vitest'
import { takeSessionToken } from '../session'

function memoryStorage(): Storage {
  const m = new Map<string, string>()
  return {
    get length() { return m.size }, clear: () => m.clear(), key: i => [...m.keys()][i] ?? null,
    getItem: k => m.get(k) ?? null, setItem: (k, v) => void m.set(k, v), removeItem: k => void m.delete(k),
  }
}

describe('takeSessionToken', () => {
  it('moves the token from the URL fragment into session storage and clears the fragment', () => {
    const storage = memoryStorage()
    const replace = vi.fn()
    const token = takeSessionToken({ hash: '#session=abc-123', pathname: '/store/editor', search: '' }, storage, replace)

    expect(token).toBe('abc-123')
    expect(storage.getItem('cos-editor-session')).toBe('abc-123')
    expect(replace).toHaveBeenCalledWith('/store/editor')
  })

  it('falls back to the stored token on reload', () => {
    const storage = memoryStorage()
    storage.setItem('cos-editor-session', 'stored')
    expect(takeSessionToken({ hash: '', pathname: '/store/editor', search: '' }, storage, vi.fn())).toBe('stored')
  })

  it('returns undefined when there is none', () => {
    expect(takeSessionToken({ hash: '', pathname: '/store/editor', search: '' }, memoryStorage(), vi.fn())).toBeUndefined()
  })
})
