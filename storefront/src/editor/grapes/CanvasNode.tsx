import type { ThemeNode } from '../../core/document'
import { nodeClass } from '../../core/style'
import { propsIn } from '../../components/translatable'
import { useRenderEnv } from '../../runtime/RenderContext'
import type { Renderer as ComponentRenderer } from '../../components/types'

/** One canvas component, showing its texts in the language being edited. */
export function CanvasNode({ Renderer, node }: { Renderer: ComponentRenderer; node: ThemeNode }) {
  const { contentLocale } = useRenderEnv()
  return <Renderer node={{ id: node.id, type: node.type }} props={propsIn(node, contentLocale)} className={nodeClass(node.id)} />
}
