import { queryOptions } from '@tanstack/react-query'
import { ApiError, type StorefrontAPI } from '../../api/storefront'

// Route data: what a product/collection/search page is about. Shared by SSR
// prefetching and the providers, so hydration reuses the server's data.

const STALE = 60_000

/** null for a missing product, so the page can render not-found instead of failing. */
async function orNull<T>(p: Promise<T>): Promise<T | null> {
  try {
    return await p
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) return null
    throw err
  }
}

export const productPageQuery = (api: StorefrontAPI, slug: string) =>
  queryOptions({ queryKey: ['product-page', slug], queryFn: () => orNull(api.getProductBySlug(slug)), staleTime: STALE })

export const collectionPageQuery = (api: StorefrontAPI, slug: string) =>
  queryOptions({ queryKey: ['collection-page', slug], queryFn: () => (slug ? orNull(api.getCollection(slug)) : Promise.resolve(null)), staleTime: STALE })

export interface ListingParams {
  collectionId?: string
  categoryId?: string
  search?: string
  page: number
  perPage: number
}

export const listingQuery = (api: StorefrontAPI, p: ListingParams) =>
  queryOptions({
    queryKey: ['listing', p],
    queryFn: () => api.listProducts({ collectionId: p.collectionId, categoryId: p.categoryId, search: p.search, page: p.page, perPage: p.perPage }),
    staleTime: STALE,
  })

export const reviewsQuery = (api: StorefrontAPI, productId: string, limit: number) =>
  queryOptions({ queryKey: ['reviews', productId, limit], queryFn: () => api.listReviews(productId, limit), staleTime: STALE })

export const recommendationsQuery = (api: StorefrontAPI, productId: string) =>
  queryOptions({ queryKey: ['recommendations', productId], queryFn: () => api.getRecommendations(productId), staleTime: STALE })

/** The store's name etc.; null when unavailable -- no page fails over it. */
export const storeQuery = (api: StorefrontAPI) =>
  queryOptions({ queryKey: ['store'], queryFn: () => api.getStore().catch(() => null), staleTime: STALE })

export const collectionsQuery = (api: StorefrontAPI) =>
  queryOptions({ queryKey: ['collections'], queryFn: () => api.listCollections(), staleTime: STALE })
