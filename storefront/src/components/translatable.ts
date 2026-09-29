import type { ThemeNode } from '../core/document'
import type { ComponentDefinition, PropField } from './types'

// Merchant-written texts can differ per store language (a node's
// `translations`); everything else -- layout, images, choices -- is the
// same in every language. The backend contract lists the same props
// (x-commerceos-component translatable); a test keeps them in step.

const TRANSLATABLE_KINDS: ReadonlySet<PropField['kind']> = new Set(['text', 'richtext', 'links'])

export function isTranslatable(field: PropField): boolean {
  return TRANSLATABLE_KINDS.has(field.kind)
}

export function translatableProps(def: ComponentDefinition): string[] {
  return Object.entries(def.props).filter(([, f]) => isTranslatable(f as PropField)).map(([name]) => name)
}

/** The node's props in one language: its translation over the defaults. */
export function propsIn(node: ThemeNode, locale: string | undefined): Record<string, unknown> {
  const props = node.props ?? {}
  const tr = locale ? node.translations?.[locale] : undefined
  return tr ? { ...props, ...tr } : props
}
