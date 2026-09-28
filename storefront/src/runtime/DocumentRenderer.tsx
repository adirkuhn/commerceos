import { useMemo } from 'react'
import type { ThemeNode } from '../core/document'
import { compileDocumentStyles, nodeClass, type CompileOptions } from '../core/style'
import { rendererFor } from '../components/renderers'

function RenderNode({ node }: { node: ThemeNode }) {
  const Renderer = rendererFor(node.type)
  if (!Renderer) {
    // A component retired after this version was published: skip it
    // rather than break the page.
    console.warn(`storefront: no renderer for component type "${node.type}"`)
    return null
  }
  return (
    <Renderer node={node} props={node.props ?? {}} className={nodeClass(node.id)}>
      {node.children?.map(child => <RenderNode key={child.id} node={child} />)}
    </Renderer>
  )
}

/** Renders one theme document: its component tree plus its compiled styles. */
export function DocumentRenderer({ root, styleOptions }: { root: ThemeNode; styleOptions?: CompileOptions }) {
  const css = useMemo(() => compileDocumentStyles(root, styleOptions), [root, styleOptions])
  return (
    <>
      {css && <style>{css}</style>}
      {root.children?.map(child => <RenderNode key={child.id} node={child} />)}
    </>
  )
}
