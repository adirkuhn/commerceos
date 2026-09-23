import { useEffect, useRef, useState } from 'react'
import { X, QrCode, Barcode as BarcodeIcon, Hash, Check } from 'lucide-react'
import { QRCodeSVG } from 'qrcode.react'
import JsBarcode from 'jsbarcode'
import * as api from './api'
import type { ScannerPairingCode } from './types'
import { Dialog } from '../components/common/Dialog'

interface PairScannerModalProps {
  token: string
  onClose: () => void
}

type PairingTab = 'digits' | 'qr' | 'barcode'

const TABS: { id: PairingTab; label: string; icon: typeof QrCode }[] = [
  { id: 'qr', label: 'QR Code', icon: QrCode },
  { id: 'barcode', label: 'Barcode', icon: BarcodeIcon },
  { id: 'digits', label: 'Digits', icon: Hash },
]

// Renders pairing.token as a Code128 barcode for a handheld's hardware
// scan-engine to read -- jsbarcode draws imperatively onto the SVG node, no
// React-managed children.
function BarcodeSVG({ value }: { value: string }) {
  const ref = useRef<SVGSVGElement>(null)
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
  return <svg ref={ref} className="jsbarcode h-auto" style={{ maxWidth: 260 }} aria-label="Pairing barcode" />
}

// Desktop side of pairing: generate a code, short-poll until the handheld
// claims it. Three equivalent claim paths render here -- digits (typed),
// QR and barcode (both encode the same high-entropy token, read by a
// handheld's camera/scan-engine, no manual typing, no brute-force
// exposure). See ../scanner/ClaimScreen.tsx for the handheld side.
export function PairScannerModal({ token, onClose }: PairScannerModalProps) {
  const [pairing, setPairing] = useState<ScannerPairingCode | null>(null)
  const [error, setError] = useState('')
  const [tab, setTab] = useState<PairingTab>('qr')
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null)

  useEffect(() => {
    api.generatePairingCode(token)
      .then(setPairing)
      .catch(err => setError(err instanceof Error ? err.message : 'Failed to generate code'))
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
          <h2 className="text-sm font-semibold text-ink">Pair a Scanner</h2>
          <button aria-label="Close" type="button" onClick={onClose} title="Close">
            <X size={16} className="text-ink-4" />
          </button>
        </div>

        {!error && pairing && pairing.status !== 'claimed' && (
          <div className="flex border-b border-line">
            {TABS.map(t => (
              <button aria-pressed={tab === t.id}
                key={t.id}
                type="button"
                onClick={() => setTab(t.id)}
                className={`flex-1 flex items-center justify-center gap-1.5 py-3 text-xs font-medium transition-colors ${
                  tab === t.id
                    ? 'text-accent border-b-2 border-accent'
                    : 'text-ink-4'
                }`}
              >
                <t.icon size={14} />
                {t.label}
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
              <p className="text-sm font-medium text-ink">Scanner paired</p>
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
                  <p className="text-xs text-ink-4">Type this code on the scanner device.</p>
                </>
              )}
              {tab === 'qr' && (
                <div className="p-3 bg-white rounded-lg">
                  <QRCodeSVG value={pairing.token} size={160} title="Scan to pair" />
                </div>
              )}
              {tab === 'barcode' && (
                <div className="p-3 bg-white rounded-lg">
                  <BarcodeSVG value={pairing.token} />
                </div>
              )}
              {tab !== 'digits' && (
                <p className="text-xs text-ink-4">Point the scanner's camera/lens at this {tab === 'qr' ? 'QR code' : 'barcode'}.</p>
              )}
              <p className="text-xs text-ink-4">Expires in 5 minutes.</p>
              <p className="text-xs text-warning">Waiting for scanner…</p>
            </>
          )}
          {!error && !pairing && <p className="text-xs text-ink-4">Generating code…</p>}
        </div>
      </div>
    </Dialog>
  )
}
