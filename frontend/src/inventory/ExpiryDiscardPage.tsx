import { useState } from 'react'
import { X, ArrowLeft, Image as ImageIcon } from 'lucide-react'
import clsx from 'clsx'
import { useQueryEnum } from '../routing/useQueryState'
import { useExpiredBatchReview } from './hooks/useExpiredBatchReview'
import { useExpiringBatches } from './hooks/useExpiringBatches'
import { useDiscards } from './hooks/useDiscards'
import { useDiscardBatch } from './hooks/useDiscardBatch'
import type { ExpiringBatch, ExpiredBatchReview } from './types'
import { Dialog } from '../components/common/Dialog'
import { useDateFormat } from '../format/DateFormatProvider'

// A single discard target, whichever tab it was opened from -- carries
// everything DiscardBatchInput needs plus enough to refresh the right
// list(s) afterwards.
interface DiscardTarget {
  skuId: string
  warehouseId: string
  batchId?: string
  quantity: number
  label: string
  source: 'expired' | 'expiring'
}

function DiscardModal({ target, submitting, error, onCancel, onConfirm }: {
  target: DiscardTarget
  submitting: boolean
  error: string
  onCancel: () => void
  onConfirm: (reason: string, photo?: File) => void
}) {
  const [reason, setReason] = useState('')
  const [photo, setPhoto] = useState<File | undefined>(undefined)

  return (
    <Dialog onClose={onCancel} dismissible={!submitting} className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4">
      <div className="w-full max-w-sm rounded-2xl bg-surface border border-line p-5 space-y-4">
        <div className="flex items-start justify-between gap-3">
          <div>
            <h2 className="text-sm font-semibold text-ink">Discard {target.label}</h2>
            <p className="text-xs text-ink-4 mt-0.5">
              Records a Lost movement for {target.quantity} unit{target.quantity === 1 ? '' : 's'} — cannot be undone.
            </p>
          </div>
          <button aria-label="Close" onClick={onCancel} title="Close" className="shrink-0 text-ink-4 dark:hover:text-ink-2">
            <X size={16} />
          </button>
        </div>
        {error && <p className="text-xs text-danger">{error}</p>}
        <div>
          <label htmlFor="discard-reason" className="block text-xs text-ink-3 mb-1.5">Reason</label>
          <input
            id="discard-reason" value={reason} onChange={e => setReason(e.target.value)}
            className="w-full px-3 py-2 text-sm rounded-lg dark:bg-night-800 bg-gray-50 border border-line text-ink outline-none focus:ring-1 focus:ring-focus"
          />
        </div>
        <div>
          <label className="block text-xs text-ink-3 mb-1.5">Photo (optional)</label>
          <label className="flex items-center gap-2 px-3 py-2 text-xs rounded-lg dark:bg-night-800 bg-gray-50 border border-line text-ink-2 cursor-pointer">
            <ImageIcon size={14} />
            <span className="truncate">{photo ? photo.name : 'Choose a photo'}</span>
            <input
              type="file" accept="image/*" className="hidden"
              onChange={e => setPhoto(e.target.files?.[0] ?? undefined)}
            />
          </label>
        </div>
        <div className="flex gap-2">
          <button
            onClick={onCancel}
            className="flex-1 py-2 rounded-lg dark:bg-night-800 bg-gray-100 text-ink-2 text-xs font-medium hover:bg-hover"
          >
            Cancel
          </button>
          <button
            onClick={() => onConfirm(reason, photo)}
            disabled={submitting}
            className="flex-1 py-2 rounded-lg bg-red-600 text-white text-xs font-medium hover:bg-red-700 disabled:opacity-50"
          >
            {submitting ? 'Discarding…' : 'Confirm'}
          </button>
        </div>
      </div>
    </Dialog>
  )
}

const EXPIRY_TABS = ['expired', 'expiring', 'discarded'] as const

type ExpiryTab = typeof EXPIRY_TABS[number]

// The tab lives in ?tab= (see inventoryRoutes). Defaults to 'expired' --
// the permanent "Damaged / lost / discarded" submenu entry opens straight
// to 'discarded' instead, since that's the whole reason it exists: unlike
// the red "Expired batches" alert (only shown when something needs review
// right now), this history is only reachable through an explicit menu
// entry, always -- not gated behind any current-state condition.
export function ExpiryDiscardPage({
  onClose, expiredReview, expiringBatches, discardsReport, discardHook,
}: {
  onClose: () => void
  expiredReview: ReturnType<typeof useExpiredBatchReview>
  expiringBatches: ReturnType<typeof useExpiringBatches>
  discardsReport: ReturnType<typeof useDiscards>
  discardHook: ReturnType<typeof useDiscardBatch>
}) {
  const { formatDate } = useDateFormat()
  const { batches, loading, error, extend, dismiss } = expiredReview
  const [tab, setTab] = useQueryEnum<ExpiryTab>('tab', EXPIRY_TABS, 'expired')
  const [rowError, setRowError] = useState('')
  const [extendingBatchId, setExtendingBatchId] = useState<string | null>(null)
  const [newExpiryDate, setNewExpiryDate] = useState('')
  const [actingBatchId, setActingBatchId] = useState<string | null>(null)
  const [discardTarget, setDiscardTarget] = useState<DiscardTarget | null>(null)

  const openExtend = (batchId: string) => {
    setRowError('')
    setNewExpiryDate('')
    setExtendingBatchId(batchId)
  }

  function openDiscard(source: 'expired' | 'expiring', row: ExpiredBatchReview | ExpiringBatch) {
    setDiscardTarget({
      skuId: row.sku_id, warehouseId: row.warehouse_id, batchId: row.batch_id, quantity: row.quantity,
      label: `${row.product_name} (${row.sku_code})`, source,
    })
  }

  async function handleConfirmDiscard(reason: string, photo?: File) {
    if (!discardTarget) return
    try {
      await discardHook.discard(discardTarget.skuId, {
        warehouse_id: discardTarget.warehouseId, batch_id: discardTarget.batchId,
        quantity: discardTarget.quantity, reason, photo,
      })
      setDiscardTarget(null)
      if (discardTarget.source === 'expired') expiredReview.refresh()
      else expiringBatches.refresh()
      discardsReport.refresh()
    } catch {
      // discardHook.error already holds the message; modal stays open.
    }
  }

  const handleConfirmExtend = async (batchId: string) => {
    if (!newExpiryDate) return
    setRowError('')
    setActingBatchId(batchId)
    try {
      await extend(batchId, newExpiryDate)
      setExtendingBatchId(null)
    } catch (err) {
      setRowError(err instanceof Error ? err.message : 'Failed to extend expiry')
    } finally {
      setActingBatchId(null)
    }
  }

  const handleDismiss = async (batchId: string) => {
    setRowError('')
    setActingBatchId(batchId)
    try {
      await dismiss(batchId)
    } catch (err) {
      setRowError(err instanceof Error ? err.message : 'Failed to dismiss batch')
    } finally {
      setActingBatchId(null)
    }
  }

  const TABS: { id: ExpiryTab; label: string }[] = [
    { id: 'expired', label: `Expired${batches.length ? ` (${batches.length})` : ''}` },
    { id: 'expiring', label: 'Expiring soon' },
    { id: 'discarded', label: 'Discarded' },
  ]

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b border-line bg-sunken shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-2xs font-medium text-ink-4 hover:text-ink-2 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> Inventory
          </button>
          <h1 className="text-xl font-semibold text-ink leading-tight">Expiry & discards</h1>
          <p className="text-xs text-ink-4 mt-0.5">Review expired/expiring stock, discard it, and see what's already been thrown away</p>
        </div>
      </div>

      <div className="flex gap-2 px-6 pt-4 shrink-0">
        {TABS.map(t => (
          <button
            key={t.id}
            onClick={() => setTab(t.id)}
            aria-pressed={tab === t.id}
            className={clsx(
              'px-3 py-1.5 rounded-lg text-xs font-medium transition-colors',
              tab === t.id
                ? 'bg-accent-solid text-white'
                : 'bg-raised text-ink-3 hover:bg-hover',
            )}
          >
            {t.label}
          </button>
        ))}
      </div>

      <div className="flex-1 overflow-auto">
        {/* discardHook.error is deliberately not shown here -- the modal (still open on failure) already surfaces it inline. */}
        {(error || rowError) && (
          <p className="px-6 py-4 text-xs text-danger">{error || rowError}</p>
        )}

        {tab === 'expired' && (
          <>
            {!error && loading && batches.length === 0 && <p className="px-6 py-4 text-xs text-ink-4">Loading…</p>}
            {!error && !loading && batches.length === 0 && (
              <p className="px-6 py-4 text-xs text-ink-4">No expired batches pending review.</p>
            )}
            {batches.length > 0 && (
              <table className="w-full text-xs">
                <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
                  <tr>
                    {['SKU', 'Product', 'Warehouse', 'Lot', 'Expired', 'Qty', ''].map(h => (
                      <th key={h} className="px-4 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y divide-line/60">
                  {batches.map(b => (
                    <tr key={b.batch_id} className="hover:bg-hover/60 transition-colors">
                      <td className="px-4 py-2.5 font-mono text-ink-2">{b.sku_code}</td>
                      <td className="px-4 py-2.5 text-ink">{b.product_name}</td>
                      <td className="px-4 py-2.5 text-ink-3">{b.warehouse_name}</td>
                      <td className="px-4 py-2.5 text-ink-3">{b.lot_number || '—'}</td>
                      <td className="px-4 py-2.5 text-danger">{formatDate(b.expiry_date)}</td>
                      <td className="px-4 py-2.5 tabular-nums text-ink-2">{b.quantity}</td>
                      <td className="px-4 py-2.5">
                        {extendingBatchId === b.batch_id ? (
                          <div className="flex items-center gap-1.5">
                            <input
                              type="date"
                              aria-label="New expiry date"
                              value={newExpiryDate}
                              onChange={e => setNewExpiryDate(e.target.value)}
                              className="px-2 py-1 text-xs rounded-lg bg-surface border border-line text-ink outline-none focus:ring-1 focus:ring-focus"
                            />
                            <button
                              onClick={() => handleConfirmExtend(b.batch_id)}
                              disabled={!newExpiryDate || actingBatchId === b.batch_id}
                              className="px-2 py-1 rounded-lg bg-accent-solid text-white text-2xs font-medium hover:bg-accent-solid-hover disabled:opacity-50"
                            >
                              Confirm
                            </button>
                            <button
                              onClick={() => setExtendingBatchId(null)}
                              className="px-2 py-1 rounded-lg bg-fill text-ink-2 text-2xs font-medium hover:bg-hover"
                            >
                              Cancel
                            </button>
                          </div>
                        ) : (
                          <div className="flex items-center gap-1.5">
                            <button
                              onClick={() => openDiscard('expired', b)}
                              disabled={actingBatchId === b.batch_id}
                              className="px-2 py-1 rounded-lg dark:bg-red-500/10 bg-red-50 text-danger text-2xs font-medium hover:dark:bg-red-500/20 hover:bg-red-100 disabled:opacity-50"
                            >
                              Discard
                            </button>
                            <button
                              onClick={() => openExtend(b.batch_id)}
                              disabled={actingBatchId === b.batch_id}
                              className="px-2 py-1 rounded-lg bg-fill text-ink-2 text-2xs font-medium hover:bg-hover disabled:opacity-50"
                            >
                              Extend
                            </button>
                            <button
                              onClick={() => handleDismiss(b.batch_id)}
                              disabled={actingBatchId === b.batch_id}
                              className="px-2 py-1 rounded-lg bg-fill text-ink-2 text-2xs font-medium hover:bg-hover disabled:opacity-50"
                            >
                              Dismiss
                            </button>
                          </div>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </>
        )}

        {tab === 'expiring' && (
          <>
            {!expiringBatches.error && expiringBatches.loading && expiringBatches.batches.length === 0 && (
              <p className="px-6 py-4 text-xs text-ink-4">Loading…</p>
            )}
            {!expiringBatches.error && !expiringBatches.loading && expiringBatches.batches.length === 0 && (
              <p className="px-6 py-4 text-xs text-ink-4">Nothing expiring soon.</p>
            )}
            {expiringBatches.batches.length > 0 && (
              <table className="w-full text-xs">
                <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
                  <tr>
                    {['SKU', 'Product', 'Warehouse', 'Lot', 'Expires', 'Qty', ''].map(h => (
                      <th key={h} className="px-4 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y divide-line/60">
                  {expiringBatches.batches.map(b => (
                    <tr key={b.batch_id} className="hover:bg-hover/60 transition-colors">
                      <td className="px-4 py-2.5 font-mono text-ink-2">{b.sku_code}</td>
                      <td className="px-4 py-2.5 text-ink">{b.product_name}</td>
                      <td className="px-4 py-2.5 text-ink-3">{b.warehouse_name}</td>
                      <td className="px-4 py-2.5 text-ink-3">{b.lot_number || '—'}</td>
                      <td className="px-4 py-2.5 text-ink-2">{formatDate(b.expiry_date)}</td>
                      <td className="px-4 py-2.5 tabular-nums text-ink-2">{b.quantity}</td>
                      <td className="px-4 py-2.5">
                        <button
                          onClick={() => openDiscard('expiring', b)}
                          className="px-2 py-1 rounded-lg dark:bg-red-500/10 bg-red-50 text-danger text-2xs font-medium hover:dark:bg-red-500/20 hover:bg-red-100"
                        >
                          Discard
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </>
        )}

        {tab === 'discarded' && (
          <>
            {!discardsReport.error && discardsReport.loading && discardsReport.discards.length === 0 && (
              <p className="px-6 py-4 text-xs text-ink-4">Loading…</p>
            )}
            {!discardsReport.error && !discardsReport.loading && discardsReport.discards.length === 0 && (
              <p className="px-6 py-4 text-xs text-ink-4">Nothing discarded yet.</p>
            )}
            {discardsReport.discards.length > 0 && (
              <table className="w-full text-xs">
                <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
                  <tr>
                    {['Photo', 'SKU', 'Product', 'Type', 'Warehouse', 'Lot', 'Qty', 'Reason', 'By', 'Discarded'].map(h => (
                      <th key={h} className="px-4 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y divide-line/60">
                  {discardsReport.discards.map(d => (
                    <tr key={d.id} className="hover:bg-hover/60 transition-colors">
                      <td className="px-4 py-2.5">
                        {d.photo_url ? (
                          <img src={d.photo_url} alt={`${d.product_name} discard photo`} className="w-8 h-8 rounded object-cover" />
                        ) : (
                          <div className="w-8 h-8 rounded dark:bg-night-800 bg-gray-100 flex items-center justify-center">
                            <ImageIcon size={12} className="text-ink-4" />
                          </div>
                        )}
                      </td>
                      <td className="px-4 py-2.5 font-mono text-ink-2">{d.sku_code}</td>
                      <td className="px-4 py-2.5 text-ink">{d.product_name}</td>
                      <td className="px-4 py-2.5">
                        <span className={clsx(
                          'px-1.5 py-0.5 rounded text-2xs font-medium capitalize',
                          d.type === 'damaged' ? 'bg-amber-500/15 text-warning' : 'bg-red-500/15 text-danger',
                        )}>
                          {d.type}
                        </span>
                      </td>
                      <td className="px-4 py-2.5 text-ink-3">{d.warehouse_name}</td>
                      <td className="px-4 py-2.5 text-ink-3">{d.lot_number || '—'}</td>
                      <td className="px-4 py-2.5 tabular-nums text-ink-2">{d.quantity}</td>
                      <td className="px-4 py-2.5 text-ink-3">{d.reason || '—'}</td>
                      <td className="px-4 py-2.5 text-ink-3">{d.discarded_by || '—'}</td>
                      <td className="px-4 py-2.5 text-ink-3">{formatDate(d.discarded_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </>
        )}
      </div>

      {discardTarget && (
        <DiscardModal
          target={discardTarget}
          submitting={discardHook.discarding}
          error={discardHook.error ?? ''}
          onCancel={() => setDiscardTarget(null)}
          onConfirm={handleConfirmDiscard}
        />
      )}
    </div>
  )
}
