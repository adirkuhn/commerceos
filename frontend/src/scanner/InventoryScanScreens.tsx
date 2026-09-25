import { useState } from 'react'
import { ArrowLeft, Boxes } from 'lucide-react'
import * as pmsApi from '../pms/api'
import { searchSKUCandidates, type SKUCandidate } from '../pms/searchCandidates'
import { ScanCodeScreen } from './ScanCodeScreen'
import { ProductCandidateRow } from './ProductCandidateRow'
import { useQueryEnum } from '../routing/useQueryState'
import type { SKU } from '../pms/types'

export interface Resolved {
  sku: SKU
  productName: string
}

const SCAN_TABS = ['code', 'name'] as const

export function InventoryScanScreen({ token, onResolved, onExit }: { token: string; onResolved: (r: Resolved) => void; onExit: () => void }) {
  const [tab, setTab] = useQueryEnum('by', SCAN_TABS, 'code')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [candidates, setCandidates] = useState<SKUCandidate[]>([])

  // A camera misread or a hand-typed code can be truncated/mistyped -- on an
  // exact-match miss, fall back to the same partial (substring) match used
  // by "Search by name" (it already matches sku_code/barcode too, see
  // searchSKUCandidates), instead of just failing.
  async function handleSubmit(code: string) {
    setError('')
    setCandidates([])
    setLoading(true)
    try {
      const { sku, product_name } = await pmsApi.resolveBarcode(token, code)
      onResolved({ sku, productName: product_name })
    } catch (err) {
      try {
        const found = await searchSKUCandidates(token, code)
        if (found.length === 1) {
          onResolved({ sku: found[0].sku, productName: found[0].productName })
          return
        }
        if (found.length > 1) {
          setCandidates(found)
          return
        }
      } catch {
        // fall through to the original resolveBarcode error below
      }
      setError(err instanceof Error ? err.message : 'Failed to resolve code')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 flex flex-col bg-night-950 text-gray-100">
      <div className="flex items-center gap-3 px-4 py-4 border-b border-night-700 shrink-0">
        <button aria-label="Back to apps" onClick={onExit} className="w-11 h-11 -ml-3 -my-2 flex items-center justify-center rounded-lg shrink-0" title="Back to apps">
          <ArrowLeft size={18} className="text-gray-400" />
        </button>
        <h1 className="flex-1 text-base font-semibold">Inventory</h1>
      </div>
      <div className="flex gap-2 px-4 pt-4 shrink-0">
        <button aria-pressed={tab === 'code'}
          type="button"
          onClick={() => setTab('code')}
          className={`flex-1 py-2 rounded-lg text-xs font-medium ${tab === 'code' ? 'bg-accent-solid text-white' : 'bg-night-800 text-gray-400'}`}
        >
          Scan code
        </button>
        <button aria-pressed={tab === 'name'}
          type="button"
          onClick={() => setTab('name')}
          className={`flex-1 py-2 rounded-lg text-xs font-medium ${tab === 'name' ? 'bg-accent-solid text-white' : 'bg-night-800 text-gray-400'}`}
        >
          Search by name
        </button>
      </div>
      {tab === 'code' ? (
        <div className="flex-1 flex flex-col overflow-y-auto">
          <ScanCodeScreen icon={Boxes} prompt="Scan a barcode or type a SKU code" loading={loading} error={error} onSubmit={handleSubmit} />
          {candidates.length > 0 && (
            <div className="w-full max-w-xs mx-auto px-6 pb-6 space-y-1.5 shrink-0">
              <p className="text-xs text-gray-400">Multiple matches -- pick one:</p>
              {candidates.map(c => (
                <ProductCandidateRow key={c.sku.id} token={token} candidate={c} onClick={() => onResolved({ sku: c.sku, productName: c.productName })} />
              ))}
            </div>
          )}
        </div>
      ) : (
        <NameSearchScreen token={token} onResolved={onResolved} />
      )}
    </div>
  )
}

// Resolves a typed product name to a SKU: list_products' Search already
// matches product name (case/accent-insensitive), then each matched
// product's SKUs are fetched to flatten into pickable candidates -- exactly
// one candidate resolves straight through (same as a barcode scan), more
// than one shows a list to pick from.
function NameSearchScreen({ token, onResolved }: { token: string; onResolved: (r: Resolved) => void }) {
  const [name, setName] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [candidates, setCandidates] = useState<SKUCandidate[]>([])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const query = name.trim()
    if (!query) return
    setError('')
    setCandidates([])
    setLoading(true)
    try {
      const found = await searchSKUCandidates(token, query)
      if (found.length === 0) {
        setError('No products found')
        return
      }
      if (found.length === 1) {
        onResolved({ sku: found[0].sku, productName: found[0].productName })
        return
      }
      setCandidates(found)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to search')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex-1 flex flex-col items-center gap-6 p-6 overflow-y-auto">
      <form onSubmit={handleSubmit} className="w-full max-w-xs space-y-3 shrink-0">
        <input
          autoFocus
          value={name}
          onChange={e => setName(e.target.value)}
          placeholder="Product name…"
          className="w-full px-4 py-4 text-center text-sm rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus"
        />
        {error && <p className="text-sm text-red-400">{error}</p>}
        <button
          type="submit"
          disabled={loading || name.trim().length === 0}
          className="w-full py-3 rounded-xl bg-accent-solid text-white font-semibold hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
        >
          {loading ? 'Searching…' : 'Search'}
        </button>
      </form>
      {candidates.length > 0 && (
        <div className="w-full max-w-xs space-y-1.5">
          {candidates.map(c => (
            <ProductCandidateRow key={c.sku.id} token={token} candidate={c} onClick={() => onResolved({ sku: c.sku, productName: c.productName })} />
          ))}
        </div>
      )}
    </div>
  )
}
