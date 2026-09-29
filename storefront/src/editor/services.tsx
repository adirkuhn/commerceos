import { createContext, useContext, type ReactNode } from 'react'
import type { QueryClient } from '@tanstack/react-query'
import type { StorefrontAPI } from '../api/storefront'
import type { EditorAPI } from './api'
import type { AssetStore } from './assetStore'
import type { PreviewStore } from './previewStore'
import type { ThemeSettings } from '../core/document'
import type { ValueStore } from './valueStore'
import type { Locale } from '../i18n/locale'

// Everything the editor UI and canvas talk to, injected once at the top.
export interface EditorServices {
  api: EditorAPI
  /** The channel's public storefront API (catalog data for pickers/canvas). */
  store: StorefrontAPI
  assets: AssetStore
  /** Product/listing the canvas previews templates with. */
  preview: PreviewStore
  /** The draft's theme settings, for components that show them (logo). */
  settings: ValueStore<ThemeSettings>
  queryClient: QueryClient
  channelId: string
  /** The language merchant texts are edited (and previewed) in; "" is the
   *  store's default. */
  contentLocale: ValueStore<Locale | ''>
}

const Ctx = createContext<EditorServices | null>(null)

/** value is null until the draft (and so the store) is known. The provider
 *  is always rendered so the layout -- and the DOM GrapesJS mounted into --
 *  never remounts when it arrives. */
export function EditorServicesProvider({ value, children }: { value: EditorServices | null; children: ReactNode }) {
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}

export function useEditorServices(): EditorServices {
  const s = useContext(Ctx)
  if (!s) throw new Error('useEditorServices outside EditorServicesProvider')
  return s
}
