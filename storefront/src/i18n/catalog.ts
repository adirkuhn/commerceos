// Helpers for checking translation files (tests and `npm run i18n:check`).

export interface Catalog {
  [key: string]: string | Catalog
}

/** All dotted keys and their texts. */
export function flatten(catalog: Catalog, prefix = ''): Record<string, string> {
  const out: Record<string, string> = {}
  for (const [k, v] of Object.entries(catalog)) {
    const key = prefix ? `${prefix}.${k}` : k
    if (typeof v === 'string') out[key] = v
    else Object.assign(out, flatten(v, key))
  }
  return out
}

/** Every {{placeholder}} a text uses, sorted. */
export function placeholders(text: string): string[] {
  return [...new Set([...text.matchAll(/\{\{\s*(\w+)[^}]*\}\}/g)].map(m => m[1]))].sort()
}

export interface CatalogIssues {
  missing: string[]
  blank: string[]
  extra: string[]
  placeholderMismatch: string[]
}

/** How a translation differs from the English source. Plural forms
 *  (key_one, key_few …) differ per language, so only their base must exist. */
export function compareCatalog(source: Catalog, target: Catalog): CatalogIssues {
  const src = flatten(source)
  const tgt = flatten(target)
  const base = (k: string) => k.replace(/_(zero|one|two|few|many|other)$/, '')
  const srcBases = new Set(Object.keys(src).map(base))
  const tgtBases = new Set(Object.keys(tgt).map(base))
  const srcOther = (k: string) => src[k] ?? src[`${base(k)}_other`]
  return {
    missing: [...srcBases].filter(k => !tgtBases.has(k)).sort(),
    blank: Object.keys(tgt).filter(k => tgt[k].trim() === '').sort(),
    extra: [...tgtBases].filter(k => !srcBases.has(k)).sort(),
    placeholderMismatch: Object.keys(tgt)
      .filter(k => tgt[k].trim() !== '' && srcOther(k) !== undefined && placeholders(tgt[k]).join() !== placeholders(srcOther(k)!).join())
      .sort(),
  }
}
