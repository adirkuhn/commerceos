import { describe, it, expect } from 'vitest'
import { defaultSettings, fontStylesheetURL, settingsToCSS } from '../settings'

describe('settingsToCSS', () => {
  it('exposes settings as custom properties on :root', () => {
    const css = settingsToCSS(defaultSettings())
    expect(css.startsWith(':root{')).toBe(true)
    expect(css).toContain('--color-primary:#1f3fbf')
    expect(css).toContain('--layout-max-width:1280px')
    expect(css).toContain('--button-radius:var(--radius-md)')
    expect(css).toContain('--product-columns:4')
    expect(css).toContain('--font-body:ui-sans-serif,system-ui')
  })

  it('quotes Google font families and keeps a system fallback', () => {
    const s = defaultSettings()
    s.typography = { ...s.typography, source: 'google', heading_font: 'Playfair Display' }
    expect(settingsToCSS(s)).toContain(`--font-heading:"Playfair Display",ui-sans-serif`)
  })

  it('ignores invalid colours instead of emitting them', () => {
    const s = defaultSettings()
    s.colors = { ...s.colors, primary: 'red;} body{x' }
    expect(settingsToCSS(s)).not.toContain('--color-primary')
  })
})

describe('fontStylesheetURL', () => {
  it('is empty for system fonts (no third-party request)', () => {
    expect(fontStylesheetURL(defaultSettings())).toBe('')
  })

  it('requests only the chosen Google families', () => {
    const s = defaultSettings()
    s.typography = { ...s.typography, source: 'google', heading_font: 'Playfair Display', body_font: 'Inter' }
    expect(fontStylesheetURL(s)).toBe('https://fonts.googleapis.com/css2?family=Playfair+Display:wght@400;500;600;700&family=Inter:wght@400;500;600;700&display=swap')
  })
})
