import type { NodeStyle, ThemeNode } from '../../core/document'
import { nodeClass } from '../../core/style'
import type { ComponentRegistry } from '../../components/registry'
import { newNodeId } from './ids'

// The canonical theme document is our ThemeNode tree -- never GrapesJS
// project data. These two functions are the only bridge between them.

/** What we hand GrapesJS to add a component (a component definition). */
export interface GrapesDefinition {
  type: string
  tagName?: string
  classes?: string[]
  cosId: string
  cosProps?: Record<string, unknown>
  cosTranslations?: Translations
  cosStyle?: NodeStyle
  components?: GrapesDefinition[]
}

/** The slice of a GrapesJS component model the serializer reads. */
export interface GrapesComponentLike {
  get(key: string): unknown
  components(): Iterable<GrapesComponentLike> | { models: GrapesComponentLike[] }
}

export function toGrapes(root: ThemeNode, registry: ComponentRegistry): GrapesDefinition[] {
  const convert = (n: ThemeNode): GrapesDefinition => {
    const def = registry.get(n.type)
    const props = { ...(def?.defaults ?? {}), ...(n.props ?? {}) }
    const el = def?.element?.(props as never)
    return {
      type: n.type,
      ...(el ? { tagName: el.tag, classes: [...el.className.split(' '), nodeClass(n.id)] } : {}),
      cosId: n.id,
      cosProps: props,
      ...(n.translations ? { cosTranslations: n.translations } : {}),
      cosStyle: n.style ?? {},
      components: (n.children ?? []).map(convert),
    }
  }
  return (root.children ?? []).map(convert)
}

function childrenOf(c: GrapesComponentLike): GrapesComponentLike[] {
  const list = c.components()
  return 'models' in list ? list.models : [...list]
}

const isEmpty = (o: object | undefined) => !o || Object.keys(o).length === 0

function cleanStyle(style: NodeStyle | undefined): NodeStyle | undefined {
  if (!style) return undefined
  const out: NodeStyle = {}
  for (const bp of ['base', 'tablet', 'mobile'] as const) {
    if (!isEmpty(style[bp])) out[bp] = style[bp]
  }
  return isEmpty(out) ? undefined : out
}

type Translations = NonNullable<ThemeNode['translations']>

/** A blank translated text means "use the default language" -- left out. */
function cleanTranslations(tr: Translations | undefined): Translations | undefined {
  if (!tr) return undefined
  const out: Translations = {}
  for (const [locale, props] of Object.entries(tr)) {
    const kept = Object.fromEntries(Object.entries(props).filter(([, v]) => v !== '' && v !== undefined))
    if (!isEmpty(kept)) out[locale] = kept
  }
  return isEmpty(out) ? undefined : out
}

/** Reads the canvas back into a document root. Duplicate ids (a copied
 *  component keeps its source's id) are replaced so every node stays unique. */
export function fromGrapes(wrapper: GrapesComponentLike, makeId: () => string = newNodeId): ThemeNode {
  const seen = new Set<string>()
  const convert = (c: GrapesComponentLike): ThemeNode => {
    let id = c.get('cosId') as string | undefined
    if (!id || seen.has(id)) id = makeId()
    seen.add(id)
    const node: ThemeNode = { id, type: c.get('type') as string }
    const props = c.get('cosProps') as Record<string, unknown> | undefined
    if (!isEmpty(props)) node.props = props
    const translations = cleanTranslations(c.get('cosTranslations') as Translations | undefined)
    if (translations) node.translations = translations
    const style = cleanStyle(c.get('cosStyle') as NodeStyle | undefined)
    if (style) node.style = style
    const kids = childrenOf(c).map(convert)
    if (kids.length) node.children = kids
    return node
  }
  const children = childrenOf(wrapper).map(convert)
  return children.length ? { id: 'root', type: 'root', children } : { id: 'root', type: 'root' }
}
