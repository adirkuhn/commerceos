import { describe, it, expect } from 'vitest'
import { compareCatalog, type Catalog } from '../catalog'
import { LOCALES } from '../locale'
import { NAMESPACES } from '../resources'
import { editorResources as resources } from '../../editor/i18n/editorResources'

// Every shipped language must be complete: no missing, blank or extra keys,
// and the same {{placeholders}} as English.
describe.each(NAMESPACES)('%s translations', ns => {
  it.each(LOCALES.filter(l => l !== 'en'))('%s is complete', locale => {
    const issues = compareCatalog(resources.en[ns] as Catalog, resources[locale][ns] as Catalog)
    expect(issues).toEqual({ missing: [], blank: [], extra: [], placeholderMismatch: [] })
  })
})

describe('compareCatalog', () => {
  it('reports what a translator still has to do', () => {
    const en = { a: 'A {{x}}', b: 'B', items_one: '{{count}} item', items_other: '{{count}} items' }
    const de = { a: 'A', c: 'C', b: '', items_other: '{{count}} Artikel' }
    expect(compareCatalog(en, de)).toEqual({ missing: [], blank: ['b'], extra: ['c'], placeholderMismatch: ['a'] })
    expect(compareCatalog(en, {}).missing).toEqual(['a', 'b', 'items'])
  })
})
