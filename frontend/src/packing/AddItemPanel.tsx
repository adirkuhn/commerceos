import { useId, useState } from 'react'
import { Gift, PackagePlus, X } from 'lucide-react'
import clsx from 'clsx'

export interface AddItemInput {
  skuId: string
  name: string
  freebie: boolean
  quantity: number
  // Per-unit price for a paid new item; undefined for a freebie.
  priceMinorUnits: number | undefined
}

interface AddItemPanelProps {
  busy: boolean
  resolveCode: (code: string) => Promise<{ sku_id: string; product_name: string }>
  onAdd: (input: AddItemInput) => void
  onClose: () => void
}

const inputClass = 'w-full px-3 py-2 text-xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus'

// The packing screen's "Add item" panel: scan a product that isn't on the
// order and add it as a freebie or a paid new item. Staged like every other
// order change -- it becomes a real line when packing finishes.
export function AddItemPanel({ busy, resolveCode, onAdd, onClose }: AddItemPanelProps) {
  const [code, setCode] = useState('')
  const [product, setProduct] = useState<{ skuId: string; name: string } | null>(null)
  const [error, setError] = useState('')
  const [freebie, setFreebie] = useState(true)
  const [quantity, setQuantity] = useState('1')
  const [price, setPrice] = useState('0')
  const id = useId()

  async function resolve() {
    const scanned = code.trim()
    if (!scanned) return
    setError('')
    try {
      const r = await resolveCode(scanned)
      setProduct({ skuId: r.sku_id, name: r.product_name })
      setCode('')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Scan not recognized')
    }
  }

  function add() {
    if (!product) return
    onAdd({
      skuId: product.skuId, name: product.name, freebie, quantity: Number(quantity),
      priceMinorUnits: freebie ? undefined : Math.round(parseFloat(price || '0') * 100),
    })
  }

  return (
    <div className="w-80 border-l border-line bg-sunken flex flex-col shrink-0">
      <div className="flex items-center justify-between px-4 py-3 border-b border-line">
        <p className="text-xs font-semibold text-ink">Add item not on order</p>
        <button type="button" onClick={onClose} aria-label="Close" className="p-1 -m-1 rounded text-ink-4 hover:text-ink-2 hover:bg-hover">
          <X size={14} />
        </button>
      </div>
      <div className="p-4 space-y-4">
        {!product ? (
          <>
            <p className="text-2xs text-ink-3">Scan or type the product&apos;s SKU/barcode.</p>
            <input
              type="text" value={code} onChange={e => setCode(e.target.value)}
              onKeyDown={e => e.key === 'Enter' && resolve()}
              placeholder="Scan or type SKU/barcode" className={inputClass} autoFocus
            />
            {error && <p role="alert" className="text-2xs text-danger">{error}</p>}
          </>
        ) : (
          <>
            <p className="text-xs text-ink">{product.name}</p>
            <div className="flex items-center rounded-lg border border-line overflow-hidden">
              <button aria-pressed={freebie}
                type="button" onClick={() => setFreebie(true)}
                className={clsx('flex-1 flex items-center justify-center gap-1 px-3 py-1.5 text-2xs font-medium', freebie ? 'bg-accent-solid text-white' : 'bg-raised text-ink-3')}
              >
                <Gift size={11} /> Freebie
              </button>
              <button aria-pressed={!freebie}
                type="button" onClick={() => setFreebie(false)}
                className={clsx('flex-1 flex items-center justify-center gap-1 px-3 py-1.5 text-2xs font-medium', !freebie ? 'bg-accent-solid text-white' : 'bg-raised text-ink-3')}
              >
                <PackagePlus size={11} /> New item
              </button>
            </div>
            <div>
              <label htmlFor={`${id}-qty`} className="block text-2xs font-medium text-ink-4 uppercase tracking-wider mb-1.5">Quantity</label>
              <input
                id={`${id}-qty`} type="number" min={1} value={quantity}
                onChange={e => setQuantity(e.target.value)} className={inputClass}
              />
            </div>
            {!freebie && (
              <div>
                <label htmlFor={`${id}-price`} className="block text-2xs font-medium text-ink-4 uppercase tracking-wider mb-1.5">Price</label>
                <input id={`${id}-price`} type="number" min={0} step="0.01" value={price} onChange={e => setPrice(e.target.value)} className={inputClass} />
              </div>
            )}
            <button
              type="button" onClick={add} disabled={busy || !(Number(quantity) >= 1)}
              className="w-full px-3 py-2 rounded-lg bg-accent-solid text-white text-xs font-semibold hover:bg-accent-solid-hover disabled:opacity-50"
            >
              Add to order
            </button>
          </>
        )}
      </div>
    </div>
  )
}
