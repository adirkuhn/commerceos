import type { ThemeSettings } from './document'

// Theme settings become CSS custom properties on :root; components read
// the variables, so a settings change restyles everything consistently.

const SYSTEM_STACKS: Record<string, string> = {
  'system-sans': 'ui-sans-serif,system-ui,-apple-system,"Segoe UI",Roboto,"Helvetica Neue",Arial,sans-serif',
  'system-serif': 'ui-serif,Georgia,Cambria,"Times New Roman",serif',
  'system-rounded': 'ui-rounded,"SF Pro Rounded",system-ui,sans-serif',
  'system-mono': 'ui-monospace,SFMono-Regular,Menlo,Consolas,monospace',
}

const HEX = /^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/
const GOOGLE_FAMILY = /^[A-Za-z0-9][A-Za-z0-9 ]{0,63}$/

const TYPE_SCALES: Record<ThemeSettings['typography']['scale'], number> = { sm: 0.9375, md: 1, lg: 1.0625 }
const TEXT_STEPS: Record<string, number> = { xs: 0.75, sm: 0.875, base: 1, lg: 1.125, xl: 1.25, '2xl': 1.5, '3xl': 1.875, '4xl': 2.25, '5xl': 3 }

export function defaultSettings(): ThemeSettings {
  return {
    brand: {},
    colors: { primary: '#1f3fbf', secondary: '#0f172a', background: '#ffffff', surface: '#f6f7f9', text: '#111827', muted: '#6b7280', accent: '#e4572e' },
    typography: { source: 'system', heading_font: 'system-sans', body_font: 'system-sans', scale: 'md' },
    layout: { max_width: 1280, gutter: 'md' },
    buttons: { radius: 'md', size: 'md', style: 'solid' },
    product: { grid_columns: 4, image_ratio: '1:1' },
    header: { layout: 'logo-left', sticky: true },
    footer: { layout: 'columns' },
  }
}

function fontStack(font: string, source: ThemeSettings['typography']['source']): string {
  if (SYSTEM_STACKS[font]) return SYSTEM_STACKS[font]
  if (source === 'google' && GOOGLE_FAMILY.test(font)) return `"${font}",${SYSTEM_STACKS['system-sans']}`
  return SYSTEM_STACKS['system-sans']
}

export function settingsToCSS(s: ThemeSettings): string {
  const vars: [string, string][] = []
  for (const [name, value] of Object.entries(s.colors)) {
    if (HEX.test(value)) vars.push([`--color-${name}`, value])
  }
  vars.push(['--font-heading', fontStack(s.typography.heading_font, s.typography.source)])
  vars.push(['--font-body', fontStack(s.typography.body_font, s.typography.source)])
  const scale = TYPE_SCALES[s.typography.scale] ?? 1
  for (const [step, rem] of Object.entries(TEXT_STEPS)) vars.push([`--text-${step}`, `${+(rem * scale).toFixed(4)}rem`])
  if (Number.isFinite(s.layout.max_width)) vars.push(['--layout-max-width', `${Math.round(s.layout.max_width)}px`])
  vars.push(['--layout-gutter', `var(--space-${s.layout.gutter})`])
  vars.push(['--button-radius', `var(--radius-${s.buttons.radius})`])
  if (Number.isInteger(s.product.grid_columns)) vars.push(['--product-columns', String(s.product.grid_columns)])
  vars.push(['--product-image-ratio', s.product.image_ratio.replace(':', '/')])
  return `:root{${vars.map(([k, v]) => `${k}:${v}`).join(';')}}`
}

export function fontStylesheetURL(s: ThemeSettings): string {
  if (s.typography.source !== 'google') return ''
  const families = [...new Set([s.typography.heading_font, s.typography.body_font])]
    .filter(f => !SYSTEM_STACKS[f] && GOOGLE_FAMILY.test(f))
  if (!families.length) return ''
  const params = families.map(f => `family=${f.replace(/ /g, '+')}:wght@400;500;600;700`).join('&')
  return `https://fonts.googleapis.com/css2?${params}&display=swap`
}
