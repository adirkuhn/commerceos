import { describe, it, expect } from 'vitest'
import { contrastRatio, readableOn, textSafe } from '../contrast'
import { defaultSettings, settingsToCSS } from '../settings'

describe('contrast', () => {
  it('measures WCAG contrast', () => {
    expect(contrastRatio('#ffffff', '#000000')).toBeCloseTo(21, 0)
    expect(contrastRatio('#ffffff', '#0aad0a')).toBeCloseTo(3.0, 1)
  })

  it('picks white or dark text, whichever reads better on a fill', () => {
    expect(readableOn('#1f3fbf')).toBe('#ffffff')
    expect(readableOn('#f9a825')).toBe('#111111')
    expect(readableOn('#e4572e')).toBe('#111111')
  })

  it('darkens a colour used as text until it reaches 4.5:1 on the background', () => {
    expect(textSafe('#1f3fbf', '#ffffff')).toBe('#1f3fbf')
    const sale = textSafe('#f9a825', '#ffffff')
    expect(contrastRatio(sale, '#ffffff')).toBeGreaterThanOrEqual(4.5)
  })
})

describe('settingsToCSS readable colours', () => {
  it('derives on-colours and text-safe accent/primary', () => {
    const s = defaultSettings()
    s.colors.accent = '#f9a825'
    const css = settingsToCSS(s)
    expect(css).toContain('--color-on-primary:#ffffff')
    expect(css).toContain('--color-on-accent:#111111')
    expect(css).toMatch(/--color-accent-text:#[0-9a-f]{6}/)
    expect(css).toContain('--color-primary-text:#1f3fbf')
  })
})
