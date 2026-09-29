import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { Collection, StorefrontAPI } from '../api/storefront'
import type { ListingPage } from '../runtime/page/ListingContext'
import type { PreviewStore } from './previewStore'
import type { DocRef } from './useDraft'

const PREVIEW_PAGE_SIZE = 12
const inert = () => '#'

async function listingFor(store: StorefrontAPI, collection: Collection | null, search: string, searchTitle: string): Promise<ListingPage> {
  const listing = await store.listProducts({ collectionId: collection?.id, search: search || undefined, perPage: PREVIEW_PAGE_SIZE })
  return {
    title: collection ? collection.name : searchTitle,
    description: collection?.description ?? '', imageUrl: collection?.image_url, listing, page: 1,
    pageCount: Math.max(1, Math.ceil(listing.total / PREVIEW_PAGE_SIZE)), hrefForPage: inert, hrefForCategory: inert,
  }
}

/** Keeps the canvas preview data in step with the open document: a product
 *  for the product template, a collection (or search) for listing
 *  templates. Starts with the first product/collection; the merchant can
 *  pick another. */
export function usePreviewData(store: StorefrontAPI | undefined, preview: PreviewStore, doc: DocRef) {
  const [productSlug, setProductSlug] = useState<string>()
  const [collectionSlug, setCollectionSlug] = useState<string>()
  const [error, setError] = useState('')
  const { t } = useTranslation('editor')
  const kind = doc.kind === 'template' ? doc.key : ''

  useEffect(() => {
    if (!store) return
    let cancelled = false
    const load = async () => {
      if (kind === 'product') {
        const slug = productSlug ?? (await store.listProducts({ perPage: 1 })).items[0]?.slug
        preview.set(slug ? { product: await store.getProductBySlug(slug) } : {})
      } else if (kind === 'collection') {
        const collections = await store.listCollections()
        const collection = collections.find(c => c.slug === collectionSlug) ?? collections[0] ?? null
        preview.set(collection ? { listing: await listingFor(store, collection, '', '') } : {})
      } else if (kind === 'search') {
        preview.set({ listing: await listingFor(store, null, 'example', t('previewBar.searchResults', { query: 'example' })) })
      } else {
        preview.set({})
      }
    }
    load().then(() => !cancelled && setError(''), e => !cancelled && setError(e instanceof Error ? e.message : t('previewBar.loadFailed')))
    return () => { cancelled = true }
  }, [store, preview, kind, productSlug, collectionSlug])

  return {
    kind, error,
    chooseProduct: useCallback((slug: string) => setProductSlug(slug), []),
    chooseCollection: useCallback((slug: string) => setCollectionSlug(slug), []),
  }
}
