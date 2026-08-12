import { useEffect, useRef, useState } from 'react'
import { BrowserMultiFormatReader } from '@zxing/browser'
import { X, Camera } from 'lucide-react'

interface CameraScannerProps {
  onDecode: (text: string) => void
  onClose: () => void
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
export function CameraScanner({ onDecode, onClose }: CameraScannerProps) {
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
        if (!cancelled) setError(err instanceof Error ? err.message : 'Camera unavailable')
      }
    }
    start()

    return () => {
      cancelled = true
      controls?.stop()
    }
  }, [onDecode])

  return (
    <div className="fixed inset-0 z-50 flex flex-col bg-black">
      <div className="flex items-center justify-between px-4 py-4 shrink-0">
        <p className="flex items-center gap-2 text-sm text-white">
          <Camera size={16} /> Point camera at code
        </p>
        <button onClick={onClose} title="Close">
          <X size={20} className="text-white" />
        </button>
      </div>
      {error && <p className="px-4 text-sm text-red-400">{error}</p>}
      <video ref={videoRef} className="flex-1 w-full object-cover" muted playsInline />
    </div>
  )
}
