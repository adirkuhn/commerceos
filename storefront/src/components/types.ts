import type { ComponentType, ReactNode } from 'react'
import type { ThemeNode } from '../core/document'

// A component definition is everything the editor and the runtime know
// about one component type. The registry (registry.ts) is the single
// source of truth; the backend contract (contract.gen.json) is checked
// against it by a test.

export type ComponentCategory = 'layout' | 'content' | 'commerce' | 'navigation' | 'customer' | 'cart'

export type DataContext = '' | 'product' | 'collection' | 'cart' | 'customer' | 'order'

export type StyleGroup = 'spacing' | 'typography' | 'colors' | 'border' | 'background' | 'layout' | 'visibility'

export interface SelectOption {
  value: string | number
  label: string
}

export type PropField =
  | { kind: 'text'; label: string; maxLength?: number; multiline?: boolean }
  | { kind: 'number'; label: string; min?: number; max?: number }
  | { kind: 'select'; label: string; options: SelectOption[] }
  | { kind: 'toggle'; label: string }
  | { kind: 'richtext'; label: string }

/** Markup for a container: one element whose children the editor manages. */
export interface ContainerElement {
  tag: 'section' | 'div'
  className: string
}

export interface ComponentDefinition<P extends object = Record<string, unknown>> {
  type: string
  name: string
  category: ComponentCategory
  icon: string
  description: string
  props: { [K in keyof P]-?: PropField }
  defaults: P
  container: boolean
  /** Containers render as a single element (runtime and editor canvas). */
  element?: (props: P) => ContainerElement
  allowedChildren: string[]
  allowedParents: string[]
  dataContext: DataContext
  requiredIn: string[]
  styleGroups: StyleGroup[]
  editor: { removable: boolean; copyable: boolean; draggable: boolean }
}

export interface RendererProps<P extends object = Record<string, unknown>> {
  node: ThemeNode
  props: P
  className: string
  children?: ReactNode
}

export type Renderer<P extends object = Record<string, unknown>> = ComponentType<RendererProps<P>>
