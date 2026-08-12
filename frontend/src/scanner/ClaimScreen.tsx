import { useState } from 'react'
import { Hash, ScanLine, Camera } from 'lucide-react'
import * as api from './api'
import { CameraScanner } from './CameraScanner'

interface ClaimScreenProps {
  onClaimed: (token: string) => void
}

// Scan is the default -- fastest path on a device with either a hardware
// scan-engine (keyboard-wedge into the input) or a camera (see cameraOpen
// below). Type code is the manual fallback.
type ClaimTab = 'digits' | 'scan'

export function ClaimScreen({ onClaimed }: ClaimScreenProps) {
  const [tab, setTab] = useState<ClaimTab>('scan')
  const [code, setCode] = useState('')
  const [scanValue, setScanValue] = useState('')
  const [cameraOpen, setCameraOpen] = useState(false)
  const [claiming, setClaiming] = useState(false)
  const [error, setError] = useState('')

  async function handleDigitsSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setClaiming(true)
    try {
      const { token } = await api.claimPairingCode(code.trim())
      onClaimed(token)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to pair')
      setClaiming(false)
    }
  }

  async function submitScannedToken(value: string) {
    setError('')
    setClaiming(true)
    try {
      const { token } = await api.claimPairingCodeByToken(value.trim())
      onClaimed(token)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to pair')
      setClaiming(false)
    }
  }

  async function handleScanSubmit(e: React.FormEvent) {
    e.preventDefault()
    await submitScannedToken(scanValue)
  }

  function handleCameraDecode(text: string) {
    setCameraOpen(false)
    setScanValue(text)
    submitScannedToken(text)
  }

  return (
    <div className="fixed inset-0 flex flex-col items-center justify-center gap-6 bg-[#0B1220] p-6 text-center">
      <div className="w-16 h-16 rounded-2xl bg-blue-600/20 flex items-center justify-center">
        {tab === 'digits' ? <Hash size={28} className="text-blue-400" /> : <ScanLine size={28} className="text-blue-400" />}
      </div>
      <div>
        <h1 className="text-lg font-semibold text-gray-100">Pair this scanner</h1>
        <p className="mt-1 text-sm text-gray-400">
          {tab === 'digits' ? 'Enter the code shown on the desktop screen' : 'Scan the QR code or barcode shown on the desktop screen'}
        </p>
      </div>

      <div className="flex w-full max-w-xs rounded-xl overflow-hidden border border-[#2D3748]">
        <button
          type="button"
          onClick={() => setTab('digits')}
          className={`flex-1 flex items-center justify-center gap-1.5 py-2.5 text-xs font-medium transition-colors ${
            tab === 'digits' ? 'bg-blue-600 text-white' : 'bg-[#1F2937] text-gray-400'
          }`}
        >
          <Hash size={13} /> Type code
        </button>
        <button
          type="button"
          onClick={() => setTab('scan')}
          className={`flex-1 flex items-center justify-center gap-1.5 py-2.5 text-xs font-medium transition-colors ${
            tab === 'scan' ? 'bg-blue-600 text-white' : 'bg-[#1F2937] text-gray-400'
          }`}
        >
          <ScanLine size={13} /> Scan
        </button>
      </div>

      {tab === 'digits' && (
        <form onSubmit={handleDigitsSubmit} className="w-full max-w-xs space-y-3">
          <input
            autoFocus
            inputMode="numeric"
            maxLength={6}
            value={code}
            onChange={e => setCode(e.target.value.replace(/\D/g, ''))}
            placeholder="000000"
            className="w-full px-4 py-4 text-center text-3xl tracking-[0.3em] rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500"
          />
          {error && <p className="text-sm text-red-400">{error}</p>}
          <button
            type="submit"
            disabled={claiming || code.length !== 6}
            className="w-full py-3 rounded-xl bg-blue-600 text-white font-semibold hover:bg-blue-700 disabled:opacity-50 transition-colors"
          >
            {claiming ? 'Pairing…' : 'Pair'}
          </button>
        </form>
      )}

      {tab === 'scan' && (
        <form onSubmit={handleScanSubmit} className="w-full max-w-xs space-y-3">
          {/* A hardware scan-engine keyboard-wedges the decoded QR/barcode
              value here followed by a terminator keystroke (Enter), which
              submits this form -- no camera/decode library needed on this
              side. The button stays as a touch-only fallback. */}
          <input
            autoFocus
            value={scanValue}
            onChange={e => setScanValue(e.target.value)}
            placeholder="Scan QR code or barcode…"
            className="w-full px-4 py-4 text-center text-sm rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-100 outline-none focus:ring-2 focus:ring-blue-500"
          />
          {error && <p className="text-sm text-red-400">{error}</p>}
          <button
            type="submit"
            disabled={claiming || scanValue.trim().length === 0}
            className="w-full py-3 rounded-xl bg-blue-600 text-white font-semibold hover:bg-blue-700 disabled:opacity-50 transition-colors"
          >
            {claiming ? 'Pairing…' : 'Pair'}
          </button>
          <button
            type="button"
            onClick={() => setCameraOpen(true)}
            className="w-full py-2.5 rounded-xl bg-[#1F2937] border border-[#2D3748] text-gray-300 text-xs font-medium flex items-center justify-center gap-1.5"
          >
            <Camera size={13} /> Use camera
          </button>
        </form>
      )}

      {cameraOpen && (
        <CameraScanner onDecode={handleCameraDecode} onClose={() => setCameraOpen(false)} />
      )}
    </div>
  )
}
