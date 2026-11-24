import type { Look } from './lookPreference'

// A shop's admin appearance (GET/PUT /settings/appearance): the default
// look plus corners, text size and spacing layered over whichever look is
// active. The look is a class on <html> (see useLook); the rest is applied
// as CSS custom properties over index.css, and the defaults add nothing at
// all -- so an untouched shop wears each look as designed.

// Mirrors backend/api/settings/openapi.yaml (AppearanceInput / Appearance).
export type AppearanceCorners = 'sharp' | 'normal' | 'round'
export type AppearanceTextSize = 'normal' | 'large' | 'larger'
export type AppearanceDensity = 'compact' | 'comfortable'

export interface AppearanceSettings {
  default_look: Look
  corners: AppearanceCorners
  text_size: AppearanceTextSize
  density: AppearanceDensity
}

export interface Appearance extends AppearanceSettings {
  updated_at: string | null
}

export const APPEARANCE_CORNERS: readonly AppearanceCorners[] = ['sharp', 'normal', 'round']
export const APPEARANCE_TEXT_SIZES: readonly AppearanceTextSize[] = ['normal', 'large', 'larger']
export const APPEARANCE_DENSITIES: readonly AppearanceDensity[] = ['compact', 'comfortable']

// Same values as the backend defaults. Values equal to these emit nothing.
export const DEFAULT_APPEARANCE: AppearanceSettings = {
  default_look: 'wallet',
  corners: 'normal',
  text_size: 'normal',
  density: 'comfortable',
}

const ROOT_SIZE: Record<AppearanceTextSize, string | null> = { normal: null, large: '106.25%', larger: '112.5%' }
// 'normal' keeps each look's own radii.
const RADII: Record<AppearanceCorners, [string, string, string, string] | null> = {
  sharp: ['2px', '3px', '4px', '4px'],
  normal: null,
  round: ['8px', '10px', '14px', '18px'],
}
const DENSITY: Record<AppearanceDensity, string | null> = { compact: '0.8', comfortable: null }

// appearanceCss is the stylesheet that turns the look's defaults into the
// shop's: only what differs from DEFAULT_APPEARANCE. html[class][class]
// (0,2,1) outranks index.css's look blocks (html.look-*, at most 0,2,1)
// whatever the stylesheet order (the pre-paint copy in index.html sits
// before the app's CSS). None of these differ between light and dark.
export function appearanceCss(a: AppearanceSettings): string {
  const vars: Record<string, string> = {}

  const rootSize = ROOT_SIZE[a.text_size]
  if (rootSize) vars['--root-size'] = rootSize

  const radii = RADII[a.corners]
  if (radii) {
    const [md, lg, xl, xxl] = radii
    Object.assign(vars, { '--radius-md': md, '--radius-lg': lg, '--radius-xl': xl, '--radius-2xl': xxl })
  }

  const density = DENSITY[a.density]
  if (density) vars['--density'] = density

  const entries = Object.entries(vars)
  if (entries.length === 0) return ''
  return `html[class][class] {\n${entries.map(([k, v]) => `  ${k}: ${v};`).join('\n')}\n}\n`
}
