import { useEffect, useState } from 'react'
import { ArrowDown, ArrowUp, X } from 'lucide-react'
import { useAuth } from '../auth/AuthContext'
import { listProducts } from '../pms/api'
import type { Product } from '../pms/types'
import { inputClass } from './styles'

/** Hand-picked products of a manual collection, in the order shown to shoppers. */
export function ManualProducts({ ids, names, onChange }: { ids: string[]; names: Record<string, string>; onChange: (ids: string[], names: Record<string, string>) => void }) {
  const { token } = useAuth()
  const [term, setTerm] = useState('')
  const [results, setResults] = useState<Product[]>([])

  useEffect(() => {
    if (!term.trim()) {
      setResults([])
      return
    }
    const t = setTimeout(() => {
      listProducts(token ?? '', { search: term, perPage: 10 }).then(r => setResults(r.products), () => setResults([]))
    }, 250)
    return () => clearTimeout(t)
  }, [token, term])

  const move = (i: number, d: number) => {
    const next = [...ids]
    ;[next[i], next[i + d]] = [next[i + d], next[i]]
    onChange(next, names)
  }

  return (
    <div className="space-y-2">
      <ol className="space-y-1" aria-label="Products in this collection">
        {ids.map((id, i) => (
          <li key={id} className="flex items-center gap-2 rounded-lg border border-line px-3 py-1.5 text-xs">
            <span className="text-ink-4 w-5">{i + 1}.</span>
            <span className="flex-1 truncate text-ink-2">{names[id] ?? id}</span>
            <button type="button" aria-label="Move up" disabled={i === 0} onClick={() => move(i, -1)} className="disabled:opacity-30"><ArrowUp size={12} /></button>
            <button type="button" aria-label="Move down" disabled={i === ids.length - 1} onClick={() => move(i, 1)} className="disabled:opacity-30"><ArrowDown size={12} /></button>
            <button type="button" aria-label={`Remove ${names[id] ?? id}`} onClick={() => onChange(ids.filter(x => x !== id), names)} className="hover:text-danger"><X size={12} /></button>
          </li>
        ))}
        {!ids.length && <li className="text-xs text-ink-4">No products yet. Search below to add some.</li>}
      </ol>
      <input className={inputClass} placeholder="Search products to add…" aria-label="Search products to add" value={term} onChange={e => setTerm(e.target.value)} />
      {results.filter(p => !ids.includes(p.id)).length > 0 && (
        <ul className="max-h-48 overflow-y-auto rounded-lg border border-line" aria-label="Search results">
          {results.filter(p => !ids.includes(p.id)).map(p => (
            <li key={p.id}>
              <button type="button" className="w-full px-3 py-1.5 text-left text-xs hover:bg-hover" onClick={() => { onChange([...ids, p.id], { ...names, [p.id]: p.name }); setTerm('') }}>
                {p.name}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
