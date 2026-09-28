import { useEffect, useState, useSyncExternalStore } from 'react'
import type { Collection } from '../api/storefront'
import { ProductSearch } from './inspector/pickers'
import { useEditorServices } from './services'

/** "Previewing with …" for product/collection templates, so merchants
 *  design against real catalog data. */
export function PreviewBar({ kind, error, onProduct, onCollection }: {
  kind: string; error: string; onProduct: (slug: string) => void; onCollection: (slug: string) => void
}) {
  const { preview, store } = useEditorServices()
  const data = useSyncExternalStore(preview.subscribe, preview.snapshot)
  const [picking, setPicking] = useState(false)
  const [collections, setCollections] = useState<Collection[]>([])
  useEffect(() => {
    if (kind === 'collection') store.listCollections().then(setCollections, () => setCollections([]))
  }, [kind, store])

  if (kind !== 'product' && kind !== 'collection') return null
  return (
    <div className="flex items-center gap-2 border-b border-neutral-200 bg-amber-50 px-4 py-1.5 text-sm" role="region" aria-label="Preview data">
      {kind === 'product' ? (
        <>
          <span>Previewing with <strong>{data.product?.name ?? (error ? '—' : '…')}</strong></span>
          <button type="button" className="underline" onClick={() => setPicking(p => !p)}>{picking ? 'Done' : 'Change product'}</button>
          {picking && <div className="w-72"><ProductSearch exclude={[]} onPick={p => { onProduct(p.slug); setPicking(false) }} /></div>}
          {!data.product && !error && <span className="text-neutral-500">Add a product to your catalog to preview this page.</span>}
        </>
      ) : (
        <label className="flex items-center gap-2">Previewing collection
          <select className="rounded border px-1 py-0.5" value={collections.find(c => c.name === data.listing?.title)?.slug ?? ''}
            onChange={e => onCollection(e.target.value)} aria-label="Preview collection">
            {!collections.length && <option value="">No published collections yet</option>}
            {collections.map(c => <option key={c.id} value={c.slug}>{c.name}</option>)}
          </select>
        </label>
      )}
      {error && <span className="text-red-700">{error}</span>}
    </div>
  )
}
