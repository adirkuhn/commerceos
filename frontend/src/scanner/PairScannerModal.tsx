import { useEffect, useRef, useState } from 'react'
import { X, QrCode, Barcode as BarcodeIcon, Hash, Check } from 'lucide-react'
import { QRCodeSVG } from 'qrcode.react'
import { useTranslation } from 'react-i18next'
import JsBarcode from 'jsbarcode'
import * as api from './api'
import type { ScannerPairingCode } from './types'
import { Dialog } from '../components/common/Dialog'
import { apiErrorText } from '../api/apiErrorText'

interface PairScannerModalProps {
  token: string
  onClose: () => void
}

type PairingTab = 'digits' | 'qr' | 'barcode'

const TABS: { id: PairingTab; icon: typeof QrCode }[] = [
  { id: 'qr', icon: QrCode },
  { id: 'barcode', icon: BarcodeIcon },
  { id: 'digits', icon: Hash },
]

// Renders pairing.token as a Code128 barcode for a handheld's hardware
// scan-engine to read -- jsbarcode draws imperatively onto the SVG node, no
// React-managed children.
function BarcodeSVG({ value }: { value: string }) {
  const ref = useRef<SVGSVGElement>(null)
  const { t } = useTranslation('scanner')
  useEffect(() => {
    if (ref.current) {
      JsBarcode(ref.current, value, { format: 'CODE128', displayValue: false, margin: 0, height: 64 })
    }
  }, [value])
  // JsBarcode sets an explicit pixel width attribute (with a matching
  // viewBox) sized to the encoded content -- for this ~22-char token that's
  // 500px+, wider than the modal card, so it overflows unless CSS caps it.
  // A fixed px max-width (not a %) is deterministic here: the wrapping div
  // shrink-to-fits inside a `items-center` flex parent, so a percentage
  // basis would be ambiguous; height:auto then scales via the SVG's
  // viewBox to keep the aspect ratio correct.
  return <svg ref={ref} className="jsbarcode h-auto" style={{ maxWidth: 260 }} aria-label={t('pair.barcodeLabel')} />
}

// Desktop side of pairing: generate a code, short-poll until the handheld
// claims it. Three equivalent claim paths render here -- digits (typed),
// QR and barcode (both encode the same high-entropy token, read by a
// handheld's camera/scan-engine, no manual typing, no brute-force
// exposure). See ../scanner/ClaimScreen.tsx for the handheld side.
export function PairScannerModal({ token, onClose }: PairScannerModalProps) {
  const { t } = useTranslation(['scanner', 'common'])
  const tabLabel = { qr: t('pair.qr'), barcode: t('pair.barcode'), digits: t('pair.digits') }
  const [pairing, setPairing] = useState<ScannerPairingCode | null>(null)
  const [error, setError] = useState('')
  const [tab, setTab] = useState<PairingTab>('qr')
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null)

  useEffect(() => {
    api.generatePairingCode(token)
      .then(setPairing)
      .catch(err => setError(apiErrorText(err) ?? t('pair.generateFailed')))
    return () => { if (pollRef.current) clearInterval(pollRef.current) }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    if (!pairing || pairing.status === 'claimed') return
    pollRef.current = setInterval(async () => {
      try {
        const updated = await api.getPairingCodeStatus(token, pairing.id)
        setPairing(updated)
      } catch {
        // Transient poll failures aren't worth surfacing -- the next tick retries.
      }
    }, 2000)
    return () => { if (pollRef.current) clearInterval(pollRef.current) }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pairing?.id, pairing?.status])

  return (
    <Dialog onClose={onClose} className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="w-full max-w-sm rounded-xl bg-surface border border-line shadow-xl">
        <div className="flex items-center justify-between px-5 py-4 border-b border-line">
          <h2 className="text-sm font-semibold text-ink">{t('pair.title')}</h2>
          <button aria-label={t('common:actions.close')} type="button" onClick={onClose} title={t('common:actions.close')}>
            <X size={16} className="text-ink-4" />
          </button>
        </div>

        {!error && pairing && pairing.status !== 'claimed' && (
          <div className="flex border-b border-line">
            {TABS.map(tb => (
              <button aria-pressed={tab === tb.id}
                key={tb.id}
                type="button"
                onClick={() => setTab(tb.id)}
                className={`flex-1 flex items-center justify-center gap-1.5 py-3 text-xs font-medium transition-colors ${
                  tab === tb.id
                    ? 'text-accent border-b-2 border-accent'
                    : 'text-ink-4'
                }`}
              >
                <tb.icon size={14} />
                {tabLabel[tb.id]}
              </button>
            ))}
          </div>
        )}

        <div className="p-6 flex flex-col items-center gap-4 text-center">
          {error && <p className="text-xs text-danger">{error}</p>}
          {!error && pairing?.status === 'claimed' && (
            <>
              <div className="w-14 h-14 rounded-2xl bg-green-500/15 flex items-center justify-center">
                <Check size={24} className="text-success" />
              </div>
              <p className="text-sm font-medium text-ink">{t('pair.paired')}</p>
            </>
          )}
          {!error && pairing && pairing.status !== 'claimed' && (
            <>
              {tab === 'digits' && (
                <>
                  <div className="w-14 h-14 rounded-2xl bg-accent/15 flex items-center justify-center">
                    <Hash size={24} className="text-accent" />
                  </div>
                  <p className="text-4xl font-bold tracking-[0.3em] text-ink">{pairing.code}</p>
                  <p className="text-xs text-ink-4">{t('pair.typeOnDevice')}</p>
                </>
              )}
              {tab === 'qr' && (
                <div className="p-3 bg-white rounded-lg">
                  <QRCodeSVG value={pairing.token} size={160} title={t('pair.scanToPair')} />
                </div>
              )}
              {tab === 'barcode' && (
                <div className="p-3 bg-white rounded-lg">
                  <BarcodeSVG value={pairing.token} />
                </div>
              )}
              {tab !== 'digits' && (
                <p className="text-xs text-ink-4">{tab === 'qr' ? t('pair.pointAtQr') : t('pair.pointAtBarcode')}</p>
              )}
              <p className="text-xs text-ink-4">{t('pair.expires')}</p>
              <p className="text-xs text-warning">{t('pair.waiting')}</p>
            </>
          )}
          {!error && !pairing && <p className="text-xs text-ink-4">{t('pair.generating')}</p>}
        </div>
      </div>
    </Dialog>
  )
}
