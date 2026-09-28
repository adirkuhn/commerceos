import { Suspense, useSyncExternalStore, type ReactNode } from 'react'
import { QueryClientProvider } from '@tanstack/react-query'
import { ApiProvider } from '../api/ApiContext'
import { RenderProvider } from '../runtime/RenderContext'
import type { EditorServices } from './services'

/** What a component rendered inside the canvas needs: catalog data, the
 *  editor's known assets, and editor mode (inert links, empty-state hints). */
export function CanvasProviders({ services, children }: { services: EditorServices; children: ReactNode }) {
  const assets = useSyncExternalStore(services.assets.subscribe, services.assets.snapshot)
  return (
    <QueryClientProvider client={services.queryClient}>
      <ApiProvider api={services.store}>
        <RenderProvider value={{ mode: 'editor', assets, storePath: p => `/store/${services.channelId}/${p}` }}>
          <Suspense fallback={<div className="cos-placeholder">Loading…</div>}>{children}</Suspense>
        </RenderProvider>
      </ApiProvider>
    </QueryClientProvider>
  )
}
