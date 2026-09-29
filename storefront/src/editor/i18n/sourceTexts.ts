import type { ComponentCategory, ComponentDefinition, PropField } from '../../components/types'
import type { ComponentRegistry } from '../../components/registry'
import { GROUP_LABELS, STYLE_CONTROLS } from '../inspector/styleControls'

// The English texts that live in code (component definitions, style
// controls) as the source of truth. en/editor.json holds a copy under the
// same keys so translators see them; a test keeps the two identical and
// `npm run i18n:registry` rewrites the copy.

/** A value as one key segment: i18next splits keys on "." and ":". */
export const keySegment = (v: string | number) => String(v).replace(/[.:]/g, '_')

/** Size and ratio tokens (xs, 2xl, 16:9, H2, 3×, 4) read the same in every
 *  language: shown as written, never translated. */
export const isTokenLabel = (label: string) =>
  /^(?:[0-9]+(?:×|:[0-9]+)?|[0-9]?x[sl]|[0-9]xl|sm|md|lg|base|h[1-6])$/i.test(label)

export const CATEGORY_TEXTS: Record<ComponentCategory, string> = {
  layout: 'Layout', content: 'Content', commerce: 'Commerce', navigation: 'Navigation', customer: 'Customer', cart: 'Cart',
}

type Texts = Record<string, unknown>

function fieldTexts(fields: Record<string, PropField>) {
  const props: Record<string, string> = {}
  const options: Record<string, Record<string, string>> = {}
  for (const [name, field] of Object.entries(fields)) {
    if (field.kind === 'hidden') continue
    props[name] = field.label
    if (field.kind === 'select') {
      const words = field.options.filter(o => !isTokenLabel(o.label))
      if (words.length) options[name] = Object.fromEntries(words.map(o => [keySegment(o.value), o.label]))
    }
  }
  return { props, options }
}

function componentTexts(def: ComponentDefinition): Texts {
  const { props, options } = fieldTexts(def.props as Record<string, PropField>)
  return {
    name: def.name,
    description: def.description,
    ...(Object.keys(props).length ? { props } : {}),
    ...(Object.keys(options).length ? { options } : {}),
  }
}

function styleTexts(): Texts {
  const properties: Record<string, string> = {}
  const options: Record<string, Record<string, string>> = {}
  for (const controls of Object.values(STYLE_CONTROLS)) {
    for (const c of controls) {
      properties[c.property] = c.label
      const words = (c.options ?? []).filter(o => !isTokenLabel(o.label))
      if (words.length) options[c.property] = Object.fromEntries(words.map(o => [keySegment(o.value), o.label]))
    }
  }
  return { groups: { ...GROUP_LABELS }, properties, options }
}

/** The code-owned sections of en/editor.json, keys sorted like the extractor. */
export function sourceTexts(registry: ComponentRegistry): Texts {
  const components = Object.fromEntries(registry.all().map(d => [d.type, componentTexts(d)]))
  return sortKeys({ categories: { ...CATEGORY_TEXTS }, components, style: styleTexts() }) as Texts
}

export function sortKeys(v: unknown): unknown {
  if (!v || typeof v !== 'object' || Array.isArray(v)) return v
  return Object.fromEntries(Object.keys(v).sort().map(k => [k, sortKeys((v as Record<string, unknown>)[k])]))
}
