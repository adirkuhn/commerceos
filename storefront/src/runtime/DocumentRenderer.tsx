import { Component, Suspense, useMemo, type ReactNode } from 'react'
import type { ThemeNode } from '../core/document'
import { compileDocumentStyles, nodeClass } from '../core/style'
import { rendererFor } from '../components/renderers'
import { RenderProvider, useRenderEnv, type AssetMap } from './RenderContext'

/** One failing component (e.g. its data didn't load) must not take the page down. */
class ComponentBoundary extends Component<{ nodeId: string; children: ReactNode }, { failed: boolean }> {
  state = { failed: false }

  static getDerivedStateFromError() {
    return { failed: true }
  }

  componentDidCatch(error: unknown) {
    console.error(`storefront: component ${this.props.nodeId} failed`, error)
  }

  render() {
    return this.state.failed ? null : this.props.children
  }
}

function RenderNode({ node }: { node: ThemeNode }) {
  const Renderer = rendererFor(node.type)
  if (!Renderer) {
    // A component retired after this version was published: skip it
    // rather than break the page.
    console.warn(`storefront: no renderer for component type "${node.type}"`)
    return null
  }
  return (
    <ComponentBoundary nodeId={node.id}>
      <Suspense fallback={null}>
        <Renderer node={node} props={node.props ?? {}} className={nodeClass(node.id)}>
          {node.children?.map(child => <RenderNode key={child.id} node={child} />)}
        </Renderer>
      </Suspense>
    </ComponentBoundary>
  )
}

function Styles({ root }: { root: ThemeNode }) {
  const { assets } = useRenderEnv()
  const css = useMemo(() => compileDocumentStyles(root, { resolveAsset: id => assets[id]?.url }), [root, assets])
  return css ? <style>{css}</style> : null
}

/** Renders one theme document: its component tree plus its compiled styles. */
export function DocumentRenderer({ root, assets }: { root: ThemeNode; assets?: AssetMap }) {
  const body = (
    <>
      <Styles root={root} />
      {root.children?.map(child => <RenderNode key={child.id} node={child} />)}
    </>
  )
  return assets ? <RenderProvider value={{ assets }}>{body}</RenderProvider> : body
}
