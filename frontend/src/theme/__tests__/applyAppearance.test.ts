// @vitest-environment jsdom
import { beforeEach, describe, expect, it } from 'vitest'
import { APPEARANCE_STORAGE_KEY, applyAppearance, renderAppearance, saveAppliedAppearance } from '../applyAppearance'
import { DEFAULT_APPEARANCE } from '../appearance'

beforeEach(() => { document.head.innerHTML = '' })

describe('renderAppearance', () => {
  it('carries the CSS and the shop default look', () => {
    expect(renderAppearance({ ...DEFAULT_APPEARANCE, default_look: 'graphite', density: 'compact' })).toEqual({
      css: 'html[class][class] {\n  --density: 0.8;\n}\n',
      look: 'graphite',
    })
  })
})

describe('applyAppearance', () => {
  it('adds the override stylesheet', () => {
    applyAppearance(document, renderAppearance({ ...DEFAULT_APPEARANCE, corners: 'round' }))

    expect(document.getElementById('appearance-css')?.textContent).toContain('--radius-lg: 10px;')
  })

  it('updates in place instead of stacking stylesheets', () => {
    applyAppearance(document, renderAppearance({ ...DEFAULT_APPEARANCE, corners: 'round' }))
    applyAppearance(document, renderAppearance({ ...DEFAULT_APPEARANCE, corners: 'sharp' }))

    expect(document.querySelectorAll('#appearance-css')).toHaveLength(1)
    expect(document.getElementById('appearance-css')?.textContent).toContain('--radius-lg: 3px;')
  })

  it('keeps the override stylesheet after the app stylesheets', () => {
    applyAppearance(document, renderAppearance({ ...DEFAULT_APPEARANCE, corners: 'round' }))
    document.head.appendChild(document.createElement('style'))

    applyAppearance(document, renderAppearance({ ...DEFAULT_APPEARANCE, corners: 'sharp' }))

    expect(document.head.lastElementChild?.id).toBe('appearance-css')
  })

  it('removes the stylesheet again for the defaults', () => {
    applyAppearance(document, renderAppearance({ ...DEFAULT_APPEARANCE, corners: 'round' }))
    applyAppearance(document, renderAppearance(DEFAULT_APPEARANCE))

    expect(document.getElementById('appearance-css')).toBeNull()
  })

  it('drops a font stylesheet left over from the old font setting', () => {
    const link = document.createElement('link')
    link.id = 'appearance-font'
    document.head.appendChild(link)

    applyAppearance(document, renderAppearance(DEFAULT_APPEARANCE))

    expect(document.getElementById('appearance-font')).toBeNull()
  })
})

describe('saveAppliedAppearance', () => {
  it('stores the css and look that index.html replays before first paint', () => {
    const store = new Map<string, string>()
    const storage = { getItem: (k: string) => store.get(k) ?? null, setItem: (k: string, v: string) => { store.set(k, v) } }

    saveAppliedAppearance(storage, renderAppearance({ ...DEFAULT_APPEARANCE, default_look: 'graphite', corners: 'sharp' }))

    const cached = JSON.parse(store.get(APPEARANCE_STORAGE_KEY)!)
    expect(Object.keys(cached).sort()).toEqual(['css', 'look'])
    expect(cached.look).toBe('graphite')
    expect(cached.css).toContain('--radius-lg: 3px;')
  })

  it('carries on when storage is blocked', () => {
    const storage = { getItem: () => null, setItem: () => { throw new Error('blocked') } }
    expect(() => saveAppliedAppearance(storage, renderAppearance(DEFAULT_APPEARANCE))).not.toThrow()
  })
})
