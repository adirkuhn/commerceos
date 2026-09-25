import { useState } from 'react'
import { ArrowLeft, Package, Tag, Trash2 } from 'lucide-react'
import * as pmsApi from '../pms/api'
import { useSKUStock } from '../inventory/hooks/useSKUStock'
import { useWarehouses } from '../inventory/hooks/useWarehouses'
import type { SKU } from '../pms/types'
import { formatPrice, parsePriceToMinorUnits } from './InventoryFormat'
import { RowActionSheet } from './InventoryRowActionSheet'
import { AddBatchSheet, PriceSheet } from './InventoryBatchSheets'

// A tapped row -- either a real batch or a warehouse's general ledger --
// scopes the action sheet to that specific batch/warehouse, instead of
// always acting on the SKU's "primary" one.
export interface SelectedRow {
  warehouseId: string
  batchId: string | null // null = general ledger
  label: string
}

export function InventorySKUScreen({ token, sku, productName, onSKUUpdated, onBack }: { token: string; sku: SKU; productName: string; onSKUUpdated: (sku: SKU) => void; onBack: () => void }) {
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
    <div className="fixed inset-0 flex flex-col bg-night-950 text-gray-100">
      <div className="flex items-center gap-3 px-4 py-4 border-b border-night-700 shrink-0">
        <button aria-label="Back to scan" onClick={onBack} className="w-11 h-11 -ml-3 -my-2 flex items-center justify-center rounded-lg shrink-0" title="Back to scan">
          <ArrowLeft size={18} className="text-gray-400" />
        </button>
        <div className="flex-1 min-w-0">
          <h1 className="text-base font-semibold truncate">
            <span>{productName}</span>{' '}
            <span className="text-xs font-normal text-gray-400">- <span>{sku.sku_code}</span></span>
          </h1>
          {level && <p className="text-xs text-gray-400">Total stock: {level.total}</p>}
        </div>
        <button type="button" onClick={() => setPriceOpen(true)} className="flex items-baseline gap-1.5 text-sm font-semibold text-accent shrink-0" title="Edit price" aria-label="Price">
          <Tag size={15} className="self-center" />
          {sku.sale_price_minor_units != null ? (
            <>
              {formatPrice(sku.sale_price_minor_units, sku.currency_code)}
              <span className="text-xs font-normal text-gray-400 line-through">{formatPrice(sku.price_minor_units, sku.currency_code)}</span>
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
          className="w-full py-2.5 rounded-xl bg-night-800 border border-night-700 text-gray-300 text-xs font-medium flex items-center justify-center gap-1.5"
        >
          + Add batch
        </button>
        {loading && <p className="text-center text-sm text-gray-400 py-8">Loading…</p>}
        {!loading && batches.length === 0 && (
          <p className="text-center text-sm text-gray-400 py-8">No batches yet</p>
        )}
        {batches.map(b => (
          <div
            key={b.is_general_ledger ? `general-${b.warehouse_id}` : b.id}
            className="w-full flex items-center gap-3 p-3.5 rounded-xl bg-night-800 border border-night-700 hover:border-accent transition-colors"
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
              <div className="w-9 h-9 rounded-lg bg-night-700 flex items-center justify-center shrink-0">
                <Package size={16} className="text-gray-400" />
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium truncate">{b.is_general_ledger ? 'General ledger' : (b.lot_number || '—')}</p>
                <p className="text-xs text-gray-400">{warehouseName(b.warehouse_id)}</p>
              </div>
              <p className="text-sm font-mono shrink-0">{b.quantity}</p>
            </button>
            {!b.is_general_ledger && (
              <button
                type="button"
                onClick={() => handleDeleteBatch(b.id)}
                aria-label={`Delete batch ${b.lot_number}`}
                className="shrink-0 w-11 h-11 -m-2 flex items-center justify-center rounded-lg text-gray-400 hover:text-red-400"
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
