import { useState, type RefObject } from 'react'
import { playErrorSound } from './errorSound'

// The scan box shared by the picking and packing screens: the typed code,
// the inline camera, and the blocking scan-error alert a mis-scan raises
// until the user acknowledges it.
export function useScanInput(codeInputRef: RefObject<HTMLInputElement>) {
  const [code, setCode] = useState('')
  const [cameraOpen, setCameraOpen] = useState(false)
  const [scanErrorAlert, setScanErrorAlert] = useState<string | null>(null)

  function showScanError(message: string) {
    playErrorSound()
    setScanErrorAlert(message)
  }

  function dismissScanError() {
    setScanErrorAlert(null)
    codeInputRef.current?.focus()
  }

  return { code, setCode, cameraOpen, setCameraOpen, scanErrorAlert, showScanError, dismissScanError }
}
