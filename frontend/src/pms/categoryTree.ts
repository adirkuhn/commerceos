import type { Category } from './types'

export interface CategoryTreeEntry extends Category {
  depth: number
}

// Depth-first walk from root categories (parent_id null/undefined) down
// through children, sorted by position then name at each level — a plain
// alphabetical/position list interleaves every level together once
// categories have real parent/child relationships (e.g. from the
// WooCommerce import), which reads as random rather than a hierarchy.
// depth lets callers indent each entry to show the nesting.
export function sortCategoriesHierarchically(categories: Category[]): CategoryTreeEntry[] {
  const byParent = new Map<string, Category[]>()
  for (const c of categories) {
    const key = c.parent_id ?? ''
    const siblings = byParent.get(key) ?? []
    siblings.push(c)
    byParent.set(key, siblings)
  }
  for (const siblings of byParent.values()) {
    siblings.sort((a, b) => a.position - b.position || a.name.localeCompare(b.name))
  }

  const out: CategoryTreeEntry[] = []
  // Categories whose parent_id doesn't resolve to any known category
  // (deleted/inconsistent data) are treated as roots rather than dropped,
  // so every category still appears in the list.
  const knownIDs = new Set(categories.map(c => c.id))

  function visit(parentKey: string, depth: number) {
    for (const c of byParent.get(parentKey) ?? []) {
      out.push({ ...c, depth })
      visit(c.id, depth + 1)
    }
  }

  visit('', 0)
  for (const c of categories) {
    if (c.parent_id && !knownIDs.has(c.parent_id) && !out.some(o => o.id === c.id)) {
      out.push({ ...c, depth: 0 })
      visit(c.id, 1)
    }
  }
  return out
}
