import { useTranslation } from 'react-i18next'
import { useState } from 'react'
import * as api from './api'
import ProviderWebhooksPanel from './ProviderWebhooksPanel'
import { apiErrorText } from '../api/apiErrorText'

// The webhook receiver itself lives at a URL outside VITE_API_BASE_URL's
// per-domain convention (see ../api.ts's base()) -- it's the single
// unauthenticated route, mounted at /webhooks/v1/{tenant_id}/{provider}
// on the API's root router, not under /integrations. Built here rather
// than exported from api.ts since it's pure display, no request involved.
function webhookURL(tenantId: string, provider: string): string {
  const apiBase = (import.meta.env.VITE_API_BASE_URL as string) ?? ''
  return `${apiBase}/webhooks/v1/${tenantId}/${provider}`
}

export function WebhooksTab({ token, connectionId, tenantId, provider }: { token: string; connectionId: string; tenantId: string; provider: string }) {
  const [secret, setSecret] = useState('')
  const { t } = useTranslation(['integrations', 'common'])
  const [generating, setGenerating] = useState(false)
  const [error, setError] = useState('')
  const [copiedField, setCopiedField] = useState<'url' | 'secret' | null>(null)

  async function handleGenerate() {
    setError('')
    setGenerating(true)
    try {
      const result = await api.setWebhookSecret(token, connectionId)
      setSecret(result.webhook_secret)
    } catch (err) {
      setError(apiErrorText(err) ?? t('wooWebhooks.failed'))
    } finally {
      setGenerating(false)
    }
  }

  async function handleCopy(field: 'url' | 'secret', value: string) {
    try {
      await navigator.clipboard.writeText(value)
      setCopiedField(field)
      setTimeout(() => setCopiedField(prev => (prev === field ? null : prev)), 1500)
    } catch {
      // Clipboard access can be denied (permissions, insecure context) --
      // the value is already shown selectable on screen, so this is a
      // convenience failure, not a blocker; no error surfaced for it.
    }
  }

  return (
    <div className="flex-1 overflow-y-auto p-6">
      <div className="max-w-3xl space-y-4">
        <div>
          <h2 className="text-sm font-semibold text-ink mb-1">{t('wooWebhooks.title')}</h2>
          <p className="text-xs text-ink-4">
            {t('wooWebhooks.intro')}
          </p>
        </div>

        <div className="bg-raised rounded-xl border border-line p-4 space-y-3">
          <div>
            <label className="block text-2xs text-ink-4 caps-label mb-1.5">
              {t('deliveryUrl')}
            </label>
            <div className="flex items-center gap-2">
              <code className="flex-1 px-3 py-2 text-2xs rounded-lg bg-sunken border border-line text-ink-2 overflow-x-auto whitespace-nowrap">
                {webhookURL(tenantId, provider)}
              </code>
              <button
                type="button"
                onClick={() => handleCopy('url', webhookURL(tenantId, provider))}
                className="px-2.5 py-2 rounded-lg bg-fill text-2xs text-ink-3 hover:text-accent transition-colors shrink-0"
              >
                {copiedField === 'url' ? t('copied') : t('copy')}
              </button>
            </div>
          </div>

          {secret ? (
            <div>
              <label className="block text-2xs text-ink-4 caps-label mb-1.5">
                {t('wooWebhooks.secretOnce')}
              </label>
              <div className="flex items-center gap-2">
                <code className="flex-1 px-3 py-2 text-2xs font-mono rounded-lg bg-sunken border border-line text-ink-2 overflow-x-auto whitespace-nowrap">
                  {secret}
                </code>
                <button
                  type="button"
                  onClick={() => handleCopy('secret', secret)}
                  className="px-2.5 py-2 rounded-lg bg-fill text-2xs text-ink-3 hover:text-accent transition-colors shrink-0"
                >
                  {copiedField === 'secret' ? t('copied') : t('copy')}
                </button>
              </div>
              <p className="mt-1.5 text-2xs text-warning">
                {t('wooWebhooks.notShownAgain')}
              </p>
            </div>
          ) : (
            <p className="text-xs text-ink-4">{t('wooWebhooks.noSecret')}</p>
          )}

          {error && <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{error}</p>}

          <button
            type="button"
            onClick={handleGenerate}
            disabled={generating}
            className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
          >
            {generating ? t('generating') : secret ? t('wooWebhooks.regenerate') : t('wooWebhooks.generate')}
          </button>
        </div>

        <ProviderWebhooksPanel token={token} connectionId={connectionId} />
      </div>
    </div>
  )
}
