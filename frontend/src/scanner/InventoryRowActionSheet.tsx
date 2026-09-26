import { useState } from 'react'
import { ArrowRightLeft, PackageMinus, Plus, X, Check } from 'lucide-react'
import { QuantityStepper } from './QuantityStepper'
import type { Batch, Warehouse } from '../inventory/types'
import { Dialog } from '../components/common/Dialog'
import { SelectedRow } from './InventorySKUScreen'
import type { DateInput } from '../format/date'
import { useDateFormat } from '../format/DateFormatProvider'

type RowActionMode = 'adjust' | 'move' | 'damaged-lost'

const ROW_ACTION_TABS: { id: RowActionMode; label: string; icon: typeof Plus; help: string }[] = [
  { id: 'adjust', label: 'Adjust', icon: Plus, help: 'Correct a count -- add or remove stock directly on this row.' },
  { id: 'move', label: 'Move', icon: ArrowRightLeft, help: 'Move stock from this row to another warehouse or batch.' },
  { id: 'damaged-lost', label: 'Damaged/lost', icon: PackageMinus, help: "It's not sellable anymore -- written off from this row." },
]

const NEW_BATCH = 'new'

function BatchOption({ selected, onClick, title, subtitle }: { selected: boolean; onClick: () => void; title: string; subtitle: string }) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={selected}
      className={`w-full flex items-center gap-2 p-2.5 rounded-lg border text-left transition-colors ${
        selected ? 'border-accent bg-accent/10' : 'border-night-700 bg-night-800'
      }`}
    >
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium truncate">{title}</p>
        <p className="text-xs text-gray-400 truncate">{subtitle}</p>
      </div>
      {selected && <Check size={14} className="text-accent shrink-0" />}
    </button>
  )
}

function formatBatchExpiry(iso: string | null | undefined, formatDate: (value: DateInput) => string): string {
  if (!iso) return 'No expiry set'
  return `Expires ${formatDate(iso)}`
}

// One sheet, three tabs -- same shape as BookingScreen's BookLineSheet
// (Receive/Remove/Damaged-Lost) for a consistent feel across every scanner
// mini-app. Scoped to the row that was tapped: Adjust/Damaged-Lost always
// act on that row's own warehouse+batch; Move starts from it and lets the
// destination be any warehouse, an existing batch, general ledger, or a
// brand-new batch (created first, then transferred into -- TransferStock
// only ever targets an already-real batch, same as the desktop Inventory
// page's Transfer modal).
export function RowActionSheet({ row, warehouses, batches, error, onCancel, onAdjust, onMove, onCreateBatchAndMove, onDamagedLost }: {
  row: SelectedRow
  warehouses: Warehouse[]
  batches: Batch[]
  error: string
  onCancel: () => void
  onAdjust: (quantityDelta: number, reason: string) => void
  onMove: (toWarehouseId: string, toBatchId: string | null, quantity: number, reason: string) => void
  onCreateBatchAndMove: (toWarehouseId: string, lotNumber: string, expiryDate: string, quantity: number, reason: string) => void
  onDamagedLost: (quantity: number, type: 'damaged' | 'lost', reason: string) => void
}) {
  const { formatDate } = useDateFormat()
  const [mode, setMode] = useState<RowActionMode>('adjust')
  const [adjustQty, setAdjustQty] = useState('')
  const [adjustReason, setAdjustReason] = useState('')
  const [toWarehouseId, setToWarehouseId] = useState(warehouses.find(w => w.id !== row.warehouseId)?.id ?? warehouses[0]?.id ?? '')
  const [toBatchId, setToBatchId] = useState('')
  const [newLotNumber, setNewLotNumber] = useState('')
  const [newExpiryDate, setNewExpiryDate] = useState('')
  const [moveQty, setMoveQty] = useState('')
  const [moveReason, setMoveReason] = useState('')
  const [dlQty, setDlQty] = useState('')
  const [dlType, setDlType] = useState<'damaged' | 'lost'>('damaged')
  const [dlReason, setDlReason] = useState('')

  const destinationBatches = batches.filter(b => !b.is_general_ledger && b.id !== row.batchId)
  const availableQty = batches.find(b =>
    b.warehouse_id === row.warehouseId && (row.batchId === null ? b.is_general_ledger : b.id === row.batchId))?.quantity ?? 0

  function handleConfirm() {
    if (mode === 'adjust') {
      onAdjust(Number(adjustQty) || 0, adjustReason)
      return
    }
    if (mode === 'move') {
      const qty = Number(moveQty) || 0
      if (toBatchId === NEW_BATCH) {
        onCreateBatchAndMove(toWarehouseId, newLotNumber, newExpiryDate, qty, moveReason)
      } else {
        onMove(toWarehouseId, toBatchId || null, qty, moveReason)
      }
      return
    }
    onDamagedLost(Number(dlQty) || 0, dlType, dlReason)
  }

  const adjustDisabled = (Number(adjustQty) || 0) === 0
  const moveDisabled = (Number(moveQty) || 0) <= 0 || !toWarehouseId || (toBatchId === NEW_BATCH && !newLotNumber.trim())
  const dlDisabled = (Number(dlQty) || 0) <= 0
  const confirmDisabled = mode === 'adjust' ? adjustDisabled : mode === 'move' ? moveDisabled : dlDisabled
  const activeTab = ROW_ACTION_TABS.find(t => t.id === mode)!

  return (
    <Dialog onClose={onCancel} className="fixed inset-0 z-50 flex items-end justify-center bg-black/40">
      <div className="w-full max-w-sm rounded-t-2xl bg-night-900 border-t border-night-700 max-h-[85vh] overflow-y-auto">
        <div className="flex items-start justify-between gap-3 px-5 pt-5">
          <div className="min-w-0">
            <h2 className="text-sm font-semibold truncate">{row.label}</h2>
            <p className="text-xs text-gray-400 mt-0.5">{warehouses.find(w => w.id === row.warehouseId)?.name ?? row.warehouseId}</p>
          </div>
          <button aria-label="Close" onClick={onCancel} title="Close" className="shrink-0 w-11 h-11 -mt-3 -mr-3 flex items-center justify-center rounded-lg text-gray-400 hover:text-gray-300">
            <X size={18} />
          </button>
        </div>

        <div className="p-5 space-y-4">
          <div className="flex gap-2">
            {ROW_ACTION_TABS.map(t => (
              <button aria-pressed={mode === t.id}
                key={t.id}
                type="button"
                onClick={() => setMode(t.id)}
                className={`flex-1 flex flex-col items-center gap-1 py-2.5 rounded-lg text-2xs font-medium leading-tight ${
                  mode === t.id ? 'bg-accent-solid text-white' : 'bg-night-800 text-gray-400'
                }`}
              >
                <t.icon size={15} />
                {t.label}
              </button>
            ))}
          </div>
          <p className="text-xs text-gray-400 -mt-2">{activeTab.help}</p>
          {error && <p className="text-sm text-red-400">{error}</p>}

          {mode === 'adjust' && (
            <>
              <QuantityStepper id="row-adjust-qty" label="Quantity change" value={adjustQty} onChange={setAdjustQty} />
              <div>
                <label htmlFor="row-adjust-reason" className="block text-xs text-gray-400 mb-1.5">Reason</label>
                <input id="row-adjust-reason" value={adjustReason} onChange={e => setAdjustReason(e.target.value)} className="w-full px-3 py-2.5 rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus" />
              </div>
            </>
          )}

          {mode === 'move' && (
            <>
              <div>
                <label htmlFor="row-move-to-warehouse" className="block text-xs text-gray-400 mb-1.5">To warehouse</label>
                <select
                  id="row-move-to-warehouse"
                  value={toWarehouseId}
                  onChange={e => setToWarehouseId(e.target.value)}
                  className="w-full px-3 py-2.5 rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus"
                >
                  {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
                </select>
              </div>
              <div>
                <p className="text-xs text-gray-400 mb-1.5">To batch</p>
                <div className="space-y-1.5 max-h-40 overflow-y-auto">
                  <BatchOption selected={toBatchId === ''} onClick={() => setToBatchId('')} title="General ledger" subtitle="No specific batch" />
                  {destinationBatches.map(b => (
                    <BatchOption
                      key={b.id}
                      selected={toBatchId === b.id}
                      onClick={() => setToBatchId(b.id)}
                      title={b.lot_number || `Batch ${b.id.slice(0, 8)}`}
                      subtitle={`${formatBatchExpiry(b.expiry_date, formatDate)} · Qty ${b.quantity}`}
                    />
                  ))}
                  <BatchOption selected={toBatchId === NEW_BATCH} onClick={() => setToBatchId(NEW_BATCH)} title="+ New batch" subtitle="Start a new lot" />
                </div>
              </div>
              {toBatchId === NEW_BATCH && (
                <div className="grid grid-cols-2 gap-2">
                  <div>
                    <label htmlFor="row-move-new-lot" className="block text-xs text-gray-400 mb-1.5">Lot number</label>
                    <input id="row-move-new-lot" value={newLotNumber} onChange={e => setNewLotNumber(e.target.value)} className="w-full px-3 py-2 text-sm rounded-lg bg-night-800 border border-night-700 text-gray-100 outline-none focus-visible:ring-2 focus-visible:ring-focus/60" />
                  </div>
                  <div>
                    <label htmlFor="row-move-new-expiry" className="block text-xs text-gray-400 mb-1.5">Expiry date</label>
                    <input id="row-move-new-expiry" type="date" value={newExpiryDate} onChange={e => setNewExpiryDate(e.target.value)} className="w-full px-3 py-2 text-sm rounded-lg bg-night-800 border border-night-700 text-gray-100 outline-none focus-visible:ring-2 focus-visible:ring-focus/60" />
                  </div>
                </div>
              )}
              <QuantityStepper id="row-move-qty" label="Quantity to move" value={moveQty} onChange={setMoveQty} min={0} max={availableQty} />
              <div>
                <label htmlFor="row-move-reason" className="block text-xs text-gray-400 mb-1.5">Reason (optional)</label>
                <input id="row-move-reason" value={moveReason} onChange={e => setMoveReason(e.target.value)} className="w-full px-3 py-2.5 rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus" />
              </div>
            </>
          )}

          {mode === 'damaged-lost' && (
            <>
              <div className="flex gap-2">
                {(['damaged', 'lost'] as const).map(t => (
                  <button aria-pressed={dlType === t}
                    key={t}
                    type="button"
                    onClick={() => setDlType(t)}
                    className={`flex-1 py-2.5 rounded-xl text-sm font-medium capitalize ${dlType === t ? 'bg-red-600 text-white' : 'bg-night-800 text-gray-400'}`}
                  >
                    {t}
                  </button>
                ))}
              </div>
              <QuantityStepper id="row-dl-qty" label="Quantity" value={dlQty} onChange={setDlQty} min={0} max={availableQty} />
              <div>
                <label htmlFor="row-dl-reason" className="block text-xs text-gray-400 mb-1.5">Reason</label>
                <input id="row-dl-reason" value={dlReason} onChange={e => setDlReason(e.target.value)} className="w-full px-3 py-2.5 rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus" />
              </div>
            </>
          )}

          <button
            onClick={handleConfirm}
            disabled={confirmDisabled}
            className={`w-full py-3 rounded-xl text-white font-semibold disabled:opacity-50 ${mode === 'damaged-lost' ? 'bg-red-600' : 'bg-accent-solid'}`}
          >
            Confirm
          </button>
        </div>
      </div>
    </Dialog>
  )
}
