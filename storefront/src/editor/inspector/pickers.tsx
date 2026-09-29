import { useEffect, useState, useSyncExternalStore } from 'react'
import { useTranslation } from 'react-i18next'
import type { Category, Collection, ProductSummary } from '../../api/storefront'
import { useEditorServices } from '../services'
import { AssetLibrary } from './AssetLibrary'
import { Field, SelectField } from './fields'
import { moveItem } from './listOps'

const ASSET_REF = /^asset:(.+)$/

/** Picks a media-library image; the value is an "asset:<id>" reference. */
export function AssetPicker({ label, value, onChange }: { label: string; value: string; onChange: (ref: string) => void }) {
  const { assets } = useEditorServices()
  const { t } = useTranslation('editor')
  const refs = useSyncExternalStore(assets.subscribe, assets.snapshot)
  const [open, setOpen] = useState(false)
  const current = refs[ASSET_REF.exec(value)?.[1] ?? '']
  return (
    <Field label={label}>
      {id => (
        <div className="flex items-center gap-2">
          <div className="h-12 w-12 shrink-0 overflow-hidden rounded border bg-neutral-100">
            {current && <img src={current.url} alt="" className="h-full w-full object-cover" />}
          </div>
          <button id={id} type="button" className="rounded-md border px-2 py-1 text-sm" onClick={() => setOpen(true)}>
            {value ? t('pickers.change') : t('pickers.chooseImage')}
          </button>
          {value && <button type="button" className="text-sm text-neutral-600" onClick={() => onChange('')}>{t('pickers.remove')}</button>}
          {open && <AssetLibrary onClose={() => setOpen(false)} onPick={a => { onChange(`asset:${a.id}`); setOpen(false) }} />}
        </div>
      )}
    </Field>
  )
}

export function CategorySelect({ label, value, onChange }: { label: string; value: string; onChange: (id: string) => void }) {
  const { store } = useEditorServices()
  const [categories, setCategories] = useState<Category[]>([])
  const { t } = useTranslation('editor')
  useEffect(() => {
    store.listCategories().then(setCategories, () => setCategories([]))
  }, [store])
  return (
    <SelectField label={label} value={value} allowEmpty={t('pickers.chooseCategory')}
      options={categories.map(c => ({ value: c.id, label: c.name }))} onChange={v => onChange(String(v))} />
  )
}

function useCollections() {
  const { store } = useEditorServices()
  const [collections, setCollections] = useState<Collection[]>([])
  useEffect(() => {
    store.listCollections().then(setCollections, () => setCollections([]))
  }, [store])
  return collections
}

/** An ordered choice of published collections. */
export function CollectionsPicker({ label, value, max, onChange }: { label: string; value: string[]; max: number; onChange: (ids: string[]) => void }) {
  const collections = useCollections()
  const { t } = useTranslation('editor')
  const name = (id: string) => collections.find(c => c.id === id)?.name ?? t('pickers.unpublishedCollection')
  const available = collections.filter(c => !value.includes(c.id))
  return (
    <Field label={label} hint={t('pickers.chosenOf', { chosen: value.length, max })}>
      {id => (
        <div className="flex flex-col gap-2">
          <ol className="flex flex-col gap-1" aria-label={t('pickers.chosenCollections')}>
            {value.map((cid, i) => (
              <li key={cid} className="flex items-center gap-1 rounded bg-neutral-100 px-2 py-1 text-sm">
                <span className="flex-1 truncate">{name(cid)}</span>
                <button type="button" aria-label={t('pickers.moveUp')} disabled={i === 0} onClick={() => onChange(moveItem(value, i, -1))}>↑</button>
                <button type="button" aria-label={t('pickers.moveDown')} disabled={i === value.length - 1} onClick={() => onChange(moveItem(value, i, 1))}>↓</button>
                <button type="button" aria-label={t('pickers.remove')} onClick={() => onChange(value.filter(v => v !== cid))}>✕</button>
              </li>
            ))}
          </ol>
          {value.length < max && (
            <select id={id} className="w-full rounded-md border border-neutral-300 px-2 py-1.5 text-sm" value=""
              onChange={e => e.target.value && onChange([...value, e.target.value])}>
              <option value="">{available.length ? t('pickers.addCollection') : collections.length ? t('pickers.allCollectionsAdded') : t('pickers.noCollections')}</option>
              {available.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
            </select>
          )}
        </div>
      )}
    </Field>
  )
}

export function CollectionSelect({ label, value, onChange }: { label: string; value: string; onChange: (id: string) => void }) {
  const collections = useCollections()
  const { t } = useTranslation('editor')
  return (
    <SelectField label={label} value={value} allowEmpty={collections.length ? t('pickers.chooseCollection') : t('pickers.noCollections')}
      options={collections.map(c => ({ value: c.id, label: c.name }))} onChange={v => onChange(String(v))} />
  )
}

function useProductSearch(term: string) {
  const { store } = useEditorServices()
  const [results, setResults] = useState<ProductSummary[]>([])
  useEffect(() => {
    let cancelled = false
    const t = setTimeout(() => {
      store.listProducts({ search: term || undefined, perPage: 10 }).then(r => !cancelled && setResults(r.items), () => !cancelled && setResults([]))
    }, 250)
    return () => { cancelled = true; clearTimeout(t) }
  }, [store, term])
  return results
}

function useProductsByIds(ids: string[]) {
  const { store } = useEditorServices()
  const [byId, setById] = useState<Record<string, ProductSummary>>({})
  const key = ids.join(',')
  useEffect(() => {
    if (!key) return
    store.listProducts({ productIds: key.split(','), perPage: 48 }).then(r => setById(Object.fromEntries(r.items.map(p => [p.id, p]))), () => {})
  }, [store, key])
  return byId
}

export function ProductSearch({ onPick, exclude }: { onPick: (p: ProductSummary) => void; exclude: string[] }) {
  const [term, setTerm] = useState('')
  const { t } = useTranslation('editor')
  const results = useProductSearch(term).filter(p => !exclude.includes(p.id))
  return (
    <div className="flex flex-col gap-1">
      <input className="w-full rounded-md border border-neutral-300 px-2 py-1.5 text-sm" placeholder={t('pickers.searchProducts')} aria-label={t('pickers.searchProducts')}
        value={term} onChange={e => setTerm(e.target.value)} />
      <ul className="max-h-40 overflow-y-auto rounded border" aria-label={t('pickers.searchResults')}>
        {results.map(p => (
          <li key={p.id}><button type="button" className="w-full px-2 py-1 text-left text-sm hover:bg-neutral-100" onClick={() => onPick(p)}>{p.name}</button></li>
        ))}
        {!results.length && <li className="px-2 py-1 text-xs text-neutral-500">{t('pickers.noProducts')}</li>}
      </ul>
    </div>
  )
}

export function ProductPicker({ label, value, onChange }: { label: string; value: string; onChange: (id: string) => void }) {
  const byId = useProductsByIds(value ? [value] : [])
  const { t } = useTranslation('editor')
  return (
    <Field label={label}>
      {() => (
        <div className="flex flex-col gap-2">
          {value && (
            <div className="flex items-center justify-between rounded bg-neutral-100 px-2 py-1 text-sm">
              <span>{byId[value]?.name ?? t('pickers.selectedProduct')}</span>
              <button type="button" className="text-xs text-neutral-600" onClick={() => onChange('')}>{t('pickers.clear')}</button>
            </div>
          )}
          <ProductSearch exclude={value ? [value] : []} onPick={p => onChange(p.id)} />
        </div>
      )}
    </Field>
  )
}

export function ProductsPicker({ label, value, max, onChange }: { label: string; value: string[]; max: number; onChange: (ids: string[]) => void }) {
  const byId = useProductsByIds(value)
  const { t } = useTranslation('editor')
  const move = (i: number, d: number) => onChange(moveItem(value, i, d))
  return (
    <Field label={label} hint={t('pickers.chosenOf', { chosen: value.length, max })}>
      {() => (
        <div className="flex flex-col gap-2">
          <ol className="flex flex-col gap-1" aria-label={t('pickers.chosenProducts')}>
            {value.map((id, i) => (
              <li key={id} className="flex items-center gap-1 rounded bg-neutral-100 px-2 py-1 text-sm">
                <span className="flex-1 truncate">{byId[id]?.name ?? id}</span>
                <button type="button" aria-label={t('pickers.moveUp')} disabled={i === 0} onClick={() => move(i, -1)}>↑</button>
                <button type="button" aria-label={t('pickers.moveDown')} disabled={i === value.length - 1} onClick={() => move(i, 1)}>↓</button>
                <button type="button" aria-label={t('pickers.remove')} onClick={() => onChange(value.filter(v => v !== id))}>✕</button>
              </li>
            ))}
          </ol>
          {value.length < max && <ProductSearch exclude={value} onPick={p => onChange([...value, p.id])} />}
        </div>
      )}
    </Field>
  )
}
