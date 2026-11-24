import { describe, expect, it } from 'vitest'
import { DEFAULT_APPEARANCE, appearanceCss, type AppearanceSettings } from '../appearance'

function withAppearance(overrides: Partial<AppearanceSettings>): AppearanceSettings {
  return { ...DEFAULT_APPEARANCE, ...overrides }
}

describe('DEFAULT_APPEARANCE', () => {
  it('matches the backend defaults', () => {
    expect(DEFAULT_APPEARANCE).toEqual({ default_look: 'wallet', corners: 'normal', text_size: 'normal', density: 'comfortable' })
  })
})

describe('appearanceCss', () => {
  it('adds nothing for the defaults, so the designed look stays exact', () => {
    expect(appearanceCss(DEFAULT_APPEARANCE)).toBe('')
  })

  it('adds nothing for the default look either -- the look is a class, not CSS', () => {
    expect(appearanceCss(withAppearance({ default_look: 'graphite' }))).toBe('')
  })

  it('restyles every look, outranking the look blocks in index.css whatever the order', () => {
    const css = appearanceCss(withAppearance({ corners: 'round' }))
    expect(css).toMatch(/^html\[class\]\[class\] \{/)
    expect(css).not.toContain('.dark')
  })

  it.each([
    ['large', '106.25%'],
    ['larger', '112.5%'],
  ] as const)('scales %s text', (textSize, size) => {
    expect(appearanceCss(withAppearance({ text_size: textSize }))).toBe(`html[class][class] {\n  --root-size: ${size};\n}\n`)
  })

  it('sharpens the corners', () => {
    const css = appearanceCss(withAppearance({ corners: 'sharp' }))
    expect(css).toContain('--radius-md: 2px;')
    expect(css).toContain('--radius-lg: 3px;')
    expect(css).toContain('--radius-xl: 4px;')
    expect(css).toContain('--radius-2xl: 4px;')
  })

  it('rounds the corners', () => {
    const css = appearanceCss(withAppearance({ corners: 'round' }))
    expect(css).toContain('--radius-md: 8px;')
    expect(css).toContain('--radius-lg: 10px;')
    expect(css).toContain('--radius-xl: 14px;')
    expect(css).toContain('--radius-2xl: 18px;')
  })

  it('tightens the spacing', () => {
    expect(appearanceCss(withAppearance({ density: 'compact' }))).toBe('html[class][class] {\n  --density: 0.8;\n}\n')
  })

  it('combines overrides in one block', () => {
    const css = appearanceCss(withAppearance({ text_size: 'larger', corners: 'sharp', density: 'compact' }))
    expect(css.match(/\{/g)).toHaveLength(1)
    expect(css).toContain('--root-size: 112.5%;')
    expect(css).toContain('--radius-lg: 3px;')
    expect(css).toContain('--density: 0.8;')
  })
})
