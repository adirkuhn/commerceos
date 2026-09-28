import type { Breakpoint, NodeStyle, StyleProperty, ThemeNode } from './document'

// Compiles node styles (the fixed style vocabulary) into scoped CSS: one
// `.n-<id>` rule per node and breakpoint. Used by both the storefront
// runtime and the editor canvas, so what the merchant sees is what ships.
// Values are re-checked here even though the backend validated them: CSS
// is only ever built from known-safe tokens.

export const BREAKPOINTS: Record<Exclude<Breakpoint, 'base'>, number> = { tablet: 1023, mobile: 767 }

type Decl = [string, string]
type Rule = (value: string, ctx: CompileOptions) => Decl[] | undefined

export interface CompileOptions {
  resolveAsset?: (assetId: string) => string | undefined
}

const enumOf = (values: string[], map: (v: string) => Decl[]): Rule =>
  value => (values.includes(value) ? map(value) : undefined)

const SPACING = ['none', 'xs', 'sm', 'md', 'lg', 'xl', '2xl', '3xl']
const COLOR_TOKENS = ['primary', 'secondary', 'background', 'surface', 'text', 'muted', 'accent']
const HEX = /^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/
const ASSET = /^asset:([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$/
const NODE_ID = /^[A-Za-z0-9_-]{1,64}$/

const space = (...props: string[]): Rule => enumOf(SPACING, v => props.map(p => [p, `var(--space-${v})`]))

const color = (prop: string): Rule => value => {
  if (value === 'transparent') return [[prop, 'transparent']]
  if (COLOR_TOKENS.includes(value)) return [[prop, `var(--color-${value})`]]
  if (HEX.test(value)) return [[prop, value]]
  return undefined
}

const FLEX_ALIGN: Record<string, string> = { start: 'flex-start', center: 'center', end: 'flex-end', stretch: 'stretch', between: 'space-between' }

const RULES: Record<StyleProperty, Rule> = {
  padding_x: space('padding-left', 'padding-right'),
  padding_y: space('padding-top', 'padding-bottom'),
  margin_top: space('margin-top'),
  margin_bottom: space('margin-bottom'),
  gap: space('gap'),
  text_align: enumOf(['left', 'center', 'right'], v => [['text-align', v]]),
  align_items: enumOf(['start', 'center', 'end', 'stretch'], v => [['align-items', FLEX_ALIGN[v]]]),
  justify_content: enumOf(['start', 'center', 'end', 'between'], v => [['justify-content', FLEX_ALIGN[v]]]),
  font_family: enumOf(['heading', 'body'], v => [['font-family', `var(--font-${v})`]]),
  font_size: enumOf(['xs', 'sm', 'base', 'lg', 'xl', '2xl', '3xl', '4xl', '5xl'], v => [['font-size', `var(--text-${v})`]]),
  font_weight: enumOf(['regular', 'medium', 'semibold', 'bold'], v => [['font-weight', { regular: '400', medium: '500', semibold: '600', bold: '700' }[v]!]]),
  line_height: enumOf(['tight', 'normal', 'relaxed'], v => [['line-height', { tight: '1.2', normal: '1.5', relaxed: '1.75' }[v]!]]),
  letter_spacing: enumOf(['tight', 'normal', 'wide'], v => [['letter-spacing', { tight: '-0.02em', normal: '0', wide: '0.05em' }[v]!]]),
  text_transform: enumOf(['none', 'uppercase'], v => [['text-transform', v]]),
  color: color('color'),
  background_color: color('background-color'),
  border_color: color('border-color'),
  border_width: enumOf(['none', 'thin', 'medium', 'thick'], v =>
    v === 'none' ? [['border-width', '0']] : [['border-style', 'solid'], ['border-width', { thin: '1px', medium: '2px', thick: '4px' }[v]!]]),
  border_radius: enumOf(['none', 'sm', 'md', 'lg', 'xl', 'full'], v => [['border-radius', `var(--radius-${v})`]]),
  shadow: enumOf(['none', 'sm', 'md', 'lg'], v => [['box-shadow', `var(--shadow-${v})`]]),
  width: enumOf(['auto', 'full', 'content', 'narrow'], v => {
    if (v === 'auto') return [['width', 'auto']]
    if (v === 'full') return [['width', '100%']]
    const max = v === 'content' ? 'var(--layout-max-width)' : '48rem'
    return [['max-width', max], ['margin-left', 'auto'], ['margin-right', 'auto']]
  }),
  min_height: enumOf(['none', 'sm', 'md', 'lg', 'screen'], v => [['min-height', { none: '0', sm: '16rem', md: '24rem', lg: '36rem', screen: '100vh' }[v]!]]),
  aspect_ratio: enumOf(['auto', '1:1', '4:3', '3:2', '16:9', '3:4', '2:3'], v => [['aspect-ratio', v === 'auto' ? 'auto' : v.replace(':', '/')]]),
  background_image: (value, ctx) => {
    const m = ASSET.exec(value)
    const url = m && ctx.resolveAsset?.(m[1])
    return url ? [['background-image', `url("${encodeURI(url).replace(/"/g, '%22')}")`]] : undefined
  },
  background_size: enumOf(['cover', 'contain'], v => [['background-size', v]]),
  background_position: enumOf(['center', 'top', 'bottom'], v => [['background-position', v]]),
  columns: enumOf(['1', '2', '3', '4', '5', '6'], v => [['grid-template-columns', `repeat(${v},minmax(0,1fr))`]]),
  // "show" reverts to the component's own display (its base styles live in
  // a cascade layer), so a node hidden on desktop can reappear on mobile.
  display: enumOf(['show', 'hide'], v => [['display', v === 'hide' ? 'none' : 'revert-layer']]),
}

export const nodeClass = (id: string) => `n-${id}`

function declarations(values: Record<string, string> | undefined, ctx: CompileOptions): string {
  if (!values) return ''
  const out: string[] = []
  for (const [prop, value] of Object.entries(values)) {
    // Untrusted keys: a property outside the vocabulary has no rule.
    const rule: Rule | undefined = Object.hasOwn(RULES, prop) ? RULES[prop as StyleProperty] : undefined
    const decls = rule && typeof value === 'string' ? rule(value, ctx) : undefined
    for (const [k, v] of decls ?? []) out.push(`${k}:${v}`)
  }
  return out.join(';')
}

function nodeRules(id: string, style: NodeStyle, ctx: CompileOptions) {
  const selector = `.${nodeClass(id)}`
  const wrap = (body: string) => (body ? `${selector}{${body}}` : '')
  return {
    base: wrap(declarations(style.base, ctx)),
    tablet: wrap(declarations(style.tablet, ctx)),
    mobile: wrap(declarations(style.mobile, ctx)),
  }
}

export function compileDocumentStyles(root: ThemeNode, ctx: CompileOptions = {}): string {
  const base: string[] = []
  const tablet: string[] = []
  const mobile: string[] = []
  const visit = (n: ThemeNode) => {
    if (n.style && NODE_ID.test(n.id)) {
      const r = nodeRules(n.id, n.style, ctx)
      if (r.base) base.push(r.base)
      if (r.tablet) tablet.push(r.tablet)
      if (r.mobile) mobile.push(r.mobile)
    }
    n.children?.forEach(visit)
  }
  visit(root)
  const media = (max: number, rules: string[]) => (rules.length ? `@media (max-width:${max}px){${rules.join('')}}` : '')
  return base.join('') + media(BREAKPOINTS.tablet, tablet) + media(BREAKPOINTS.mobile, mobile)
}
