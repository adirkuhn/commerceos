import { useEffect, useState, useSyncExternalStore } from 'react'
import type { Category, Collection, ProductSummary } from '../../api/storefront'
import { useEditorServices } from '../services'
import { AssetLibrary } from './AssetLibrary'
import { Field, SelectField } from './fields'
import { moveItem } from './listOps'

const ASSET_REF = /^asset:(.+)$/

/** Picks a media-library image; the value is an "asset:<id>" reference. */
export function AssetPicker({ label, value, onChange }: { label: string; value: string; onChange: (ref: string) => void }) {
  const { assets } = useEditorServices()
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
            {value ? 'Change' : 'Choose image'}
          </button>
          {value && <button type="button" className="text-sm text-neutral-600" onClick={() => onChange('')}>Remove</button>}
          {open && <AssetLibrary onClose={() => setOpen(false)} onPick={a => { onChange(`asset:${a.id}`); setOpen(false) }} />}
        </div>
      )}
    </Field>
  )
}

export function CategorySelect({ label, value, onChange }: { label: string; value: string; onChange: (id: string) => void }) {
  const { store } = useEditorServices()
  const [categories, setCategories] = useState<Category[]>([])
  useEffect(() => {
    store.listCategories().then(setCategories, () => setCategories([]))
  }, [store])
  return (
    <SelectField label={label} value={value} allowEmpty="Choose a category…"
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
  const name = (id: string) => collections.find(c => c.id === id)?.name ?? 'Unpublished collection'
  const available = collections.filter(c => !value.includes(c.id))
  return (
    <Field label={label} hint={`${value.length} of ${max}`}>
      {id => (
        <div className="flex flex-col gap-2">
          <ol className="flex flex-col gap-1" aria-label="Chosen collections">
            {value.map((cid, i) => (
              <li key={cid} className="flex items-center gap-1 rounded bg-neutral-100 px-2 py-1 text-sm">
                <span className="flex-1 truncate">{name(cid)}</span>
                <button type="button" aria-label="Move up" disabled={i === 0} onClick={() => onChange(moveItem(value, i, -1))}>↑</button>
                <button type="button" aria-label="Move down" disabled={i === value.length - 1} onClick={() => onChange(moveItem(value, i, 1))}>↓</button>
                <button type="button" aria-label="Remove" onClick={() => onChange(value.filter(v => v !== cid))}>✕</button>
              </li>
            ))}
          </ol>
          {value.length < max && (
            <select id={id} className="w-full rounded-md border border-neutral-300 px-2 py-1.5 text-sm" value=""
              onChange={e => e.target.value && onChange([...value, e.target.value])}>
              <option value="">{available.length ? 'Add a collection…' : collections.length ? 'All collections added' : 'No published collections yet'}</option>
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
  return (
    <SelectField label={label} value={value} allowEmpty={collections.length ? 'Choose a collection…' : 'No published collections yet'}
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
  const results = useProductSearch(term).filter(p => !exclude.includes(p.id))
  return (
    <div className="flex flex-col gap-1">
      <input className="w-full rounded-md border border-neutral-300 px-2 py-1.5 text-sm" placeholder="Search products…" aria-label="Search products"
        value={term} onChange={e => setTerm(e.target.value)} />
      <ul className="max-h-40 overflow-y-auto rounded border" aria-label="Search results">
        {results.map(p => (
          <li key={p.id}><button type="button" className="w-full px-2 py-1 text-left text-sm hover:bg-neutral-100" onClick={() => onPick(p)}>{p.name}</button></li>
        ))}
        {!results.length && <li className="px-2 py-1 text-xs text-neutral-500">No products found</li>}
      </ul>
    </div>
  )
}

export function ProductPicker({ label, value, onChange }: { label: string; value: string; onChange: (id: string) => void }) {
  const byId = useProductsByIds(value ? [value] : [])
  return (
    <Field label={label}>
      {() => (
        <div className="flex flex-col gap-2">
          {value && (
            <div className="flex items-center justify-between rounded bg-neutral-100 px-2 py-1 text-sm">
              <span>{byId[value]?.name ?? 'Selected product'}</span>
              <button type="button" className="text-xs text-neutral-600" onClick={() => onChange('')}>Clear</button>
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
  const move = (i: number, d: number) => onChange(moveItem(value, i, d))
  return (
    <Field label={label} hint={`${value.length} of ${max}`}>
      {() => (
        <div className="flex flex-col gap-2">
          <ol className="flex flex-col gap-1" aria-label="Chosen products">
            {value.map((id, i) => (
              <li key={id} className="flex items-center gap-1 rounded bg-neutral-100 px-2 py-1 text-sm">
                <span className="flex-1 truncate">{byId[id]?.name ?? id}</span>
                <button type="button" aria-label="Move up" disabled={i === 0} onClick={() => move(i, -1)}>↑</button>
                <button type="button" aria-label="Move down" disabled={i === value.length - 1} onClick={() => move(i, 1)}>↓</button>
                <button type="button" aria-label="Remove" onClick={() => onChange(value.filter(v => v !== id))}>✕</button>
              </li>
            ))}
          </ol>
          {value.length < max && <ProductSearch exclude={value} onPick={p => onChange([...value, p.id])} />}
        </div>
      )}
    </Field>
  )
}
