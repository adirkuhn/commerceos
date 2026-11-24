import { describe, it, expect } from 'vitest'
import {
  DEFAULT_LOOK,
  LOOKS,
  LOOK_STORAGE_KEY,
  lookClass,
  loadLookPreference,
  resolveLook,
  saveLookPreference,
} from '../lookPreference'
import type { ThemeStorage } from '../themePreference'

function memoryStorage(initial: Record<string, string> = {}): ThemeStorage & { data: Record<string, string> } {
  const data = { ...initial }
  return {
    data,
    getItem: key => (key in data ? data[key] : null),
    setItem: (key, value) => { data[key] = value },
  }
}

const throwingStorage: ThemeStorage = {
  getItem: () => { throw new Error('blocked') },
  setItem: () => { throw new Error('blocked') },
}

describe('looks', () => {
  it('offers ticket wallet and graphite, ticket wallet by default', () => {
    expect(LOOKS).toEqual(['wallet', 'graphite'])
    expect(DEFAULT_LOOK).toBe('wallet')
  })

  it('gives every look its own class on <html>, the one index.html and index.css use', () => {
    expect(lookClass('wallet')).toBe('look-wallet')
    expect(lookClass('graphite')).toBe('look-graphite')
  })
})

describe('loadLookPreference', () => {
  it('is null when this browser never picked a look (the shop default applies)', () => {
    expect(loadLookPreference(memoryStorage())).toBeNull()
  })

  it('returns a stored choice', () => {
    expect(loadLookPreference(memoryStorage({ [LOOK_STORAGE_KEY]: 'graphite' }))).toBe('graphite')
  })

  it('ignores an unknown stored value, such as a retired design', () => {
    expect(loadLookPreference(memoryStorage({ [LOOK_STORAGE_KEY]: 'depot' }))).toBeNull()
  })

  it('is null when storage is unavailable', () => {
    expect(loadLookPreference(throwingStorage)).toBeNull()
    expect(loadLookPreference(null)).toBeNull()
  })
})

describe('saveLookPreference', () => {
  it('stores the choice', () => {
    const storage = memoryStorage()
    saveLookPreference(storage, 'graphite')
    expect(storage.data[LOOK_STORAGE_KEY]).toBe('graphite')
  })

  it('forgets the choice for null, so the shop default applies again', () => {
    const storage = memoryStorage({ [LOOK_STORAGE_KEY]: 'graphite' })
    const removing = { ...storage, removeItem: (k: string) => { delete storage.data[k] } }
    saveLookPreference(removing, null)
    expect(storage.data[LOOK_STORAGE_KEY]).toBeUndefined()
  })

  it('does not throw when storage is unavailable', () => {
    expect(() => saveLookPreference(throwingStorage, 'wallet')).not.toThrow()
  })
})

describe('resolveLook', () => {
  it('prefers the person, then the shop, then Ticket Wallet', () => {
    expect(resolveLook('graphite', 'wallet')).toBe('graphite')
    expect(resolveLook(null, 'graphite')).toBe('graphite')
    expect(resolveLook(null, null)).toBe('wallet')
  })
})
