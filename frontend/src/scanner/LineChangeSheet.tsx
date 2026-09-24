import { useState } from 'react'
import { Check, Repeat, Search, Undo2, X } from 'lucide-react'
import { searchSKUCandidates, type SKUCandidate } from '../pms/searchCandidates'
import { QuantityStepper } from './QuantityStepper'
import { ProductCandidateRow } from './ProductCandidateRow'
import type { PackScan, PickingSessionLineItem, SubstituteLine } from './types'
import { Dialog } from '../components/common/Dialog'

// Tap a line item row while packing to either confirm how many went in the
// box, or change the order -- swap (part of) the line for one or more
// replacement products, or refund part of it. Order changes happen in
// packing now; nothing is real until the session finishes. A single original unit can be split across
// several substitute SKUs (e.g. 1 unit for 3 of one cheaper item + 4 of
// another) -- each row picks its own SKU, how many of it is being added,
// how much of the original line it covers, and whether that covered share
// goes back to stock. The rows' covered quantities must sum to the
// original line's own quantity exactly before submitting.
type SheetMode = 'pack' | 'substitute' | 'refund'

let substituteRowKeySeq = 0
function nextSubstituteRowKey(): string { return `sub-row-${++substituteRowKeySeq}` }

interface SubstituteRowState {
  key: string
  query: string
  candidates: SKUCandidate[]
  searching: boolean
  selected?: SKUCandidate
  substituteQty: string
  coveredQty: string
  returnToStock: boolean
}

function newSubstituteRow(): SubstituteRowState {
  return { key: nextSubstituteRowKey(), query: '', candidates: [], searching: false, substituteQty: '1', coveredQty: '', returnToStock: false }
}

export function LineChangeSheet({ token, lineItem, packedSoFar, scans, onDeleteScan, onCancel, onPack, onSubstitute, onRefund }: {
  token: string
  lineItem: PickingSessionLineItem
  packedSoFar: number
  scans: PackScan[]
  onDeleteScan: (scanId: string) => void
  onCancel: () => void
  onPack: (quantity: number) => void
  onSubstitute: (lines: SubstituteLine[]) => void
  onRefund: (quantity: number, reason?: string) => void
}) {
  const [mode, setMode] = useState<SheetMode>('pack')
  const [qty, setQty] = useState(String(Math.max(lineItem.quantity - packedSoFar, 0)))
  const [substituteRows, setSubstituteRows] = useState<SubstituteRowState[]>([newSubstituteRow()])
  const [refundQty, setRefundQty] = useState(String(Math.max(lineItem.quantity - packedSoFar, 0)))
  const [refundReason, setRefundReason] = useState('')
  const n = Number(qty) || 0
  const refundN = Number(refundQty) || 0
  const remaining = Math.max(lineItem.quantity - packedSoFar, 0)

  const coveredTotal = substituteRows.reduce((sum, r) => sum + (Number(r.coveredQty) || 0), 0)
  // Must cover exactly what's left, not the line's full original quantity
  // -- a line already partially packed can still have the rest substituted.
  const substituteReady = substituteRows.length > 0
    && substituteRows.every(r => r.selected && Number(r.substituteQty) > 0 && Number(r.coveredQty) > 0)
    && coveredTotal === remaining

  function updateRow(key: string, patch: Partial<SubstituteRowState>) {
    setSubstituteRows(rows => rows.map(r => (r.key === key ? { ...r, ...patch } : r)))
  }

  async function handleRowSearch(key: string, e: React.FormEvent) {
    e.preventDefault()
    const row = substituteRows.find(r => r.key === key)
    if (!row || !row.query.trim()) return
    updateRow(key, { searching: true })
    try {
      const candidates = await searchSKUCandidates(token, row.query.trim())
      updateRow(key, { candidates, searching: false })
    } catch {
      updateRow(key, { searching: false })
    }
  }

  function handleSubmitSubstitute() {
    if (!substituteReady) return
    onSubstitute(substituteRows.map(r => ({
      sku_id: r.selected!.sku.id,
      substitute_quantity: Number(r.substituteQty),
      original_quantity_covered: Number(r.coveredQty),
      return_to_stock: r.returnToStock,
    })))
  }

  return (
    <Dialog onClose={onCancel} className="fixed inset-0 z-50 flex items-end justify-center bg-black/40">
      <div className="w-full max-w-sm rounded-t-2xl bg-night-900 border-t border-night-700 max-h-[85vh] overflow-y-auto">
        <div className="flex items-start justify-between gap-3 px-5 pt-5">
          <div className="min-w-0">
            <h2 className="text-sm font-semibold truncate">{lineItem.name}</h2>
            <p className="text-xs text-gray-400 mt-0.5">Packed {packedSoFar} of {lineItem.quantity}</p>
          </div>
          <button aria-label="Close" onClick={onCancel} title="Close" className="shrink-0 w-11 h-11 -mt-3 -mr-3 flex items-center justify-center rounded-lg text-gray-400 hover:text-gray-300">
            <X size={18} />
          </button>
        </div>

        {scans.length > 0 && (
          <div className="px-5 pt-3 space-y-1.5">
            <p className="text-xs text-gray-400">Already staged for this line:</p>
            {scans.map(s => (
              <div key={s.id} className="flex items-center gap-2 rounded-lg bg-night-800 border border-night-700 px-3 py-2">
                <div className="flex-1 min-w-0">
                  <p className="text-sm text-gray-200">
                    {s.action_type === 'substitute'
                      ? `Substituted × ${s.quantity} (covers ${s.original_quantity_covered ?? '?'})`
                      : s.action_type === 'refund'
                        ? `Refunded × ${s.quantity}`
                        : `Packed × ${s.quantity}`}
                  </p>
                  {s.reason && <p className="text-xs text-gray-400 truncate">{s.reason}</p>}
                </div>
                <button aria-label="Delete this staged scan"
                  type="button"
                  onClick={() => onDeleteScan(s.id)}
                  title="Delete this staged scan"
                  className="shrink-0 w-11 h-11 -m-2 flex items-center justify-center rounded-lg text-gray-400 hover:text-red-400"
                >
                  <X size={14} />
                </button>
              </div>
            ))}
          </div>
        )}

        <div className="p-5 space-y-4">
          <div className="flex gap-2">
            <button aria-pressed={mode === 'pack'}
              type="button"
              onClick={() => setMode('pack')}
              className={`flex-1 flex flex-col items-center gap-1 py-2.5 rounded-lg text-2xs font-medium leading-tight ${mode === 'pack' ? 'bg-accent-solid text-white' : 'bg-night-800 text-gray-400'}`}
            >
              <Check size={15} /> Pack
            </button>
            <button aria-pressed={mode === 'substitute'}
              type="button"
              onClick={() => setMode('substitute')}
              className={`flex-1 flex flex-col items-center gap-1 py-2.5 rounded-lg text-2xs font-medium leading-tight ${mode === 'substitute' ? 'bg-accent-solid text-white' : 'bg-night-800 text-gray-400'}`}
            >
              <Repeat size={15} /> Substitute
            </button>
            <button aria-pressed={mode === 'refund'}
              type="button"
              onClick={() => setMode('refund')}
              className={`flex-1 flex flex-col items-center gap-1 py-2.5 rounded-lg text-2xs font-medium leading-tight ${mode === 'refund' ? 'bg-accent-solid text-white' : 'bg-night-800 text-gray-400'}`}
            >
              <Undo2 size={15} /> Refund
            </button>
          </div>

          {mode === 'refund' ? (
            <>
              <p className="text-xs text-gray-400">
                Can't fulfill this and don't want to substitute? Refund part or all of this line's quantity
                ({lineItem.quantity}) instead -- money only moves once packing is finished.
              </p>
              <QuantityStepper id="refund-quantity" label="Quantity to refund" value={refundQty} onChange={setRefundQty} min={0} max={lineItem.quantity} />
              <div>
                <label htmlFor="refund-reason" className="block text-xs text-gray-400 mb-1.5">Reason (optional)</label>
                <input
                  id="refund-reason"
                  value={refundReason}
                  onChange={e => setRefundReason(e.target.value)}
                  className="w-full px-3 py-2.5 rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus"
                />
              </div>
              <button
                onClick={() => onRefund(refundN, refundReason || undefined)}
                disabled={refundN <= 0}
                className="w-full py-3 rounded-xl bg-accent-solid text-white font-semibold disabled:opacity-50"
              >
                Confirm refund
              </button>
            </>
          ) : mode === 'pack' ? (
            <>
              <QuantityStepper id="pack-quantity" label="Quantity to pack" value={qty} onChange={setQty} min={0} max={remaining} />
              <button
                onClick={() => onPack(n)}
                disabled={n <= 0}
                className="w-full py-3 rounded-xl bg-accent-solid text-white font-semibold disabled:opacity-50"
              >
                Confirm pack
              </button>
            </>
          ) : (
            <>
              <p className="text-xs text-gray-400">
                Out of stock? Replace it with one or more products -- each row's "covers" quantity must add up to
                this line's remaining quantity ({remaining}) before you can confirm.
              </p>
              {substituteRows.map((row, i) => (
                <div key={row.key} className="rounded-xl bg-night-800 border border-night-700 p-3 space-y-2">
                  {row.selected ? (
                    <div className="flex items-center gap-2">
                      <div className="flex-1 min-w-0 text-sm text-gray-200 truncate">{row.selected.sku.sku_code}</div>
                      <button aria-label="Change product"
                        type="button"
                        onClick={() => updateRow(row.key, { selected: undefined, candidates: [] })}
                        className="shrink-0 w-11 h-11 -m-2 flex items-center justify-center rounded-lg text-gray-400 hover:text-red-400"
                        title="Change product"
                      >
                        <X size={14} />
                      </button>
                    </div>
                  ) : (
                    <>
                      <form onSubmit={e => handleRowSearch(row.key, e)} className="flex gap-2">
                        <div className="relative flex-1">
                          <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
                          <input
                            value={row.query}
                            onChange={e => updateRow(row.key, { query: e.target.value })}
                            placeholder="Search or scan barcode…"
                            className="w-full pl-8 pr-3 py-2 text-sm rounded-xl bg-night-900 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus"
                          />
                        </div>
                        <button type="submit" disabled={row.searching || !row.query.trim()} className="px-4 rounded-xl bg-night-900 border border-night-700 text-sm text-gray-300 disabled:opacity-50">
                          {row.searching ? '…' : 'Find'}
                        </button>
                      </form>
                      <div className="space-y-1.5">
                        {row.candidates.map(c => (
                          <ProductCandidateRow key={c.sku.id} token={token} candidate={c} onClick={() => updateRow(row.key, { selected: c, candidates: [] })} />
                        ))}
                      </div>
                    </>
                  )}
                  {row.selected && (
                    <>
                      <div className="grid grid-cols-2 gap-2">
                        <label className="block">
                          <span className="block text-2xs text-gray-400 mb-1">Qty added</span>
                          <input
                            type="number" min={1} value={row.substituteQty}
                            onChange={e => updateRow(row.key, { substituteQty: e.target.value })}
                            className="w-full px-3 py-2 text-sm rounded-xl bg-night-900 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus"
                          />
                        </label>
                        <label className="block">
                          <span className="block text-2xs text-gray-400 mb-1">Covers (of original)</span>
                          <input
                            type="number" min={1} value={row.coveredQty}
                            onChange={e => updateRow(row.key, { coveredQty: e.target.value })}
                            className="w-full px-3 py-2 text-sm rounded-xl bg-night-900 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus"
                          />
                        </label>
                      </div>
                      <label className="flex items-center gap-2 text-xs text-gray-400">
                        <input
                          type="checkbox" checked={row.returnToStock}
                          onChange={e => updateRow(row.key, { returnToStock: e.target.checked })}
                          className="rounded border-night-700 bg-night-900"
                        />
                        Return original quantity to stock
                      </label>
                    </>
                  )}
                  {substituteRows.length > 1 && (
                    <button
                      type="button"
                      onClick={() => setSubstituteRows(rows => rows.filter((_, idx) => idx !== i))}
                      className="text-xs text-gray-400 hover:text-red-400"
                    >
                      Remove this row
                    </button>
                  )}
                </div>
              ))}
              <button
                type="button"
                onClick={() => setSubstituteRows(rows => [...rows, newSubstituteRow()])}
                className="w-full py-2 rounded-xl border border-dashed border-night-700 text-sm text-gray-400 hover:text-gray-200"
              >
                + Add another substitute
              </button>
              <p className="text-xs text-gray-400 text-center">
                Covers {coveredTotal} of {remaining}
              </p>
              <button
                onClick={handleSubmitSubstitute}
                disabled={!substituteReady}
                className="w-full py-3 rounded-xl bg-accent-solid text-white font-semibold disabled:opacity-50"
              >
                Confirm substitute
              </button>
            </>
          )}
        </div>
      </div>
    </Dialog>
  )
}
