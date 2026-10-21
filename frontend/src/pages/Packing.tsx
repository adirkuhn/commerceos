import { Package } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import { SessionErrorPanel } from '../oms/SessionErrorPanel'
import { PackingOrderQueue } from '../packing/PackingOrderQueue'
import { PickListPrintView } from '../packing/PickListPrintView'
import { canTakeOverLocks } from '../packing/lockDisplay'
import { usePackingQueue } from '../oms/hooks/usePackingQueue'
import { useMemberNames } from '../team/hooks/useMemberNames'
import { useQueryString } from '../routing/useQueryState'
import * as packingApi from '../scanner/api'
import { StockShortfallDialog } from '../scanner/StockShortfallDialog'
import { LineChangePanel } from '../packing/LineChangePanel'
import { AddItemPanel } from '../packing/AddItemPanel'
import { usePackingSession } from '../packing/usePackingSession'
import { usePackActions } from '../packing/usePackActions'
import { usePackingFinish } from '../packing/usePackingFinish'
import { useShippingLabel } from '../packing/useShippingLabel'
import { packProgress } from '../packing/packProgress'
import { ShippingLabelButton } from '../packing/ShippingLabelButton'
import { ShippingLabelFlow } from '../packing/ShippingLabelFlow'
import { PackingDoneView } from '../packing/PackingDoneView'
import { PackingReviewView } from '../packing/PackingReviewView'
import { PackingSessionHeader } from '../packing/PackingSessionHeader'
import { PackingLockBanner } from '../packing/PackingLockBanner'
import { PackScanInput } from '../packing/PackScanInput'
import { PackProgressBar } from '../packing/PackProgressBar'
import { PackLineItemList } from '../packing/PackLineItemList'
import { PackScanLog } from '../packing/PackScanLog'
import { ScanErrorDialog, ResetSessionDialog } from '../packing/PackingDialogs'

// Each open session has its own URL (/packing/:sessionId) -- see
// usePackingSession. The page wires the session, its actions, finishing and
// the shipping label flow into the queue + pack area + side panels.
export default function Packing() {
  const { token, user } = useAuth()
  const { t } = useTranslation(['packing', 'common'])
  // ?pick=<id,id> shows the multi-order pick list. Opening pushes (Back
  // closes it, refresh keeps it); the Close button replaces, so closing
  // doesn't leave a Back step that reopens it.
  const [pickParam, setPickParam] = useQueryString('pick', { history: 'push' })
  const [, clearPickParam] = useQueryString('pick')
  const pickIds = pickParam ? pickParam.split(',').filter(Boolean) : []

  const queue = usePackingQueue(token ?? '')
  const nameFor = useMemberNames(token ?? '')
  const isAdmin = canTakeOverLocks(user?.role)

  // The hooks below reference each other's state; these run from event
  // handlers/effects only, after every hook has been called.
  const pack = usePackingSession({
    token, me: user?.user_id ?? '',
    onSelect: () => { finish.clearResult(); actions.closeLine() },
    onCleared: () => { finish.clear(); actions.closeLine(); label.clear() },
  })
  const { activeOrder, session, error, readOnly } = pack
  const actions = usePackActions({ token, session, loadSession: pack.loadSession, setError: pack.setError })
  const finish = usePackingFinish({
    token, session, loadSession: pack.loadSession, setError: pack.setError,
    closePanels: actions.closePanels, onCompleted: queue.refresh,
  })
  // Before packing is finished, making the label is working the order --
  // take the lock first. Once packing is complete there's no session left
  // to claim (completeResult is set but the loaded session still says
  // in_progress).
  const label = useShippingLabel({
    token, activeOrder, onError: pack.setError,
    beforeCreate: async () => (session && !finish.completeResult && session.session.status !== 'completed') ? pack.claim() : true,
  })

  function labelButton(className: string) {
    return (
      <ShippingLabelButton
        createdShipmentId={label.createdShipmentId} autoPrintEnabled={label.autoPrintEnabled} canCreate={!!label.shippingConnection}
        onOpen={label.openLabel} onCreate={label.createLabel} className={className}
      />
    )
  }

  const progress = packProgress(session)
  const { completeResult, confirmingFinish } = finish
  const packLine = actions.packLine

  if (token && pickIds.length > 0) {
    return <PickListPrintView token={token} orderIds={pickIds} onClose={() => clearPickParam(null)} />
  }

  return (
    <div className="flex h-full">
      <PackingOrderQueue
        entries={queue.entries} loading={queue.loading} error={queue.error}
        currentUserId={user?.user_id ?? ''} isAdmin={isAdmin} nameFor={nameFor}
        activeOrderId={activeOrder?.id ?? null} activeInProgress={session?.session.status === 'in_progress'}
        onSelect={pack.selectOrder} onOpenPickList={ids => setPickParam(ids.join(','))}
      />

      {/* Main pack area */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {!activeOrder && !((pack.urlSessionId || pack.orderToStart) && (pack.sessionLoading || error)) && (
          <div className="flex-1 flex flex-col items-center justify-center text-ink-4">
            <Package size={32} className="mb-3 opacity-40" />
            <p className="text-sm">{t('page.selectOrder')}</p>
          </div>
        )}

        {!session && !pack.sessionLoading && error && <SessionErrorPanel error={error} onBack={pack.backToOrders} />}

        {(activeOrder || pack.urlSessionId || pack.orderToStart) && pack.sessionLoading && !session && (
          <div className="flex-1 flex items-center justify-center text-ink-4 text-sm">{t('common:status.loading')}</div>
        )}

        {activeOrder && completeResult && (
          <PackingDoneView
            order={activeOrder} missingItems={completeResult.missing_items} onBack={pack.backToOrders}
            labelAction={labelButton('mt-5 px-4 py-2 rounded-lg dark:bg-night-800 bg-gray-100 text-ink-2 text-xs font-medium hover:bg-hover transition-colors')}
          />
        )}

        <ShippingLabelFlow token={token} activeOrder={activeOrder} label={label} />

        {activeOrder && session && confirmingFinish && !completeResult && (
          <PackingReviewView
            order={activeOrder} session={session} error={error} completing={finish.completing}
            labelAction={labelButton('w-full py-2.5 rounded-xl dark:bg-night-800 bg-gray-100 text-ink-2 text-sm font-medium hover:bg-hover transition-colors')}
            onConfirm={() => finish.complete()} onBack={finish.cancelFinish}
          />
        )}

        {activeOrder && session && !completeResult && !confirmingFinish && (
          <>
            <PackingSessionHeader
              order={activeOrder} readOnly={readOnly}
              showCreateLabel={!!label.shippingConnection && !label.createdShipmentId} onCreateLabel={label.createLabel}
              onAddItem={actions.openAddItem} onStartOver={finish.openResetConfirm} onComplete={finish.startFinish}
            />

            {pack.heldByOther && (
              <PackingLockBanner
                holderName={nameFor(pack.heldByOther)} lockedSince={session.session.locked_since}
                canTakeOver={isAdmin} onTakeOver={pack.takeOver}
              />
            )}

            <PackScanInput
              inputRef={actions.codeRef} code={actions.code} onChange={actions.setCode} onSubmit={actions.submitScan} error={error}
              disabled={actions.busy || readOnly || !!packLine || actions.addItemOpen || !!actions.scanErrorAlert}
            />

            <PackProgressBar packedItems={progress.packedItems} totalItems={progress.totalItems} percent={progress.percent} allPacked={progress.allPacked} />

            <PackLineItemList
              token={token ?? ''} sessionId={session.session.id} lineItems={session.line_items} packedByLineItem={progress.packedByLineItem}
              readOnly={readOnly} onOpenLine={actions.openLine}
            />

            <PackScanLog scans={session.scans} onDelete={actions.deleteScan} />
          </>
        )}
      </div>

      {packLine && (
        <LineChangePanel
          key={packLine.lineItem.id}
          lineItem={packLine.lineItem} coveredSoFar={packLine.packedSoFar} busy={actions.busy}
          scans={(session?.scans ?? []).filter(sc => sc.line_item_id === packLine.lineItem.id)}
          onDeleteScan={id => { actions.closeLine(); actions.deleteScan(id) }}
          onPack={actions.confirmPack} onSubstitute={actions.confirmSubstitute} onRefund={actions.confirmRefund} onClose={actions.closeLine}
        />
      )}

      {actions.addItemOpen && session && token && (
        <AddItemPanel
          busy={actions.busy}
          resolveCode={c => packingApi.resolvePackScanCode(token, session.session.id, c)}
          onAdd={actions.confirmAddItem} onClose={actions.closeAddItem}
        />
      )}

      {finish.stockShortfalls && (
        <StockShortfallDialog
          shortfalls={finish.stockShortfalls} submitting={finish.completing}
          onConfirm={resolutions => finish.complete(resolutions)} onCancel={finish.cancelShortfalls}
        />
      )}

      {actions.scanErrorAlert && <ScanErrorDialog message={actions.scanErrorAlert} onDismiss={actions.dismissScanError} />}

      {finish.resetConfirmOpen && (
        <ResetSessionDialog resetting={finish.resetting} onCancel={finish.closeResetConfirm} onConfirm={finish.resetSession} />
      )}
    </div>
  )
}
