import { useState } from 'react'
import { Camera, type LucideIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { CameraScanner } from './CameraScanner'

interface ScanCodeScreenProps {
  icon: LucideIcon
  prompt: string
  placeholder?: string
  submitLabel?: string
  loadingLabel?: string
  loading: boolean
  error: string
  onSubmit: (code: string) => void | Promise<void>
}

// Shared scan-entry screen for mini-apps that resolve a single code to a
// result (Inventory lookup, Price Lookup): a text input a hardware
// scan-engine can keyboard-wedge into, plus a camera fallback for phones/
// tablets with no dedicated scan engine. Both paths converge on the same
// onSubmit.
export function ScanCodeScreen({
  icon: Icon, prompt, placeholder, submitLabel, loadingLabel,
  loading, error, onSubmit,
}: ScanCodeScreenProps) {
  const { t } = useTranslation('scanner')
  const [code, setCode] = useState('')
  const [cameraOpen, setCameraOpen] = useState(false)

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!code.trim()) return
    onSubmit(code.trim())
  }

  function handleCameraDecode(text: string) {
    setCameraOpen(false)
    setCode(text)
    onSubmit(text)
  }

  return (
    <div className="flex-1 flex flex-col p-6 text-center overflow-y-auto">
      {cameraOpen && (
        <div className="w-full shrink-0 mb-4">
          <CameraScanner onDecode={handleCameraDecode} onClose={() => setCameraOpen(false)} frameShape="rect" variant="inline" />
        </div>
      )}
      <div className="flex-1 flex flex-col items-center justify-center gap-6">
        <div className="w-16 h-16 rounded-2xl bg-accent/20 flex items-center justify-center">
          <Icon size={28} className="text-accent" />
        </div>
        <p className="text-sm text-gray-400">{prompt}</p>
        <form onSubmit={handleSubmit} className="w-full max-w-xs space-y-3">
          <input
            autoFocus
            value={code}
            onChange={e => setCode(e.target.value)}
            placeholder={placeholder ?? t('scanCode.placeholder')}
            className="w-full px-4 py-4 text-center text-sm rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus"
          />
          {error && <p className="text-sm text-red-400">{error}</p>}
          <button
            type="submit"
            disabled={loading || code.trim().length === 0}
            className="w-full py-3 rounded-xl bg-accent-solid text-white font-semibold hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
          >
            {loading ? (loadingLabel ?? t('scanCode.lookingUp')) : (submitLabel ?? t('scanCode.lookUp'))}
          </button>
          <button
            type="button"
            onClick={() => setCameraOpen(true)}
            className="w-full py-2.5 rounded-xl bg-night-800 border border-night-700 text-gray-300 text-xs font-medium flex items-center justify-center gap-1.5"
          >
            <Camera size={13} /> {t('shared.useCamera')}
          </button>
        </form>
      </div>
    </div>
  )
}
