import { createContext, useContext, type ReactNode } from 'react'
import type { QueryClient } from '@tanstack/react-query'
import type { StorefrontAPI } from '../api/storefront'
import type { EditorAPI } from './api'
import type { AssetStore } from './assetStore'
import type { PreviewStore } from './previewStore'

// Everything the editor UI and canvas talk to, injected once at the top.
export interface EditorServices {
  api: EditorAPI
  /** The channel's public storefront API (catalog data for pickers/canvas). */
  store: StorefrontAPI
  assets: AssetStore
  /** Product/listing the canvas previews templates with. */
  preview: PreviewStore
  queryClient: QueryClient
  channelId: string
}

const Ctx = createContext<EditorServices | null>(null)

export function EditorServicesProvider({ value, children }: { value: EditorServices; children: ReactNode }) {
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}

export function useEditorServices(): EditorServices {
  const s = useContext(Ctx)
  if (!s) throw new Error('useEditorServices outside EditorServicesProvider')
  return s
}
