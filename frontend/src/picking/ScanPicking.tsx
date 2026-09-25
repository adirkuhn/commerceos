import { ScanBarcode } from 'lucide-react'
import { useMemberNames } from '../team/hooks/useMemberNames'
import { canTakeOverLocks } from '../packing/lockDisplay'
import { useAuth } from '../auth/AuthContext'
import { usePickingQueue } from '../oms/hooks/usePackingQueue'
import { ReadOnlyOrderItems } from './ReadOnlyOrderItems'
import { useQueryString } from '../routing/useQueryState'
import type { QueueEntry } from '../oms/types'
import { StockShortfallDialog } from '../scanner/StockShortfallDialog'
import { SessionErrorPanel } from '../oms/SessionErrorPanel'
import { pickProgress } from './pickingProgress'
import { usePickLineSheet } from './usePickLineSheet'
import { usePickingSession } from './usePickingSession'
import { usePickScanEntry } from './usePickScanEntry'
import { PickingQueueSidebar } from './PickingQueueSidebar'
import { PickingCompleteResult } from './PickingCompleteResult'
import { PickingReviewStep } from './PickingReviewStep'
import { PickingSessionHeader, PickingLockBanner } from './PickingSessionHeader'
import { PickScanInput } from './PickScanInput'
import { PickProgressBar, PickLineList, PickScanLog } from './PickLineList'
import { PickLineSheet } from './PickLineSheet'
import { ScanErrorDialog, StartOverDialog } from './PickingDialogs'

// Each open session has its own URL (/picking/:sessionId) so a refresh or the
// browser Back button keeps the picker on the order they were working on.
// Session id rather than order id: reopening is a read (get session -> get
// its order), never a startOrResume, which would fail once the order is
// fulfilled by completing the session.
export function ScanPicking() {
  const { token, user } = useAuth()

  // The picking queue: orders still to pick, with who is working each and
  // whether packing has it -- those open as a read-only item list
  // (?view=<orderId>, so refresh/Back work) instead of a picking session.
  const queue = usePickingQueue(token ?? '')
  const [viewId, openView] = useQueryString('view', { history: 'push' })
  const viewEntry = queue.entries.find(e => e.sales_order.id === viewId && e.in_packing)
  const nameFor = useMemberNames(token ?? '')

  const sheet = usePickLineSheet()
  const ps = usePickingSession({ token, userId: user?.user_id ?? '', refreshQueue: queue.refresh, onSelectionReset: sheet.close })
  const { activeOrder, session, error } = ps
  const entry = usePickScanEntry({ token, session, loadSession: ps.loadSession, setError: ps.setError, sheet })

  function selectEntry({ sales_order: order, in_packing }: QueueEntry) {
    if (in_packing) {
      // An order packing has: show its items read-only, no picking session.
      if (activeOrder) ps.backToOrders()
      openView(order.id)
      return
    }
    if (viewId) openView(null)
    ps.selectOrder(order)
  }

  const progress = pickProgress(session)

  return (
    <div className="flex h-full">
      <PickingQueueSidebar
        entries={queue.entries} loading={queue.loading} error={queue.error}
        activeOrderId={activeOrder?.id ?? null} activeInProgress={session?.session.status === 'in_progress'}
        viewId={viewId} nameFor={nameFor} onSelect={selectEntry}
      />

      {/* Main pick area */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {viewEntry && (
          <div className="flex-1 overflow-y-auto p-6">
            <ReadOnlyOrderItems token={token ?? ''} orderId={viewEntry.sales_order.id} lock={viewEntry.lock} nameFor={nameFor} />
          </div>
        )}

        {!viewEntry && !activeOrder && !(ps.urlSessionId && (ps.sessionLoading || error)) && (
          <div className="flex-1 flex flex-col items-center justify-center text-ink-4">
            <ScanBarcode size={32} className="mb-3 opacity-40" />
            <p className="text-sm">Select an order to start picking</p>
          </div>
        )}

        {!session && !ps.sessionLoading && error && <SessionErrorPanel error={error} onBack={ps.backToOrders} />}

        {(activeOrder || ps.urlSessionId) && ps.sessionLoading && !session && (
          <div className="flex-1 flex items-center justify-center text-ink-4 text-sm">Loading…</div>
        )}

        {activeOrder && ps.completeResult && (
          <PickingCompleteResult order={activeOrder} result={ps.completeResult} onContinueToPacking={ps.continueToPacking} onBack={ps.backToOrders} />
        )}

        {ps.stockShortfalls && (
          <StockShortfallDialog
            shortfalls={ps.stockShortfalls}
            submitting={ps.completing}
            onCancel={() => ps.setStockShortfalls(null)}
            onConfirm={resolutions => ps.handleConfirmComplete(resolutions)}
          />
        )}

        {activeOrder && session && ps.confirmingFinish && !ps.completeResult && !ps.stockShortfalls && (
          <PickingReviewStep
            order={activeOrder} session={session} error={error} completing={ps.completing}
            onConfirm={() => ps.handleConfirmComplete()} onBack={() => ps.setConfirmingFinish(false)}
          />
        )}

        {activeOrder && session && !ps.completeResult && !ps.confirmingFinish && (
          <>
            <PickingSessionHeader
              order={activeOrder} pausing={ps.pausing} readOnly={ps.readOnly}
              onPause={ps.handlePause} onStartOver={() => ps.setResetConfirmOpen(true)} onComplete={() => ps.setConfirmingFinish(true)}
            />

            {ps.heldByOther && (
              <PickingLockBanner
                holderName={nameFor(ps.heldByOther)} lockedSince={session.session.locked_since}
                canTakeOver={canTakeOverLocks(user?.role)} onTakeOver={ps.handleTakeOver}
              />
            )}

            <PickScanInput
              inputRef={entry.codeRef} code={entry.code} onCodeChange={entry.setCode} onSubmit={entry.handleScanSubmit}
              disabled={entry.busy || ps.readOnly || !!sheet.pickLine || !!entry.scanErrorAlert} error={error}
            />

            <PickProgressBar progress={progress} />

            <PickLineList
              token={token ?? ''} lineItems={session.line_items} pickedByLineItem={progress.pickedByLineItem}
              readOnly={ps.readOnly} onOpenLine={sheet.open}
            />

            <PickScanLog scans={session.scans} onDelete={entry.handleDeleteScan} />
          </>
        )}
      </div>

      {sheet.pickLine && (
        <PickLineSheet
          pickLine={sheet.pickLine} pickQty={sheet.pickQty} onPickQtyChange={sheet.setPickQty}
          onConfirm={entry.confirmPick} onClose={sheet.close} busy={entry.busy}
        />
      )}

      {entry.scanErrorAlert && <ScanErrorDialog message={entry.scanErrorAlert} onDismiss={entry.dismissScanError} />}

      {ps.resetConfirmOpen && (
        <StartOverDialog resetting={ps.resetting} onCancel={() => ps.setResetConfirmOpen(false)} onConfirm={ps.handleResetSession} />
      )}

    </div>
  )
}
