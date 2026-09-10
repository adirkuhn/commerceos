import { describe, expect, it } from 'vitest'
import { MockProductRepository } from '../MockProductRepository'
import { PRODUCTS, CATEGORIES } from '@/config/demoData'

const repo = new MockProductRepository()

describe('MockProductRepository', () => {
  it('lists every product with no filters', async () => {
    const { items, total } = await repo.list({})
    expect(total).toBe(PRODUCTS.length)
    expect(items).toHaveLength(PRODUCTS.length)
  })

  it('filters by category', async () => {
    const category = CATEGORIES[0]
    const { items } = await repo.list({ categoryId: category.id })
    expect(items.length).toBeGreaterThan(0)
    for (const item of items) expect(item.categoryIds).toContain(category.id)
  })

  it('filters by price range', async () => {
    const { items } = await repo.list({ minPrice: 20000, maxPrice: 30000 })
    for (const item of items) {
      expect(item.price).toBeGreaterThanOrEqual(20000)
      expect(item.price).toBeLessThanOrEqual(30000)
    }
  })

  it('filters to in-stock only', async () => {
    const { items } = await repo.list({ inStockOnly: true })
    for (const item of items) expect(item.stockStatus).not.toBe('out_of_stock')
  })

  it('sorts by price ascending', async () => {
    const { items } = await repo.list({ sort: 'price_asc' })
    for (let i = 1; i < items.length; i++) expect(items[i].price).toBeGreaterThanOrEqual(items[i - 1].price)
  })

  it('paginates results', async () => {
    const page1 = await repo.list({ page: 1, perPage: 5 })
    const page2 = await repo.list({ page: 2, perPage: 5 })
    expect(page1.items).toHaveLength(5)
    expect(page2.items).toHaveLength(5)
    expect(page1.items[0].id).not.toBe(page2.items[0].id)
    expect(page1.total).toBe(PRODUCTS.length)
  })

  it('finds a product by slug', async () => {
    const product = await repo.getBySlug(PRODUCTS[0].slug)
    expect(product?.id).toBe(PRODUCTS[0].id)
  })

  it('returns null for an unknown slug', async () => {
    expect(await repo.getBySlug('not-a-real-product')).toBeNull()
  })

  it('gets multiple products by id', async () => {
    const ids = [PRODUCTS[0].id, PRODUCTS[1].id]
    const products = await repo.getByIds(ids)
    expect(products.map(p => p.id).sort()).toEqual(ids.sort())
  })
})
