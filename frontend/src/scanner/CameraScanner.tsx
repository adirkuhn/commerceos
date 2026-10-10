import { useEffect, useRef, useState } from 'react'
import { BrowserMultiFormatReader } from '@zxing/browser'
import { X, Camera } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { apiErrorText } from '../api/apiErrorText'

interface CameraScannerProps {
  onDecode: (text: string) => void
  onClose: () => void
  /** Viewfinder shape: 'square' for QR codes (default), 'rect' for 1D barcodes. Cosmetic only -- zxing decodes the full frame regardless. */
  frameShape?: 'square' | 'rect'
  /** 'fullscreen' (default) covers the whole screen -- used by the QR pairing flow, the app's entry screen. 'inline' renders as a bounded box in normal page flow instead, for scan entry points embedded in an existing screen (barcode lookups). */
  variant?: 'fullscreen' | 'inline'
}

interface ScannerControls {
  stop: () => void
}

// Reusable camera-based scanner for phones/tablets without a hardware
// scan-engine -- decodes QR codes and common 1D barcode formats (Code128,
// EAN, UPC...) from the live camera feed via zxing. Every scan input in the
// app that accepts a hardware-scanner keyboard-wedge also offers this as an
// alternative entry point, full-screen overlay so it works on small phone
// screens.
export function CameraScanner({ onDecode, onClose, frameShape = 'square', variant = 'fullscreen' }: CameraScannerProps) {
  const { t } = useTranslation(['scanner', 'common'])
  const videoRef = useRef<HTMLVideoElement>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false
    let controls: ScannerControls | undefined
    const reader = new BrowserMultiFormatReader()

    async function start() {
      try {
        controls = await reader.decodeFromVideoDevice(undefined, videoRef.current ?? undefined, result => {
          if (cancelled || !result) return
          cancelled = true
          controls?.stop()
          onDecode(result.getText())
        })
      } catch (err) {
        if (!cancelled) setError(apiErrorText(err) ?? t('camera.unavailable'))
      }
    }
    start()

    return () => {
      cancelled = true
      controls?.stop()
    }
  }, [onDecode])

  return (
    <div
      data-testid="camera-scanner"
      className={variant === 'fullscreen' ? 'fixed inset-0 z-50 flex flex-col bg-black' : 'relative w-full max-w-xs mx-auto rounded-xl overflow-hidden flex flex-col bg-black'}
    >
      <div className="flex items-center justify-between px-4 py-4 shrink-0">
        <p className="flex items-center gap-2 text-sm text-white">
          <Camera size={16} /> {t('camera.pointAtCode')}
        </p>
        <button aria-label={t('common:actions.close')} onClick={onClose} title={t('common:actions.close')}>
          <X size={20} className="text-white" />
        </button>
      </div>
      {error && <p className="px-4 text-sm text-red-400">{error}</p>}
      <div className={`relative w-full overflow-hidden ${variant === 'fullscreen' ? 'flex-1' : 'h-56 shrink-0'}`}>
        <video ref={videoRef} className="absolute inset-0 w-full h-full object-cover" muted playsInline />
        <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
          <div
            data-testid="scan-frame"
            data-shape={frameShape}
            className={`relative rounded-lg ${frameShape === 'square' ? 'w-60 h-60' : 'w-64 h-28'}`}
          >
            <span className="absolute -top-0.5 -left-0.5 w-6 h-6 border-t-2 border-l-2 border-accent rounded-tl-lg" />
            <span className="absolute -top-0.5 -right-0.5 w-6 h-6 border-t-2 border-r-2 border-accent rounded-tr-lg" />
            <span className="absolute -bottom-0.5 -left-0.5 w-6 h-6 border-b-2 border-l-2 border-accent rounded-bl-lg" />
            <span className="absolute -bottom-0.5 -right-0.5 w-6 h-6 border-b-2 border-r-2 border-accent rounded-br-lg" />
            <div
              data-testid="scan-line"
              className="absolute left-0 right-0 h-0.5 bg-accent shadow-[0_0_6px_2px_rgb(var(--accent)/0.8)] animate-scan-line"
            />
          </div>
        </div>
      </div>
    </div>
  )
}
