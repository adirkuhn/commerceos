import { useState } from 'react'
import * as api from './api'
import ProviderWebhooksPanel from './ProviderWebhooksPanel'

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
      setError(err instanceof Error ? err.message : 'Failed to generate webhook secret')
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
          <h2 className="text-sm font-semibold text-ink mb-1">Webhook signature secret</h2>
          <p className="text-xs text-ink-4">
            Generate a secret, then configure it as this webhook's "Secret" when creating it under WooCommerce → Settings →
            Advanced → Webhooks. Regenerating replaces the old value immediately -- update WooCommerce's webhook config with
            the new one, or deliveries will start failing signature verification.
          </p>
        </div>

        <div className="bg-raised rounded-xl border border-line p-4 space-y-3">
          <div>
            <label className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
              Delivery URL
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
                {copiedField === 'url' ? 'Copied' : 'Copy'}
              </button>
            </div>
          </div>

          {secret ? (
            <div>
              <label className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                Secret (shown once -- copy it now)
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
                  {copiedField === 'secret' ? 'Copied' : 'Copy'}
                </button>
              </div>
              <p className="mt-1.5 text-2xs text-warning">
                This won't be shown again -- if you lose it, generate a new one and update WooCommerce.
              </p>
            </div>
          ) : (
            <p className="text-xs text-ink-4">No secret generated yet -- webhooks from this connection can't be verified until you generate one.</p>
          )}

          {error && <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{error}</p>}

          <button
            type="button"
            onClick={handleGenerate}
            disabled={generating}
            className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
          >
            {generating ? 'Generating…' : secret ? 'Regenerate secret' : 'Generate secret'}
          </button>
        </div>

        <ProviderWebhooksPanel token={token} connectionId={connectionId} />
      </div>
    </div>
  )
}
