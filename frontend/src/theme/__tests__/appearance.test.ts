import { describe, expect, it } from 'vitest'
import {
  DEFAULT_APPEARANCE, appearanceCss, contrastRatio, darkenToContrast, fontStylesheetHref, meetsAccentContrast,
  type AppearanceSettings,
} from '../appearance'

function withAppearance(overrides: Partial<AppearanceSettings>): AppearanceSettings {
  return { ...DEFAULT_APPEARANCE, ...overrides }
}

describe('contrastRatio', () => {
  it('follows WCAG relative luminance', () => {
    expect(contrastRatio('#000000', '#FFFFFF')).toBeCloseTo(21, 1)
    expect(contrastRatio('#FFFFFF', '#FFFFFF')).toBeCloseTo(1, 5)
    expect(contrastRatio('#767676', '#FFFFFF')).toBeCloseTo(4.54, 2)
  })
})

describe('meetsAccentContrast', () => {
  it('accepts an accent white button text is readable on', () => {
    expect(meetsAccentContrast('#2F33B8')).toBe(true)
    expect(meetsAccentContrast('#767676')).toBe(true)
  })

  it('rejects an accent too light for white text', () => {
    expect(meetsAccentContrast('#777777')).toBe(false)
    expect(meetsAccentContrast('#FFFF00')).toBe(false)
  })
})

describe('darkenToContrast', () => {
  it('suggests the closest darker shade that passes, keeping the hue', () => {
    const suggestion = darkenToContrast('#FFB000')
    expect(meetsAccentContrast(suggestion)).toBe(true)
    const [r, g, b] = [1, 3, 5].map(i => parseInt(suggestion.slice(i, i + 2), 16))
    expect(r).toBeGreaterThan(g)
    expect(g).toBeGreaterThan(b)
  })

  it('leaves a passing colour as it is', () => {
    expect(darkenToContrast('#2F33B8')).toBe('#2F33B8')
  })
})

describe('appearanceCss', () => {
  it('adds nothing for the defaults, so the designed look stays exact', () => {
    expect(appearanceCss(DEFAULT_APPEARANCE)).toBe('')
  })

  it('only restyles the new design, outranking index.css whatever the order', () => {
    const css = appearanceCss(withAppearance({ corners: 'round' }))
    expect(css).toMatch(/^html\.design-depot \{/)
    expect(css).not.toMatch(/(^|\})\s*\.dark \{/)
  })

  it('sets the font family and loads nothing else for it', () => {
    const css = appearanceCss(withAppearance({ font: 'ibm_plex_sans' }))
    expect(css).toContain("--font-sans: 'IBM Plex Sans', system-ui, sans-serif;")
    expect(css).toContain("--font-mono: 'IBM Plex Mono', monospace;")
    expect(css).not.toContain('--accent')
  })

  it('uses the accent for buttons and a lighter tint of it as text in dark mode', () => {
    const css = appearanceCss(withAppearance({ accent_color: '#0B6E4F' }))
    expect(css).toContain('--accent: 11 110 79;')
    expect(css).toContain('--accent-solid: 11 110 79;')
    const dark = css.slice(css.indexOf('html.design-depot.dark'))
    const tint = /--accent: (\d+) (\d+) (\d+);/.exec(dark)
    expect(tint).not.toBeNull()
    const hex = '#' + tint!.slice(1).map(n => Number(n).toString(16).padStart(2, '0')).join('')
    expect(contrastRatio(hex, '#1B1F36')).toBeGreaterThanOrEqual(4.5)
  })

  it('gives a light sidebar dark text and a dark sidebar light text', () => {
    const light = appearanceCss(withAppearance({ sidebar_color: '#F4F1E8' }))
    const dark = appearanceCss(withAppearance({ sidebar_color: '#3A0D12' }))
    const ink = (css: string) => /--depot-ink: (\d+) (\d+) (\d+);/.exec(css)!.slice(1).map(Number)
    expect(ink(light)[0]).toBeLessThan(80)
    expect(ink(dark)[0]).toBeGreaterThan(200)
  })

  it('keeps the sidebar secondary text readable on the chosen colour', () => {
    for (const sidebar of ['#F4F1E8', '#3A0D12', '#7A7A7A', '#0B6E4F']) {
      const css = appearanceCss(withAppearance({ sidebar_color: sidebar }))
      const [r, g, b] = /--depot-ink-2: (\d+) (\d+) (\d+);/.exec(css)!.slice(1).map(Number)
      const hex = '#' + [r, g, b].map(n => n.toString(16).padStart(2, '0')).join('')
      expect(contrastRatio(hex, sidebar)).toBeGreaterThanOrEqual(4.5)
    }
  })

  it('scales text, corners and spacing', () => {
    const css = appearanceCss(withAppearance({ text_size: 'larger', corners: 'sharp', density: 'compact' }))
    expect(css).toContain('--root-size: 112.5%;')
    expect(css).toContain('--radius-lg: 3px;')
    expect(css).toContain('--density: 0.8;')
  })
})

describe('fontStylesheetHref', () => {
  it('loads the chosen family from Google Fonts', () => {
    expect(fontStylesheetHref('source_sans')).toBe(
      'https://fonts.googleapis.com/css2?family=Source+Sans+3:wght@400;500;600;700&family=Source+Code+Pro:wght@400;500&display=swap',
    )
  })

  it('needs nothing extra for fonts the page already loads', () => {
    expect(fontStylesheetHref('atkinson')).toBeNull()
    expect(fontStylesheetHref('inter')).toBeNull()
  })
})
