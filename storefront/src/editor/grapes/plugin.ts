import type { Component, Editor } from 'grapesjs'
import { createElement } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import type { ComponentRegistry } from '../../components/registry'
import type { ComponentDefinition } from '../../components/types'
import { rendererFor } from '../../components/renderers'
import { nodeClass } from '../../core/style'
import { newNodeId } from './ids'

// Teaches GrapesJS the CommerceOS components: one component type and one
// block per registry definition, placement rules from the registry, and
// leaf components rendered by their real React renderer inside the canvas.
// Containers are plain elements whose children GrapesJS manages.

const WRAPPER = 'wrapper'
const typeOf = (c: Component) => (c.get('type') === WRAPPER ? 'root' : String(c.get('type')))

export const CATEGORY_LABELS: Record<ComponentDefinition['category'], string> = {
  layout: 'Layout', content: 'Content', commerce: 'Commerce', navigation: 'Navigation', customer: 'Customer', cart: 'Cart',
}

function registerType(editor: Editor, registry: ComponentRegistry, def: ComponentDefinition) {
  const el = def.element?.(def.defaults)
  const roots = new WeakMap<object, Root>()

  editor.Components.addType(def.type, {
    // Never let GrapesJS claim pasted/imported HTML as one of ours.
    isComponent: () => false,
    model: {
      defaults: {
        name: def.name,
        tagName: el?.tag ?? 'div',
        droppable: def.container
          ? (source: Component) => registry.canPlace(typeOf(source), def.type)
          : false,
        draggable: (_source: Component, target: Component) => def.editor.draggable && registry.canPlace(def.type, typeOf(target)),
        copyable: def.editor.copyable,
        removable: def.editor.removable,
        stylable: false,
        editable: false,
        cosProps: { ...def.defaults },
        cosStyle: {},
      },
      init(this: Component) {
        if (!this.get('cosId')) this.set('cosId', newNodeId())
        if (el) this.setClass([...el.className.split(' '), nodeClass(String(this.get('cosId')))])
      },
    },
    view: def.container ? {} : {
      init(this: { listenTo: (o: unknown, e: string, f: () => void) => void; model: Component; render: () => void }) {
        this.listenTo(this.model, 'change:cosProps change:cosId', () => this.render())
      },
      onRender({ el: host, model }: { el: HTMLElement; model: Component }) {
        const Renderer = rendererFor(def.type)
        if (!Renderer) return
        roots.get(model)?.unmount()
        const root = createRoot(host)
        roots.set(model, root)
        const id = String(model.get('cosId'))
        root.render(createElement(Renderer, { node: { id, type: def.type }, props: model.get('cosProps') ?? {}, className: nodeClass(id) }))
      },
      removed({ model }: { model: Component }) {
        roots.get(model)?.unmount()
        roots.delete(model)
      },
    },
  })

  editor.Blocks.add(def.type, {
    label: def.name,
    category: CATEGORY_LABELS[def.category],
    content: { type: def.type },
    attributes: { title: def.description },
  })
}

export function installRegistry(editor: Editor, registry: ComponentRegistry) {
  for (const def of registry.all()) registerType(editor, registry, def)

  // A copied component keeps its source's id; give it its own.
  editor.on('component:clone', (c: Component) => {
    const id = newNodeId()
    c.set('cosId', id)
    const def = registry.get(String(c.get('type')))
    const el = def?.element?.(c.get('cosProps') ?? def.defaults)
    if (el) c.setClass([...el.className.split(' '), nodeClass(id)])
  })
}
