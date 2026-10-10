import { useState } from 'react'
import { Hash, ScanLine, Camera } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import * as api from './api'
import { CameraScanner } from './CameraScanner'
import { hasLoginUrl, APP_ID, lm2Origin } from '../auth/loginConfig'
import { apiErrorText } from '../api/apiErrorText'

interface ClaimScreenProps {
  onClaimed: (token: string) => void
}

const SCANNER_CALLBACK_PATH = '/scanner/auth/callback'

// Scan is the default -- fastest path on a device with either a hardware
// scan-engine (keyboard-wedge into the input) or a camera (see cameraOpen
// below). Type code is the manual fallback.
type ClaimTab = 'digits' | 'scan'

export function ClaimScreen({ onClaimed }: ClaimScreenProps) {
  const { t } = useTranslation('scanner')
  const [tab, setTab] = useState<ClaimTab>('scan')
  const [code, setCode] = useState('')
  const [scanValue, setScanValue] = useState('')
  const [cameraOpen, setCameraOpen] = useState(false)
  const [claiming, setClaiming] = useState(false)
  const [error, setError] = useState('')
  const [googleLoading, setGoogleLoading] = useState(false)
  const [googleError, setGoogleError] = useState('')

  async function handleDigitsSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setClaiming(true)
    try {
      const { token } = await api.claimPairingCode(code.trim())
      onClaimed(token)
    } catch (err) {
      setError(apiErrorText(err) ?? t('claim.failed'))
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
      setError(apiErrorText(err) ?? t('claim.failed'))
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

  // Alternative to pairing off a desktop admin session -- signs the device
  // itself in via lm2's Google OAuth, the same flow ../auth/LoginPage uses.
  // Full-page redirect, so there's nothing to await here: the browser leaves
  // this screen and comes back at ScannerCallbackPage once Google (and lm2)
  // finish, which stores the resulting token+refresh_token and returns here.
  async function handleGoogleLogin() {
    setGoogleError('')
    setGoogleLoading(true)
    try {
      const callbackUrl = `${window.location.origin}${SCANNER_CALLBACK_PATH}`
      const res = await fetch(`${lm2Origin()}/api/v1/auth/app/google/url?app=${APP_ID}&redirect=${encodeURIComponent(callbackUrl)}`)
      const body = await res.json()
      if (!res.ok) throw new Error(body.error?.message ?? body.message ?? t('claim.googleUnavailable'))
      window.location.href = body.data.url
    } catch (err) {
      setGoogleError(apiErrorText(err) ?? t('claim.googleUnavailable'))
      setGoogleLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 flex flex-col items-center justify-center gap-6 bg-night-950 p-6 text-center">
      <div className="w-16 h-16 rounded-2xl bg-accent/20 flex items-center justify-center">
        {tab === 'digits' ? <Hash size={28} className="text-accent" /> : <ScanLine size={28} className="text-accent" />}
      </div>
      <div>
        <h1 className="text-lg font-semibold text-gray-100">{t('claim.title')}</h1>
        <p className="mt-1 text-sm text-gray-400">
          {tab === 'digits' ? t('claim.enterCode') : t('claim.scanCode')}
        </p>
      </div>

      <div className="flex w-full max-w-xs rounded-xl overflow-hidden border border-night-700">
        <button aria-pressed={tab === 'digits'}
          type="button"
          onClick={() => setTab('digits')}
          className={`flex-1 flex items-center justify-center gap-1.5 py-2.5 text-xs font-medium transition-colors ${
            tab === 'digits' ? 'bg-accent-solid text-white' : 'bg-night-800 text-gray-400'
          }`}
        >
          <Hash size={13} /> {t('claim.typeCode')}
        </button>
        <button aria-pressed={tab === 'scan'}
          type="button"
          onClick={() => setTab('scan')}
          className={`flex-1 flex items-center justify-center gap-1.5 py-2.5 text-xs font-medium transition-colors ${
            tab === 'scan' ? 'bg-accent-solid text-white' : 'bg-night-800 text-gray-400'
          }`}
        >
          <ScanLine size={13} /> {t('claim.scan')}
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
            className="w-full px-4 py-4 text-center text-3xl tracking-[0.3em] rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus"
          />
          {error && <p className="text-sm text-red-400">{error}</p>}
          <button
            type="submit"
            disabled={claiming || code.length !== 6}
            className="w-full py-3 rounded-xl bg-accent-solid text-white font-semibold hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
          >
            {claiming ? t('claim.pairing') : t('claim.pair')}
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
            placeholder={t('claim.scanPlaceholder')}
            className="w-full px-4 py-4 text-center text-sm rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus"
          />
          {error && <p className="text-sm text-red-400">{error}</p>}
          <button
            type="submit"
            disabled={claiming || scanValue.trim().length === 0}
            className="w-full py-3 rounded-xl bg-accent-solid text-white font-semibold hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
          >
            {claiming ? t('claim.pairing') : t('claim.pair')}
          </button>
          <button
            type="button"
            onClick={() => setCameraOpen(true)}
            className="w-full py-2.5 rounded-xl bg-night-800 border border-night-700 text-gray-300 text-xs font-medium flex items-center justify-center gap-1.5"
          >
            <Camera size={13} /> {t('shared.useCamera')}
          </button>
        </form>
      )}

      {cameraOpen && (
        <CameraScanner onDecode={handleCameraDecode} onClose={() => setCameraOpen(false)} />
      )}

      {hasLoginUrl(import.meta.env) && (
        <div className="w-full max-w-xs space-y-3">
          <div className="flex items-center gap-3 text-2xs text-gray-400">
            <div className="h-px flex-1 bg-night-700" />
            {t('claim.or')}
            <div className="h-px flex-1 bg-night-700" />
          </div>
          {googleError && <p className="text-sm text-red-400">{googleError}</p>}
          <button
            type="button"
            onClick={handleGoogleLogin}
            disabled={googleLoading}
            className="w-full py-3 rounded-xl bg-night-800 border border-night-700 text-gray-200 text-sm font-medium disabled:opacity-50 transition-colors"
          >
            {googleLoading ? t('claim.redirecting') : t('claim.google')}
          </button>
        </div>
      )}
    </div>
  )
}
