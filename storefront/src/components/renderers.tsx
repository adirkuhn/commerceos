import { createElement } from 'react'
import type { Renderer, RendererProps } from './types'
import { registry } from './registry'
import type { HeadingProps } from './content/heading'
import type { TextProps } from './content/text'
import { RichTextView } from './content/RichTextView'
import type { SpacerProps } from './content/spacer'
import { ImageView } from './content/ImageView'
import { ButtonView } from './content/ButtonView'
import { ProductGridView } from './commerce/ProductGridView'
import { ProductCardView } from './commerce/ProductCardView'

// Containers share one renderer: their definition's element() gives the
// markup, which is exactly what the editor canvas builds too.
function ContainerRenderer({ node, props, className, children }: RendererProps) {
  const def = registry.get(node.type)
  const el = def?.element?.(props) ?? { tag: 'div' as const, className: '' }
  return createElement(el.tag, { className: `${el.className} ${className}`.trim(), 'data-node': node.id }, children)
}

function HeadingRenderer({ node, props, className }: RendererProps<HeadingProps>) {
  const level = Number.isInteger(props.level) && props.level >= 1 && props.level <= 6 ? props.level : 2
  return createElement(`h${level}`, { className: `cos-heading ${className}`, 'data-node': node.id }, props.text ?? '')
}

function TextRenderer({ node, props, className }: RendererProps<TextProps>) {
  return (
    <div className={`cos-text ${className}`} data-node={node.id}>
      <RichTextView content={props.content ?? []} />
    </div>
  )
}

const SPACER_SIZES = new Set(['xs', 'sm', 'md', 'lg', 'xl', '2xl', '3xl'])

function SpacerRenderer({ node, props, className }: RendererProps<SpacerProps>) {
  const size = SPACER_SIZES.has(props.size) ? props.size : 'lg'
  return <div className={`cos-spacer cos-spacer-${size} ${className}`} data-node={node.id} aria-hidden="true" />
}

// A node's props were validated against its component's contract schema
// before the document was saved, so each renderer can rely on its own
// props shape.
const typed = <P extends object>(r: Renderer<P>) => r as unknown as Renderer

const renderers: Record<string, Renderer> = {
  section: ContainerRenderer,
  container: ContainerRenderer,
  columns: ContainerRenderer,
  column: ContainerRenderer,
  heading: typed(HeadingRenderer),
  text: typed(TextRenderer),
  spacer: typed(SpacerRenderer),
  image: typed(ImageView),
  button: typed(ButtonView),
  'product-grid': typed(ProductGridView),
  'product-card': typed(ProductCardView),
}

export function rendererFor(type: string): Renderer | undefined {
  return renderers[type]
}

export function registeredRendererTypes(): string[] {
  return Object.keys(renderers)
}

