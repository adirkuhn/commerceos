import type { Component, Editor } from 'grapesjs'
import { createElement, type ReactElement } from 'react'
import type { ComponentRegistry } from '../../components/registry'
import type { ComponentDefinition } from '../../components/types'
import { rendererFor } from '../../components/renderers'
import { nodeClass } from '../../core/style'
import { newNodeId } from './ids'
import { ReactMounts } from './reactMount'
import { CanvasNode } from './CanvasNode'
import type { EditorLabels } from '../i18n/labels'

// Teaches GrapesJS the CommerceOS components: one component type and one
// block per registry definition, placement rules from the registry, and
// leaf components rendered by their real React renderer inside the canvas.
// Containers are plain elements whose children GrapesJS manages.

const WRAPPER = 'wrapper'
const typeOf = (c: Component) => (c.get('type') === WRAPPER ? 'root' : String(c.get('type')))

export interface PluginOptions {
  /** Wraps each canvas-rendered component in the providers it needs. */
  wrap: (el: ReactElement) => ReactElement
  /** Component and category names in the editor's language. */
  labels: EditorLabels
}

function registerType(editor: Editor, registry: ComponentRegistry, def: ComponentDefinition, opts: PluginOptions) {
  const el = def.element?.(def.defaults)
  const mounts = new ReactMounts<Component>()
  const renderReact = (host: HTMLElement, model: Component) => {
    const Renderer = rendererFor(String(model.get('type')))
    if (!Renderer) return
    const id = String(model.get('cosId'))
    mounts.render(model, host, opts.wrap(createElement(CanvasNode, {
      Renderer, node: { id, type: String(model.get('type')), props: model.get('cosProps') ?? {}, translations: model.get('cosTranslations') },
    })))
  }

  editor.Components.addType(def.type, {
    // Never let GrapesJS claim pasted/imported HTML as one of ours.
    isComponent: () => false,
    model: {
      defaults: {
        name: opts.labels.componentName(def),
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
        if (!def.element) return
        const applyClasses = () => {
          const current = def.element!({ ...def.defaults, ...(this.get('cosProps') ?? {}) } as never)
          this.setClass([...current.className.split(' '), nodeClass(String(this.get('cosId')))])
        }
        applyClasses()
        this.on('change:cosProps change:cosId', applyClasses)
      },
    },
    view: def.container ? {} : {
      init(this: { listenTo: (o: unknown, e: string, f: () => void) => void; model: Component; el: HTMLElement }) {
        // Prop edits re-render the React tree in place, not the whole view.
        this.listenTo(this.model, 'change:cosProps change:cosTranslations change:cosId', () => renderReact(this.el, this.model))
      },
      onRender({ el, model }: { el: HTMLElement; model: Component }) {
        renderReact(el, model)
      },
      removed({ model }: { model: Component }) {
        mounts.unmount(model)
      },
    },
  })

  editor.Blocks.add(def.type, {
    label: opts.labels.componentName(def),
    category: opts.labels.category(def.category),
    content: { type: def.type },
    attributes: { title: opts.labels.componentDescription(def) },
  })
}

export function installRegistry(editor: Editor, registry: ComponentRegistry, opts: PluginOptions) {
  for (const def of registry.all()) registerType(editor, registry, def, opts)

  // A copied component keeps its source's id; give it its own.
  editor.on('component:clone', (c: Component) => {
    const id = newNodeId()
    c.set('cosId', id)
  })
}
