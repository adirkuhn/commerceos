import { useTranslation } from 'react-i18next'
import { useState } from 'react'
import { X } from 'lucide-react'
import { useAuth } from '../auth/AuthContext'
import * as api from './api'
import type { Connection } from './types'
import { Dialog } from '../components/common/Dialog'

// Edits a WhatsApp connection's base URL/instance name/API key in place --
// the only way to rotate a wrong/expired API key without disconnecting
// (which would reset test mode/AI parsing and rotate the webhook secret as
// a side effect, see backend's UpdateWhatsAppCredentialsUseCase doc
// comment). The API key is never shown here -- there's nothing to prefill
// it with (it never round-trips from the backend), so the field starts
// blank and is left blank to keep the currently stored key unchanged.
export function WhatsAppCredentialsModal({ connection, onClose, onSaved }: {
  connection: Connection
  onClose: () => void
  onSaved: (connection: Connection) => void
}) {
  const { token } = useAuth()
  const { t } = useTranslation(['integrations', 'common'])
  const [baseUrl, setBaseUrl] = useState(connection.store_url)
  const [instanceName, setInstanceName] = useState(connection.whatsapp_instance_name ?? '')
  const [apiKey, setApiKey] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  async function handleSave() {
    setSaving(true)
    setError('')
    try {
      const updated = await api.updateWhatsAppCredentials(token ?? '', connection.id, { baseUrl, instanceName, apiKey })
      onSaved(updated)
      onClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('saveFailed'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog onClose={onClose} className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={onClose} />
      <div className="relative w-full max-w-sm bg-raised rounded-xl shadow-2xl border border-line p-5">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-sm font-semibold text-ink">{t('credentials.title')}</h3>
          <button onClick={onClose} aria-label={t('common:actions.close')}>
            <X size={14} className="text-ink-4" />
          </button>
        </div>

        <div className="space-y-3 mb-4">
          <div>
            <label htmlFor="wa-base-url" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
              {t('credentials.baseUrl')}
            </label>
            <input
              id="wa-base-url"
              type="text"
              required
              value={baseUrl}
              onChange={e => setBaseUrl(e.target.value)}
              placeholder={'https://evolution.example.com'}
              className="w-full px-3 py-2 text-xs font-mono rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
            />
          </div>

          <div>
            <label htmlFor="wa-instance-name" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
              {t('whatsapp.instanceName')}
            </label>
            <input
              id="wa-instance-name"
              type="text"
              required
              value={instanceName}
              onChange={e => setInstanceName(e.target.value)}
              className="w-full px-3 py-2 text-xs font-mono rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
            />
          </div>

          <div>
            <label htmlFor="wa-api-key" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
              {t('whatsapp.apiKey')}
            </label>
            <input
              id="wa-api-key"
              type="text"
              autoFocus
              value={apiKey}
              onChange={e => setApiKey(e.target.value)}
              placeholder={t('credentials.keepKey')}
              className="w-full px-3 py-2 text-xs font-mono rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
            />
            <p className="mt-1.5 text-2xs text-ink-4">
              {t('credentials.keyHint')}
            </p>
          </div>
        </div>

        {error && <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5 mb-3">{error}</p>}

        <button
          onClick={handleSave}
          disabled={saving || baseUrl.trim() === '' || instanceName.trim() === ''}
          className="w-full py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
        >
          {saving ? t('common:actions.saving') : t('common:actions.save')}
        </button>
      </div>
    </Dialog>
  )
}
