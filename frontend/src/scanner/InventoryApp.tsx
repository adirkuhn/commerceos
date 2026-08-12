import { useState } from 'react'
import { ArrowLeft, Boxes, Package, ArrowRightLeft, PackageMinus, Plus, Minus } from 'lucide-react'
import * as pmsApi from '../pms/api'
import { useSKUStock } from '../inventory/hooks/useSKUStock'
import { useWarehouses } from '../inventory/hooks/useWarehouses'
import { ScanCodeScreen } from './ScanCodeScreen'
import type { SKU } from '../pms/types'

interface InventoryAppProps {
  token: string
  onExit: () => void
}

// Scanner mini-app: scan/type a code to resolve a SKU, then adjust, move, or
// write off its stock -- every action also works by tapping alone (numeric
// keypad inputs + steppers + confirm buttons), not scan-only.
export function InventoryApp({ token, onExit }: InventoryAppProps) {
  const [resolvedSKU, setResolvedSKU] = useState<SKU | null>(null)

  if (!resolvedSKU) {
    return <InventoryScanScreen token={token} onResolved={setResolvedSKU} onExit={onExit} />
  }

  return <InventorySKUScreen token={token} sku={resolvedSKU} onBack={() => setResolvedSKU(null)} />
}

function InventoryScanScreen({ token, onResolved, onExit }: { token: string; onResolved: (sku: SKU) => void; onExit: () => void }) {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  async function handleSubmit(code: string) {
    setError('')
    setLoading(true)
    try {
      const { sku } = await pmsApi.resolveBarcode(token, code)
      onResolved(sku)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to resolve code')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 flex flex-col bg-[#0B1220] text-gray-100">
      <div className="flex items-center gap-3 px-4 py-4 border-b border-[#2D3748] shrink-0">
        <button onClick={onExit} className="p-1 -ml-1" title="Back to apps">
          <ArrowLeft size={18} className="text-gray-400" />
        </button>
        <h1 className="flex-1 text-base font-semibold">Inventory</h1>
      </div>
      <ScanCodeScreen icon={Boxes} prompt="Scan a barcode or type a SKU code" loading={loading} error={error} onSubmit={handleSubmit} />
    </div>
  )
}

type Sheet = 'adjust' | 'move' | 'damaged-lost' | null

function InventorySKUScreen({ token, sku, onBack }: { token: string; sku: SKU; onBack: () => void }) {
  const { level, batches, loading, error, adjust, transfer, markDamagedOrLost } = useSKUStock(token, sku.id)
  const { warehouses } = useWarehouses(token)
  const [sheet, setSheet] = useState<Sheet>(null)
  const [actionError, setActionError] = useState('')

  const warehouseName = (id: string) => warehouses.find(w => w.id === id)?.name ?? id
  // Defaults every action to the SKU's primary warehouse -- the first
  // batch/general-ledger row loaded. A SKU with stock in more than one
  // warehouse can still pick a different one inside the Move sheet.
  const primaryWarehouseId = batches[0]?.warehouse_id ?? warehouses[0]?.id ?? ''

  async function runAction(fn: () => Promise<unknown>) {
    setActionError('')
    try {
      await fn()
      setSheet(null)
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Action failed')
    }
  }

  return (
    <div className="fixed inset-0 flex flex-col bg-[#0B1220] text-gray-100">
      <div className="flex items-center gap-3 px-4 py-4 border-b border-[#2D3748] shrink-0">
        <button onClick={onBack} className="p-1 -ml-1" title="Back to scan">
          <ArrowLeft size={18} className="text-gray-400" />
        </button>
        <div className="flex-1 min-w-0">
          <h1 className="text-base font-semibold truncate">{sku.sku_code}</h1>
          {level && <p className="text-xs text-gray-500">Total stock: {level.total}</p>}
        </div>
      </div>

      <div className="grid grid-cols-3 gap-2 p-3 shrink-0">
        <button onClick={() => setSheet('adjust')} className="flex flex-col items-center gap-1.5 py-3 rounded-xl bg-[#1F2937] border border-[#2D3748] hover:border-blue-500 transition-colors">
          <Plus size={18} className="text-blue-400" />
          <span className="text-xs font-medium">Adjust</span>
        </button>
        <button onClick={() => setSheet('move')} className="flex flex-col items-center gap-1.5 py-3 rounded-xl bg-[#1F2937] border border-[#2D3748] hover:border-blue-500 transition-colors">
          <ArrowRightLeft size={18} className="text-blue-400" />
          <span className="text-xs font-medium">Move</span>
        </button>
        <button onClick={() => setSheet('damaged-lost')} className="flex flex-col items-center gap-1.5 py-3 rounded-xl bg-[#1F2937] border border-[#2D3748] hover:border-red-500 transition-colors">
          <PackageMinus size={18} className="text-red-400" />
          <span className="text-xs font-medium">Damaged/Lost</span>
        </button>
      </div>

      {error && <p className="mx-4 mb-2 text-sm text-red-400">{error}</p>}

      <div className="flex-1 overflow-y-auto px-4 pb-4 space-y-2">
        {loading && <p className="text-center text-sm text-gray-500 py-8">Loading…</p>}
        {!loading && batches.length === 0 && (
          <p className="text-center text-sm text-gray-500 py-8">No batches yet</p>
        )}
        {batches.map(b => (
          <div key={b.is_general_ledger ? `general-${b.warehouse_id}` : b.id} className="flex items-center gap-3 p-3.5 rounded-xl bg-[#1F2937] border border-[#2D3748]">
            <div className="w-9 h-9 rounded-lg bg-[#2D3748] flex items-center justify-center shrink-0">
              <Package size={16} className="text-gray-400" />
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium truncate">{b.is_general_ledger ? 'General ledger' : (b.lot_number || '—')}</p>
              <p className="text-xs text-gray-500">{warehouseName(b.warehouse_id)}</p>
            </div>
            <p className="text-sm font-mono shrink-0">{b.quantity}</p>
          </div>
        ))}
      </div>

      {sheet === 'adjust' && (
        <AdjustSheet
          error={actionError}
          onCancel={() => setSheet(null)}
          onConfirm={(quantityDelta, reason) => runAction(() => adjust({
            warehouse_id: primaryWarehouseId, type: 'adjustment', quantity_delta: quantityDelta, reason,
          }))}
        />
      )}
      {sheet === 'move' && (
        <MoveSheet
          fromWarehouseId={primaryWarehouseId}
          warehouses={warehouses}
          error={actionError}
          onCancel={() => setSheet(null)}
          onConfirm={(toWarehouseId, quantity) => runAction(() => transfer({
            from_warehouse_id: primaryWarehouseId, to_warehouse_id: toWarehouseId, quantity,
          }))}
        />
      )}
      {sheet === 'damaged-lost' && (
        <DamagedLostSheet
          error={actionError}
          onCancel={() => setSheet(null)}
          onConfirm={(quantity, type, reason) => runAction(() => markDamagedOrLost({
            warehouse_id: primaryWarehouseId, quantity, type, reason,
          }))}
        />
      )}
    </div>
  )
}

function SheetShell({ title, error, onCancel, children }: { title: string; error: string; onCancel: () => void; children: React.ReactNode }) {
  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-black/40">
      <div className="w-full max-w-sm rounded-t-2xl bg-[#111827] border-t border-[#2D3748] p-5 space-y-4">
        <h2 className="text-sm font-semibold">{title}</h2>
        {error && <p className="text-sm text-red-400">{error}</p>}
        {children}
        <button onClick={onCancel} className="w-full py-2.5 rounded-xl bg-[#1F2937] text-gray-300 text-sm font-medium">
          Cancel
        </button>
      </div>
    </div>
  )
}

function QuantityStepper({ id, label, value, onChange }: { id: string; label: string; value: string; onChange: (v: string) => void }) {
  const n = Number(value) || 0
  return (
    <div>
      <label htmlFor={id} className="block text-xs text-gray-400 mb-1.5">{label}</label>
      <div className="flex items-center gap-2">
        <button type="button" onClick={() => onChange(String(n - 1))} className="w-11 h-11 shrink-0 rounded-xl bg-[#1F2937] flex items-center justify-center" aria-label={`Decrease ${label}`}>
          <Minus size={16} />
        </button>
        <input
          id={id}
          inputMode="numeric"
          value={value}
          onChange={e => onChange(e.target.value.replace(/(?!^-)[^\d]/g, ''))}
          className="flex-1 px-3 py-2.5 text-center text-lg rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500"
        />
        <button type="button" onClick={() => onChange(String(n + 1))} className="w-11 h-11 shrink-0 rounded-xl bg-[#1F2937] flex items-center justify-center" aria-label={`Increase ${label}`}>
          <Plus size={16} />
        </button>
      </div>
    </div>
  )
}

function AdjustSheet({ error, onCancel, onConfirm }: { error: string; onCancel: () => void; onConfirm: (quantityDelta: number, reason: string) => void }) {
  const [quantity, setQuantity] = useState('')
  const [reason, setReason] = useState('')
  return (
    <SheetShell title="Adjust stock" error={error} onCancel={onCancel}>
      <QuantityStepper id="adjust-quantity" label="Quantity change" value={quantity} onChange={setQuantity} />
      <div>
        <label htmlFor="adjust-reason" className="block text-xs text-gray-400 mb-1.5">Reason</label>
        <input id="adjust-reason" value={reason} onChange={e => setReason(e.target.value)} className="w-full px-3 py-2.5 rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500" />
      </div>
      <button
        onClick={() => onConfirm(Number(quantity) || 0, reason)}
        disabled={!quantity || Number(quantity) === 0}
        className="w-full py-3 rounded-xl bg-blue-600 text-white font-semibold disabled:opacity-50"
      >
        Confirm adjustment
      </button>
    </SheetShell>
  )
}

function MoveSheet({ fromWarehouseId, warehouses, error, onCancel, onConfirm }: {
  fromWarehouseId: string
  warehouses: { id: string; name: string }[]
  error: string
  onCancel: () => void
  onConfirm: (toWarehouseId: string, quantity: number) => void
}) {
  const destinations = warehouses.filter(w => w.id !== fromWarehouseId)
  const [toWarehouseId, setToWarehouseId] = useState(destinations[0]?.id ?? '')
  const [quantity, setQuantity] = useState('')
  return (
    <SheetShell title="Move stock" error={error} onCancel={onCancel}>
      <div>
        <label htmlFor="move-to-warehouse" className="block text-xs text-gray-400 mb-1.5">To warehouse</label>
        <select
          id="move-to-warehouse"
          value={toWarehouseId}
          onChange={e => setToWarehouseId(e.target.value)}
          className="w-full px-3 py-2.5 rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500"
        >
          {destinations.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
        </select>
      </div>
      <QuantityStepper id="move-quantity" label="Quantity to move" value={quantity} onChange={setQuantity} />
      <button
        onClick={() => onConfirm(toWarehouseId, Number(quantity) || 0)}
        disabled={!toWarehouseId || !quantity || Number(quantity) <= 0}
        className="w-full py-3 rounded-xl bg-blue-600 text-white font-semibold disabled:opacity-50"
      >
        Confirm move
      </button>
    </SheetShell>
  )
}

function DamagedLostSheet({ error, onCancel, onConfirm }: {
  error: string
  onCancel: () => void
  onConfirm: (quantity: number, type: 'damaged' | 'lost', reason: string) => void
}) {
  const [quantity, setQuantity] = useState('')
  const [type, setType] = useState<'damaged' | 'lost'>('damaged')
  const [reason, setReason] = useState('')
  return (
    <SheetShell title="Damaged / Lost" error={error} onCancel={onCancel}>
      <div className="flex gap-2">
        {(['damaged', 'lost'] as const).map(t => (
          <button
            key={t}
            type="button"
            onClick={() => setType(t)}
            className={`flex-1 py-2.5 rounded-xl text-sm font-medium capitalize ${type === t ? 'bg-red-600 text-white' : 'bg-[#1F2937] text-gray-400'}`}
          >
            {t}
          </button>
        ))}
      </div>
      <QuantityStepper id="damaged-lost-quantity" label="Quantity" value={quantity} onChange={setQuantity} />
      <div>
        <label htmlFor="damaged-lost-reason" className="block text-xs text-gray-400 mb-1.5">Reason</label>
        <input id="damaged-lost-reason" value={reason} onChange={e => setReason(e.target.value)} className="w-full px-3 py-2.5 rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500" />
      </div>
      <button
        onClick={() => onConfirm(Number(quantity) || 0, type, reason)}
        disabled={!quantity || Number(quantity) <= 0}
        className="w-full py-3 rounded-xl bg-red-600 text-white font-semibold disabled:opacity-50"
      >
        Confirm
      </button>
    </SheetShell>
  )
}
