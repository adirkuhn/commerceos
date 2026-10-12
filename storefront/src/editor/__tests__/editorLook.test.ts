import { describe, expect, it } from 'vitest'
import {
  DEFAULT_EDITOR_APPEARANCE, editorAppearanceCss, editorFontHref, isDarkLook, takeThemePreference, type EditorAppearance,
} from '../editorLook'

function memoryStorage() {
  const m = new Map<string, string>()
  return { getItem: (k: string) => m.get(k) ?? null, setItem: (k: string, v: string) => { m.set(k, v) } } as unknown as Storage
}

const look = (o: Partial<EditorAppearance>): EditorAppearance => ({ ...DEFAULT_EDITOR_APPEARANCE, ...o })

describe('takeThemePreference', () => {
  it('takes the admin light/dark choice from the opening link and keeps it for the tab', () => {
    const storage = memoryStorage()
    expect(takeThemePreference('#session=abc&theme=dark', storage)).toBe('dark')
    expect(takeThemePreference('', storage)).toBe('dark')
  })

  it('follows the system when nothing (or nonsense) was passed', () => {
    expect(takeThemePreference('#session=abc', memoryStorage())).toBe('system')
    expect(takeThemePreference('#theme=purple', memoryStorage())).toBe('system')
  })
})

describe('isDarkLook', () => {
  it('resolves system with the OS setting', () => {
    expect(isDarkLook('system', true)).toBe(true)
    expect(isDarkLook('system', false)).toBe(false)
    expect(isDarkLook('light', true)).toBe(false)
    expect(isDarkLook('dark', false)).toBe(true)
  })
})

describe('editorAppearanceCss', () => {
  it('adds nothing for the default appearance', () => {
    expect(editorAppearanceCss(DEFAULT_EDITOR_APPEARANCE)).toBe('')
  })

  it('uses the shop font for the editor controls', () => {
    expect(editorAppearanceCss(look({ font: 'source_sans' }))).toContain("--ed-font: 'Source Sans 3', system-ui, sans-serif;")
  })

  it('uses the shop accent, with a readable tint in dark mode', () => {
    const css = editorAppearanceCss(look({ accent_color: '#0B6E4F' }))
    expect(css).toContain('--ed-accent: 11 110 79;')
    expect(css).toContain('--ed-accent-solid: 11 110 79;')
    expect(css).toMatch(/:root\.dark \{[^}]*--ed-accent: (?!11 110 79)\d+ \d+ \d+;/)
  })

  it('uses the shop corners', () => {
    expect(editorAppearanceCss(look({ corners: 'sharp' }))).toContain('--ed-radius: 3px;')
  })
})

describe('editorFontHref', () => {
  it('loads the shop font from Google Fonts', () => {
    expect(editorFontHref('atkinson')).toBe('https://fonts.googleapis.com/css2?family=Atkinson+Hyperlegible+Next:wght@400;500;600;700&display=swap')
    expect(editorFontHref('inter')).toContain('family=Inter:')
  })
})
