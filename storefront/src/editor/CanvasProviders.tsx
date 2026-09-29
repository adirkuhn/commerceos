import { Suspense, useSyncExternalStore, type ReactNode } from 'react'
import { QueryClientProvider, useQuery } from '@tanstack/react-query'
import { I18nextProvider, useTranslation } from 'react-i18next'
import { useMemo } from 'react'
import { createI18n } from '../i18n/createI18n'
import { resolveLocale } from '../i18n/locale'
import { ApiProvider } from '../api/ApiContext'
import { RenderProvider } from '../runtime/RenderContext'
import type { EditorServices } from './services'
import { ProductProvider } from '../runtime/page/ProductContext'
import { ListingProvider } from '../runtime/page/ListingContext'
import { StoreInfoProvider } from '../runtime/page/StoreContext'
import { ComponentBoundary } from '../components/ComponentBoundary'
import { storeQuery } from '../runtime/page/queries'

function DataFailed() {
  const { t } = useTranslation()
  return <div className="cos-placeholder">{t('hint.dataFailed')}</div>
}

function CanvasStoreInfo({ services, children }: { services: EditorServices; children: ReactNode }) {
  const settings = useSyncExternalStore(services.settings.subscribe, services.settings.snapshot)
  const assets = useSyncExternalStore(services.assets.subscribe, services.assets.snapshot)
  const { data: store } = useQuery(storeQuery(services.store))
  // Components preview in the store's own language.
  const locale = resolveLocale(store?.locale)
  const i18n = useMemo(() => createI18n(locale), [locale])
  return (
    <I18nextProvider i18n={i18n}>
      <StoreInfoProvider value={{ name: store?.name ?? '', settings, assets }}>{children}</StoreInfoProvider>
    </I18nextProvider>
  )
}

/** What a component rendered inside the canvas needs: catalog data, the
 *  editor's known assets, and editor mode (inert links, empty-state hints). */
export function CanvasProviders({ services, children }: { services: EditorServices; children: ReactNode }) {
  const assets = useSyncExternalStore(services.assets.subscribe, services.assets.snapshot)
  const preview = useSyncExternalStore(services.preview.subscribe, services.preview.snapshot)
  let content = children
  if (preview.product) content = <ProductProvider key={preview.product.id} product={preview.product}>{content}</ProductProvider>
  if (preview.listing) content = <ListingProvider value={preview.listing}>{content}</ListingProvider>
  return (
    <QueryClientProvider client={services.queryClient}>
      <ApiProvider api={services.store}>
        <RenderProvider value={{ mode: 'editor', assets, storePath: p => `/store/${services.channelId}/${p}` }}>
          <CanvasStoreInfo services={services}>
            <ComponentBoundary nodeId="canvas" fallback={<DataFailed />}>
              <Suspense fallback={<div className="cos-placeholder">Loading…</div>}>{content}</Suspense>
            </ComponentBoundary>
          </CanvasStoreInfo>
        </RenderProvider>
      </ApiProvider>
    </QueryClientProvider>
  )
}
