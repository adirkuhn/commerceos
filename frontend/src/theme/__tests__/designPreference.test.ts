import { describe, it, expect } from 'vitest'
import {
  DESIGN_CLASS,
  DESIGN_STORAGE_KEY,
  defaultDesign,
  loadDesignPreference,
  saveDesignPreference,
} from '../designPreference'
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

describe('defaultDesign', () => {
  it('is depot unless the build pins classic', () => {
    expect(defaultDesign(undefined)).toBe('depot')
    expect(defaultDesign('')).toBe('depot')
    expect(defaultDesign('depot')).toBe('depot')
    expect(defaultDesign('classic')).toBe('classic')
  })

  it('ignores an unknown build value', () => {
    expect(defaultDesign('%VITE_DESIGN_DEFAULT%')).toBe('depot')
  })
})

describe('loadDesignPreference', () => {
  it('uses the default when nothing is stored', () => {
    expect(loadDesignPreference(memoryStorage(), 'depot')).toBe('depot')
    expect(loadDesignPreference(memoryStorage(), 'classic')).toBe('classic')
  })

  it('returns a stored choice over the default', () => {
    expect(loadDesignPreference(memoryStorage({ [DESIGN_STORAGE_KEY]: 'classic' }), 'depot')).toBe('classic')
  })

  it('ignores an unknown stored value', () => {
    expect(loadDesignPreference(memoryStorage({ [DESIGN_STORAGE_KEY]: 'retro' }), 'depot')).toBe('depot')
  })

  it('falls back to the default when storage is unavailable', () => {
    expect(loadDesignPreference(throwingStorage, 'depot')).toBe('depot')
    expect(loadDesignPreference(null, 'classic')).toBe('classic')
  })
})

describe('saveDesignPreference', () => {
  it('stores the choice', () => {
    const storage = memoryStorage()
    saveDesignPreference(storage, 'classic')
    expect(storage.data[DESIGN_STORAGE_KEY]).toBe('classic')
  })

  it('does not throw when storage is unavailable', () => {
    expect(() => saveDesignPreference(throwingStorage, 'depot')).not.toThrow()
  })
})

describe('DESIGN_CLASS', () => {
  it('is the class index.html and the stylesheet key the depot design on', () => {
    expect(DESIGN_CLASS).toBe('design-depot')
  })
})
