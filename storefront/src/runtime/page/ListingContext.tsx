import { createContext, useContext, type ReactNode } from 'react'
import type { ProductList } from '../../api/storefront'

// A collection or search page: what it's about and the current page of
// products (filters and pagination live in the URL).
export interface ListingPage {
  title: string
  description: string
  imageUrl?: string | null
  listing: ProductList
  page: number
  pageCount: number
  activeCategoryId?: string
  hrefForPage: (page: number) => string
  hrefForCategory: (categoryId: string | null) => string
}

const Ctx = createContext<ListingPage | null>(null)

export function ListingProvider({ value, children }: { value: ListingPage; children: ReactNode }) {
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}

export const useListingPage = () => useContext(Ctx)
