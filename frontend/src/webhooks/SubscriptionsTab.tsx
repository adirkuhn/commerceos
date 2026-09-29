import { useTranslation } from 'react-i18next'
import { useState } from 'react'
import { useAllProviderWebhooks } from '../integrations/hooks/useAllProviderWebhooks'

// Screen 2: what's actually subscribed right now, across every connection
// -- the "give me visibility" ask. Read-only list + Remove; Create still
// lives on each connection's own settings page (WooCommerceSyncSettingsPage
// / MyParcelSyncSettingsPage's Webhooks tab), since creating one requires
// picking which connection it belongs to and that page already has that
// context, plus the webhook-secret generation step right next to it.
export function SubscriptionsTab({
  webhooks, loading, error, remove, onRegisterWebhook,
}: ReturnType<typeof useAllProviderWebhooks> & { onRegisterWebhook: () => void }) {
  const { t } = useTranslation(['webhooks', 'common'])
  const [busyKey, setBusyKey] = useState<string | null>(null)
  const [rowError, setRowError] = useState('')

  // A CommerceOS-registered webhook's delivery_url always hits our own
  // ingest route (see application/integrations.providerWebhookIngestURL) --
  // if none of the rows do, every subscription here points somewhere else
  // entirely (e.g. an ERP, a WordPress plugin) and CommerceOS itself isn't
  // receiving anything yet.
  const noneTargetCommerceOS = webhooks.length > 0 && webhooks.every(w => !w.delivery_url.includes('/webhooks/v1/'))

  async function handleRemove(connectionId: string, webhookId: string) {
    const key = `${connectionId}:${webhookId}`
    setRowError('')
    setBusyKey(key)
    try {
      await remove(connectionId, webhookId)
    } catch (err) {
      setRowError(err instanceof Error ? err.message : t('subscriptions.removeFailed'))
    } finally {
      setBusyKey(null)
    }
  }

  return (
    <div className="max-w-4xl">
      <div className="bg-raised rounded-xl border border-line p-4">
        <div className="flex items-start justify-between gap-3 mb-3">
          <div>
            <h2 className="text-sm font-semibold text-ink mb-1">{t('subscriptions.title')}</h2>
            <p className="text-xs text-ink-4">
              {t('subscriptions.intro')}
            </p>
          </div>
          <button
            type="button" onClick={onRegisterWebhook}
            className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover transition-colors shrink-0"
          >
            {t('subscriptions.register')}
          </button>
        </div>

        {noneTargetCommerceOS && (
          <p className="text-xs text-warning bg-amber-50 dark:bg-amber-500/10 rounded-lg px-3 py-2 mb-3">
            {t('subscriptions.noneForUs')}
          </p>
        )}

        {error && <p className="text-xs text-danger mb-2">{error}</p>}
        {rowError && <p className="text-xs text-danger mb-2">{rowError}</p>}

        {loading ? (
          <p className="text-xs text-ink-4">{t('common:status.loading')}</p>
        ) : webhooks.length === 0 ? (
          <p className="text-xs text-ink-4">{t('subscriptions.empty')}</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr className="text-left text-ink-4 border-b border-line">
                  <th className="py-2 pr-3 font-medium">{t('connection')}</th>
                  <th className="py-2 pr-3 font-medium">{t('topic')}</th>
                  <th className="py-2 pr-3 font-medium">{t('deliveryUrl')}</th>
                  <th className="py-2 pr-3 font-medium">{t('common:fields.status')}</th>
                  <th className="py-2 pr-3 font-medium"></th>
                </tr>
              </thead>
              <tbody>
                {webhooks.map(w => {
                  const key = `${w.connection_id}:${w.id}`
                  return (
                    <tr key={key} className="border-b border-line last:border-0">
                      <td className="py-2 pr-3 text-ink-2">
                        <span className="capitalize">{w.provider}</span>
                        {w.store_label && <span className="text-ink-4"> · {w.store_label}</span>}
                      </td>
                      <td className="py-2 pr-3 text-ink-3">{w.topic || t('any')}</td>
                      <td className="py-2 pr-3 text-ink-3 font-mono truncate max-w-[240px]" title={w.delivery_url}>{w.delivery_url}</td>
                      <td className="py-2 pr-3 text-ink-3">{w.status}</td>
                      <td className="py-2 pr-3">
                        <button
                          onClick={() => w.connection_id && handleRemove(w.connection_id, w.id)}
                          disabled={busyKey === key || !w.connection_id}
                          className="px-2 py-1 rounded bg-fill text-ink-3 hover:text-danger disabled:opacity-50 transition-colors"
                        >
                          {t('common:actions.remove')}
                        </button>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}
