// A shop's admin appearance (GET/PUT /settings/appearance): font, colours,
// corners, text size and spacing, layered over whichever look is active.
// It is applied as CSS custom properties over index.css, and the defaults
// add nothing at all -- so an untouched shop wears each look as designed.

// Mirrors backend/api/settings/openapi.yaml (AppearanceInput / Appearance).
export type AppearanceFont = 'atkinson' | 'inter' | 'ibm_plex_sans' | 'source_sans' | 'barlow'
export type AppearanceCorners = 'sharp' | 'normal' | 'round'
export type AppearanceTextSize = 'normal' | 'large' | 'larger'
export type AppearanceDensity = 'compact' | 'comfortable'

export interface AppearanceSettings {
  font: AppearanceFont
  accent_color: string
  sidebar_color: string
  corners: AppearanceCorners
  text_size: AppearanceTextSize
  density: AppearanceDensity
}

export interface Appearance extends AppearanceSettings {
  updated_at: string | null
}

export const APPEARANCE_FONTS: readonly AppearanceFont[] = ['atkinson', 'inter', 'ibm_plex_sans', 'source_sans', 'barlow']
export const APPEARANCE_CORNERS: readonly AppearanceCorners[] = ['sharp', 'normal', 'round']
export const APPEARANCE_TEXT_SIZES: readonly AppearanceTextSize[] = ['normal', 'large', 'larger']
export const APPEARANCE_DENSITIES: readonly AppearanceDensity[] = ['compact', 'comfortable']

// Same values as the backend defaults. Values equal to these emit nothing.
export const DEFAULT_APPEARANCE: AppearanceSettings = {
  font: 'atkinson',
  accent_color: '#2F33B8',
  sidebar_color: '#14163F',
  corners: 'normal',
  text_size: 'normal',
  density: 'comfortable',
}

// WCAG AA for normal text: white button text on the accent.
export const MIN_CONTRAST = 4.5
const WHITE = '#FFFFFF'
const BLACK = '#000000'
// The darkest "raised" surface of the looks (index.css), where dark-mode accent text sits.
const DARK_RAISED = '#1B1F36'

interface FontSpec {
  label: string
  sans: string
  mono: string
  data?: string
  // Families index.html does not already load, as Google Fonts css2 params.
  load?: string[]
  // Small x-height fonts need one step up to stay legible.
  scale?: number
}

export const FONT_SPECS: Readonly<Record<AppearanceFont, FontSpec>> = {
  atkinson: { label: 'Atkinson Hyperlegible Next', sans: "'Atkinson Hyperlegible Next'", mono: "'Atkinson Hyperlegible Mono'" },
  inter: { label: 'Inter', sans: "'Inter'", mono: "'JetBrains Mono'" },
  ibm_plex_sans: {
    label: 'IBM Plex Sans', sans: "'IBM Plex Sans'", mono: "'IBM Plex Mono'",
    load: ['IBM+Plex+Sans:wght@400;500;600;700', 'IBM+Plex+Mono:wght@400;500'],
  },
  source_sans: {
    label: 'Source Sans 3', sans: "'Source Sans 3'", mono: "'Source Code Pro'",
    load: ['Source+Sans+3:wght@400;500;600;700', 'Source+Code+Pro:wght@400;500'],
  },
  barlow: {
    label: 'Barlow', sans: "'Barlow'", mono: "'Barlow Semi Condensed', 'Barlow'", data: "'Barlow Semi Condensed', 'Barlow'",
    load: ['Barlow:wght@400;500;600;700', 'Barlow+Semi+Condensed:wght@400;500;600'], scale: 1.0625,
  },
}

const TEXT_SCALE: Record<AppearanceTextSize, number> = { normal: 1, large: 1.0625, larger: 1.125 }
const RADII: Record<AppearanceCorners, [string, string, string, string]> = {
  sharp: ['2px', '3px', '4px', '4px'],
  normal: ['6px', '6px', '10px', '10px'],
  round: ['8px', '10px', '14px', '18px'],
}
const DENSITY: Record<AppearanceDensity, number> = { compact: 0.8, comfortable: 1 }

type RGB = [number, number, number]

function rgb(hex: string): RGB {
  const n = parseInt(hex.slice(1), 16)
  return [(n >> 16) & 255, (n >> 8) & 255, n & 255]
}

function hex([r, g, b]: RGB): string {
  return '#' + [r, g, b].map(c => Math.round(c).toString(16).padStart(2, '0')).join('').toUpperCase()
}

function channels(color: string): string {
  return rgb(color).join(' ')
}

function luminance(color: string): number {
  const [r, g, b] = rgb(color).map(c => {
    const s = c / 255
    return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4
  })
  return 0.2126 * r + 0.7152 * g + 0.0722 * b
}

export function contrastRatio(a: string, b: string): number {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x)
  return (hi + 0.05) / (lo + 0.05)
}

export function meetsAccentContrast(accent: string): boolean {
  return contrastRatio(accent, WHITE) >= MIN_CONTRAST
}

function mix(from: string, to: string, amount: number): string {
  const a = rgb(from)
  const b = rgb(to)
  return hex([0, 1, 2].map(i => a[i] + (b[i] - a[i]) * amount) as RGB)
}

// Moves color towards target in small steps until it reaches min contrast
// against background: the closest passing shade of the same hue.
function towardsContrast(color: string, target: string, background: string, min: number): string {
  for (let step = 0; step <= 50; step++) {
    const candidate = step === 0 ? color.toUpperCase() : mix(color, target, step / 50)
    if (contrastRatio(candidate, background) >= min) return candidate
  }
  return target
}

// darkenToContrast suggests the closest shade of accent that white button
// text is readable on (the accent itself when it already is).
export function darkenToContrast(accent: string): string {
  return towardsContrast(accent, BLACK, WHITE, MIN_CONTRAST)
}

function isDark(color: string): boolean {
  // Where white and black text reach the same contrast.
  return luminance(color) < 0.179
}

function sidebarVars(sidebar: string, accent: string): Record<string, string> {
  const dark = isDark(sidebar)
  // Aim for 7:1; a mid-tone sidebar gets pure black or white (≥4.58:1).
  const ink = towardsContrast(dark ? '#F3F4FA' : '#1A1C2E', dark ? WHITE : BLACK, sidebar, 7)
  const ink2 = towardsContrast(mix(ink, sidebar, 0.35), ink, sidebar, MIN_CONTRAST)
  const mark = towardsContrast(accent, dark ? WHITE : BLACK, sidebar, 3)
  return {
    '--rail': channels(sidebar),
    '--rail-active': channels(mix(sidebar, dark ? WHITE : BLACK, 0.12)),
    '--rail-ink': channels(ink),
    '--rail-ink-2': channels(ink2),
    '--rail-hover': dark ? '255 255 255' : '0 0 0',
    '--rail-mark': channels(mark),
  }
}

function block(selector: string, vars: Record<string, string>): string {
  const entries = Object.entries(vars)
  if (entries.length === 0) return ''
  return `${selector} {\n${entries.map(([k, v]) => `  ${k}: ${v};`).join('\n')}\n}\n`
}

// appearanceCss is the stylesheet that turns the look's defaults into the
// shop's: only what differs from DEFAULT_APPEARANCE. html[class][class]
// (0,2,1) outranks index.css's look blocks (html.look-*, at most 0,2,1 for
// light tokens the dark block does not repeat) and html[class][class].dark
// (0,3,1) outranks every dark block, whatever the stylesheet order (the pre-paint copy in index.html sits before the app's
// CSS).
export function appearanceCss(a: AppearanceSettings): string {
  const d = DEFAULT_APPEARANCE
  const light: Record<string, string> = {}
  const dark: Record<string, string> = {}

  if (a.font !== d.font) {
    const f = FONT_SPECS[a.font]
    light['--font-sans'] = `${f.sans}, system-ui, sans-serif`
    light['--font-data'] = f.data ? `${f.data}, system-ui, sans-serif` : 'var(--font-sans)'
    light['--font-mono'] = `${f.mono}, monospace`
  }
  const scale = (FONT_SPECS[a.font].scale ?? 1) * TEXT_SCALE[a.text_size]
  if (scale !== 1) light['--root-size'] = `${+(scale * 100).toFixed(4)}%`

  if (a.accent_color.toUpperCase() !== d.accent_color) {
    const accent = a.accent_color.toUpperCase()
    const solid = channels(accent)
    light['--accent'] = solid
    light['--accent-solid'] = solid
    light['--accent-solid-hover'] = channels(mix(accent, BLACK, 0.15))
    light['--focus'] = solid
    const tint = channels(towardsContrast(accent, WHITE, DARK_RAISED, MIN_CONTRAST))
    dark['--accent'] = tint
    dark['--accent-solid'] = solid
    dark['--accent-solid-hover'] = channels(mix(accent, BLACK, 0.15))
    dark['--focus'] = tint
  }

  if (a.sidebar_color.toUpperCase() !== d.sidebar_color || a.accent_color.toUpperCase() !== d.accent_color) {
    const sidebar = sidebarVars(a.sidebar_color.toUpperCase(), a.accent_color.toUpperCase())
    Object.assign(light, sidebar)
    // The sidebar is the same colour in both appearances.
    Object.assign(dark, sidebar)
  }

  if (a.corners !== d.corners) {
    const [md, lg, xl, xxl] = RADII[a.corners]
    Object.assign(light, { '--radius-md': md, '--radius-lg': lg, '--radius-xl': xl, '--radius-2xl': xxl })
  }
  if (a.density !== d.density) light['--density'] = String(DENSITY[a.density])

  return block('html[class][class]', light) + block('html[class][class].dark', dark)
}

// fontStylesheetHref is the Google Fonts stylesheet a font needs, or null
// when index.html already loads it.
export function fontStylesheetHref(font: AppearanceFont): string | null {
  const load = FONT_SPECS[font].load
  if (!load) return null
  return `https://fonts.googleapis.com/css2?${load.map(f => `family=${f}`).join('&')}&display=swap`
}
