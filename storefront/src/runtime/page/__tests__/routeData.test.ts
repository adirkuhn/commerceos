import { describe, it, expect } from 'vitest'
import { listingParamsFor } from '../routeData'
import type { ThemeDocument } from '../../../core/document'

const doc = (perPage?: number): ThemeDocument => ({
  kind: 'template', key: 'collection', title: '', seo: {},
  root: { id: 'root', type: 'root', children: [{ id: 's', type: 'section', children: [{ id: 'p', type: 'collection-products', props: perPage ? { per_page: perPage } : {} }] }] },
})

describe('listingParamsFor', () => {
  it('reads page and category from the URL and page size from the template', () => {
    const p = listingParamsFor({ kind: 'template', key: 'collection', param: 'summer', notFound: false }, new URLSearchParams('page=3&category=c9'), doc(12), { id: 'col-1' } as never)
    expect(p).toEqual({ page: 3, perPage: 12, categoryId: 'c9', collectionId: 'col-1' })
  })

  it('defaults and clamps bad input', () => {
    const p = listingParamsFor({ kind: 'template', key: 'search', notFound: false }, new URLSearchParams('page=-4&q=red'), doc())
    expect(p).toEqual({ page: 1, perPage: 24, categoryId: undefined, search: 'red' })
  })
})
