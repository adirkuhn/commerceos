import { describe, it, expect } from 'vitest'
import { compareCatalog, type Catalog } from '../catalog'
import { LOCALES } from '../locale'
import { NAMESPACES, resources } from '../resources'
import { typesSource } from '../typesSource'

// Every shipped language must be complete: no missing, blank or extra keys,
// and the same {{placeholders}} as English.
describe.each(NAMESPACES)('%s translations', ns => {
  it.each(LOCALES.filter(l => l !== 'en'))('%s is complete', locale => {
    const issues = compareCatalog(resources.en[ns] as Catalog, (resources[locale][ns] ?? {}) as Catalog)
    expect(issues).toEqual({ missing: [], blank: [], extra: [], placeholderMismatch: [] })
  })
})

describe('generated key types', () => {
  it('match the English files (run `npm run i18n:types`)', () => {
    const english = Object.keys(import.meta.glob('../locales/en/*.json')).map(p => p.split('/').pop()!)
    const generated = Object.values(import.meta.glob<string>('../resources.gen.d.ts', { query: '?raw', import: 'default', eager: true }))[0]
    expect(generated).toBe(typesSource(english))
  })
})
