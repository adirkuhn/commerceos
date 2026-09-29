import i18next from 'i18next'
import type { Collection, CollectionInput, CollectionRule, CollectionRuleField } from './types'

/** "Summer Sale 2026!" -> "summer-sale-2026" (the storefront URL). */
export function slugify(name: string): string {
  return name.normalize('NFKD').replace(/[̀-ͯ]/g, '').toLowerCase()
    .replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '').slice(0, 64).replace(/-+$/, '')
}

export function emptyInput(): CollectionInput {
  return { name: '', slug: '', description: '', seo_title: '', seo_description: '', type: 'manual', match: 'all', rules: [], sort: 'manual', published: false }
}

export function toInput(c: Collection): CollectionInput {
  return {
    name: c.name, slug: c.slug, description: c.description, seo_title: c.seo_title, seo_description: c.seo_description,
    type: c.type, match: c.match, rules: c.rules, sort: c.sort, published: c.published,
  }
}

export function newRule(field: CollectionRuleField): CollectionRule {
  switch (field) {
    case 'category': return { field, value: '', include_subcategories: true }
    case 'tag': return { field, value: '' }
    case 'price': return { field, min_minor_units: null, max_minor_units: null }
    case 'added_within_days': return { field, days: 30 }
    case 'on_sale': return { field }
  }
}

/** Switching type keeps the settings valid: smart collections can't be sorted by hand. */
export function withType(input: CollectionInput, type: CollectionInput['type']): CollectionInput {
  if (type === 'smart') {
    return { ...input, type, match: input.match ?? 'all', rules: input.rules?.length ? input.rules : [newRule('category')], sort: input.sort === 'manual' ? 'newest' : input.sort }
  }
  return { ...input, type, rules: [] }
}

/** What's wrong before sending, in the merchant's words ('' when fine). */
export function validationMessage(input: CollectionInput): string {
  const t = i18next.t
  if (!input.name.trim()) return t('collections:validation.name')
  if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(input.slug)) return t('collections:validation.slug')
  if (input.type === 'smart') {
    if (!input.rules?.length) return t('collections:validation.noConditions')
    for (const r of input.rules) {
      if (r.field === 'category' && !r.value?.trim()) return t('collections:validation.chooseCategory')
      if (r.field === 'tag' && !r.value?.trim()) return t('collections:validation.chooseTag')
      if (r.field === 'price' && r.min_minor_units == null && r.max_minor_units == null) return t('collections:validation.priceBounds')
      if (r.field === 'price' && r.min_minor_units != null && r.max_minor_units != null && r.min_minor_units > r.max_minor_units) return t('collections:validation.priceOrder')
    }
  }
  return ''
}

/** Money typed in major units ("12.50") to minor units (1250); empty -> null. */
export function toMinorUnits(value: string): number | null {
  const n = Number(value.replace(',', '.'))
  return value.trim() === '' || !Number.isFinite(n) || n < 0 ? null : Math.round(n * 100)
}

export const fromMinorUnits = (v: number | null | undefined) => (v == null ? '' : (v / 100).toFixed(2))
