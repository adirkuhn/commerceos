// @vitest-environment jsdom
import { beforeEach, describe, expect, it } from 'vitest'
import { APPEARANCE_STORAGE_KEY, applyAppearance, renderAppearance, saveAppliedAppearance } from '../applyAppearance'
import { DEFAULT_APPEARANCE } from '../appearance'

beforeEach(() => { document.head.innerHTML = '' })

describe('applyAppearance', () => {
  it('adds the override stylesheet and the font the page lacks', () => {
    applyAppearance(document, renderAppearance({ ...DEFAULT_APPEARANCE, font: 'barlow', corners: 'round' }))

    expect(document.getElementById('appearance-css')?.textContent).toContain('--radius-lg: 10px;')
    expect(document.getElementById('appearance-font')?.getAttribute('href')).toContain('family=Barlow')
  })

  it('updates in place instead of stacking stylesheets', () => {
    applyAppearance(document, renderAppearance({ ...DEFAULT_APPEARANCE, corners: 'round' }))
    applyAppearance(document, renderAppearance({ ...DEFAULT_APPEARANCE, corners: 'sharp' }))

    expect(document.querySelectorAll('#appearance-css')).toHaveLength(1)
    expect(document.getElementById('appearance-css')?.textContent).toContain('--radius-lg: 3px;')
  })

  it('removes everything again for the defaults', () => {
    applyAppearance(document, renderAppearance({ ...DEFAULT_APPEARANCE, font: 'barlow', corners: 'round' }))
    applyAppearance(document, renderAppearance(DEFAULT_APPEARANCE))

    expect(document.getElementById('appearance-css')).toBeNull()
    expect(document.getElementById('appearance-font')).toBeNull()
  })
})

describe('saveAppliedAppearance', () => {
  it('stores what index.html replays before first paint', () => {
    const store = new Map<string, string>()
    const storage = { getItem: (k: string) => store.get(k) ?? null, setItem: (k: string, v: string) => { store.set(k, v) } }
    const applied = renderAppearance({ ...DEFAULT_APPEARANCE, font: 'inter' })

    saveAppliedAppearance(storage, applied)

    expect(JSON.parse(store.get(APPEARANCE_STORAGE_KEY)!)).toEqual(applied)
  })

  it('carries on when storage is blocked', () => {
    const storage = { getItem: () => null, setItem: () => { throw new Error('blocked') } }
    expect(() => saveAppliedAppearance(storage, renderAppearance(DEFAULT_APPEARANCE))).not.toThrow()
  })
})
