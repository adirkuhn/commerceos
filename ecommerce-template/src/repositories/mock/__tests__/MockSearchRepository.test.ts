import { describe, expect, it } from 'vitest'
import { MockSearchRepository } from '../MockSearchRepository'

const repo = new MockSearchRepository()

describe('MockSearchRepository', () => {
  it('matches products by name (case-insensitive)', async () => {
    const result = await repo.search({ query: 'trail runner' })
    expect(result.products.length).toBeGreaterThan(0)
    expect(result.products.every(p => p.name.toLowerCase().includes('trail runner'))).toBe(true)
  })

  it('matches products by tag', async () => {
    const result = await repo.search({ query: 'merino' })
    expect(result.products.length).toBeGreaterThan(0)
  })

  it('returns categories and brands that match the query', async () => {
    const result = await repo.search({ query: 'outerwear' })
    expect(result.categories.some(c => c.slug === 'outerwear')).toBe(true)
  })

  it('suggests a fuzzy correction for a near-miss query with no results', async () => {
    const result = await repo.search({ query: 'jaket' })
    expect(result.total).toBe(0)
    expect(result.suggestedQuery).toBe('Jacket')
  })

  it('does not suggest a correction for a query with real results', async () => {
    const result = await repo.search({ query: 'jacket' })
    expect(result.total).toBeGreaterThan(0)
    expect(result.suggestedQuery).toBeUndefined()
  })

  it('paginates search results', async () => {
    const result = await repo.search({ query: 'a', page: 1, perPage: 3 })
    expect(result.products.length).toBeLessThanOrEqual(3)
    expect(result.total).toBeGreaterThan(3)
  })

  it('returns autocomplete suggestions', async () => {
    const suggestions = await repo.suggestions('head')
    expect(suggestions.some(s => s.type === 'product')).toBe(true)
  })

  it('returns a curated popular searches list', async () => {
    const popular = await repo.popularSearches()
    expect(popular.length).toBeGreaterThan(0)
  })
})
