import { useEffect, useRef, useState } from 'react'
import { X, QrCode, Check } from 'lucide-react'
import * as api from './api'
import type { ScannerPairingCode } from './types'

interface PairScannerModalProps {
  token: string
  onClose: () => void
}

// Desktop side of pairing: generate a code, short-poll until the handheld
// claims it. See ../scanner/ClaimScreen.tsx for the other end of this flow.
export function PairScannerModal({ token, onClose }: PairScannerModalProps) {
  const [pairing, setPairing] = useState<ScannerPairingCode | null>(null)
  const [error, setError] = useState('')
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
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="w-full max-w-sm rounded-xl dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 shadow-xl">
        <div className="flex items-center justify-between px-5 py-4 border-b dark:border-[#2D3748] border-gray-200">
          <h2 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Pair a Scanner</h2>
          <button type="button" onClick={onClose} title="Close">
            <X size={16} className="dark:text-gray-500 text-gray-400" />
          </button>
        </div>
        <div className="p-6 flex flex-col items-center gap-4 text-center">
          {error && <p className="text-xs text-red-400">{error}</p>}
          {!error && pairing?.status === 'claimed' && (
            <>
              <div className="w-14 h-14 rounded-2xl bg-green-500/15 flex items-center justify-center">
                <Check size={24} className="text-green-500" />
              </div>
              <p className="text-sm font-medium dark:text-gray-100 text-gray-900">Scanner paired</p>
            </>
          )}
          {!error && pairing && pairing.status !== 'claimed' && (
            <>
              <div className="w-14 h-14 rounded-2xl bg-blue-500/15 flex items-center justify-center">
                <QrCode size={24} className="text-blue-500" />
              </div>
              <p className="text-4xl font-bold tracking-[0.3em] dark:text-gray-100 text-gray-900">{pairing.code}</p>
              <p className="text-xs dark:text-gray-500 text-gray-400">
                Enter this code on the scanner device. Expires in 5 minutes.
              </p>
              <p className="text-xs text-amber-400">Waiting for scanner…</p>
            </>
          )}
          {!error && !pairing && <p className="text-xs dark:text-gray-500 text-gray-400">Generating code…</p>}
        </div>
      </div>
    </div>
  )
}
