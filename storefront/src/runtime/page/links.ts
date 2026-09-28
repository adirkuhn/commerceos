import { safeHref } from '../../core/richText'

/** A merchant-entered link as a real href: store paths ("/pages/about")
 *  are made relative to this store; web links pass through; anything
 *  unsafe becomes undefined. */
export function storeHref(href: string | undefined, storePath: (p: string) => string): string | undefined {
  const safe = safeHref(href)
  if (!safe) return undefined
  if (safe.startsWith('/') && !safe.startsWith('//')) return storePath(safe.slice(1))
  return safe
}
