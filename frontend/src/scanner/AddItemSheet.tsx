import { useState } from 'react'
import { Gift, PackagePlus, Search, X } from 'lucide-react'
import { searchSKUCandidates, type SKUCandidate } from '../pms/searchCandidates'
import { QuantityStepper } from './QuantityStepper'
import { ProductCandidateRow } from './ProductCandidateRow'
import { Dialog } from '../components/common/Dialog'

export type ExtraItemAction = 'freebie' | 'new_item'

// Scan/search any catalog product not on the order while packing, then
// decide whether it's a no-charge freebie or a real, priced addition. Both
// become a real SalesOrderLineItem and decrement real stock when packing
// finishes (see OrderChangeApplier) -- a freebie carries no price, so the
// price field is hidden rather than shown-and-ignored.
export function AddItemSheet({ token, onCancel, onAdd }: {
  token: string
  onCancel: () => void
  onAdd: (skuId: string, actionType: ExtraItemAction, quantity: number, priceMinorUnits: number | undefined, reason: string | undefined) => void
}) {
  const [query, setQuery] = useState('')
  const [searching, setSearching] = useState(false)
  const [candidates, setCandidates] = useState<SKUCandidate[]>([])
  const [selected, setSelected] = useState<SKUCandidate | null>(null)
  const [isFreebie, setIsFreebie] = useState(true)
  const [qty, setQty] = useState('1')
  const [price, setPrice] = useState('')
  const [reason, setReason] = useState('')
  const n = Number(qty) || 0
  const priceMinorUnits = Math.round((Number(price) || 0) * 100)

  async function handleSearch(e: React.FormEvent) {
    e.preventDefault()
    if (!query.trim()) return
    setSearching(true)
    try {
      setCandidates(await searchSKUCandidates(token, query.trim()))
    } finally {
      setSearching(false)
    }
  }

  function handleConfirm() {
    if (!selected) return
    onAdd(
      selected.sku.id,
      isFreebie ? 'freebie' : 'new_item',
      n,
      isFreebie ? undefined : priceMinorUnits,
      reason || selected.productName,
    )
  }

  return (
    <Dialog onClose={onCancel} className="fixed inset-0 z-50 flex items-end justify-center bg-black/40">
      <div className="w-full max-w-sm rounded-t-2xl bg-night-900 border-t border-night-700 max-h-[85vh] overflow-y-auto">
        <div className="flex items-start justify-between gap-3 px-5 pt-5">
          <h2 className="text-sm font-semibold">Add freebie or new item</h2>
          <button aria-label="Close" onClick={onCancel} title="Close" className="shrink-0 w-11 h-11 -mt-3 -mr-3 flex items-center justify-center rounded-lg text-gray-400 hover:text-gray-300">
            <X size={18} />
          </button>
        </div>

        <div className="p-5 space-y-4">
          {!selected ? (
            <>
              <form onSubmit={handleSearch} className="flex gap-2">
                <div className="relative flex-1">
                  <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
                  <input
                    value={query}
                    onChange={e => setQuery(e.target.value)}
                    placeholder="Search or scan barcode…"
                    className="w-full pl-8 pr-3 py-2.5 text-sm rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus"
                    autoFocus
                  />
                </div>
                <button type="submit" disabled={searching || !query.trim()} className="px-4 rounded-xl bg-night-800 border border-night-700 text-sm text-gray-300 disabled:opacity-50">
                  {searching ? '…' : 'Find'}
                </button>
              </form>
              <div className="space-y-1.5">
                {candidates.map(c => (
                  <ProductCandidateRow key={c.sku.id} token={token} candidate={c} onClick={() => { setSelected(c); setReason(c.productName) }} />
                ))}
              </div>
            </>
          ) : (
            <>
              <div className="flex items-center gap-3 rounded-xl bg-night-800 border border-night-700 px-3.5 py-3">
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium truncate">{selected.productName}</p>
                  <p className="text-xs text-gray-400 truncate">{selected.sku.sku_code}</p>
                </div>
                <button type="button" onClick={() => setSelected(null)} className="text-xs text-accent shrink-0">Change</button>
              </div>

              <div className="flex gap-2">
                <button aria-pressed={isFreebie}
                  type="button"
                  onClick={() => setIsFreebie(true)}
                  className={`flex-1 flex flex-col items-center gap-1 py-2.5 rounded-lg text-2xs font-medium leading-tight ${isFreebie ? 'bg-purple-600 text-white' : 'bg-night-800 text-gray-400'}`}
                >
                  <Gift size={15} /> Freebie
                </button>
                <button aria-pressed={!isFreebie}
                  type="button"
                  onClick={() => setIsFreebie(false)}
                  className={`flex-1 flex flex-col items-center gap-1 py-2.5 rounded-lg text-2xs font-medium leading-tight ${!isFreebie ? 'bg-accent-solid text-white' : 'bg-night-800 text-gray-400'}`}
                >
                  <PackagePlus size={15} /> New item
                </button>
              </div>

              <QuantityStepper id="add-item-quantity" label="Quantity" value={qty} onChange={setQty} min={1} />

              {!isFreebie && (
                <div>
                  <label htmlFor="add-item-price" className="block text-xs text-gray-400 mb-1.5">Price (each)</label>
                  <input
                    id="add-item-price"
                    inputMode="decimal"
                    value={price}
                    onChange={e => setPrice(e.target.value)}
                    placeholder="0.00"
                    className="w-full px-3 py-2.5 rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus"
                  />
                </div>
              )}

              <div>
                <label htmlFor="add-item-reason" className="block text-xs text-gray-400 mb-1.5">Name / note</label>
                <input
                  id="add-item-reason"
                  value={reason}
                  onChange={e => setReason(e.target.value)}
                  className="w-full px-3 py-2.5 rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus"
                />
              </div>

              <button
                onClick={handleConfirm}
                disabled={n <= 0 || (!isFreebie && priceMinorUnits < 0)}
                className={`w-full py-3 rounded-xl text-white font-semibold disabled:opacity-50 ${isFreebie ? 'bg-purple-600' : 'bg-accent-solid'}`}
              >
                Add to order
              </button>
            </>
          )}
        </div>
      </div>
    </Dialog>
  )
}
