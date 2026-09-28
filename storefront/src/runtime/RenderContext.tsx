import { createContext, useContext, type ReactNode } from 'react'

import type { AssetRef } from '../core/document'

export type { AssetRef }

export type AssetMap = Record<string, AssetRef>

// What a component needs to know about where it renders: the live store
// or the editor canvas, the images its document refers to, and how to
// link to other store pages.
export interface RenderEnv {
  mode: 'live' | 'editor'
  assets: AssetMap
  /** Absolute path of a store page, e.g. storePath('products/red-shoes'). */
  storePath: (path: string) => string
  /** Full page load of another URL (store page or payment provider). */
  navigate: (url: string, opts?: { replace?: boolean }) => void
}

function navigate(url: string, opts?: { replace?: boolean }) {
  if (opts?.replace) window.location.replace(url)
  else window.location.assign(url)
}

const defaultEnv: RenderEnv = { mode: 'live', assets: {}, storePath: p => `/${p}`, navigate }

const RenderContext = createContext<RenderEnv>(defaultEnv)

export function RenderProvider({ value, children }: { value: Partial<RenderEnv>; children: ReactNode }) {
  const parent = useContext(RenderContext)
  return <RenderContext.Provider value={{ ...parent, ...value }}>{children}</RenderContext.Provider>
}

export const useRenderEnv = () => useContext(RenderContext)

const ASSET_REF = /^asset:([0-9a-f-]{36})$/

/** Resolves an "asset:<id>" reference against the document's assets. */
export function resolveAssetRef(ref: unknown, assets: AssetMap): AssetRef | undefined {
  const m = typeof ref === 'string' ? ASSET_REF.exec(ref) : null
  return m ? assets[m[1]] : undefined
}
