import { useState } from 'react'
import { ArrowLeft, Boxes, Package, ArrowRightLeft, PackageMinus, Plus, Minus, Tag, X, Check } from 'lucide-react'
import * as pmsApi from '../pms/api'
import { useSKUStock } from '../inventory/hooks/useSKUStock'
import { useWarehouses } from '../inventory/hooks/useWarehouses'
import { ScanCodeScreen } from './ScanCodeScreen'
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
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  async function handleSubmit(code: string) {
    setError('')
    setLoading(true)
    try {
      const { sku, product_name } = await pmsApi.resolveBarcode(token, code)
      onResolved({ sku, productName: product_name })
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

// A tapped row -- either a real batch or a warehouse's general ledger --
// scopes the action sheet to that specific batch/warehouse, instead of
// always acting on the SKU's "primary" one.
interface SelectedRow {
  warehouseId: string
  batchId: string | null // null = general ledger
  label: string
}

function InventorySKUScreen({ token, sku, productName, onSKUUpdated, onBack }: { token: string; sku: SKU; productName: string; onSKUUpdated: (sku: SKU) => void; onBack: () => void }) {
  const { level, batches, loading, error, addBatch, adjust, transfer, markDamagedOrLost } = useSKUStock(token, sku.id)
  const { warehouses } = useWarehouses(token)
  const [selectedRow, setSelectedRow] = useState<SelectedRow | null>(null)
  const [priceOpen, setPriceOpen] = useState(false)
  const [actionError, setActionError] = useState('')
  const [priceError, setPriceError] = useState('')

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
  async function handleSavePrice(price: string, salePrice: string, saleEndDate: string) {
    setPriceError('')
    try {
      const updated = await pmsApi.updateSKU(token, sku.id, {
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

      <div className="flex-1 overflow-y-auto px-4 py-3 space-y-2">
        {loading && <p className="text-center text-sm text-gray-500 py-8">Loading…</p>}
        {!loading && batches.length === 0 && (
          <p className="text-center text-sm text-gray-500 py-8">No batches yet</p>
        )}
        {batches.map(b => (
          <button
            key={b.is_general_ledger ? `general-${b.warehouse_id}` : b.id}
            type="button"
            onClick={() => setSelectedRow({
              warehouseId: b.warehouse_id,
              batchId: b.is_general_ledger ? null : b.id,
              label: b.is_general_ledger ? 'General ledger' : (b.lot_number || 'Batch'),
            })}
            className="w-full flex items-center gap-3 p-3.5 rounded-xl bg-[#1F2937] border border-[#2D3748] text-left hover:border-blue-500 transition-colors"
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

function QuantityStepper({ id, label, value, onChange }: { id: string; label: string; value: string; onChange: (v: string) => void }) {
  const n = Number(value) || 0
  return (
    <div>
      <label htmlFor={id} className="block text-xs text-gray-400 mb-1.5">{label}</label>
      <div className="flex items-center gap-2">
        <button type="button" onClick={() => onChange(String(n - 1))} className="w-9 h-9 shrink-0 rounded-xl bg-[#1F2937] flex items-center justify-center" aria-label={`Decrease ${label}`}>
          <Minus size={14} />
        </button>
        <input
          id={id}
          inputMode="numeric"
          value={value}
          onChange={e => onChange(e.target.value.replace(/(?!^-)[^\d]/g, ''))}
          className="flex-1 min-w-0 px-2 py-2 text-center text-base rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500"
        />
        <button type="button" onClick={() => onChange(String(n + 1))} className="w-9 h-9 shrink-0 rounded-xl bg-[#1F2937] flex items-center justify-center" aria-label={`Increase ${label}`}>
          <Plus size={14} />
        </button>
      </div>
    </div>
  )
}

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
              <QuantityStepper id="row-move-qty" label="Quantity to move" value={moveQty} onChange={setMoveQty} />
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
              <QuantityStepper id="row-dl-qty" label="Quantity" value={dlQty} onChange={setDlQty} />
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
