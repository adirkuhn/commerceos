import { useTranslation } from 'react-i18next'
import { useState } from 'react'
import { X } from 'lucide-react'
import { useAuth } from '../auth/AuthContext'
import * as api from './api'
import type { Connection } from './types'
import { Dialog } from '../components/common/Dialog'

// The webhook receiver lives outside VITE_API_BASE_URL's per-domain
// convention -- see integrations/WooCommerceSyncSettingsPage.tsx's own
// webhookURL helper, same base path. WhatsApp's shared secret travels as a
// URL path segment (not a separate signature header like Woo's), see
// backend's providerWebhookIngestURLWithToken -- so the full URL below IS
// the secret, and it's view-once by design (only the encrypted form
// persists server-side, see SetWebhookSecretUseCase). There's no way to
// see the CURRENT one again, only to rotate and see the new one -- doing
// so breaks the old URL immediately, so the old value must be replaced in
// evolution-go's webhook settings right after.
function webhookURL(tenantId: string, secret: string): string {
  const apiBase = (import.meta.env.VITE_API_BASE_URL as string) ?? ''
  return `${apiBase}/webhooks/v1/${tenantId}/whatsapp/${secret}`
}

export function WhatsAppWebhookModal({ connection, tenantId, onClose }: {
  connection: Connection
  tenantId: string
  onClose: () => void
}) {
  const { token } = useAuth()
  const { t } = useTranslation(['integrations', 'common'])
  const [secret, setSecret] = useState('')
  const [generating, setGenerating] = useState(false)
  const [error, setError] = useState('')
  const [copied, setCopied] = useState(false)

  async function handleGenerate() {
    setGenerating(true)
    setError('')
    try {
      const result = await api.setWebhookSecret(token ?? '', connection.id)
      setSecret(result.webhook_secret)
    } catch (err) {
      setError(err instanceof Error ? err.message : t('whatsappWebhook.failed'))
    } finally {
      setGenerating(false)
    }
  }

  async function handleCopy(url: string) {
    try {
      await navigator.clipboard.writeText(url)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      // Clipboard access can be denied -- the value is already shown
      // selectable on screen, a convenience failure, not a blocker.
    }
  }

  const url = secret ? webhookURL(tenantId, secret) : ''

  return (
    <Dialog onClose={onClose} className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={onClose} />
      <div className="relative w-full max-w-md bg-raised rounded-xl shadow-2xl border border-line p-5">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-sm font-semibold text-ink">{t('whatsappWebhook.title')}</h3>
          <button onClick={onClose} aria-label={t('common:actions.close')}>
            <X size={14} className="text-ink-4" />
          </button>
        </div>

        <p className="text-xs text-ink-4 mb-4">
          {t('whatsappWebhook.intro')}
        </p>

        {url ? (
          <div className="mb-4">
            <label className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
              {t('whatsappWebhook.newUrl')}
            </label>
            <div className="flex items-center gap-2">
              <code className="flex-1 px-3 py-2 text-2xs rounded-lg bg-sunken border border-line text-ink-2 overflow-x-auto whitespace-nowrap">
                {url}
              </code>
              <button
                type="button"
                onClick={() => handleCopy(url)}
                className="px-2.5 py-2 rounded-lg bg-fill text-2xs text-ink-3 hover:text-accent transition-colors shrink-0"
              >
                {copied ? t('copied') : t('copy')}
              </button>
            </div>
          </div>
        ) : null}

        {error && <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5 mb-3">{error}</p>}

        <button
          onClick={handleGenerate}
          disabled={generating}
          className="w-full py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
        >
          {generating ? t('generating') : secret ? t('whatsappWebhook.regenerateAgain') : t('whatsappWebhook.regenerate')}
        </button>
      </div>
    </Dialog>
  )
}
