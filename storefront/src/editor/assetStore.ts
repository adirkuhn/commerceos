import type { AssetRef } from '../core/document'

/** Assets known to the editor (from loaded documents, settings and
 *  uploads), observable so canvas components re-render when one arrives. */
export class AssetStore {
  private refs: Record<string, AssetRef> = {}
  private readonly listeners = new Set<() => void>()

  snapshot = (): Record<string, AssetRef> => this.refs

  subscribe = (fn: () => void): (() => void) => {
    this.listeners.add(fn)
    return () => this.listeners.delete(fn)
  }

  merge(refs: Record<string, AssetRef> | undefined) {
    if (!refs || !Object.keys(refs).length) return
    this.refs = { ...this.refs, ...refs }
    this.listeners.forEach(l => l())
  }
}
