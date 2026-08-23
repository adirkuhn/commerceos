import { useState } from 'react'
import { ArrowLeft } from 'lucide-react'
import { useAuth } from '../auth/AuthContext'
import * as api from './api'

// Bol's Retailer API uses client_credentials (client_id/client_secret
// pasted from Bol's own portal) -- unlike WooCommerce there's no OAuth
// redirect/callback flow, so connecting is just this one form. No
// Sync/Taxes/Categories tabs yet (v1 scope: connect + per-product overrides
// on the product edit page's Bol tab) -- add tabs here if that grows.
export default function BolSyncSettingsPage({ onClose, onConnected }: { onClose: () => void; onConnected: () => void }) {
  const { token } = useAuth()
  const [clientId, setClientId] = useState('')
  const [clientSecret, setClientSecret] = useState('')
  const [economicOperatorId, setEconomicOperatorId] = useState('')
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!token) return
    setError('')
    setSaving(true)
    try {
      await api.connectBol(token, { clientId, clientSecret, economicOperatorId })
      onConnected()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to connect')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      <div className="flex items-center gap-3 px-5 py-4 border-b dark:border-[#2D3748] border-gray-200">
        <button onClick={onClose} aria-label="Back" className="p-1.5 rounded-lg dark:hover:bg-[#2D3748] hover:bg-gray-100 transition-colors">
          <ArrowLeft size={18} className="dark:text-gray-400 text-gray-500" />
        </button>
        <h2 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Connect Bol.com</h2>
      </div>
      <div className="flex-1 overflow-y-auto p-5 max-w-lg">
        <form onSubmit={handleSubmit} className="space-y-4">
          <p className="text-xs dark:text-gray-500 text-gray-400">
            Paste the credentials from your Bol.com Retailer API portal. No redirect step -- Bol authenticates via client_credentials.
          </p>
          <div>
            <label htmlFor="bol-client-id" className="block text-xs font-medium dark:text-gray-400 text-gray-500 mb-1">Client ID</label>
            <input
              id="bol-client-id"
              value={clientId}
              onChange={e => setClientId(e.target.value)}
              required
              className="w-full px-3 py-2 rounded-lg text-sm dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-100 text-gray-900"
            />
          </div>
          <div>
            <label htmlFor="bol-client-secret" className="block text-xs font-medium dark:text-gray-400 text-gray-500 mb-1">Client Secret</label>
            <input
              id="bol-client-secret"
              type="password"
              value={clientSecret}
              onChange={e => setClientSecret(e.target.value)}
              required
              className="w-full px-3 py-2 rounded-lg text-sm dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-100 text-gray-900"
            />
          </div>
          <div>
            <label htmlFor="bol-economic-operator-id" className="block text-xs font-medium dark:text-gray-400 text-gray-500 mb-1">Economic Operator ID</label>
            <input
              id="bol-economic-operator-id"
              value={economicOperatorId}
              onChange={e => setEconomicOperatorId(e.target.value)}
              required
              className="w-full px-3 py-2 rounded-lg text-sm dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-100 text-gray-900"
            />
            <p className="text-[10px] dark:text-gray-500 text-gray-400 mt-1">Your EU operator identifier, required on every offer.</p>
          </div>
          {error && <p className="text-xs text-red-400">{error}</p>}
          <button
            type="submit"
            disabled={saving}
            className="px-4 py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors"
          >
            {saving ? 'Connecting…' : 'Connect'}
          </button>
        </form>
      </div>
    </div>
  )
}
