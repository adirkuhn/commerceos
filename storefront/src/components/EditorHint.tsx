import { useRenderEnv } from '../runtime/RenderContext'

/** Shown instead of a component in the editor when it has nothing to show
 *  yet; renders nothing on the live store. */
export function EditorHint({ nodeId, className, children }: { nodeId: string; className: string; children: string }) {
  const env = useRenderEnv()
  return env.mode === 'editor' ? <div className={`cos-placeholder ${className}`} data-node={nodeId}>{children}</div> : null
}
