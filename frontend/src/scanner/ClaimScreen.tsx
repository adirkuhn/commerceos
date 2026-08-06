import { useState } from 'react'
import { QrCode } from 'lucide-react'
import * as api from './api'

interface ClaimScreenProps {
  onClaimed: (token: string) => void
}

export function ClaimScreen({ onClaimed }: ClaimScreenProps) {
  const [code, setCode] = useState('')
  const [claiming, setClaiming] = useState(false)
  const [error, setError] = useState('')

  async function handleSubmit(e: React.FormEvent) {
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

  return (
    <div className="fixed inset-0 flex flex-col items-center justify-center gap-6 bg-[#0B1220] p-6 text-center">
      <div className="w-16 h-16 rounded-2xl bg-blue-600/20 flex items-center justify-center">
        <QrCode size={28} className="text-blue-400" />
      </div>
      <div>
        <h1 className="text-lg font-semibold text-gray-100">Pair this scanner</h1>
        <p className="mt-1 text-sm text-gray-400">Enter the code shown on the desktop screen</p>
      </div>
      <form onSubmit={handleSubmit} className="w-full max-w-xs space-y-3">
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
    </div>
  )
}
