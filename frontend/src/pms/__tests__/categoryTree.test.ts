import { describe, it, expect } from 'vitest'
import { sortCategoriesHierarchically } from '../categoryTree'
import type { Category } from '../types'

function category(overrides: Partial<Category> = {}): Category {
  return {
    id: 'c1', tenant_id: 't1', parent_id: null, name: 'Category', slug: 'category',
    position: 0, created_at: '', updated_at: '', ...overrides,
  }
}

describe('sortCategoriesHierarchically', () => {
  it('orders a flat list of roots alphabetically', () => {
    const cats = [
      category({ id: 'b', name: 'Bebidas' }),
      category({ id: 'a', name: 'Alimentos' }),
    ]

    const result = sortCategoriesHierarchically(cats)

    expect(result.map(c => c.id)).toEqual(['a', 'b'])
    expect(result.every(c => c.depth === 0)).toBe(true)
  })

  it('places children immediately after their parent, not interleaved alphabetically', () => {
    const cats = [
      category({ id: 'root2', name: 'Bebidas', parent_id: null }),
      category({ id: 'root1', name: 'Alimentos', parent_id: null }),
      category({ id: 'child1', name: 'Salgadinhos', parent_id: 'root1' }),
    ]

    const result = sortCategoriesHierarchically(cats)

    // Without hierarchy, alphabetical order would be Alimentos, Bebidas,
    // Salgadinhos — interleaving the child between unrelated roots. This
    // asserts the actual bug being fixed: the child must stay under its
    // parent regardless of name.
    expect(result.map(c => c.id)).toEqual(['root1', 'child1', 'root2'])
    expect(result.find(c => c.id === 'child1')?.depth).toBe(1)
    expect(result.find(c => c.id === 'root1')?.depth).toBe(0)
  })

  it('supports multiple levels of nesting', () => {
    const cats = [
      category({ id: 'a', name: 'Alimentos', parent_id: null }),
      category({ id: 'b', name: 'Biscoitos e snacks', parent_id: 'a' }),
      category({ id: 'c', name: 'Salgadinhos', parent_id: 'b' }),
    ]

    const result = sortCategoriesHierarchically(cats)

    expect(result.map(c => c.id)).toEqual(['a', 'b', 'c'])
    expect(result.map(c => c.depth)).toEqual([0, 1, 2])
  })

  it('sorts siblings by position before name', () => {
    const cats = [
      category({ id: 'z', name: 'Zeta', position: 0 }),
      category({ id: 'a', name: 'Alpha', position: 1 }),
    ]

    const result = sortCategoriesHierarchically(cats)

    expect(result.map(c => c.id)).toEqual(['z', 'a'])
  })

  it('treats a category with a dangling parent_id as a root rather than dropping it', () => {
    const cats = [category({ id: 'orphan', name: 'Orphan', parent_id: 'does-not-exist' })]

    const result = sortCategoriesHierarchically(cats)

    expect(result.map(c => c.id)).toEqual(['orphan'])
    expect(result[0].depth).toBe(0)
  })

  it('returns an empty array for an empty input', () => {
    expect(sortCategoriesHierarchically([])).toEqual([])
  })
})
