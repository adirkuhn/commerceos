import type { TFunction } from 'i18next'
import type { ComponentCategory, ComponentDefinition, PropField, SelectOption, StyleGroup } from '../../components/types'
import type { DocumentSummary } from '../../core/document'
import type { StyleControl } from '../inspector/styleControls'
import { CATEGORY_TEXTS, isTokenLabel, keySegment } from './sourceTexts'

// Merchant-facing names for code-owned texts (component definitions, style
// controls, theme documents) in the editor's language. English lives in
// the code and is the fallback; translations under the same keys in
// locales/<lang>/editor.json.

export interface EditorLabels {
  componentName(def: ComponentDefinition): string
  componentDescription(def: ComponentDefinition): string
  category(category: ComponentCategory): string
  field(def: ComponentDefinition, prop: string, field: Exclude<PropField, { kind: 'hidden' }>): string
  option(def: ComponentDefinition, prop: string, option: SelectOption): string
  styleGroup(group: StyleGroup, english: string): string
  styleProperty(control: StyleControl): string
  styleOption(control: StyleControl, option: { value: string; label: string }): string
  document(doc: DocumentSummary): string
}

const titleCase = (key: string) => key.split('-').map(w => w[0].toUpperCase() + w.slice(1)).join(' ')

export function createEditorLabels(t: TFunction<'editor'>): EditorLabels {
  // Keys here are built from code data, so they're typed loosely; every one
  // has an English default.
  const tx = (key: string, defaultValue: string) => (t as unknown as (k: string, o: object) => string)(key, { defaultValue })
  return {
    componentName: def => tx(`components.${def.type}.name`, def.name),
    componentDescription: def => tx(`components.${def.type}.description`, def.description),
    category: c => tx(`categories.${c}`, CATEGORY_TEXTS[c]),
    field: (def, prop, field) => tx(`components.${def.type}.props.${prop}`, field.label),
    option: (def, prop, o) => (isTokenLabel(o.label) ? o.label : tx(`components.${def.type}.options.${prop}.${keySegment(o.value)}`, o.label)),
    styleGroup: (g, english) => tx(`style.groups.${g}`, english),
    styleProperty: c => tx(`style.properties.${c.property}`, c.label),
    styleOption: (c, o) => (isTokenLabel(o.label) ? o.label : tx(`style.options.${c.property}.${keySegment(o.value)}`, o.label)),
    // Pages are named by the merchant; built-in documents by their key.
    document: d => (d.kind === 'page' ? d.title || titleCase(d.key) : tx(`documents.${d.kind}.${d.key}`, d.title || titleCase(d.key))),
  }
}
