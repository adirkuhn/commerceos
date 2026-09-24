import { useId, useState } from 'react'
import { Repeat, Trash2, Undo2, X } from 'lucide-react'
import clsx from 'clsx'
import type { PackScan, PickingSessionLineItem } from '../scanner/types'
import { packScanLabel } from './packScanLabel'

export interface SubstituteRowInput {
  code: string
  substituteQuantity: number
  coveredQuantity: number
  returnToStock: boolean
}

interface LineChangePanelProps {
  lineItem: PickingSessionLineItem
  // How much of the line is already packed, substituted or refunded.
  coveredSoFar: number
  // What's already recorded for this line (packed/replaced/refunded) --
  // removable, so a fully packed line can still be replaced.
  scans: PackScan[]
  busy: boolean
  onPack: (quantity: number) => void
  onSubstitute: (rows: SubstituteRowInput[]) => void
  onRefund: (quantity: number, reason: string) => void
  onClose: () => void
  onDeleteScan: (scanId: string) => void
}

type Mode = 'pack' | 'substitute' | 'refund'

interface SubstituteRowState {
  key: number
  code: string
  substituteQty: string
  coveredQty: string
  returnToStock: boolean
}

let rowKeySeq = 0
function newRow(): SubstituteRowState {
  return { key: ++rowKeySeq, code: '', substituteQty: '1', coveredQty: '', returnToStock: false }
}

const inputClass = 'w-full px-3 py-2 text-xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus'
const primaryButton = 'w-full px-3 py-2 rounded-lg bg-accent-solid text-white text-xs font-semibold hover:bg-accent-solid-hover disabled:opacity-50'

// The packing screen's side panel for one order line: pack a quantity, or
// change the order -- substitute (part of) the line with one or more other
// products, or refund (remove) part of it. Order changes happen in packing
// now that picking only picks; nothing is real until the session finishes.
// Pure UI: the page resolves codes and calls the API.
export function LineChangePanel({ lineItem, coveredSoFar, scans, busy, onPack, onSubstitute, onRefund, onClose, onDeleteScan }: LineChangePanelProps) {
  const remaining = Math.max(0, lineItem.quantity - coveredSoFar)
  const [mode, setMode] = useState<Mode>('pack')
  const [packQty, setPackQty] = useState(String(Math.max(1, remaining)))
  const [rows, setRows] = useState<SubstituteRowState[]>([newRow()])
  const [refundQty, setRefundQty] = useState(String(Math.max(1, remaining)))
  const [refundReason, setRefundReason] = useState('')
  const id = useId()

  const coveredTotal = rows.reduce((sum, r) => sum + (Number(r.coveredQty) || 0), 0)
  const substituteReady = rows.length > 0
    && rows.every(r => r.code.trim() && Number(r.substituteQty) > 0 && Number(r.coveredQty) > 0)
    && coveredTotal === remaining
  const refundQtyNum = Number(refundQty) || 0
  const refundReady = refundQtyNum > 0 && refundQtyNum <= lineItem.quantity

  function updateRow(key: number, patch: Partial<SubstituteRowState>) {
    setRows(rs => rs.map(r => (r.key === key ? { ...r, ...patch } : r)))
  }

  function modeButton(m: Mode, label: React.ReactNode) {
    return (
      <button aria-pressed={mode === m}
        type="button" onClick={() => setMode(m)}
        className={clsx('flex-1 flex items-center justify-center gap-1 px-3 py-1.5 text-2xs font-medium', mode === m ? 'bg-accent-solid text-white' : 'bg-raised text-ink-3')}
      >
        {label}
      </button>
    )
  }

  return (
    <div className="w-80 border-l border-line bg-sunken flex flex-col shrink-0">
      <div className="flex items-center justify-between px-4 py-3 border-b border-line">
        <p className="text-xs font-semibold text-ink">{lineItem.name}</p>
        <button type="button" onClick={onClose} aria-label="Close" className="p-1 -m-1 rounded text-ink-4 hover:text-ink-2 hover:bg-hover">
          <X size={14} />
        </button>
      </div>
      <div className="p-4 space-y-4 overflow-y-auto">
        {scans.length > 0 && (
          <div className="space-y-1.5">
            <p className="text-2xs font-medium text-ink-4 uppercase tracking-wider">Already recorded</p>
            {scans.map(s => (
              <div key={s.id} className="flex items-center justify-between gap-2 rounded-lg border border-line bg-raised px-2.5 py-1.5">
                <span className="text-2xs text-ink-2">{packScanLabel(s)}</span>
                <button type="button" onClick={() => onDeleteScan(s.id)} aria-label={`Remove ${packScanLabel(s)}`} className="text-ink-4 hover:text-danger">
                  <Trash2 size={11} />
                </button>
              </div>
            ))}
            {remaining === 0 && <p className="text-2xs text-ink-4">This line is complete. Remove what&apos;s recorded to replace it.</p>}
          </div>
        )}
        <div className="flex items-center rounded-lg border border-line overflow-hidden">
          {modeButton('pack', 'Pack')}
          {modeButton('substitute', <><Repeat size={11} /> Replace</>)}
          {modeButton('refund', <><Undo2 size={11} /> Refund</>)}
        </div>

        {mode === 'pack' && (
          <>
            <div>
              <label htmlFor={`${id}-pack-qty`} className="block text-2xs font-medium text-ink-4 uppercase tracking-wider mb-1.5">Quantity</label>
              <input
                id={`${id}-pack-qty`} type="number" min={1} max={remaining || undefined}
                value={packQty} onChange={e => setPackQty(e.target.value)}
                className={inputClass}
              />
            </div>
            <button type="button" onClick={() => onPack(Number(packQty))} disabled={busy || !(Number(packQty) >= 1)} className={primaryButton}>Confirm pack</button>
          </>
        )}

        {mode === 'substitute' && (
          <>
            <p className="text-2xs text-ink-3">
              Replace with one or more products -- the rows&apos; &quot;covers&quot; quantities must add up to what&apos;s left of this line ({remaining}).
            </p>
            {rows.map((row, i) => (
              <div key={row.key} className="space-y-2 rounded-lg border border-line p-2.5">
                <input
                  type="text" value={row.code} onChange={e => updateRow(row.key, { code: e.target.value })}
                  placeholder="Scan or type replacement SKU/barcode" className={inputClass} autoFocus={i === 0}
                />
                <div className="grid grid-cols-2 gap-2">
                  <div>
                    <label htmlFor={`${id}-sub-qty-${row.key}`} className="block text-2xs text-ink-4 mb-1">Qty added</label>
                    <input
                      id={`${id}-sub-qty-${row.key}`} type="number" min={1} value={row.substituteQty}
                      onChange={e => updateRow(row.key, { substituteQty: e.target.value })} className={inputClass}
                    />
                  </div>
                  <div>
                    <label htmlFor={`${id}-sub-cov-${row.key}`} className="block text-2xs text-ink-4 mb-1">Covers (of original)</label>
                    <input
                      id={`${id}-sub-cov-${row.key}`} type="number" min={1} value={row.coveredQty}
                      onChange={e => updateRow(row.key, { coveredQty: e.target.value })} className={inputClass}
                    />
                  </div>
                </div>
                <label className="flex items-center gap-2 text-2xs text-ink-3">
                  <input type="checkbox" checked={row.returnToStock} onChange={e => updateRow(row.key, { returnToStock: e.target.checked })} />
                  Return original quantity to stock
                </label>
                {rows.length > 1 && (
                  <button type="button" onClick={() => setRows(rs => rs.filter(r => r.key !== row.key))} className="text-2xs text-ink-4 hover:text-danger">
                    Remove this row
                  </button>
                )}
              </div>
            ))}
            <button
              type="button" onClick={() => setRows(rs => [...rs, newRow()])}
              className="w-full py-2 rounded-lg border border-dashed border-line text-2xs text-ink-3"
            >
              + Add another replacement
            </button>
            <p className="text-2xs text-center text-ink-4">Covers {coveredTotal} of {remaining}</p>
            <button
              type="button" disabled={busy || !substituteReady} className={primaryButton}
              onClick={() => onSubstitute(rows.map(r => ({
                code: r.code.trim(), substituteQuantity: Number(r.substituteQty), coveredQuantity: Number(r.coveredQty), returnToStock: r.returnToStock,
              })))}
            >
              Confirm replace
            </button>
          </>
        )}

        {mode === 'refund' && (
          <>
            <p className="text-2xs text-ink-3">
              Can&apos;t send this and don&apos;t want to substitute? Refund part or all of this line ({lineItem.quantity}) --
              money only moves once packing is finished.
            </p>
            <div>
              <label htmlFor={`${id}-refund-qty`} className="block text-2xs font-medium text-ink-4 uppercase tracking-wider mb-1.5">Quantity to refund</label>
              <input
                id={`${id}-refund-qty`} type="number" min={1} max={lineItem.quantity}
                value={refundQty} onChange={e => setRefundQty(e.target.value)} className={inputClass}
              />
            </div>
            <div>
              <label htmlFor={`${id}-refund-reason`} className="block text-2xs font-medium text-ink-4 uppercase tracking-wider mb-1.5">Reason (optional)</label>
              <input id={`${id}-refund-reason`} type="text" value={refundReason} onChange={e => setRefundReason(e.target.value)} className={inputClass} />
            </div>
            <button type="button" onClick={() => onRefund(refundQtyNum, refundReason)} disabled={busy || !refundReady} className={primaryButton}>
              Confirm refund
            </button>
          </>
        )}
      </div>
    </div>
  )
}
