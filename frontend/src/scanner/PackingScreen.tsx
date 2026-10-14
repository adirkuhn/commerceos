import { useState } from 'react'
import i18next from 'i18next'
import { useTranslation } from 'react-i18next'
import { ArrowLeft, Gift, Tag } from 'lucide-react'
import { canTakeOverLocks } from '../packing/lockDisplay'
import * as api from './api'
import { CameraScanner } from './CameraScanner'
import { computeMissingPackItems, packedQuantityByLineItem } from './packingProgress'
import type { PackingSessionDetail, PickingSessionLineItem } from './types'
import { LineChangeSheet } from './LineChangeSheet'
import { AddItemSheet } from './AddItemSheet'
import { StockShortfallDialog } from './StockShortfallDialog'
import type { SalesOrder } from '../oms/types'
import { useScanSession, type ScanSessionApi } from './useScanSession'
import { useSessionCompletion } from './useSessionCompletion'
import { useScanInput } from './useScanInput'
import { usePackingLabel } from './usePackingLabel'
import { PackingLabelButton, PackingLabelModals } from './PackingLabelModals'
import { PackingReviewScreen } from './PackingReviewScreen'
import { PackingDoneScreen } from './PackingDoneScreen'
import { PackingExtraItems } from './PackingExtraItems'
import { ScanSessionOrderBand, StartOverButton } from './ScanSessionOrderBand'
import { ScanSessionLockBanner } from './ScanSessionLockBanner'
import { ScanSessionScanForm } from './ScanSessionScanForm'
import { PackingLineList } from './PackingLineList'
import { ScanErrorDialog, StartOverDialog } from './ScanSessionDialogs'

interface PackingScreenProps {
  token: string
  sessionId: string
  order: SalesOrder
  onExit: () => void
  onAuthError: () => void
}

const PACKING_SESSION_API: ScanSessionApi<PackingSessionDetail> = {
  get: (token, id) => api.getPackingSession(token, id),
  touch: (token, id) => api.touchPackingSession(token, id),
  release: (token, id) => api.releasePackingSessionLock(token, id),
  takeOver: (token, id) => api.takeOverPackingSession(token, id),
  deleteScan: (token, id, scanId) => api.deletePackScan(token, id, scanId),
  reset: (token, id) => api.resetPackingSession(token, id),
  messages: {
    get loadFailed() { return i18next.t('scanner:errors.loadPackingSession') },
    get lockLost() { return i18next.t('scanner:errors.packingLockLost') },
    get resetFailed() { return i18next.t('scanner:errors.resetPackingSession') },
  },
}

export function PackingScreen({ token, sessionId, order, onExit, onAuthError }: PackingScreenProps) {
  const { t } = useTranslation(['scanner', 'common'])
  const session = useScanSession(token, sessionId, onAuthError, PACKING_SESSION_API)
  const { detail, error, setError, readOnly, codeInputRef, run } = session
  const completion = useSessionCompletion(
    resolutions => api.completePacking(token, sessionId, resolutions), onAuthError, setError,
  )
  const scanInput = useScanInput(codeInputRef)
  const { code, setCode, scanErrorAlert, showScanError } = scanInput
  const label = usePackingLabel({ token, sessionId, order, finished: !!completion.result, reload: session.load, onAuthError, setError })
  const [packLine, setPackLine] = useState<{ lineItem: PickingSessionLineItem; packedSoFar: number } | null>(null)
  const [addItemOpen, setAddItemOpen] = useState(false)

  function stage(input: Parameters<typeof api.scanPackItem>[2]): Promise<boolean> {
    return run(() => api.scanPackItem(token, sessionId, input), t('errors.scanFailed'))
  }

  // Unlike picking's two-step resolve-then-stage, scanPackItem resolves and
  // records in one call (see ScanPackItemUseCase) -- any failure (SKU not on
  // this order, quantity would exceed ordered, barcode unrecognized) must
  // block the same way picking's mis-scans do, not fall through silently.
  async function handleScan(e: React.FormEvent) {
    e.preventDefault()
    const scannedCode = code.trim()
    if (!scannedCode || scanErrorAlert) return
    setCode('')
    const ok = await stage({ code: scannedCode, quantity: 1 })
    if (!ok) showScanError(error || t('errors.notOnOrder'))
  }

  function handleCameraDecode(text: string) {
    scanInput.setCameraOpen(false)
    setCode(text)
    stage({ code: text, quantity: 1 }).then(ok => {
      setCode('')
      if (!ok) showScanError(t('errors.notOnOrder'))
    })
  }

  const labelButton = <PackingLabelButton label={label} />
  const labelModals = <PackingLabelModals token={token} order={order} label={label} />

  if (completion.confirmingFinish && detail) {
    const { stockShortfalls, finishing } = completion
    return (
      <PackingReviewScreen
        order={order} missing={computeMissingPackItems(detail.line_items, detail.scans)}
        error={error} finishing={finishing} labelButton={labelButton}
        onBack={() => completion.setConfirmingFinish(false)} onConfirm={() => completion.confirmComplete()}
      >
        {stockShortfalls && (
          <StockShortfallDialog
            shortfalls={stockShortfalls} submitting={finishing}
            onConfirm={resolutions => completion.confirmComplete(resolutions)} onCancel={() => completion.setStockShortfalls(null)}
          />
        )}
        {labelModals}
      </PackingReviewScreen>
    )
  }

  if (completion.result) {
    return (
      <PackingDoneScreen missingItems={completion.result.missing_items} labelButton={labelButton} onDone={onExit}>
        {labelModals}
      </PackingDoneScreen>
    )
  }

  const packedByLineItem = packedQuantityByLineItem(detail?.scans ?? [])
  const extras = (detail?.scans ?? []).filter(s => s.action_type === 'freebie' || s.action_type === 'new_item')

  return (
    <div className="fixed inset-0 flex flex-col bg-night-950 text-gray-100">
      <div className="flex items-center gap-3 px-4 py-4 border-b border-night-700 shrink-0">
        <button aria-label={t('shared.back')} onClick={onExit} className="w-11 h-11 -ml-3 -my-2 flex items-center justify-center rounded-lg shrink-0" title={t('shared.back')}>
          <ArrowLeft size={18} className="text-gray-400" />
        </button>
        <h1 className="flex-1 text-base font-semibold">{t('packScreen.title')}</h1>
        {label.myParcelConnection && !label.createdShipmentId && (
          <button aria-label={t('packScreen.createLabel')}
            type="button" onClick={label.createLabel} disabled={readOnly} title={t('packScreen.createLabel')}
            className="min-h-11 px-3 -my-2 flex items-center gap-1.5 rounded-lg bg-night-800 border border-night-700 text-sm font-medium text-gray-200 disabled:opacity-50"
          >
            <Tag size={16} /> {t('packScreen.label')}
          </button>
        )}
        <button
          onClick={() => completion.setConfirmingFinish(true)} disabled={readOnly}
          className="px-4 min-h-11 -my-2 rounded-lg bg-green-600 text-white text-sm font-semibold hover:bg-green-700 disabled:opacity-50"
        >
          {t('shared.finish')}
        </button>
      </div>

      <ScanSessionOrderBand order={order}>
        <StartOverButton onClick={() => session.setResetConfirmOpen(true)} disabled={readOnly} />
      </ScanSessionOrderBand>

      {session.heldByOther && (
        <ScanSessionLockBanner
          activity="packing" holderName={session.nameFor(session.heldByOther)} lockedSince={detail?.session.locked_since}
          canTakeOver={canTakeOverLocks(session.me?.role)} onTakeOver={session.takeOver}
        />
      )}

      {scanInput.cameraOpen && (
        <div className="px-4 pt-3 shrink-0">
          <CameraScanner onDecode={handleCameraDecode} onClose={() => scanInput.setCameraOpen(false)} frameShape="rect" variant="inline" />
        </div>
      )}

      <div className="flex-1 overflow-y-auto px-4 py-3 space-y-2">
        {session.loading && <p className="text-center text-sm text-gray-400 py-8">{t('common:status.loading')}</p>}
        {detail && (
          <PackingLineList
            token={token} sessionId={sessionId} lineItems={detail.line_items} packedByLineItem={packedByLineItem} readOnly={readOnly}
            onOpenLine={(lineItem, packedSoFar) => setPackLine({ lineItem, packedSoFar })}
          />
        )}

        <PackingExtraItems extras={extras} onRemove={session.deleteScan} />

        <button
          type="button"
          onClick={() => setAddItemOpen(true)} disabled={readOnly}
          className="w-full flex items-center justify-center gap-1.5 rounded-xl border border-dashed border-night-700 px-3.5 py-3 text-sm text-accent hover:border-accent transition-colors"
        >
          <Gift size={14} /> {t('packScreen.addExtra')}
        </button>
      </div>

      <ScanSessionScanForm
        inputRef={codeInputRef} code={code} onCodeChange={setCode} onSubmit={handleScan}
        onOpenCamera={() => scanInput.setCameraOpen(true)}
        error={error} scanning={session.scanning} readOnly={readOnly} blocked={!!scanErrorAlert}
      />

      {packLine && (
        <LineChangeSheet
          token={token}
          lineItem={packLine.lineItem}
          packedSoFar={packLine.packedSoFar}
          scans={(detail?.scans ?? []).filter(s => s.line_item_id === packLine.lineItem.id)}
          onDeleteScan={session.deleteScan}
          onCancel={() => setPackLine(null)}
          onPack={quantity => {
            setPackLine(null)
            stage({ sku_id: packLine.lineItem.sku_id ?? undefined, quantity })
          }}
          onSubstitute={lines => {
            setPackLine(null)
            run(() => api.substitutePackLine(token, sessionId, { line_item_id: packLine.lineItem.id, lines }), t('errors.substituteFailed'))
          }}
          onRefund={(quantity, reason) => {
            setPackLine(null)
            run(() => api.refundPackLine(token, sessionId, { line_item_id: packLine.lineItem.id, quantity, reason }), t('errors.refundFailed'))
          }}
        />
      )}

      {addItemOpen && (
        <AddItemSheet
          token={token}
          onCancel={() => setAddItemOpen(false)}
          onAdd={(skuId, actionType, quantity, priceMinorUnits, reason) => {
            setAddItemOpen(false)
            stage({ sku_id: skuId, action_type: actionType, quantity, price_minor_units: priceMinorUnits, reason })
          }}
        />
      )}

      {labelModals}

      {scanErrorAlert && <ScanErrorDialog message={scanErrorAlert} onDismiss={scanInput.dismissScanError} />}

      {session.resetConfirmOpen && (
        <StartOverDialog
          activity="packing" resetting={session.resetting}
          onCancel={() => session.setResetConfirmOpen(false)}
          onConfirm={() => session.resetSession(() => { setPackLine(null); setAddItemOpen(false) })}
        />
      )}
    </div>
  )
}
