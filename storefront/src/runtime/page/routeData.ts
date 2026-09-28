import type { QueryClient } from '@tanstack/react-query'
import type { Collection, ProductDetail, StorefrontAPI } from '../../api/storefront'
import type { StorefrontTheme, ThemeDocument, ThemeNode } from '../../core/document'
import { documentFor, documentQuery } from '../queries'
import { plainDescription } from '../seo'
import type { ResolvedRoute } from '../route'
import { collectionPageQuery, listingQuery, productPageQuery, type ListingParams } from './queries'

export const DEFAULT_PER_PAGE = 24
const NOT_FOUND: ResolvedRoute = { kind: 'template', key: 'not-found', notFound: true }

function findNode(root: ThemeNode, type: string): ThemeNode | undefined {
  if (root.type === type) return root
  for (const c of root.children ?? []) {
    const hit = findNode(c, type)
    if (hit) return hit
  }
  return undefined
}

/** Listing settings for a collection/search page: page + filter from the
 *  URL, page size from the template's Collection products component. */
export function listingParamsFor(route: ResolvedRoute, query: URLSearchParams, doc: ThemeDocument, collection?: Collection | null): ListingParams {
  const perPage = Number(findNode(doc.root, 'collection-products')?.props?.per_page) || DEFAULT_PER_PAGE
  const page = Math.max(1, Math.floor(Number(query.get('page'))) || 1)
  return {
    page, perPage: Math.min(Math.max(perPage, 1), 48),
    categoryId: query.get('category') || undefined,
    ...(route.key === 'collection' && collection ? { collectionId: collection.id } : {}),
    ...(route.key === 'search' ? { search: query.get('q') ?? '' } : {}),
  }
}

export interface PageData {
  route: ResolvedRoute
  product?: ProductDetail
  collection?: Collection
  listing?: ListingParams
  seoTitle?: string
  seoDescription?: string
}

/** Loads everything a page needs into the query cache (used by SSR) and
 *  says which document renders it -- not-found when the product or
 *  collection doesn't exist. */
export async function resolvePageData(qc: QueryClient, api: StorefrontAPI, theme: StorefrontTheme, path: string, query: URLSearchParams): Promise<PageData> {
  let route = documentFor(theme, path)
  if (route.builtin) return { route, seoTitle: 'Newsletter' }
  if (route.key === 'product' && route.param) {
    const product = await qc.fetchQuery(productPageQuery(api, route.param))
    if (!product) return { route: NOT_FOUND }
    return { route, product, seoTitle: product.name, seoDescription: plainDescription(product.description) || undefined }
  }
  if (route.key === 'collection' || route.key === 'search') {
    let collection: Collection | null = null
    if (route.key === 'collection') {
      collection = route.param ? await qc.fetchQuery(collectionPageQuery(api, route.param)) : null
      if (!collection) return { route: NOT_FOUND }
    }
    const doc = await qc.fetchQuery(documentQuery(api, theme, route.kind, route.key))
    const listing = listingParamsFor(route, query, doc, collection)
    await qc.fetchQuery(listingQuery(api, listing))
    return {
      route, listing, collection: collection ?? undefined,
      seoTitle: collection ? collection.seo_title || collection.name : `Search: ${listing.search ?? ''}`,
      seoDescription: collection?.seo_description || undefined,
    }
  }
  if (route.notFound) route = NOT_FOUND
  return { route }
}
