// The editor's own look (its controls, not the store page in the canvas):
// light or dark as the admin was when it opened the editor, and the shop's
// admin appearance (Settings > Appearance) -- font, accent, corners.

export type ThemePreference = 'system' | 'light' | 'dark'

/** Mirrors the draft's `appearance` (backend api/theme/openapi.yaml EditorAppearance). */
export interface EditorAppearance {
  font: 'atkinson' | 'inter' | 'ibm_plex_sans' | 'source_sans' | 'barlow'
  accent_color: string
  sidebar_color: string
  corners: 'sharp' | 'normal' | 'round'
  text_size: 'normal' | 'large' | 'larger'
  density: 'compact' | 'comfortable'
}

export const DEFAULT_EDITOR_APPEARANCE: EditorAppearance = {
  font: 'atkinson', accent_color: '#2F33B8', sidebar_color: '#14163F', corners: 'normal', text_size: 'normal', density: 'comfortable',
}

const THEME_KEY = 'cos-editor-theme'
const PREFERENCES: readonly ThemePreference[] = ['system', 'light', 'dark']

/** The admin passes its light/dark choice in the opening link
 *  (/store/editor#session=…&theme=dark); kept for the tab like the session. */
export function takeThemePreference(hash: string, storage: Storage): ThemePreference {
  const passed = new URLSearchParams(hash.replace(/^#/, '')).get('theme')
  if (passed && (PREFERENCES as readonly string[]).includes(passed)) {
    storage.setItem(THEME_KEY, passed)
    return passed as ThemePreference
  }
  const kept = storage.getItem(THEME_KEY)
  return kept && (PREFERENCES as readonly string[]).includes(kept) ? kept as ThemePreference : 'system'
}

export function isDarkLook(preference: ThemePreference, systemPrefersDark: boolean): boolean {
  return preference === 'dark' || (preference === 'system' && systemPrefersDark)
}

const FONTS: Record<EditorAppearance['font'], { family: string; css: string }> = {
  atkinson: { family: 'Atkinson+Hyperlegible+Next', css: "'Atkinson Hyperlegible Next'" },
  inter: { family: 'Inter', css: "'Inter'" },
  ibm_plex_sans: { family: 'IBM+Plex+Sans', css: "'IBM Plex Sans'" },
  source_sans: { family: 'Source+Sans+3', css: "'Source Sans 3'" },
  barlow: { family: 'Barlow', css: "'Barlow'" },
}

export function editorFontHref(font: EditorAppearance['font']): string {
  return `https://fonts.googleapis.com/css2?family=${FONTS[font].family}:wght@400;500;600;700&display=swap`
}

const RADIUS: Record<EditorAppearance['corners'], string> = { sharp: '3px', normal: '6px', round: '10px' }
// The dark editor panel, where dark-mode accent text sits.
const DARK_RAISED = '#1B1F36'

type RGB = [number, number, number]
const rgb = (hex: string): RGB => { const n = parseInt(hex.slice(1), 16); return [(n >> 16) & 255, (n >> 8) & 255, n & 255] }
const channels = (c: RGB) => c.map(Math.round).join(' ')
const luminance = ([r, g, b]: RGB) => {
  const lin = (v: number) => { const s = v / 255; return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4 }
  return 0.2126 * lin(r) + 0.7152 * lin(g) + 0.0722 * lin(b)
}
const contrast = (a: RGB, b: RGB) => { const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x); return (hi + 0.05) / (lo + 0.05) }
const mix = (a: RGB, b: RGB, t: number): RGB => [0, 1, 2].map(i => a[i] + (b[i] - a[i]) * t) as RGB

/** The accent lightened just enough to read on the dark panel. */
function darkTint(accent: RGB): RGB {
  const bg = rgb(DARK_RAISED)
  for (let step = 0; step <= 50; step++) {
    const c = mix(accent, [255, 255, 255], step / 50)
    if (contrast(c, bg) >= 4.5) return c
  }
  return [255, 255, 255]
}

function block(selector: string, vars: Record<string, string>): string {
  const entries = Object.entries(vars)
  return entries.length ? `${selector} {\n${entries.map(([k, v]) => `  ${k}: ${v};`).join('\n')}\n}\n` : ''
}

/** Overrides of the --ed-* tokens (editor.css) for the shop's appearance;
 *  empty for the defaults. */
export function editorAppearanceCss(a: EditorAppearance): string {
  const d = DEFAULT_EDITOR_APPEARANCE
  const light: Record<string, string> = {}
  const dark: Record<string, string> = {}
  if (a.font !== d.font) light['--ed-font'] = `${FONTS[a.font].css}, system-ui, sans-serif`
  if (a.corners !== d.corners) light['--ed-radius'] = RADIUS[a.corners]
  if (a.accent_color.toUpperCase() !== d.accent_color) {
    const accent = rgb(a.accent_color)
    const solid = channels(accent)
    const hover = channels(mix(accent, [0, 0, 0], 0.15))
    Object.assign(light, { '--ed-accent': solid, '--ed-accent-solid': solid, '--ed-accent-solid-hover': hover })
    Object.assign(dark, { '--ed-accent': channels(darkTint(accent)), '--ed-accent-solid': solid, '--ed-accent-solid-hover': hover })
  }
  return block(':root', light) + block(':root.dark', dark)
}
