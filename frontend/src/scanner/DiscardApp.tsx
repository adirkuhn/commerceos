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
import type { Resolved } from './InventoryApp'
import type { Batch } from '../inventory/types'
import { Dialog } from '../components/common/Dialog'

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
        <h1 className="flex-1 text-base font-semibold">Discard</h1>
      </div>
      <div className="flex-1 flex flex-col overflow-y-auto">
        <ScanCodeScreen icon={Trash2} prompt="Scan a barcode or type a SKU code" loading={loading} error={error} onSubmit={handleSubmit} />
        {candidates.length > 0 && (
          <div className="w-full max-w-xs mx-auto px-6 pb-6 space-y-1.5 shrink-0">
            <p className="text-xs text-gray-500">Multiple matches -- pick one:</p>
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
  const { level, batches, loading, error, refresh } = useSKUStock(token, sku.id)
  const discardHook = useDiscardBatch(token)
  const [selectedRow, setSelectedRow] = useState<SelectedRow | null>(null)

  function selectRow(b: Batch) {
    setSelectedRow({
      warehouseId: b.warehouse_id,
      batchId: b.is_general_ledger ? null : b.id,
      label: b.is_general_ledger ? 'General ledger' : (b.lot_number || 'Batch'),
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
            onClick={() => selectRow(b)}
            className="w-full flex items-center gap-3 p-3.5 rounded-xl bg-[#1F2937] border border-[#2D3748] text-left hover:border-red-500 transition-colors"
          >
            <div className="w-9 h-9 rounded-lg bg-[#2D3748] flex items-center justify-center shrink-0">
              <Package size={16} className="text-gray-400" />
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium truncate">{b.is_general_ledger ? 'General ledger' : (b.lot_number || '—')}</p>
              <p className="text-xs text-gray-500">{b.quantity} in stock</p>
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
  const [quantity, setQuantity] = useState(String(row.quantity))
  const [reason, setReason] = useState('')
  const [photo, setPhoto] = useState<File | undefined>(undefined)

  const qty = Number(quantity) || 0
  const disabled = qty <= 0 || submitting

  return (
    <Dialog onClose={onCancel} className="fixed inset-0 z-50 flex items-end justify-center bg-black/40">
      <div className="w-full max-w-sm rounded-t-2xl bg-[#111827] border-t border-[#2D3748] max-h-[85vh] overflow-y-auto">
        <div className="flex items-start justify-between gap-3 px-5 pt-5">
          <div className="min-w-0">
            <h2 className="text-sm font-semibold truncate">Discard {row.label}</h2>
            <p className="text-xs text-gray-500 mt-0.5">{row.quantity} in stock</p>
          </div>
          <button onClick={onCancel} title="Close" className="shrink-0 p-1 -mt-1 -mr-1 text-gray-500 hover:text-gray-300">
            <X size={18} />
          </button>
        </div>

        <div className="p-5 space-y-4">
          {error && <p className="text-sm text-red-400">{error}</p>}

          <QuantityStepper id="discard-qty" label="Quantity to discard" value={quantity} onChange={setQuantity} min={0} max={row.quantity} />

          <div>
            <label htmlFor="discard-reason" className="block text-xs text-gray-400 mb-1.5">Reason</label>
            <input
              id="discard-reason" value={reason} onChange={e => setReason(e.target.value)}
              className="w-full px-3 py-2.5 rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <div>
            <p className="block text-xs text-gray-400 mb-1.5">Photo (optional)</p>
            <label className="flex items-center gap-2 px-3 py-2.5 rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-300 text-sm cursor-pointer">
              <CameraIcon size={15} className="shrink-0" />
              <span className="truncate">{photo ? photo.name : 'Take or choose a photo'}</span>
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
            {submitting ? 'Discarding…' : 'Discard'}
          </button>
        </div>
      </div>
    </Dialog>
  )
}
