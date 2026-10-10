import { useTranslation } from 'react-i18next'
import { useState } from 'react'
import { Route, Routes } from 'react-router-dom'
import { ArrowLeft, Package, Trash2, X, Camera as CameraIcon } from 'lucide-react'
import * as pmsApi from '../pms/api'
import { useSKUStock } from '../inventory/hooks/useSKUStock'
import { useDiscardBatch } from '../inventory/hooks/useDiscardBatch'
import { searchSKUCandidates, type SKUCandidate } from '../pms/searchCandidates'
import { ScanCodeScreen } from './ScanCodeScreen'
import { ProductCandidateRow } from './ProductCandidateRow'
import { QuantityStepper } from './QuantityStepper'
import { SkuRoute } from './SkuRoute'
import { scannerPaths } from './scannerRoutes'
import { useStackNavigation } from '../routing/useStackNavigation'
import type { Resolved } from './InventoryScanScreens'
import type { Batch } from '../inventory/types'
import { Dialog } from '../components/common/Dialog'
import { apiErrorText } from '../api/apiErrorText'

interface DiscardAppProps {
  token: string
  onExit: () => void
}

// Scanner mini-app: scan/type a code to resolve a SKU, tap a batch or
// general-ledger row, then discard some or all of it -- quantity, reason,
// and an optional photo of the discarded goods. Same DiscardBatchUseCase
// endpoint the desktop Inventory module's Expiry & Discards page uses, so
// every discard (desktop review queue, desktop "expiring soon", or here)
// lands in the same report.
//
// Mounted at /scanner/discard/* -- scan screen at the index, the SKU's
// discard screen at /scanner/discard/:code (see ./scannerRoutes).
export function DiscardApp({ token, onExit }: DiscardAppProps) {
  const { open } = useStackNavigation()

  return (
    <Routes>
      <Route
        index
        element={<DiscardScanScreen token={token} onResolved={r => open(scannerPaths.discardSku(r.sku.sku_code))} onExit={onExit} />}
      />
      <Route
        path=":code"
        element={
          <SkuRoute token={token} backTo={scannerPaths.discard}>
            {(resolved, { back }) => (
              <DiscardSKUScreen token={token} sku={resolved.sku} productName={resolved.productName} onBack={back} />
            )}
          </SkuRoute>
        }
      />
    </Routes>
  )
}

function DiscardScanScreen({ token, onResolved, onExit }: { token: string; onResolved: (r: Resolved) => void; onExit: () => void }) {
  const { t } = useTranslation('scanner')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [candidates, setCandidates] = useState<SKUCandidate[]>([])

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
      setError(apiErrorText(err) ?? t('errors.resolveCode'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 flex flex-col bg-night-950 text-gray-100">
      <div className="flex items-center gap-3 px-4 py-4 border-b border-night-700 shrink-0">
        <button aria-label={t('shared.backToApps')} onClick={onExit} className="w-11 h-11 -ml-3 -my-2 flex items-center justify-center rounded-lg shrink-0" title={t('shared.backToApps')}>
          <ArrowLeft size={18} className="text-gray-400" />
        </button>
        <h1 className="flex-1 text-base font-semibold">{t('discard.title')}</h1>
      </div>
      <div className="flex-1 flex flex-col overflow-y-auto">
        <ScanCodeScreen icon={Trash2} prompt={t('shared.scanPrompt')} loading={loading} error={error} onSubmit={handleSubmit} />
        {candidates.length > 0 && (
          <div className="w-full max-w-xs mx-auto px-6 pb-6 space-y-1.5 shrink-0">
            <p className="text-xs text-gray-400">{t('shared.multipleMatches')}</p>
            {candidates.map(c => (
              <ProductCandidateRow key={c.sku.id} token={token} candidate={c} onClick={() => onResolved({ sku: c.sku, productName: c.productName })} />
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

interface SelectedRow {
  warehouseId: string
  batchId: string | null // null = general ledger
  label: string
  quantity: number
}

function DiscardSKUScreen({ token, sku, productName, onBack }: { token: string; sku: Resolved['sku']; productName: string; onBack: () => void }) {
  const { t } = useTranslation(['scanner', 'common'])
  const { level, batches, loading, error, refresh } = useSKUStock(token, sku.id)
  const discardHook = useDiscardBatch(token)
  const [selectedRow, setSelectedRow] = useState<SelectedRow | null>(null)

  function selectRow(b: Batch) {
    setSelectedRow({
      warehouseId: b.warehouse_id,
      batchId: b.is_general_ledger ? null : b.id,
      label: b.is_general_ledger ? t('batch.generalLedger') : (b.lot_number || t('booking.batch')),
      quantity: b.quantity,
    })
  }

  async function handleDiscard(quantity: number, reason: string, photo?: File) {
    if (!selectedRow) return
    try {
      await discardHook.discard(sku.id, {
        warehouse_id: selectedRow.warehouseId, batch_id: selectedRow.batchId ?? undefined,
        quantity, reason, photo,
      })
      setSelectedRow(null)
      refresh()
    } catch {
      // discardHook.error already holds the message; sheet stays open.
    }
  }

  return (
    <div className="fixed inset-0 flex flex-col bg-night-950 text-gray-100">
      <div className="flex items-center gap-3 px-4 py-4 border-b border-night-700 shrink-0">
        <button aria-label={t('shared.backToScan')} onClick={onBack} className="w-11 h-11 -ml-3 -my-2 flex items-center justify-center rounded-lg shrink-0" title={t('shared.backToScan')}>
          <ArrowLeft size={18} className="text-gray-400" />
        </button>
        <div className="flex-1 min-w-0">
          <h1 className="text-base font-semibold truncate">
            <span>{productName}</span>{' '}
            <span className="text-xs font-normal text-gray-400">- <span>{sku.sku_code}</span></span>
          </h1>
          {level && <p className="text-xs text-gray-400">{t('shared.totalStock', { total: level.total })}</p>}
        </div>
      </div>

      {error && <p className="mx-4 mt-3 text-sm text-red-400">{error}</p>}

      <div className="flex-1 overflow-y-auto px-4 py-3 space-y-2">
        {loading && <p className="text-center text-sm text-gray-400 py-8">{t('common:status.loading')}</p>}
        {!loading && batches.length === 0 && (
          <p className="text-center text-sm text-gray-400 py-8">{t('shared.noBatches')}</p>
        )}
        {batches.map(b => (
          <button
            key={b.is_general_ledger ? `general-${b.warehouse_id}` : b.id}
            type="button"
            onClick={() => selectRow(b)}
            className="w-full flex items-center gap-3 p-3.5 rounded-xl bg-night-800 border border-night-700 text-left hover:border-red-500 transition-colors"
          >
            <div className="w-9 h-9 rounded-lg bg-night-700 flex items-center justify-center shrink-0">
              <Package size={16} className="text-gray-400" />
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium truncate">{b.is_general_ledger ? t('batch.generalLedger') : (b.lot_number || '—')}</p>
              <p className="text-xs text-gray-400">{t('shared.inStock', { quantity: b.quantity })}</p>
            </div>
          </button>
        ))}
      </div>

      {selectedRow && (
        <DiscardSheet
          row={selectedRow}
          submitting={discardHook.discarding}
          error={discardHook.error ?? ''}
          onCancel={() => setSelectedRow(null)}
          onConfirm={handleDiscard}
        />
      )}
    </div>
  )
}

function DiscardSheet({ row, submitting, error, onCancel, onConfirm }: {
  row: SelectedRow
  submitting: boolean
  error: string
  onCancel: () => void
  onConfirm: (quantity: number, reason: string, photo?: File) => void
}) {
  const { t } = useTranslation(['scanner', 'common'])
  const [quantity, setQuantity] = useState(String(row.quantity))
  const [reason, setReason] = useState('')
  const [photo, setPhoto] = useState<File | undefined>(undefined)

  const qty = Number(quantity) || 0
  const disabled = qty <= 0 || submitting

  return (
    <Dialog onClose={onCancel} className="fixed inset-0 z-50 flex items-end justify-center bg-black/40">
      <div className="w-full max-w-sm rounded-t-2xl bg-night-900 border-t border-night-700 max-h-[85vh] overflow-y-auto">
        <div className="flex items-start justify-between gap-3 px-5 pt-5">
          <div className="min-w-0">
            <h2 className="text-sm font-semibold truncate">{t('discard.discardRow', { row: row.label })}</h2>
            <p className="text-xs text-gray-400 mt-0.5">{t('shared.inStock', { quantity: row.quantity })}</p>
          </div>
          <button aria-label={t('common:actions.close')} onClick={onCancel} title={t('common:actions.close')} className="shrink-0 w-11 h-11 -mt-3 -mr-3 flex items-center justify-center rounded-lg text-gray-400 hover:text-gray-300">
            <X size={18} />
          </button>
        </div>

        <div className="p-5 space-y-4">
          {error && <p className="text-sm text-red-400">{error}</p>}

          <QuantityStepper id="discard-qty" label={t('discard.qty')} value={quantity} onChange={setQuantity} min={0} max={row.quantity} />

          <div>
            <label htmlFor="discard-reason" className="block text-xs text-gray-400 mb-1.5">{t('shared.reason')}</label>
            <input
              id="discard-reason" value={reason} onChange={e => setReason(e.target.value)}
              className="w-full px-3 py-2.5 rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus"
            />
          </div>

          <div>
            <p className="block text-xs text-gray-400 mb-1.5">{t('discard.photo')}</p>
            <label className="flex items-center gap-2 px-3 py-2.5 rounded-xl bg-night-800 border border-night-700 text-gray-300 text-sm cursor-pointer">
              <CameraIcon size={15} className="shrink-0" />
              <span className="truncate">{photo ? photo.name : t('discard.choosePhoto')}</span>
              <input
                type="file" accept="image/*" capture="environment" className="hidden"
                onChange={e => setPhoto(e.target.files?.[0] ?? undefined)}
              />
            </label>
          </div>

          <button
            onClick={() => onConfirm(qty, reason, photo)}
            disabled={disabled}
            className="w-full py-3 rounded-xl bg-red-600 text-white font-semibold disabled:opacity-50"
          >
            {submitting ? t('discard.discarding') : t('discard.submit')}
          </button>
        </div>
      </div>
    </Dialog>
  )
}
