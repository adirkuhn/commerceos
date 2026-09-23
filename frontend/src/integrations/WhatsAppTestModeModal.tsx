import { useState } from 'react'
import { X } from 'lucide-react'
import { useAuth } from '../auth/AuthContext'
import * as api from './api'
import type { Connection } from './types'
import { Dialog } from '../components/common/Dialog'

// While enabled, every outbound WhatsApp send from this connection
// (inbound-message replies, order-status/shipment-delivery notifications,
// Message Center automation) is redirected to the test number instead of
// the real customer -- see backend's SetWhatsAppTestModeUseCase.
export function WhatsAppTestModeModal({ connection, onClose, onSaved }: {
  connection: Connection
  onClose: () => void
  onSaved: (connection: Connection) => void
}) {
  const { token } = useAuth()
  const [enabled, setEnabled] = useState(connection.whatsapp_test_mode_enabled ?? false)
  const [testPhone, setTestPhone] = useState(connection.whatsapp_test_phone_e164 ?? '')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  async function handleSave() {
    setSaving(true)
    setError('')
    try {
      const updated = await api.setWhatsAppTestMode(token ?? '', connection.id, enabled, testPhone)
      onSaved(updated)
      onClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog onClose={onClose} className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={onClose} />
      <div className="relative w-full max-w-sm bg-raised rounded-xl shadow-2xl border border-line p-5">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-sm font-semibold text-ink">WhatsApp test mode</h3>
          <button onClick={onClose} aria-label="Close">
            <X size={14} className="text-ink-4" />
          </button>
        </div>

        <p className="text-xs text-ink-4 mb-4">
          While enabled, every outbound WhatsApp message from this connection — replies, order/shipment
          notifications, automation rules — is redirected to the number below instead of the real customer.
          Nothing reaches a real customer until you turn this off.
        </p>

        <label className="flex items-center gap-2 mb-3 cursor-pointer">
          <input
            type="checkbox"
            checked={enabled}
            onChange={e => setEnabled(e.target.checked)}
            className="rounded"
          />
          <span className="text-xs text-ink-2">Enable test mode</span>
        </label>

        {enabled && (
          <div className="mb-4">
            <label htmlFor="wa-test-phone" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
              Test phone number
            </label>
            <input
              id="wa-test-phone"
              type="text"
              required
              autoFocus
              value={testPhone}
              onChange={e => setTestPhone(e.target.value)}
              placeholder="31612345678"
              className="w-full px-3 py-2 text-xs font-mono rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
            />
            <p className="mt-1.5 text-2xs text-ink-4">
              E.164, no leading '+' (e.g. 31612345678, not +31612345678).
            </p>
          </div>
        )}

        {error && <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5 mb-3">{error}</p>}

        <button
          onClick={handleSave}
          disabled={saving || (enabled && testPhone.trim() === '')}
          className="w-full py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
        >
          {saving ? 'Saving…' : 'Save'}
        </button>
      </div>
    </Dialog>
  )
}
