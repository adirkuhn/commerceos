import { useEffect, useState, useSyncExternalStore } from 'react'
import { Trans, useTranslation } from 'react-i18next'
import type { Collection } from '../api/storefront'
import { ProductSearch } from './inspector/pickers'
import { useEditorServices } from './services'

/** "Previewing with …" for product/collection templates, so merchants
 *  design against real catalog data. */
export function PreviewBar({ kind, error, onProduct, onCollection }: {
  kind: string; error: string; onProduct: (slug: string) => void; onCollection: (slug: string) => void
}) {
  const { preview, store } = useEditorServices()
  const { t } = useTranslation('editor')
  const data = useSyncExternalStore(preview.subscribe, preview.snapshot)
  const [picking, setPicking] = useState(false)
  const [collections, setCollections] = useState<Collection[]>([])
  useEffect(() => {
    if (kind === 'collection') store.listCollections().then(setCollections, () => setCollections([]))
  }, [kind, store])

  if (kind !== 'product' && kind !== 'collection') return null
  return (
    <div className="flex items-center gap-2 border-b border-neutral-200 bg-amber-50 px-4 py-1.5 text-sm" role="region" aria-label={t('previewBar.region')}>
      {kind === 'product' ? (
        <>
          <span><Trans t={t} ns="editor" i18nKey="previewBar.previewingWith" values={{ name: data.product?.name ?? (error ? '—' : '…') }} components={{ strong: <strong /> }} /></span>
          <button type="button" className="underline" onClick={() => setPicking(p => !p)}>{picking ? t('previewBar.done') : t('previewBar.changeProduct')}</button>
          {picking && <div className="w-72"><ProductSearch exclude={[]} onPick={p => { onProduct(p.slug); setPicking(false) }} /></div>}
          {!data.product && !error && <span className="text-neutral-500">{t('previewBar.addProduct')}</span>}
        </>
      ) : (
        <label className="flex items-center gap-2">{t('previewBar.previewingCollection')}
          <select className="rounded border px-1 py-0.5" value={collections.find(c => c.name === data.listing?.title)?.slug ?? ''}
            onChange={e => onCollection(e.target.value)} aria-label={t('previewBar.previewCollection')}>
            {!collections.length && <option value="">{t('previewBar.noCollections')}</option>}
            {collections.map(c => <option key={c.id} value={c.slug}>{c.name}</option>)}
          </select>
        </label>
      )}
      {error && <span className="text-red-700">{error}</span>}
    </div>
  )
}
