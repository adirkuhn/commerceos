import { useState } from 'react'
import { Plus, ArrowRightLeft, Trash2, Edit3, PackageMinus, ClipboardList, Tag, History, Boxes, Scale } from 'lucide-react'
import clsx from 'clsx'
import { Trans, useTranslation } from 'react-i18next'
import { useSkuActionRoute } from './useSkuActionRoute'
import type { SkuAction } from './inventoryRoutes'
import { useQuerySet } from '../routing/useQueryState'
import { useSKUStock } from './hooks/useSKUStock'
import * as pmsApi from '../pms/api'
import type { SKU } from '../pms/types'
import type { Batch } from './types'
import { PortionFromBulkModal } from './PortionFromBulkModal'
import { SplitIntoPackagesModal } from '../pms/SplitIntoPackagesModal'
import { isScaleSoldWeightPool } from '../pms/weightPackages'
import { stockUnitFor } from './stockUnit'
import { AdjustStockModal, BatchFormModal, BatchScope, RecountModal } from './BatchStockModals'
import { formatSKUQuantity, parseOptionalPriceToMinorUnits, parsePriceToMinorUnits } from './inventoryFormat'
import { ExpiryStamp } from './ExpiryStamp'
import { BatchRowActionModal, NewBatchModal, PriceModal } from './BatchActionModals'
import { DamagedLostModal, SplitCaseModal, TransferModal } from './TransferStockModals'
import { apiErrorText } from '../api/apiErrorText'

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
  const { t } = useTranslation(['inventory', 'common'])
  const { level, batches, loading, error, addBatch, editBatch, removeBatch, adjust, recount, transfer, splitCase, portionFromBulk, markDamagedOrLost } =
    useSKUStock(token, sku.id)
  // The open modal, and the batch row it's locked to, come from the URL
  // (see useSkuActionRoute), as does "show zero-stock batches"
  // (?allBatches=) -- a refresh or Back keeps or closes them like pages.
  const route = useSkuActionRoute(productId, sku.id)
  const canSplitPackages = isScaleSoldWeightPool(sku)
  const unit = stockUnitFor(sku)
  // Packing adds package variants to this product, so its SKU list needs
  // a full reload (onSKUUpdated) -- deferred until the split modal closes,
  // since that reload remounts this panel and would drop the modal's
  // "Packages created" summary.
  const [packedSinceOpen, setPackedSinceOpen] = useState(false)
  const closeSplit = () => {
    route.close()
    if (packedSinceOpen) {
      setPackedSinceOpen(false)
      onSKUUpdated()
    }
  }
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
      ? { warehouseId: route.row.ledger, batchId: '', label: t('history.generalLedger') }
      : rowBatch && { warehouseId: rowBatch.warehouse_id, batchId: rowBatch.id, label: rowBatch.lot_number || t('subject.batch') }
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
      setDeleteError(apiErrorText(e) ?? t('sku.deleteFailed'))
    }
  }

  return (
    <div className="pl-8 pr-4 py-3 border-t border-line/60 bg-sunken">
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-3">
          <span className="font-mono text-2xs text-ink-2">
            {variantName && <span className="font-sans text-ink-3">{variantName} · </span>}
            {sku.sku_code}
          </span>
          <button
            type="button"
            onClick={() => setModal('price')}
            title={t('sku.editPrice')}
            className="flex items-center gap-1 text-2xs text-ink-4 hover:text-accent"
          >
            <Tag size={11} /> {t('sku.price')}
          </button>
          <button
            type="button"
            onClick={() => onViewHistory(sku)}
            title={t('sku.viewHistory')}
            className="flex items-center gap-1 text-2xs text-ink-4 hover:text-accent"
          >
            <History size={11} /> {t('row.history')}
          </button>
          {level && (
            <span data-testid="stock-level" className="text-2xs text-ink-3">
              <Trans t={t} i18nKey="sku.levels" values={{ total: formatSKUQuantity(sku, level.total), syncable: formatSKUQuantity(sku, level.syncable) }}
                components={{ total: <strong className="text-ink" />, syncable: <strong className="text-ink" /> }} />
            </span>
          )}
          {loading && <span className="text-2xs text-ink-4">{t('common:status.loading')}</span>}
        </div>
        <div className="flex items-center gap-1.5">
          <button type="button" onClick={() => setModal('adjust')} className="flex items-center gap-1 px-2 py-1 whitespace-nowrap rounded-md text-2xs bg-raised border border-line text-ink-2 hover:bg-hover">
            <Plus size={11} /> {t('sku.adjust')}
          </button>
          <button type="button" onClick={() => setModal('recount')} className="flex items-center gap-1 px-2 py-1 whitespace-nowrap rounded-md text-2xs bg-raised border border-line text-ink-2 hover:bg-hover">
            <ClipboardList size={11} /> {t('sku.recount')}
          </button>
          <button type="button" onClick={() => setModal('transfer')} className="flex items-center gap-1 px-2 py-1 whitespace-nowrap rounded-md text-2xs bg-raised border border-line text-ink-2 hover:bg-hover">
            <ArrowRightLeft size={11} /> {t('sku.transfer')}
          </button>
          {sku.case_of_sku_id && (
            <button type="button" onClick={() => setModal('split-case')} title={t('sku.splitCaseTitle')} className="flex items-center gap-1 px-2 py-1 whitespace-nowrap rounded-md text-2xs bg-raised border border-line text-ink-2 hover:bg-hover">
              <Boxes size={11} /> {t('sku.splitCase')}
            </button>
          )}
          {sku.portioned_from_sku_id && (
            <button type="button" onClick={() => setModal('portion')} title={t('sku.portionTitle')} className="flex items-center gap-1 px-2 py-1 whitespace-nowrap rounded-md text-2xs bg-raised border border-line text-ink-2 hover:bg-hover">
              <Scale size={11} /> {t('sku.portion')}
            </button>
          )}
          {canSplitPackages && (
            <button type="button" onClick={() => setModal('split-packages')} title={t('sku.splitPackagesTitle')} className="flex items-center gap-1 px-2 py-1 whitespace-nowrap rounded-md text-2xs bg-raised border border-line text-ink-2 hover:bg-hover">
              <Scale size={11} /> {t('sku.splitPackages')}
            </button>
          )}
          <button type="button" onClick={() => setModal('damaged-lost')} className="flex items-center gap-1 px-2 py-1 whitespace-nowrap rounded-md text-2xs bg-raised border border-line text-danger hover:bg-hover">
            <PackageMinus size={11} /> {t('sku.damagedLost')}
          </button>
        </div>
      </div>

      {error && <p className="text-2xs text-danger mb-2">{error}</p>}
      {deleteError && <p className="text-2xs text-danger mb-2">{deleteError}</p>}

      <div className="rounded-lg border border-line overflow-hidden">
        <table className="w-full text-2xs">
          <thead className="bg-fill">
            <tr>
              {[t('sku.colLot'), t('sku.colWarehouse'), t('sku.colQty'), t('sku.colExpiry'), ''].map((h, i) => (
                <th key={i} className="px-2.5 py-1.5 text-left font-medium text-ink-4 caps-label">{h}</th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-line/60">
            {visibleBatches.map(b => (
              <tr
                key={b.is_general_ledger ? `general-${b.warehouse_id}` : b.id}
                onClick={() => openRowAction(b)}
                title={t('sku.rowTitle')}
                className={clsx(
                  'cursor-pointer hover:bg-hover',
                  b.is_general_ledger && 'bg-fill',
                )}
              >
                <td className="px-2.5 py-1.5">
                  {b.is_general_ledger
                    ? <span className="text-ink-2">{t('history.generalLedger')}</span>
                    : b.lot_number
                      ? <span className="inline-block rounded-sm border border-line bg-surface px-1.5 py-px font-mono text-2xs text-ink-2">{b.lot_number}</span>
                      : <span className="text-ink-4">—</span>}
                </td>
                <td className="px-2.5 py-1.5 text-ink-3">{warehouses.find(w => w.id === b.warehouse_id)?.name ?? b.warehouse_id}</td>
                <td className="px-2.5 py-1.5 font-semibold tabular-nums text-figure">{b.quantity}</td>
                <td className="px-2.5 py-1.5">
                  {!b.is_general_ledger && <ExpiryStamp date={b.expiry_date} />}
                </td>
                <td className="px-2.5 py-1.5" onClick={e => e.stopPropagation()}>
                  {!b.is_general_ledger && (
                    <div className="flex items-center justify-end gap-2">
                      <button type="button" onClick={() => onViewBatchHistory(sku, b)} aria-label={t('sku.batchHistory', { lot: b.lot_number })} title={t('sku.viewHistory')} className="text-ink-4 hover:text-accent">
                        <History size={11} />
                      </button>
                      <button type="button" onClick={() => route.openAction('edit-batch', { batch: b.id })} aria-label={t('sku.editBatch', { lot: b.lot_number })} className="text-ink-4 hover:text-ink-2">
                        <Edit3 size={11} />
                      </button>
                      <button type="button" onClick={() => handleDeleteBatch(b.id)} aria-label={t('sku.deleteBatch', { lot: b.lot_number })} className="text-ink-4 hover:text-danger">
                        <Trash2 size={11} />
                      </button>
                    </div>
                  )}
                </td>
              </tr>
            ))}
            {batches.length === 0 && (
              <tr>
                <td colSpan={5} className="px-2.5 py-3 text-center text-ink-4">{t('sku.noBatches')}</td>
              </tr>
            )}
            {batches.length > 0 && visibleBatches.length === 0 && (
              <tr>
                <td colSpan={5} className="px-2.5 py-3 text-center text-ink-4">
                  {t('sku.allZero', { count: zeroStockBatchCount })}
                </td>
              </tr>
            )}
          </tbody>
        </table>
        <div className="px-2.5 py-1.5 border-t border-line flex items-center gap-3">
          <button type="button" onClick={() => setModal('new-batch')} className="text-2xs text-accent hover:text-accent/80">
            {t('sku.newBatch')}
          </button>
          {zeroStockBatchCount > 0 && (
            <button type="button" onClick={() => allBatches.toggle(sku.id)} className="text-2xs text-ink-4 hover:text-accent">
              {showAllBatches ? t('sku.hideZero') : t('sku.showZero', { count: zeroStockBatchCount })}
            </button>
          )}
        </div>
      </div>

      {modal === 'choose' && rowScope && (
        <BatchRowActionModal scope={rowScope} canSplitCase={!!sku.case_of_sku_id} canSplitPackages={canSplitPackages} onChoose={route.switchAction} onClose={closeModal} />
      )}
      {modal === 'adjust' && (
        <AdjustStockModal warehouses={warehouses} batches={batches} scope={rowScope ?? undefined} unit={unit} onClose={closeModal}
          onSave={input => adjust(input).then(() => onStockChanged())} />
      )}
      {modal === 'recount' && (
        <RecountModal warehouses={warehouses} batches={batches} scope={rowScope ?? undefined} unit={unit} onClose={closeModal}
          onSave={input => recount(input).then(() => onStockChanged())} />
      )}
      {modal === 'transfer' && (
        <TransferModal warehouses={warehouses} batches={batches} scope={rowScope ?? undefined} unit={unit} onClose={closeModal}
          onSave={input => transfer(input).then(() => onStockChanged())} />
      )}
      {modal === 'split-case' && (
        <SplitCaseModal warehouses={warehouses} batches={batches} scope={rowScope ?? undefined} onClose={closeModal}
          onSave={input => splitCase(input).then(() => onStockChanged())} />
      )}
      {modal === 'portion' && sku.portioned_from_sku_id && (
        <PortionFromBulkModal
          token={token} bulkSkuId={sku.portioned_from_sku_id} warehouses={warehouses} nominalWeightGrams={sku.weight_grams}
          onClose={closeModal} onSave={input => portionFromBulk(input).then(() => onStockChanged())}
        />
      )}
      {modal === 'split-packages' && canSplitPackages && sku.price_per_kilo_minor_units != null && (
        <SplitIntoPackagesModal
          token={token} bulkSkuId={sku.id} pricePerKiloMinorUnits={sku.price_per_kilo_minor_units} currencyCode={sku.currency_code}
          initialScope={rowScope ?? undefined} onPacked={() => { setPackedSinceOpen(true); onStockChanged() }} onClose={closeSplit}
        />
      )}
      {modal === 'damaged-lost' && (
        <DamagedLostModal warehouses={warehouses} batches={batches} scope={rowScope ?? undefined} unit={unit} onClose={closeModal}
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
