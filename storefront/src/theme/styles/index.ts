import type { ThemeSettings } from '../../core/document'
import { supermarketCSS } from './supermarket'

// The stylesheet each theme style adds on top of the base component CSS
// (see ThemeSettings.style). The default style adds nothing.
const STYLES: Record<NonNullable<ThemeSettings['style']>, string> = {
  default: '',
  supermarket: supermarketCSS,
}

export const THEME_STYLES = Object.keys(STYLES) as NonNullable<ThemeSettings['style']>[]

export function styleCSS(settings: Pick<ThemeSettings, 'style'>): string {
  return STYLES[settings.style ?? 'default'] ?? ''
}
