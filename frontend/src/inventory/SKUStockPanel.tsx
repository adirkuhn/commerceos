import { useState } from 'react'
import { Plus, ArrowRightLeft, Trash2, Edit3, PackageMinus, ClipboardList, Tag, History, Boxes, Scale } from 'lucide-react'
import clsx from 'clsx'
import { useSkuActionRoute } from './useSkuActionRoute'
import type { SkuAction } from './inventoryRoutes'
import { useQuerySet } from '../routing/useQueryState'
import { useSKUStock } from './hooks/useSKUStock'
import * as pmsApi from '../pms/api'
import type { SKU } from '../pms/types'
import type { Batch } from './types'
import { PortionFromBulkModal } from './PortionFromBulkModal'
import { AdjustStockModal, BatchFormModal, BatchScope, RecountModal } from './BatchStockModals'
import { formatSKUQuantity, isExpired, isExpiringSoon, parseOptionalPriceToMinorUnits, parsePriceToMinorUnits } from './inventoryFormat'
import { BatchRowActionModal, NewBatchModal, PriceModal } from './BatchActionModals'
import { DamagedLostModal, SplitCaseModal, TransferModal } from './TransferStockModals'
import { formatDate } from '../format/date'

export function SKUStockPanel({ token, productId, sku, variantName, warehouses, onSKUUpdated, onStockChanged, onViewHistory, onViewBatchHistory }: {
  token: string; productId: string; sku: SKU; variantName?: string; warehouses: { id: string; name: string }[]
  // onSKUUpdated refetches product/variant/SKU metadata (name, price, ...)
  // -- reserved for price edits. Stock-quantity mutations must use
  // onStockChanged instead: onSKUUpdated flips the parent's `loading` flag,
  // which unmounts/remounts this whole panel (see ProductStockRow's
  // `{!loading && skus.map(...)}`), discarding showAllBatches/modal state
  // on every adjust/recount/transfer/etc. onStockChanged only refetches the
  // collapsed row's aggregate total, no unmount.
  onSKUUpdated: () => void
  onStockChanged: () => void
  onViewHistory: (sku: SKU) => void; onViewBatchHistory: (sku: SKU, batch: Batch) => void
}) {
  const { level, batches, loading, error, addBatch, editBatch, removeBatch, adjust, recount, transfer, splitCase, portionFromBulk, markDamagedOrLost } =
    useSKUStock(token, sku.id)
  // The open modal, and the batch row it's locked to, come from the URL
  // (see useSkuActionRoute), as does "show zero-stock batches"
  // (?allBatches=) -- a refresh or Back keeps or closes them like pages.
  const route = useSkuActionRoute(productId, sku.id)
  const allBatches = useQuerySet('allBatches')
  const showAllBatches = allBatches.has(sku.id)
  const [deleteError, setDeleteError] = useState<string | null>(null)

  // Resolves ?batch= against the loaded batches (the general-ledger row
  // has no id, so it's addressed by warehouse via ?ledger=). A modal that
  // needs its row waits until the batches are in rather than opening
  // unlocked.
  const rowBatch = route.row && 'batch' in route.row ? batches.find(b => b.id === (route.row as { batch: string }).batch) ?? null : null
  const rowScope: BatchScope | null = !route.row
    ? null
    : 'ledger' in route.row
      ? { warehouseId: route.row.ledger, batchId: '', label: 'General ledger' }
      : rowBatch && { warehouseId: rowBatch.warehouse_id, batchId: rowBatch.id, label: rowBatch.lot_number || 'Batch' }
  const modal = !route.row || rowScope ? route.action : null
  const setModal = (action: SkuAction) => route.openAction(action)
  const closeModal = route.close
  const openRowAction = (b: Batch) => {
    route.openAction('choose', b.is_general_ledger ? { ledger: b.warehouse_id } : { batch: b.id })
  }

  // Old zero-stock batches (even with movement history) clutter this table
  // by default -- hidden unless explicitly shown. The general-ledger row
  // and any batch still carrying stock are always shown. Modals below still
  // get the full, unfiltered `batches` -- e.g. Transfer needs to offer a
  // zero-stock batch as a valid destination.
  const zeroStockBatchCount = batches.filter(b => !b.is_general_ledger && b.quantity === 0).length
  const visibleBatches = showAllBatches ? batches : batches.filter(b => b.is_general_ledger || b.quantity !== 0)

  const handleSavePrice = async (price: string, salePrice: string, saleEndDate: string) => {
    // Full round-trip of every SKU field, not just price -- UpdateSKU is a
    // full replace, not a patch, so omitting a field (tracks_expiry,
    // case_of_sku_id, unit_type, ...) would silently wipe it.
    await pmsApi.updateSKU(token, sku.id, {
      sku_code: sku.sku_code,
      barcode: sku.barcode ?? undefined,
      price_minor_units: parsePriceToMinorUnits(price),
      sale_price_minor_units: parseOptionalPriceToMinorUnits(salePrice),
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
    onSKUUpdated()
  }

  const handleDeleteBatch = async (id: string) => {
    setDeleteError(null)
    try {
      await removeBatch(id)
      onStockChanged()
    } catch (e) {
      setDeleteError(e instanceof Error ? e.message : 'Failed to delete batch')
    }
  }

  return (
    <div className="pl-8 pr-4 py-3 border-t border-line/60 dark:bg-night-950/40 bg-gray-50/60">
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-3">
          <span className="font-mono text-2xs text-ink-2">
            {variantName && <span className="font-sans text-ink-3">{variantName} · </span>}
            {sku.sku_code}
          </span>
          <button
            type="button"
            onClick={() => setModal('price')}
            title="Edit price"
            className="flex items-center gap-1 text-2xs text-ink-4 hover:text-accent"
          >
            <Tag size={11} /> Price
          </button>
          <button
            type="button"
            onClick={() => onViewHistory(sku)}
            title="View movement history"
            className="flex items-center gap-1 text-2xs text-ink-4 hover:text-accent"
          >
            <History size={11} /> History
          </button>
          {level && (
            <span data-testid="stock-level" className="text-2xs text-ink-3">
              Total: <strong className="text-ink">{formatSKUQuantity(sku, level.total)}</strong>
              {' · '}Syncable: <strong className="text-ink">{formatSKUQuantity(sku, level.syncable)}</strong>
            </span>
          )}
          {loading && <span className="text-2xs text-ink-4">Loading…</span>}
        </div>
        <div className="flex items-center gap-1.5">
          <button type="button" onClick={() => setModal('adjust')} className="flex items-center gap-1 px-2 py-1 rounded-md text-2xs bg-raised border border-line text-ink-2 hover:bg-hover">
            <Plus size={11} /> Adjust
          </button>
          <button type="button" onClick={() => setModal('recount')} className="flex items-center gap-1 px-2 py-1 rounded-md text-2xs bg-raised border border-line text-ink-2 hover:bg-hover">
            <ClipboardList size={11} /> Recount
          </button>
          <button type="button" onClick={() => setModal('transfer')} className="flex items-center gap-1 px-2 py-1 rounded-md text-2xs bg-raised border border-line text-ink-2 hover:bg-hover">
            <ArrowRightLeft size={11} /> Transfer
          </button>
          {sku.case_of_sku_id && (
            <button type="button" onClick={() => setModal('split-case')} title="Split sealed cases into individual units" className="flex items-center gap-1 px-2 py-1 rounded-md text-2xs bg-raised border border-line text-ink-2 hover:bg-hover">
              <Boxes size={11} /> Split case
            </button>
          )}
          {sku.portioned_from_sku_id && (
            <button type="button" onClick={() => setModal('portion')} title="Add a sellable item peeled off the linked weight pool" className="flex items-center gap-1 px-2 py-1 rounded-md text-2xs bg-raised border border-line text-ink-2 hover:bg-hover">
              <Scale size={11} /> Add item from weight pool
            </button>
          )}
          <button type="button" onClick={() => setModal('damaged-lost')} className="flex items-center gap-1 px-2 py-1 rounded-md text-2xs bg-raised border border-line text-danger hover:bg-hover">
            <PackageMinus size={11} /> Damaged/lost
          </button>
        </div>
      </div>

      {error && <p className="text-2xs text-danger mb-2">{error}</p>}
      {deleteError && <p className="text-2xs text-danger mb-2">{deleteError}</p>}

      <div className="rounded-lg border border-line overflow-hidden">
        <table className="w-full text-2xs">
          <thead className="bg-fill">
            <tr>
              {['Lot', 'Warehouse', 'Qty', 'Expiry', ''].map(h => (
                <th key={h} className="px-2.5 py-1.5 text-left font-medium text-ink-4 uppercase tracking-wider">{h}</th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-line/60">
            {visibleBatches.map(b => (
              <tr
                key={b.is_general_ledger ? `general-${b.warehouse_id}` : b.id}
                onClick={() => openRowAction(b)}
                title="Click to adjust, recount, transfer, or mark damaged/lost"
                className={clsx(
                  'cursor-pointer dark:hover:bg-night-800 hover:bg-gray-100/80',
                  b.is_general_ledger && 'dark:bg-night-900/40 bg-gray-100/60',
                )}
              >
                <td className="px-2.5 py-1.5 font-mono text-ink-2">
                  {b.is_general_ledger ? 'General ledger' : (b.lot_number || '—')}
                </td>
                <td className="px-2.5 py-1.5 text-ink-3">{warehouses.find(w => w.id === b.warehouse_id)?.name ?? b.warehouse_id}</td>
                <td className="px-2.5 py-1.5 font-mono text-ink-2">{b.quantity}</td>
                <td className="px-2.5 py-1.5">
                  {!b.is_general_ledger && (
                    <span className={clsx(
                      'font-mono',
                      isExpired(b.expiry_date) ? 'text-danger' : isExpiringSoon(b.expiry_date) ? 'text-warning' : 'text-ink-3',
                    )}>
                      {formatDate(b.expiry_date)}
                    </span>
                  )}
                </td>
                <td className="px-2.5 py-1.5" onClick={e => e.stopPropagation()}>
                  {!b.is_general_ledger && (
                    <div className="flex items-center justify-end gap-2">
                      <button type="button" onClick={() => onViewBatchHistory(sku, b)} aria-label={`View movement history for batch ${b.lot_number}`} title="View movement history" className="text-ink-4 hover:text-accent">
                        <History size={11} />
                      </button>
                      <button type="button" onClick={() => route.openAction('edit-batch', { batch: b.id })} aria-label={`Edit batch ${b.lot_number}`} className="text-ink-4 hover:text-ink-2">
                        <Edit3 size={11} />
                      </button>
                      <button type="button" onClick={() => handleDeleteBatch(b.id)} aria-label={`Delete batch ${b.lot_number}`} className="text-ink-4 hover:text-danger">
                        <Trash2 size={11} />
                      </button>
                    </div>
                  )}
                </td>
              </tr>
            ))}
            {batches.length === 0 && (
              <tr>
                <td colSpan={5} className="px-2.5 py-3 text-center text-ink-4">No batches — all stock is in the general ledger.</td>
              </tr>
            )}
            {batches.length > 0 && visibleBatches.length === 0 && (
              <tr>
                <td colSpan={5} className="px-2.5 py-3 text-center text-ink-4">
                  All {zeroStockBatchCount} batch{zeroStockBatchCount === 1 ? '' : 'es'} for this SKU {zeroStockBatchCount === 1 ? 'has' : 'have'} zero stock.
                </td>
              </tr>
            )}
          </tbody>
        </table>
        <div className="px-2.5 py-1.5 border-t border-line flex items-center gap-3">
          <button type="button" onClick={() => setModal('new-batch')} className="text-2xs text-accent hover:text-accent/80">
            + New batch
          </button>
          {zeroStockBatchCount > 0 && (
            <button type="button" onClick={() => allBatches.toggle(sku.id)} className="text-2xs text-ink-4 hover:text-accent">
              {showAllBatches ? 'Hide zero-stock batches' : `Show ${zeroStockBatchCount} zero-stock batch${zeroStockBatchCount === 1 ? '' : 'es'}`}
            </button>
          )}
        </div>
      </div>

      {modal === 'choose' && rowScope && (
        <BatchRowActionModal scope={rowScope} onChoose={route.switchAction} onClose={closeModal} />
      )}
      {modal === 'adjust' && (
        <AdjustStockModal warehouses={warehouses} batches={batches} scope={rowScope ?? undefined} onClose={closeModal}
          onSave={input => adjust(input).then(() => onStockChanged())} />
      )}
      {modal === 'recount' && (
        <RecountModal warehouses={warehouses} batches={batches} scope={rowScope ?? undefined} onClose={closeModal}
          onSave={input => recount(input).then(() => onStockChanged())} />
      )}
      {modal === 'transfer' && (
        <TransferModal warehouses={warehouses} batches={batches} scope={rowScope ?? undefined} onClose={closeModal}
          onSave={input => transfer(input).then(() => onStockChanged())} />
      )}
      {modal === 'split-case' && (
        <SplitCaseModal warehouses={warehouses} batches={batches} onClose={closeModal}
          onSave={input => splitCase(input).then(() => onStockChanged())} />
      )}
      {modal === 'portion' && sku.portioned_from_sku_id && (
        <PortionFromBulkModal
          token={token} bulkSkuId={sku.portioned_from_sku_id} warehouses={warehouses} nominalWeightGrams={sku.weight_grams}
          onClose={closeModal} onSave={input => portionFromBulk(input).then(() => onStockChanged())}
        />
      )}
      {modal === 'damaged-lost' && (
        <DamagedLostModal warehouses={warehouses} batches={batches} scope={rowScope ?? undefined} onClose={closeModal}
          onSave={input => markDamagedOrLost(input).then(() => onStockChanged())} />
      )}
      {modal === 'new-batch' && (
        <NewBatchModal warehouses={warehouses} onClose={closeModal} onCreated={route.closeShowingAllBatches}
          onSave={input => addBatch(input).then(() => onStockChanged())} />
      )}
      {modal === 'price' && (
        <PriceModal sku={sku} onClose={closeModal} onSave={handleSavePrice} />
      )}
      {modal === 'edit-batch' && rowBatch && (
        <BatchFormModal batch={rowBatch} warehouses={warehouses} onClose={closeModal}
          onSave={input => editBatch(rowBatch.id, input).then(() => onStockChanged())} />
      )}
    </div>
  )
}
