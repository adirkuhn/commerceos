import { useState } from 'react'
import { ArrowLeft, Boxes, Package, ArrowRightLeft, PackageMinus, Plus, Tag, X, Check, Trash2 } from 'lucide-react'
import * as pmsApi from '../pms/api'
import { useSKUStock } from '../inventory/hooks/useSKUStock'
import { useWarehouses } from '../inventory/hooks/useWarehouses'
import { searchSKUCandidates, type SKUCandidate } from '../pms/searchCandidates'
import { ScanCodeScreen } from './ScanCodeScreen'
import { ProductCandidateRow } from './ProductCandidateRow'
import { QuantityStepper } from './QuantityStepper'
import type { SKU } from '../pms/types'
import type { Batch, Warehouse } from '../inventory/types'

export interface Resolved {
  sku: SKU
  productName: string
}

interface InventoryAppProps {
  token: string
  onExit: () => void
  // Skips straight to the SKU screen for an already-resolved SKU (e.g.
  // jumping here from Price Lookup's "View in Inventory") instead of
  // showing the scan screen first.
  initialResolved?: Resolved
}

function parsePriceToMinorUnits(price: string): number {
  return Math.round(parseFloat(price || '0') * 100)
}
function formatMinorUnitsAsInput(minorUnits: number): string {
  return (minorUnits / 100).toFixed(2)
}
function formatPrice(minorUnits: number, currency: string): string {
  return `${currency} ${(minorUnits / 100).toFixed(2)}`
}

// Scanner mini-app: scan/type a code to resolve a SKU, then adjust, move, or
// write off its stock -- every action also works by tapping alone (numeric
// keypad inputs + steppers + confirm buttons), not scan-only.
export function InventoryApp({ token, onExit, initialResolved }: InventoryAppProps) {
  const [resolved, setResolved] = useState<Resolved | null>(initialResolved ?? null)

  if (!resolved) {
    return <InventoryScanScreen token={token} onResolved={setResolved} onExit={onExit} />
  }

  return (
    <InventorySKUScreen
      token={token}
      sku={resolved.sku}
      productName={resolved.productName}
      onSKUUpdated={updatedSKU => setResolved(r => r && { ...r, sku: updatedSKU })}
      onBack={() => setResolved(null)}
    />
  )
}

function InventoryScanScreen({ token, onResolved, onExit }: { token: string; onResolved: (r: Resolved) => void; onExit: () => void }) {
  const [tab, setTab] = useState<'code' | 'name'>('code')
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
    <div className="fixed inset-0 flex flex-col bg-[#0B1220] text-gray-100">
      <div className="flex items-center gap-3 px-4 py-4 border-b border-[#2D3748] shrink-0">
        <button onClick={onExit} className="p-1 -ml-1" title="Back to apps">
          <ArrowLeft size={18} className="text-gray-400" />
        </button>
        <h1 className="flex-1 text-base font-semibold">Inventory</h1>
      </div>
      <div className="flex gap-2 px-4 pt-4 shrink-0">
        <button
          type="button"
          onClick={() => setTab('code')}
          className={`flex-1 py-2 rounded-lg text-xs font-medium ${tab === 'code' ? 'bg-blue-600 text-white' : 'bg-[#1F2937] text-gray-400'}`}
        >
          Scan code
        </button>
        <button
          type="button"
          onClick={() => setTab('name')}
          className={`flex-1 py-2 rounded-lg text-xs font-medium ${tab === 'name' ? 'bg-blue-600 text-white' : 'bg-[#1F2937] text-gray-400'}`}
        >
          Search by name
        </button>
      </div>
      {tab === 'code' ? (
        <div className="flex-1 flex flex-col overflow-y-auto">
          <ScanCodeScreen icon={Boxes} prompt="Scan a barcode or type a SKU code" loading={loading} error={error} onSubmit={handleSubmit} />
          {candidates.length > 0 && (
            <div className="w-full max-w-xs mx-auto px-6 pb-6 space-y-1.5 shrink-0">
              <p className="text-xs text-gray-500">Multiple matches -- pick one:</p>
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
          className="w-full px-4 py-4 text-center text-sm rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500"
        />
        {error && <p className="text-sm text-red-400">{error}</p>}
        <button
          type="submit"
          disabled={loading || name.trim().length === 0}
          className="w-full py-3 rounded-xl bg-blue-600 text-white font-semibold hover:bg-blue-700 disabled:opacity-50 transition-colors"
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

// A tapped row -- either a real batch or a warehouse's general ledger --
// scopes the action sheet to that specific batch/warehouse, instead of
// always acting on the SKU's "primary" one.
interface SelectedRow {
  warehouseId: string
  batchId: string | null // null = general ledger
  label: string
}

function InventorySKUScreen({ token, sku, productName, onSKUUpdated, onBack }: { token: string; sku: SKU; productName: string; onSKUUpdated: (sku: SKU) => void; onBack: () => void }) {
  const { level, batches, loading, error, addBatch, removeBatch, adjust, transfer, markDamagedOrLost } = useSKUStock(token, sku.id)
  const { warehouses } = useWarehouses(token)
  const [selectedRow, setSelectedRow] = useState<SelectedRow | null>(null)
  const [priceOpen, setPriceOpen] = useState(false)
  const [addBatchOpen, setAddBatchOpen] = useState(false)
  const [actionError, setActionError] = useState('')
  const [priceError, setPriceError] = useState('')
  const [addBatchError, setAddBatchError] = useState('')
  const [deleteError, setDeleteError] = useState('')

  const warehouseName = (id: string) => warehouses.find(w => w.id === id)?.name ?? id

  async function runAction(fn: () => Promise<unknown>) {
    setActionError('')
    try {
      await fn()
      setSelectedRow(null)
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Action failed')
    }
  }

  // Full round-trip of every SKU field, not just price -- UpdateSKU is a
  // full replace, not a patch, so omitting a field (tracks_expiry,
  // case_of_sku_id, unit_type, ...) would silently wipe it.
  async function handleDeleteBatch(batchId: string) {
    setDeleteError('')
    try {
      await removeBatch(batchId)
    } catch (err) {
      setDeleteError(err instanceof Error ? err.message : 'Failed to delete batch')
    }
  }

  async function handleAddBatch(warehouseId: string, lotNumber: string, expiryDate: string, quantity: number, reason: string) {
    setAddBatchError('')
    try {
      const created = await addBatch({ warehouse_id: warehouseId, lot_number: lotNumber, expiry_date: expiryDate || null })
      if (quantity > 0) {
        await adjust({ warehouse_id: warehouseId, batch_id: created.id, type: 'receipt', quantity_delta: quantity, reason })
      }
      setAddBatchOpen(false)
    } catch (err) {
      setAddBatchError(err instanceof Error ? err.message : 'Failed to add batch')
    }
  }

  async function handleSavePrice(price: string, salePrice: string, saleEndDate: string) {
    setPriceError('')
    try {
      const updated = await pmsApi.updateSKU(token, sku.id, {
        sku_code: sku.sku_code,
        barcode: sku.barcode ?? undefined,
        price_minor_units: parsePriceToMinorUnits(price),
        sale_price_minor_units: salePrice.trim() === '' ? null : parsePriceToMinorUnits(salePrice),
        sale_end_date: saleEndDate || null,
        currency_code: sku.currency_code,
        weight_grams: sku.weight_grams,
        length_mm: sku.length_mm,
        width_mm: sku.width_mm,
        height_mm: sku.height_mm,
        is_virtual: sku.is_virtual,
        tracks_expiry: sku.tracks_expiry,
        case_of_sku_id: sku.case_of_sku_id,
        units_per_case: sku.units_per_case,
        unit_type: sku.unit_type,
        portioned_from_sku_id: sku.portioned_from_sku_id,
        status: sku.status,
      })
      onSKUUpdated(updated)
      setPriceOpen(false)
    } catch (err) {
      setPriceError(err instanceof Error ? err.message : 'Failed to update price')
    }
  }

  return (
    <div className="fixed inset-0 flex flex-col bg-[#0B1220] text-gray-100">
      <div className="flex items-center gap-3 px-4 py-4 border-b border-[#2D3748] shrink-0">
        <button onClick={onBack} className="p-1 -ml-1" title="Back to scan">
          <ArrowLeft size={18} className="text-gray-400" />
        </button>
        <div className="flex-1 min-w-0">
          <h1 className="text-base font-semibold truncate">
            <span>{productName}</span>{' '}
            <span className="text-xs font-normal text-gray-500">- <span>{sku.sku_code}</span></span>
          </h1>
          {level && <p className="text-xs text-gray-500">Total stock: {level.total}</p>}
        </div>
        <button type="button" onClick={() => setPriceOpen(true)} className="flex items-baseline gap-1.5 text-sm font-semibold text-blue-400 shrink-0" title="Edit price" aria-label="Price">
          <Tag size={15} className="self-center" />
          {sku.sale_price_minor_units != null ? (
            <>
              {formatPrice(sku.sale_price_minor_units, sku.currency_code)}
              <span className="text-xs font-normal text-gray-500 line-through">{formatPrice(sku.price_minor_units, sku.currency_code)}</span>
            </>
          ) : (
            formatPrice(sku.price_minor_units, sku.currency_code)
          )}
        </button>
      </div>

      {error && <p className="mx-4 mt-3 text-sm text-red-400">{error}</p>}
      {deleteError && <p className="mx-4 mt-3 text-sm text-red-400">{deleteError}</p>}

      <div className="flex-1 overflow-y-auto px-4 py-3 space-y-2">
        <button
          type="button"
          onClick={() => setAddBatchOpen(true)}
          className="w-full py-2.5 rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-300 text-xs font-medium flex items-center justify-center gap-1.5"
        >
          + Add batch
        </button>
        {loading && <p className="text-center text-sm text-gray-500 py-8">Loading…</p>}
        {!loading && batches.length === 0 && (
          <p className="text-center text-sm text-gray-500 py-8">No batches yet</p>
        )}
        {batches.map(b => (
          <div
            key={b.is_general_ledger ? `general-${b.warehouse_id}` : b.id}
            className="w-full flex items-center gap-3 p-3.5 rounded-xl bg-[#1F2937] border border-[#2D3748] hover:border-blue-500 transition-colors"
          >
            <button
              type="button"
              onClick={() => setSelectedRow({
                warehouseId: b.warehouse_id,
                batchId: b.is_general_ledger ? null : b.id,
                label: b.is_general_ledger ? 'General ledger' : (b.lot_number || 'Batch'),
              })}
              className="flex-1 flex items-center gap-3 min-w-0 text-left"
            >
              <div className="w-9 h-9 rounded-lg bg-[#2D3748] flex items-center justify-center shrink-0">
                <Package size={16} className="text-gray-400" />
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium truncate">{b.is_general_ledger ? 'General ledger' : (b.lot_number || '—')}</p>
                <p className="text-xs text-gray-500">{warehouseName(b.warehouse_id)}</p>
              </div>
              <p className="text-sm font-mono shrink-0">{b.quantity}</p>
            </button>
            {!b.is_general_ledger && (
              <button
                type="button"
                onClick={() => handleDeleteBatch(b.id)}
                aria-label={`Delete batch ${b.lot_number}`}
                className="shrink-0 p-1.5 text-gray-500 hover:text-red-400"
              >
                <Trash2 size={15} />
              </button>
            )}
          </div>
        ))}
      </div>

      {selectedRow && (
        <RowActionSheet
          row={selectedRow}
          warehouses={warehouses}
          batches={batches}
          error={actionError}
          onCancel={() => setSelectedRow(null)}
          onAdjust={(quantityDelta, reason) => runAction(() => adjust({
            warehouse_id: selectedRow.warehouseId, batch_id: selectedRow.batchId, type: 'adjustment', quantity_delta: quantityDelta, reason,
          }))}
          onMove={(toWarehouseId, toBatchId, quantity, reason) => runAction(() => transfer({
            from_warehouse_id: selectedRow.warehouseId, from_batch_id: selectedRow.batchId,
            to_warehouse_id: toWarehouseId, to_batch_id: toBatchId, quantity, reason,
          }))}
          onCreateBatchAndMove={(toWarehouseId, lotNumber, expiryDate, quantity, reason) => runAction(async () => {
            const created = await addBatch({ warehouse_id: toWarehouseId, lot_number: lotNumber, expiry_date: expiryDate || null })
            return transfer({
              from_warehouse_id: selectedRow.warehouseId, from_batch_id: selectedRow.batchId,
              to_warehouse_id: toWarehouseId, to_batch_id: created.id, quantity, reason,
            })
          })}
          onDamagedLost={(quantity, type, reason) => runAction(() => markDamagedOrLost({
            warehouse_id: selectedRow.warehouseId, batch_id: selectedRow.batchId, quantity, type, reason,
          }))}
        />
      )}

      {priceOpen && (
        <PriceSheet sku={sku} error={priceError} onCancel={() => setPriceOpen(false)} onSave={handleSavePrice} />
      )}

      {addBatchOpen && (
        <AddBatchSheet
          warehouses={warehouses}
          error={addBatchError}
          onCancel={() => setAddBatchOpen(false)}
          onCreate={handleAddBatch}
        />
      )}
    </div>
  )
}

type RowActionMode = 'adjust' | 'move' | 'damaged-lost'

const ROW_ACTION_TABS: { id: RowActionMode; label: string; icon: typeof Plus; help: string }[] = [
  { id: 'adjust', label: 'Adjust', icon: Plus, help: 'Correct a count -- add or remove stock directly on this row.' },
  { id: 'move', label: 'Move', icon: ArrowRightLeft, help: 'Move stock from this row to another warehouse or batch.' },
  { id: 'damaged-lost', label: 'Damaged/Lost', icon: PackageMinus, help: "It's not sellable anymore -- written off from this row." },
]

const NEW_BATCH = 'new'

function BatchOption({ selected, onClick, title, subtitle }: { selected: boolean; onClick: () => void; title: string; subtitle: string }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`w-full flex items-center gap-2 p-2.5 rounded-lg border text-left transition-colors ${
        selected ? 'border-blue-500 bg-blue-500/10' : 'border-[#2D3748] bg-[#1F2937]'
      }`}
    >
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium truncate">{title}</p>
        <p className="text-xs text-gray-500 truncate">{subtitle}</p>
      </div>
      {selected && <Check size={14} className="text-blue-400 shrink-0" />}
    </button>
  )
}

function formatBatchExpiry(iso?: string | null): string {
  if (!iso) return 'No expiry set'
  return `Expires ${new Date(iso).toLocaleDateString()}`
}

// One sheet, three tabs -- same shape as BookingScreen's BookLineSheet
// (Receive/Remove/Damaged-Lost) for a consistent feel across every scanner
// mini-app. Scoped to the row that was tapped: Adjust/Damaged-Lost always
// act on that row's own warehouse+batch; Move starts from it and lets the
// destination be any warehouse, an existing batch, general ledger, or a
// brand-new batch (created first, then transferred into -- TransferStock
// only ever targets an already-real batch, same as the desktop Inventory
// page's Transfer modal).
function RowActionSheet({ row, warehouses, batches, error, onCancel, onAdjust, onMove, onCreateBatchAndMove, onDamagedLost }: {
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
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-black/40">
      <div className="w-full max-w-sm rounded-t-2xl bg-[#111827] border-t border-[#2D3748] max-h-[85vh] overflow-y-auto">
        <div className="flex items-start justify-between gap-3 px-5 pt-5">
          <div className="min-w-0">
            <h2 className="text-sm font-semibold truncate">{row.label}</h2>
            <p className="text-xs text-gray-500 mt-0.5">{warehouses.find(w => w.id === row.warehouseId)?.name ?? row.warehouseId}</p>
          </div>
          <button onClick={onCancel} title="Close" className="shrink-0 p-1 -mt-1 -mr-1 text-gray-500 hover:text-gray-300">
            <X size={18} />
          </button>
        </div>

        <div className="p-5 space-y-4">
          <div className="flex gap-2">
            {ROW_ACTION_TABS.map(t => (
              <button
                key={t.id}
                type="button"
                onClick={() => setMode(t.id)}
                className={`flex-1 flex flex-col items-center gap-1 py-2.5 rounded-lg text-[11px] font-medium leading-tight ${
                  mode === t.id ? 'bg-blue-600 text-white' : 'bg-[#1F2937] text-gray-400'
                }`}
              >
                <t.icon size={15} />
                {t.label}
              </button>
            ))}
          </div>
          <p className="text-xs text-gray-500 -mt-2">{activeTab.help}</p>
          {error && <p className="text-sm text-red-400">{error}</p>}

          {mode === 'adjust' && (
            <>
              <QuantityStepper id="row-adjust-qty" label="Quantity change" value={adjustQty} onChange={setAdjustQty} />
              <div>
                <label htmlFor="row-adjust-reason" className="block text-xs text-gray-400 mb-1.5">Reason</label>
                <input id="row-adjust-reason" value={adjustReason} onChange={e => setAdjustReason(e.target.value)} className="w-full px-3 py-2.5 rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500" />
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
                  className="w-full px-3 py-2.5 rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500"
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
                      subtitle={`${formatBatchExpiry(b.expiry_date)} · Qty ${b.quantity}`}
                    />
                  ))}
                  <BatchOption selected={toBatchId === NEW_BATCH} onClick={() => setToBatchId(NEW_BATCH)} title="+ New batch" subtitle="Start a new lot" />
                </div>
              </div>
              {toBatchId === NEW_BATCH && (
                <div className="grid grid-cols-2 gap-2">
                  <div>
                    <label htmlFor="row-move-new-lot" className="block text-xs text-gray-400 mb-1.5">Lot number</label>
                    <input id="row-move-new-lot" value={newLotNumber} onChange={e => setNewLotNumber(e.target.value)} className="w-full px-3 py-2 text-sm rounded-lg bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none" />
                  </div>
                  <div>
                    <label htmlFor="row-move-new-expiry" className="block text-xs text-gray-400 mb-1.5">Expiry date</label>
                    <input id="row-move-new-expiry" type="date" value={newExpiryDate} onChange={e => setNewExpiryDate(e.target.value)} className="w-full px-3 py-2 text-sm rounded-lg bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none" />
                  </div>
                </div>
              )}
              <QuantityStepper id="row-move-qty" label="Quantity to move" value={moveQty} onChange={setMoveQty} min={0} max={availableQty} />
              <div>
                <label htmlFor="row-move-reason" className="block text-xs text-gray-400 mb-1.5">Reason (optional)</label>
                <input id="row-move-reason" value={moveReason} onChange={e => setMoveReason(e.target.value)} className="w-full px-3 py-2.5 rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500" />
              </div>
            </>
          )}

          {mode === 'damaged-lost' && (
            <>
              <div className="flex gap-2">
                {(['damaged', 'lost'] as const).map(t => (
                  <button
                    key={t}
                    type="button"
                    onClick={() => setDlType(t)}
                    className={`flex-1 py-2.5 rounded-xl text-sm font-medium capitalize ${dlType === t ? 'bg-red-600 text-white' : 'bg-[#1F2937] text-gray-400'}`}
                  >
                    {t}
                  </button>
                ))}
              </div>
              <QuantityStepper id="row-dl-qty" label="Quantity" value={dlQty} onChange={setDlQty} min={0} max={availableQty} />
              <div>
                <label htmlFor="row-dl-reason" className="block text-xs text-gray-400 mb-1.5">Reason</label>
                <input id="row-dl-reason" value={dlReason} onChange={e => setDlReason(e.target.value)} className="w-full px-3 py-2.5 rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500" />
              </div>
            </>
          )}

          <button
            onClick={handleConfirm}
            disabled={confirmDisabled}
            className={`w-full py-3 rounded-xl text-white font-semibold disabled:opacity-50 ${mode === 'damaged-lost' ? 'bg-red-600' : 'bg-blue-600'}`}
          >
            Confirm
          </button>
        </div>
      </div>
    </div>
  )
}

// Standalone batch creation, distinct from Move's inline "+ New batch": that
// one always immediately transfers stock in from a source row (there must
// be a from-row to move out of). This one is for new stock with no source
// -- e.g. a fresh delivery, or a brand-new SKU with nothing to move from
// yet -- so quantity is optional and, when given, is recorded as a receipt
// straight into the new batch rather than a transfer.
function AddBatchSheet({ warehouses, error, onCancel, onCreate }: {
  warehouses: Warehouse[]
  error: string
  onCancel: () => void
  onCreate: (warehouseId: string, lotNumber: string, expiryDate: string, quantity: number, reason: string) => void
}) {
  const [warehouseId, setWarehouseId] = useState(warehouses[0]?.id ?? '')
  const [lotNumber, setLotNumber] = useState('')
  const [expiryDate, setExpiryDate] = useState('')
  const [quantity, setQuantity] = useState('')
  const [reason, setReason] = useState('')

  const disabled = !warehouseId || !lotNumber.trim()

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-black/40">
      <div className="w-full max-w-sm rounded-t-2xl bg-[#111827] border-t border-[#2D3748] max-h-[85vh] overflow-y-auto">
        <div className="flex items-start justify-between gap-3 px-5 pt-5">
          <h2 className="text-sm font-semibold">Add batch</h2>
          <button onClick={onCancel} title="Close" className="shrink-0 p-1 -mt-1 -mr-1 text-gray-500 hover:text-gray-300">
            <X size={18} />
          </button>
        </div>
        <div className="p-5 space-y-4">
          {error && <p className="text-sm text-red-400">{error}</p>}
          <div>
            <label htmlFor="add-batch-warehouse" className="block text-xs text-gray-400 mb-1.5">Warehouse</label>
            <select
              id="add-batch-warehouse"
              value={warehouseId}
              onChange={e => setWarehouseId(e.target.value)}
              className="w-full px-3 py-2.5 rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500"
            >
              {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
            </select>
          </div>
          <div>
            <label htmlFor="add-batch-lot" className="block text-xs text-gray-400 mb-1.5">Lot number</label>
            <input id="add-batch-lot" value={lotNumber} onChange={e => setLotNumber(e.target.value)} className="w-full px-3 py-2.5 rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500" />
          </div>
          <div>
            <label htmlFor="add-batch-expiry" className="block text-xs text-gray-400 mb-1.5">Expiry date (optional)</label>
            <input id="add-batch-expiry" type="date" value={expiryDate} onChange={e => setExpiryDate(e.target.value)} className="w-full px-3 py-2.5 rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500" />
          </div>
          <QuantityStepper id="add-batch-qty" label="Initial quantity" value={quantity} onChange={setQuantity} />
          <div>
            <label htmlFor="add-batch-reason" className="block text-xs text-gray-400 mb-1.5">Reason</label>
            <input id="add-batch-reason" value={reason} onChange={e => setReason(e.target.value)} className="w-full px-3 py-2.5 rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500" />
          </div>
          <button
            onClick={() => onCreate(warehouseId, lotNumber.trim(), expiryDate, Number(quantity) || 0, reason)}
            disabled={disabled}
            className="w-full py-3 rounded-xl bg-blue-600 text-white font-semibold disabled:opacity-50"
          >
            Create
          </button>
        </div>
      </div>
    </div>
  )
}

function PriceSheet({ sku, error, onCancel, onSave }: {
  sku: SKU
  error: string
  onCancel: () => void
  onSave: (price: string, salePrice: string, saleEndDate: string) => void
}) {
  const [price, setPrice] = useState(formatMinorUnitsAsInput(sku.price_minor_units))
  const [salePrice, setSalePrice] = useState(sku.sale_price_minor_units != null ? formatMinorUnitsAsInput(sku.sale_price_minor_units) : '')
  const [saleEndDate, setSaleEndDate] = useState(sku.sale_end_date ?? '')

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-black/40">
      <div className="w-full max-w-sm rounded-t-2xl bg-[#111827] border-t border-[#2D3748] max-h-[85vh] overflow-y-auto">
        <div className="flex items-start justify-between gap-3 px-5 pt-5">
          <h2 className="text-sm font-semibold">Edit price</h2>
          <button onClick={onCancel} title="Close" className="shrink-0 p-1 -mt-1 -mr-1 text-gray-500 hover:text-gray-300">
            <X size={18} />
          </button>
        </div>
        <div className="p-5 space-y-4">
          {error && <p className="text-sm text-red-400">{error}</p>}
          <div>
            <label htmlFor="price-sheet-price" className="block text-xs text-gray-400 mb-1.5">Regular price ({sku.currency_code})</label>
            <input id="price-sheet-price" type="number" step="0.01" min="0" value={price} onChange={e => setPrice(e.target.value)} className="w-full px-3 py-2.5 rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500" />
          </div>
          <div>
            <label htmlFor="price-sheet-sale-price" className="block text-xs text-gray-400 mb-1.5">Sale price (optional)</label>
            <input
              id="price-sheet-sale-price" type="number" step="0.01" min="0" value={salePrice}
              onChange={e => { setSalePrice(e.target.value); if (e.target.value.trim() === '') setSaleEndDate('') }}
              className="w-full px-3 py-2.5 rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          <div>
            <label htmlFor="price-sheet-sale-end" className="block text-xs text-gray-400 mb-1.5">Sale ends (optional)</label>
            <input
              id="price-sheet-sale-end" type="date" value={saleEndDate} onChange={e => setSaleEndDate(e.target.value)}
              disabled={salePrice.trim() === ''}
              className="w-full px-3 py-2.5 rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
            />
          </div>
          <button
            onClick={() => onSave(price, salePrice, saleEndDate)}
            className="w-full py-3 rounded-xl bg-blue-600 text-white font-semibold"
          >
            Save
          </button>
        </div>
      </div>
    </div>
  )
}
