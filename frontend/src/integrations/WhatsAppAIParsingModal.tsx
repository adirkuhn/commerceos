import { useTranslation } from 'react-i18next'
import { useState } from 'react'
import { X } from 'lucide-react'
import { useAuth } from '../auth/AuthContext'
import { useAIConnections } from './hooks/useAIConnections'
import * as api from './api'
import type { Connection } from './types'
import { Dialog } from '../components/common/Dialog'
import { apiErrorText } from '../api/apiErrorText'

// AI-assisted parsing: when a customer's message doesn't match the rigid
// "STOCK <code>"/"PRICE <code>"/etc format, the tenant's connected AI
// provider classifies it (strict, closed-schema output only -- never a
// free-form chat reply) and extracts the product they meant. Requires an
// AI connection (Settings -> Integrations -> AI) to already exist.
export function WhatsAppAIParsingModal({ connection, onClose, onSaved }: {
  connection: Connection
  onClose: () => void
  onSaved: (connection: Connection) => void
}) {
  const { token } = useAuth()
  const { t } = useTranslation(['integrations', 'common'])
  const { connections: aiConnections, loading: loadingAIConnection } = useAIConnections(token ?? '')
  const [enabled, setEnabled] = useState(connection.whatsapp_ai_parsing_enabled ?? false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  // Any enabled AI connection will do: the router tries them in order.
  const aiConfigured = aiConnections.some(c => c.enabled)

  async function handleSave() {
    setSaving(true)
    setError('')
    try {
      const updated = await api.setWhatsAppAIParsing(token ?? '', connection.id, enabled)
      onSaved(updated)
      onClose()
    } catch (err) {
      setError(apiErrorText(err) ?? t('saveFailed'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog onClose={onClose} className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={onClose} />
      <div className="relative w-full max-w-sm bg-raised rounded-xl shadow-2xl border border-line p-5">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-sm font-semibold text-ink">{t('aiParsing.title')}</h3>
          <button onClick={onClose} aria-label={t('common:actions.close')}>
            <X size={14} className="text-ink-4" />
          </button>
        </div>

        <p className="text-xs text-ink-4 mb-4">
          {t('aiParsing.intro')}
        </p>

        {!loadingAIConnection && !aiConfigured && (
          <p className="text-xs text-warning bg-amber-500/10 rounded-lg px-2 py-1.5 mb-3">
            {t('aiParsing.noProvider')}
          </p>
        )}

        <label className={`flex items-center gap-2 mb-4 ${aiConfigured ? 'cursor-pointer' : 'cursor-not-allowed opacity-50'}`}>
          <input
            type="checkbox"
            checked={enabled}
            disabled={!aiConfigured}
            onChange={e => setEnabled(e.target.checked)}
            className="rounded"
          />
          <span className="text-xs text-ink-2">{t('aiParsing.enable')}</span>
        </label>

        {error && <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5 mb-3">{error}</p>}

        <button
          onClick={handleSave}
          disabled={saving || (enabled && !aiConfigured)}
          className="w-full py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
        >
          {saving ? t('common:actions.saving') : t('common:actions.save')}
        </button>
      </div>
    </Dialog>
  )
}
