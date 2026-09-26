import { useState } from 'react'
import { ArrowLeft } from 'lucide-react'
import { canTakeOverLocks } from '../packing/lockDisplay'
import * as api from './api'
import { CameraScanner } from './CameraScanner'
import { ProductCandidateRow } from './ProductCandidateRow'
import { computeMissingItems, pickedQuantityByLineItem } from './pickingMissingItems'
import type { PickingSessionDetail, PickingSessionLineItem } from './types'
import { StockShortfallDialog } from './StockShortfallDialog'
import type { SalesOrder } from '../oms/types'
import { PickLineSheet } from './PickingLineSheet'
import { useDateFormat } from '../format/DateFormatProvider'
import { isAuthError } from '../auth/isAuthError'
import { useScanSession, type ScanSessionApi } from './useScanSession'
import { useSessionCompletion } from './useSessionCompletion'
import { usePickingScan } from './usePickingScan'
import { PickingReviewScreen } from './PickingReviewScreen'
import { PickingDoneScreen } from './PickingDoneScreen'
import { PickingAddedItems } from './PickingAddedItems'
import { PickingBillingLine } from './PickingOrderDetails'
import { ScanSessionOrderBand, StartOverButton } from './ScanSessionOrderBand'
import { ScanSessionLockBanner } from './ScanSessionLockBanner'
import { ScanSessionScanForm } from './ScanSessionScanForm'
import { ScanSessionLineRow } from './ScanSessionLines'
import { ScanErrorDialog, StartOverDialog } from './ScanSessionDialogs'

interface PickingScreenProps {
  token: string
  sessionId: string
  order: SalesOrder
  onExit: () => void
  onAuthError: () => void
  // Hands off to the packing screen for this same order -- see Picking.tsx's
  // continueToPacking doc. Starts (or resumes) the packing session itself
  // (picking already has the order/token in hand) and lets ScannerApp just
  // switch screens with the result.
  onContinueToPacking: (sessionId: string, order: SalesOrder) => void
}

const PICKING_SESSION_API: ScanSessionApi<PickingSessionDetail> = {
  get: (token, id) => api.getPickingSession(token, id),
  touch: (token, id) => api.touchPickingSession(token, id),
  release: (token, id) => api.releasePickingSessionLock(token, id),
  takeOver: (token, id) => api.takeOverPickingSession(token, id),
  deleteScan: (token, id, scanId) => api.deletePickScan(token, id, scanId),
  reset: (token, id) => api.resetPickingSession(token, id),
  messages: {
    loadFailed: 'Failed to load picking session',
    lockLost: 'Another user has taken over this picking session',
    resetFailed: 'Failed to reset picking session',
  },
}

export function PickingScreen({ token, sessionId, order, onExit, onAuthError, onContinueToPacking }: PickingScreenProps) {
  const { formatDate } = useDateFormat()
  const session = useScanSession(token, sessionId, onAuthError, PICKING_SESSION_API)
  const { detail, error, setError, readOnly, codeInputRef } = session
  const completion = useSessionCompletion(
    resolutions => api.completePicking(token, sessionId, resolutions), onAuthError, setError,
  )
  const [pickLine, setPickLine] = useState<{ lineItem: PickingSessionLineItem; pickedSoFar: number } | null>(null)
  const [startingPacking, setStartingPacking] = useState(false)

  function stage(input: Parameters<typeof api.scanPickItem>[2]): Promise<boolean> {
    return session.run(() => api.scanPickItem(token, sessionId, input), 'Scan failed')
  }

  const scan = usePickingScan({
    token, sessionId, detail, codeInputRef, stage, setError, setScanning: session.setScanning, onAuthError,
  })

  async function handlePause() {
    try {
      await api.pausePicking(token, sessionId)
      onExit()
    } catch (err) {
      if (isAuthError(err)) return onAuthError()
      setError(err instanceof Error ? err.message : 'Failed to pause')
    }
  }

  async function handleContinueToPacking() {
    setStartingPacking(true)
    try {
      const packingSession = await api.startOrResumePacking(token, order.id)
      onContinueToPacking(packingSession.id, order)
    } catch (err) {
      if (isAuthError(err)) return onAuthError()
      setError(err instanceof Error ? err.message : 'Failed to start packing')
      setStartingPacking(false)
    }
  }

  if (completion.stockShortfalls) {
    return (
      <StockShortfallDialog
        shortfalls={completion.stockShortfalls}
        submitting={completion.finishing}
        onCancel={() => completion.setStockShortfalls(null)}
        onConfirm={resolutions => completion.confirmComplete(resolutions)}
      />
    )
  }

  if (completion.confirmingFinish && detail) {
    return (
      <PickingReviewScreen
        order={order} missing={computeMissingItems(detail.line_items, detail.scans)}
        error={error} finishing={completion.finishing}
        onBack={() => completion.setConfirmingFinish(false)} onConfirm={() => completion.confirmComplete()}
      />
    )
  }

  if (completion.result) {
    return (
      <PickingDoneScreen
        missingItems={completion.result.missing_items} error={error} startingPacking={startingPacking}
        onContinueToPacking={handleContinueToPacking} onDone={onExit}
      />
    )
  }

  const pickedByLineItem = pickedQuantityByLineItem(detail?.scans ?? [])
  const addedItems = (detail?.scans ?? []).filter(s => s.action_type === 'freebie' || s.action_type === 'new_item')

  return (
    <div className="fixed inset-0 flex flex-col bg-night-950 text-gray-100">
      <div className="flex items-center gap-3 px-4 py-4 border-b border-night-700 shrink-0">
        <button aria-label="Pause and go back" onClick={handlePause} className="w-11 h-11 -ml-3 -my-2 flex items-center justify-center rounded-lg shrink-0" title="Pause and go back">
          <ArrowLeft size={18} className="text-gray-400" />
        </button>
        <h1 className="flex-1 text-base font-semibold">Picking</h1>
        <button
          onClick={() => completion.setConfirmingFinish(true)} disabled={readOnly}
          className="px-4 min-h-11 -my-2 rounded-lg bg-green-600 text-white text-sm font-semibold hover:bg-green-700 disabled:opacity-50"
        >
          Finish
        </button>
      </div>

      {/* Order info -- its own full-width band, separate from the control
          row above, so the address has room to wrap instead of getting
          truncated next to the back/Finish buttons. */}
      <ScanSessionOrderBand order={order}>
        <PickingBillingLine order={order} />
        <StartOverButton onClick={() => session.setResetConfirmOpen(true)} disabled={readOnly} />
      </ScanSessionOrderBand>

      {session.heldByOther && (
        <ScanSessionLockBanner
          activity="picking" holderName={session.nameFor(session.heldByOther)} lockedSince={detail?.session.locked_since}
          canTakeOver={canTakeOverLocks(session.me?.role)} onTakeOver={session.takeOver}
        />
      )}

      {scan.cameraOpen && (
        <div className="px-4 pt-3 shrink-0">
          <CameraScanner onDecode={scan.handleCameraDecode} onClose={() => scan.setCameraOpen(false)} frameShape="rect" variant="inline" />
        </div>
      )}

      <div className="flex-1 overflow-y-auto px-4 py-3 space-y-2">
        {session.loading && <p className="text-center text-sm text-gray-400 py-8">Loading…</p>}
        {detail?.line_items.map(li => {
          const pickedQty = pickedByLineItem.get(li.id) ?? 0
          return (
            <ScanSessionLineRow
              key={li.id} token={token} lineItem={li} doneQty={pickedQty} disabled={readOnly}
              onClick={() => setPickLine({ lineItem: li, pickedSoFar: pickedQty })}
            >
              {li.sold_expiry_date && (
                <p className="text-xs text-gray-400 truncate" title="Reported by WooCommerce at checkout -- may not match the batch actually allocated">
                  Sold exp. {formatDate(li.sold_expiry_date)}
                </p>
              )}
              {!li.sku_id && <p className="text-xs text-amber-400">Not linked to a product yet</p>}
            </ScanSessionLineRow>
          )
        })}

        <PickingAddedItems items={addedItems} />
      </div>

      <ScanSessionScanForm
        inputRef={codeInputRef} code={scan.code} onCodeChange={scan.setCode} onSubmit={scan.handleScan}
        onOpenCamera={() => scan.setCameraOpen(true)}
        error={error} scanning={session.scanning} readOnly={readOnly} blocked={!!scan.scanErrorAlert}
      >
        {scan.candidates.length > 0 && (
          <div className="space-y-1.5">
            <p className="text-xs text-gray-400">Multiple matches -- pick one:</p>
            {scan.candidates.map(c => (
              <ProductCandidateRow key={c.sku.id} token={token} candidate={c} onClick={() => scan.selectCandidate(c)} />
            ))}
          </div>
        )}
      </ScanSessionScanForm>

      {pickLine && (
        <PickLineSheet
          lineItem={pickLine.lineItem}
          pickedSoFar={pickLine.pickedSoFar}
          scans={(detail?.scans ?? []).filter(s => s.line_item_id === pickLine.lineItem.id)}
          onDeleteScan={session.deleteScan}
          onCancel={() => setPickLine(null)}
          onPick={(quantity, reason) => {
            setPickLine(null)
            stage({ sku_id: pickLine.lineItem.sku_id ?? undefined, action_type: 'pick', quantity, reason })
          }}
        />
      )}

      {scan.scanErrorAlert && <ScanErrorDialog message={scan.scanErrorAlert} onDismiss={scan.dismissScanError} />}

      {session.resetConfirmOpen && (
        <StartOverDialog
          activity="picking" resetting={session.resetting}
          onCancel={() => session.setResetConfirmOpen(false)}
          onConfirm={() => session.resetSession(() => setPickLine(null))}
        />
      )}
    </div>
  )
}
