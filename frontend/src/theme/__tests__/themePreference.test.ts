import { describe, it, expect } from 'vitest'
import {
  THEME_STORAGE_KEY,
  loadThemePreference,
  nextThemePreference,
  resolveTheme,
  saveThemePreference,
  type ThemeStorage,
} from '../themePreference'

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

describe('resolveTheme', () => {
  it('follows the system appearance when the preference is system', () => {
    expect(resolveTheme('system', true)).toBe('dark')
    expect(resolveTheme('system', false)).toBe('light')
  })

  it('uses an explicit preference regardless of the system appearance', () => {
    expect(resolveTheme('light', true)).toBe('light')
    expect(resolveTheme('dark', false)).toBe('dark')
  })
})

describe('nextThemePreference', () => {
  it('cycles system → light → dark → system', () => {
    expect(nextThemePreference('system')).toBe('light')
    expect(nextThemePreference('light')).toBe('dark')
    expect(nextThemePreference('dark')).toBe('system')
  })
})

describe('loadThemePreference', () => {
  it('defaults to system when nothing is stored', () => {
    expect(loadThemePreference(memoryStorage())).toBe('system')
  })

  it('returns a valid stored preference', () => {
    expect(loadThemePreference(memoryStorage({ [THEME_STORAGE_KEY]: 'dark' }))).toBe('dark')
  })

  it('ignores an unknown stored value', () => {
    expect(loadThemePreference(memoryStorage({ [THEME_STORAGE_KEY]: 'sepia' }))).toBe('system')
  })

  it('falls back to system when storage is unavailable', () => {
    expect(loadThemePreference(throwingStorage)).toBe('system')
    expect(loadThemePreference(null)).toBe('system')
  })
})

describe('saveThemePreference', () => {
  it('stores the preference', () => {
    const storage = memoryStorage()
    saveThemePreference(storage, 'light')
    expect(storage.data[THEME_STORAGE_KEY]).toBe('light')
  })

  it('does not throw when storage is unavailable', () => {
    expect(() => saveThemePreference(throwingStorage, 'light')).not.toThrow()
    expect(() => saveThemePreference(null, 'light')).not.toThrow()
  })
})
