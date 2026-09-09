import { describe, expect, it } from 'vitest'
import { THEME_PALETTES, paletteCustomCSS } from '../palettes'

describe('paletteCustomCSS', () => {
  it('emits a :root block setting every brand CSS variable from the palette', () => {
    const palette = THEME_PALETTES.find(p => p.id === 'verde')!

    const css = paletteCustomCSS(palette)

    expect(css).toContain(':root {')
    expect(css).toContain(`--brand-50: ${palette.colors[50]};`)
    expect(css).toContain(`--brand-100: ${palette.colors[100]};`)
    expect(css).toContain(`--brand-500: ${palette.colors[500]};`)
    expect(css).toContain(`--brand-600: ${palette.colors[600]};`)
    expect(css).toContain(`--brand-700: ${palette.colors[700]};`)
  })

  it('has more than one palette to choose from, each with a unique id', () => {
    expect(THEME_PALETTES.length).toBeGreaterThan(1)
    expect(new Set(THEME_PALETTES.map(p => p.id)).size).toBe(THEME_PALETTES.length)
  })
})
